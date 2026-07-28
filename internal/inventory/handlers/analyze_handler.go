package handlers

import (
	"io"

	"github.com/gin-gonic/gin"
	"github.com/lunetterie/backend/internal/inventory/services"
	"github.com/lunetterie/backend/internal/shared"
)

// AnalyzeHandler gère l'analyse IA à la volée (aperçu, sans écriture en base)
type AnalyzeHandler struct {
	aiService *services.AIService
}

// NewAnalyzeHandler crée une nouvelle instance
func NewAnalyzeHandler(aiService *services.AIService) *AnalyzeHandler {
	return &AnalyzeHandler{aiService: aiService}
}

// HandleAnalyze analyse une photo de monture et renvoie les caractéristiques détectées
// POST /api/v1/inventory/analyze
func (h *AnalyzeHandler) HandleAnalyze(c *gin.Context) {
	file, header, err := c.Request.FormFile("image")
	if err != nil {
		shared.BadRequest(c, "Image requise")
		return
	}
	defer file.Close()

	contentType := header.Header.Get("Content-Type")
	if contentType != "image/jpeg" && contentType != "image/png" && contentType != "image/webp" {
		shared.BadRequest(c, "Format d'image non supporté (JPEG, PNG ou WebP requis)")
		return
	}

	imageBytes, err := io.ReadAll(file)
	if err != nil {
		shared.BadRequest(c, "Erreur lecture image")
		return
	}

	result, err := h.aiService.Analyze(imageBytes, header.Filename, contentType)
	if err != nil {
		shared.InternalError(c, "Erreur lors de l'analyse: "+err.Error())
		return
	}

	shared.Success(c, 200, result)
}
