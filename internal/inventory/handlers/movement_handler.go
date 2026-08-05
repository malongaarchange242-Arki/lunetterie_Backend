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

// ListMovements liste l'historique des mouvements, filtrable par poste (départ/arrivée séparés),
// action, code-barres, utilisateur et période.
// GET /api/v1/inventory/movements?from_station_id=1&to_station_id=5&action=RESERVATION&barcode=LUN&user_id=3&date_from=2026-07-01&date_to=2026-07-31&limit=100&offset=0
func (h *MovementHandler) ListMovements(c *gin.Context) {
	filters := models.MovementFilters{}

	if raw := c.Query("from_station_id"); raw != "" {
		stationID, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			shared.BadRequest(c, "from_station_id invalide")
			return
		}
		filters.FromStationID = &stationID
	}
	if raw := c.Query("to_station_id"); raw != "" {
		stationID, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			shared.BadRequest(c, "to_station_id invalide")
			return
		}
		filters.ToStationID = &stationID
	}
	if raw := c.Query("action"); raw != "" {
		filters.Action = &raw
	}
	if raw := c.Query("barcode"); raw != "" {
		filters.Barcode = &raw
	}
	if raw := c.Query("user_id"); raw != "" {
		userID, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			shared.BadRequest(c, "user_id invalide")
			return
		}
		filters.UserID = &userID
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
