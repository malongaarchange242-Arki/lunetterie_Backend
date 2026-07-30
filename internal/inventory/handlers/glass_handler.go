package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/lunetterie/backend/internal/inventory/repositories"
	"github.com/lunetterie/backend/internal/inventory/services"
	"github.com/lunetterie/backend/internal/shared"
)

// GlassHandler gère la consultation des montures en stock
type GlassHandler struct {
	repo    *repositories.GlassRepository
	display *services.DisplayService
}

// NewGlassHandler crée une nouvelle instance
func NewGlassHandler(repo *repositories.GlassRepository, display *services.DisplayService) *GlassHandler {
	return &GlassHandler{repo: repo, display: display}
}

// ListGlasses liste les montures d'une station, filtrées par statut
// GET /api/v1/inventory/glasses?station_id=1&status=EN_STOCK_GENERAL,EN_STOCK_SOUS_STATION
func (h *GlassHandler) ListGlasses(c *gin.Context) {
	stationID, err := strconv.ParseInt(c.Query("station_id"), 10, 64)
	if err != nil {
		shared.BadRequest(c, "station_id requis")
		return
	}

	statuses := []string{"EN_STOCK_GENERAL", "EN_STOCK_SOUS_STATION"}
	if raw := c.Query("status"); raw != "" {
		statuses = strings.Split(raw, ",")
	}

	glasses, err := h.repo.FindByStationAndStatuses(stationID, statuses)
	if err != nil {
		shared.InternalError(c, "Impossible de récupérer les montures")
		return
	}

	shared.Success(c, http.StatusOK, gin.H{"glasses": glasses})
}

// GetGlassByBarcode recherche une monture par code-barres (toutes stations confondues).
// Si station_id est fourni et que la monture est à ce poste, sa recherche vaut confirmation de
// sa présence physique sur le présentoir : son statut et son emplacement sont mis à jour
// automatiquement (le code-barres, lui, ne change jamais).
// GET /api/v1/inventory/glasses/:barcode?station_id=1
func (h *GlassHandler) GetGlassByBarcode(c *gin.Context) {
	barcode := c.Param("barcode")

	if stationIDStr := c.Query("station_id"); stationIDStr != "" {
		stationID, err := strconv.ParseInt(stationIDStr, 10, 64)
		if err != nil {
			shared.BadRequest(c, "station_id invalide")
			return
		}
		userID, ok := currentUserID(c)
		if !ok {
			return
		}
		if err := h.display.PlaceOnDisplay(barcode, stationID, userID); err != nil {
			shared.NotFound(c, "Aucune monture ne correspond à ce code-barres")
			return
		}
	}

	glass, err := h.repo.FindDetailByBarcode(barcode)
	if err != nil {
		shared.NotFound(c, "Aucune monture ne correspond à ce code-barres")
		return
	}

	shared.Success(c, http.StatusOK, gin.H{"glass": glass})
}
