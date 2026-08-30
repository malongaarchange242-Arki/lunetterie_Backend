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
		"aucun emplacement libre",
		"aucun emplacement disponible",
		"carton de pre-enregistrement requis",
		"carton de pre-enregistrement introuvable",
		"capacite du carton",
		"capacit",
		"commande de r",
		"introuvable",
		"erreur allocation",
	}
	for _, item := range clientErrors {
		if strings.Contains(message, item) {
			return true
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
	// Récupérer l'utilisateur connecté (depuis JWT middleware)
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

	// Parser la requête multipart
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

	// Récupérer la photo de la monture
	montureFile, montureHeader, err := c.Request.FormFile("image")
	if err != nil {
		shared.BadRequest(c, "Image de monture requise")
		return
	}
	defer montureFile.Close()

	// Vérifier le type MIME de la monture
	montureContentType := montureHeader.Header.Get("Content-Type")
	if montureContentType != "image/jpeg" && montureContentType != "image/png" && montureContentType != "image/webp" {
		shared.BadRequest(c, "Format d'image monture non supporté (JPEG, PNG ou WebP requis)")
		return
	}

	// Récupérer la photo de la branche
	brancheFile, brancheHeader, err := c.Request.FormFile("branch_image")
	if err != nil {
		shared.BadRequest(c, "Image de branche requise")
		return
	}
	defer brancheFile.Close()

	brancheContentType := brancheHeader.Header.Get("Content-Type")
	if brancheContentType != "image/jpeg" && brancheContentType != "image/png" && brancheContentType != "image/webp" {
		shared.BadRequest(c, "Format d'image branche non supporté (JPEG, PNG ou WebP requis)")
		return
	}

	var arriereFile multipart.File
	var arriereHeader *multipart.FileHeader
	if formFile, header, err := c.Request.FormFile("rear_image"); err == nil {
		arriereFile = formFile
		arriereHeader = header
		defer arriereFile.Close()
		arriereContentType := arriereHeader.Header.Get("Content-Type")
		if arriereContentType != "image/jpeg" && arriereContentType != "image/png" && arriereContentType != "image/webp" {
			shared.BadRequest(c, "Format d'image arrière non supporté (JPEG, PNG ou WebP requis)")
			return
		}
	}

	// Exécuter le workflow
	result, err := h.workflow.Execute(
		req,
		montureFile,
		brancheFile,
		arriereFile,
		userID,
	)
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
