package handlers

import (
	"mime/multipart"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/lunetterie/backend/internal/inventory/dto"
	"github.com/lunetterie/backend/internal/shared"
	"github.com/lunetterie/backend/internal/workflows"
)

// ReceptionHandler gère les endpoints de réception
type ReceptionHandler struct {
	workflow *workflows.ReceptionWorkflow
}

// NewReceptionHandler crée une nouvelle instance
func NewReceptionHandler(workflow *workflows.ReceptionWorkflow) *ReceptionHandler {
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
	file, header, err := c.Request.FormFile("image")
	if err != nil {
		shared.BadRequest(c, "Image requise")
		return
	}
	defer file.Close()

	// Vérifier le type MIME
	contentType := header.Header.Get("Content-Type")
	if contentType != "image/jpeg" && contentType != "image/png" && contentType != "image/webp" {
		shared.BadRequest(c, "Format d'image non supporté (JPEG, PNG ou WebP requis)")
		return
	}

	// Récupérer la photo de la branche (optionnelle)
	var brancheFile multipart.File
	if bFile, _, err := c.Request.FormFile("branche_image"); err == nil {
		brancheFile = bFile
		defer brancheFile.Close()
	}

	// Exécuter le workflow
	result, err := h.workflow.Execute(
		req,
		file,
		brancheFile,
		userID,
	)
	if err != nil {
		shared.InternalError(c, "Erreur lors de la réception: "+err.Error())
		return
	}

	shared.Created(c, result)
}
