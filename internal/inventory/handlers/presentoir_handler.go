package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/lunetterie/backend/internal/inventory/repositories"
	"github.com/lunetterie/backend/internal/shared"
)

// PresentoirHandler gère les endpoints spécifiques au poste Présentoir
type PresentoirHandler struct {
	locationRepo *repositories.LocationRepository
}

// NewPresentoirHandler crée une nouvelle instance
func NewPresentoirHandler(locationRepo *repositories.LocationRepository) *PresentoirHandler {
	return &PresentoirHandler{locationRepo: locationRepo}
}

// EmptySlotsToday liste les emplacements présentoir d'une station libérés aujourd'hui suite à
// une vente ou une réserve, pour savoir quoi remplacer en fin de journée.
// GET /api/v1/inventory/presentoir/empty-slots?station_id=5
func (h *PresentoirHandler) EmptySlotsToday(c *gin.Context) {
	stationID, err := strconv.ParseInt(c.Query("station_id"), 10, 64)
	if err != nil {
		shared.BadRequest(c, "station_id requis")
		return
	}

	slots, err := h.locationRepo.FindEmptyPresentoirSlotsToday(stationID)
	if err != nil {
		shared.InternalError(c, "Impossible de récupérer les emplacements vides")
		return
	}

	shared.Success(c, http.StatusOK, gin.H{"slots": slots})
}
