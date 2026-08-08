package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/lunetterie/backend/internal/inventory/dto"
	"github.com/lunetterie/backend/internal/inventory/repositories"
	"github.com/lunetterie/backend/internal/inventory/services"
	"github.com/lunetterie/backend/internal/shared"
)

// PresentoirHandler gère les endpoints spécifiques au poste Présentoir
type PresentoirHandler struct {
	locationRepo *repositories.LocationRepository
	displaySvc   *services.DisplayService
}

// NewPresentoirHandler crée une nouvelle instance
func NewPresentoirHandler(locationRepo *repositories.LocationRepository, displaySvc *services.DisplayService) *PresentoirHandler {
	return &PresentoirHandler{locationRepo: locationRepo, displaySvc: displaySvc}
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

// SendToCaisse envoie au comptoir d'encaissement les montures sélectionnées sur le présentoir
// (EN_PRESENTOIR -> EN_CAISSE, même station).
// POST /api/v1/inventory/presentoir/send-to-caisse
// body: { station_id, barcodes: [...] }
func (h *PresentoirHandler) SendToCaisse(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	var req dto.SendToCaisseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.BadRequest(c, "Données invalides: "+err.Error())
		return
	}

	sent, skipped, err := h.displaySvc.SendToCaisse(req.StationID, req.Barcodes, userID)
	if err != nil {
		// Les refus partent avec l'erreur : sans eux, l'UI ne peut pas dire POURQUOI rien
		// n'est parti.
		shared.BadRequestWithData(c, err.Error(), gin.H{"sent": sent, "skipped": skipped})
		return
	}

	shared.Success(c, http.StatusOK, gin.H{"sent": sent, "skipped": skipped})
}
