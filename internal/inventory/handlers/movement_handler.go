package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/lunetterie/backend/internal/inventory/models"
	"github.com/lunetterie/backend/internal/inventory/repositories"
	"github.com/lunetterie/backend/internal/shared"
)

// MovementHandler gère la consultation de l'historique des mouvements de montures
type MovementHandler struct {
	repo *repositories.MovementRepository
}

// NewMovementHandler crée une nouvelle instance
func NewMovementHandler(repo *repositories.MovementRepository) *MovementHandler {
	return &MovementHandler{repo: repo}
}

// ListMovements liste l'historique des mouvements, filtrable par poste/action/code-barres/période.
// GET /api/v1/inventory/movements?station_id=1&action=RESERVATION&barcode=LUN&date_from=2026-07-01&date_to=2026-07-31&limit=100&offset=0
func (h *MovementHandler) ListMovements(c *gin.Context) {
	filters := models.MovementFilters{}

	if raw := c.Query("station_id"); raw != "" {
		stationID, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			shared.BadRequest(c, "station_id invalide")
			return
		}
		filters.StationID = &stationID
	}
	if raw := c.Query("action"); raw != "" {
		filters.Action = &raw
	}
	if raw := c.Query("barcode"); raw != "" {
		filters.Barcode = &raw
	}
	if raw := c.Query("date_from"); raw != "" {
		filters.DateFrom = &raw
	}
	if raw := c.Query("date_to"); raw != "" {
		filters.DateTo = &raw
	}
	if raw := c.Query("limit"); raw != "" {
		if limit, err := strconv.Atoi(raw); err == nil {
			filters.Limit = limit
		}
	}
	if raw := c.Query("offset"); raw != "" {
		if offset, err := strconv.Atoi(raw); err == nil {
			filters.Offset = offset
		}
	}

	items, total, err := h.repo.List(filters)
	if err != nil {
		shared.InternalError(c, "Impossible de récupérer l'historique des mouvements")
		return
	}

	shared.Success(c, http.StatusOK, gin.H{"movements": items, "total": total})
}
