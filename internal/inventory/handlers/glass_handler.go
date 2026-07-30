package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/lunetterie/backend/internal/inventory/repositories"
	"github.com/lunetterie/backend/internal/shared"
)

// GlassHandler gère la consultation des montures en stock
type GlassHandler struct {
	repo *repositories.GlassRepository
}

// NewGlassHandler crée une nouvelle instance
func NewGlassHandler(repo *repositories.GlassRepository) *GlassHandler {
	return &GlassHandler{repo: repo}
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
