package handlers

import (
	"mime/multipart"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/lunetterie/backend/internal/inventory/dto"
	"github.com/lunetterie/backend/internal/shared"
)

type receptionExecutor interface {
	Execute(req dto.ReceptionRequest, montureImage multipart.File, brancheImage multipart.File, arriereImage multipart.File, userID int64) (*dto.ReceptionResponse, error)
}

func isReceptionClientError(err error) bool {
	message := strings.ToLower(err.Error())
	clientErrors := []string{
		"aucun carton disponible",
		"carton disponible",
		"capacit",
		"commande de r",
		"introuvable",
	}
	for _, item := range clientErrors {
		if strings.Contains(message, item) {
			return true // ✅ CORRIGÉ : return sur une ligne
		}
	}
	return false
}

// ReceptionHandler gère les endpoints de réception
type ReceptionHandler struct {
	workflow receptionExecutor
}

// NewReceptionHandler crée une nouvelle instance
func NewReceptionHandler(workflow receptionExecutor) *ReceptionHandler {
	return &ReceptionHandler{workflow: workflow}
}

// HandleReception gère la réception d'une nouvelle monture
// POST /api/v1/inventory/reception
func (h *ReceptionHandler) HandleReception(c *gin.Context) {
	userIDStr, exists := c.Get("user_id")
	if !exists {
		shared.Unauthorized(c, "Utilisateur non authentifié")
		return
	}

	userID, err := strconv.ParseInt(userIDStr.(string), 10, 64)
	if err != nil {
		shared.BadRequest(c, "ID utilisateur invalide")
		return
	}

	var req dto.ReceptionRequest
	if err := c.ShouldBindWith(&req, binding.FormMultipart); err != nil {
		shared.BadRequest(c, "Données invalides: "+err.Error())
		return
	}
	if req.ReceptionCommandCode == nil {
		if code := strings.TrimSpace(c.PostForm("reception_command_code")); code != "" {
			req.ReceptionCommandCode = &code
		}
	}

	isSupportedImage := func(contentType string) bool {
		return contentType == "image/jpeg" || contentType == "image/png" || contentType == "image/webp"
	}
	isBlank := func(value *string) bool {
		return value == nil || strings.TrimSpace(*value) == ""
	}

	var montureFile multipart.File
	var montureURL *string
	if formFile, montureHeader, err := c.Request.FormFile("image"); err == nil {
		montureFile = formFile
		defer montureFile.Close()
		if !isSupportedImage(montureHeader.Header.Get("Content-Type")) {
			shared.BadRequest(c, "Format d'image monture non supporté (JPEG, PNG ou WebP requis)")
			return
		}
	} else if isBlank(req.PhotoMontureURL) {
		shared.BadRequest(c, "Image de monture requise")
		return
	} else {
		montureURL = req.PhotoMontureURL
	}

	var brancheFile multipart.File
	if formFile, brancheHeader, err := c.Request.FormFile("branch_image"); err == nil {
		brancheFile = formFile
		defer brancheFile.Close()
		if !isSupportedImage(brancheHeader.Header.Get("Content-Type")) {
			shared.BadRequest(c, "Format d'image branche non supporté (JPEG, PNG ou WebP requis)")
			return
		}
	}

	var arriereFile multipart.File
	if formFile, arriereHeader, err := c.Request.FormFile("rear_image"); err == nil {
		arriereFile = formFile
		defer arriereFile.Close()
		if !isSupportedImage(arriereHeader.Header.Get("Content-Type")) {
			shared.BadRequest(c, "Format d'image arrière non supporté (JPEG, PNG ou WebP requis)")
			return
		}
	}

	if montureFile == nil && montureURL != nil {
		req.PhotoMontureURL = montureURL
	}
	result, err := h.workflow.Execute(req, montureFile, brancheFile, arriereFile, userID)
	if err != nil {
		message := "Erreur lors de la réception: " + err.Error()
		if isReceptionClientError(err) {
			shared.BadRequest(c, message)
			return
		}
		shared.InternalError(c, message)
		return
	}

	shared.Created(c, result)
}