package handlers

import (
	"io"
	"log"
	"net/http"

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
	// 1. Récupération du fichier
	file, header, err := c.Request.FormFile("image")
	if err != nil {
		log.Printf("Erreur récupération fichier: %v", err)
		shared.BadRequest(c, "Image requise")
		return
	}
	defer file.Close()

	// 2. Vérification du type MIME
	contentType := header.Header.Get("Content-Type")
	if contentType != "image/jpeg" && contentType != "image/png" && contentType != "image/webp" {
		log.Printf("Format non supporté: %s", contentType)
		shared.BadRequest(c, "Format d'image non supporté (JPEG, PNG ou WebP requis)")
		return
	}

	// 3. Lecture du fichier
	imageBytes, err := io.ReadAll(file)
	if err != nil {
		log.Printf("Erreur lecture image: %v", err)
		shared.BadRequest(c, "Erreur lecture image")
		return
	}

	// 4. Vérification que l'image n'est pas vide
	if len(imageBytes) == 0 {
		log.Printf("Image vide reçue")
		shared.BadRequest(c, "Image vide")
		return
	}

	// 5. Log des infos
	log.Printf("Analyse image - Taille: %d bytes, Type: %s, Nom: %s",
		len(imageBytes), contentType, header.Filename)

	// 6. Appel du service AI avec gestion d'erreur détaillée
	result, err := h.aiService.Analyze(imageBytes, header.Filename, contentType)
	if err != nil {
		// Log complet de l'erreur
		log.Printf("ERREUR AI Analyze: %v", err)

		// Message d'erreur plus spécifique
		errMsg := "Erreur lors de l'analyse de l'image"
		shared.InternalError(c, errMsg+": "+err.Error())
		return
	}

	// 7. Vérification que le résultat n'est pas nul
	if result == nil {
		log.Printf("Résultat AI null")
		shared.InternalError(c, "Erreur: résultat d'analyse vide")
		return
	}

	// 8. Succès
	log.Printf("Analyse réussie: %+v", result)
	shared.Success(c, http.StatusOK, result)
}

// HandleAnalyzeBranche analyse une photo de branche et renvoie la référence/marque lues par OCR
// POST /api/v1/inventory/analyze-branche
func (h *AnalyzeHandler) HandleAnalyzeBranche(c *gin.Context) {
	// 1. Récupération du fichier
	file, header, err := c.Request.FormFile("image")
	if err != nil {
		log.Printf("Erreur récupération fichier: %v", err)
		shared.BadRequest(c, "Image requise")
		return
	}
	defer file.Close()

	// 2. Vérification du type MIME
	contentType := header.Header.Get("Content-Type")
	if contentType != "image/jpeg" && contentType != "image/png" && contentType != "image/webp" {
		log.Printf("Format non supporté: %s", contentType)
		shared.BadRequest(c, "Format d'image non supporté (JPEG, PNG ou WebP requis)")
		return
	}

	// 3. Lecture du fichier
	imageBytes, err := io.ReadAll(file)
	if err != nil {
		log.Printf("Erreur lecture image: %v", err)
		shared.BadRequest(c, "Erreur lecture image")
		return
	}

	// 4. Vérification que l'image n'est pas vide
	if len(imageBytes) == 0 {
		log.Printf("Image vide reçue")
		shared.BadRequest(c, "Image vide")
		return
	}

	// 5. Log des infos
	log.Printf("Analyse branche - Taille: %d bytes, Type: %s, Nom: %s",
		len(imageBytes), contentType, header.Filename)

	// 6. Appel du service AI avec gestion d'erreur détaillée
	result, err := h.aiService.AnalyzeBranche(imageBytes, header.Filename, contentType)
	if err != nil {
		// Log complet de l'erreur
		log.Printf("ERREUR AI AnalyzeBranche: %v", err)

		// Message d'erreur plus spécifique
		errMsg := "Erreur lors de l'analyse de l'image de branche"
		shared.InternalError(c, errMsg+": "+err.Error())
		return
	}

	// 7. Vérification que le résultat n'est pas nul
	if result == nil {
		log.Printf("Résultat AI Branche null")
		shared.InternalError(c, "Erreur: résultat d'analyse vide")
		return
	}

	// 8. Succès
	log.Printf("Analyse branche réussie: %+v", result)
	shared.Success(c, http.StatusOK, result)
}
