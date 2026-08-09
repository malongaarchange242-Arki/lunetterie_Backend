package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/lunetterie/backend/internal/inventory/models"
	"github.com/lunetterie/backend/internal/shared"
)

type savFollowupRepository interface {
	List() ([]models.SavFollowup, error)
	Save(proformaID int64, req models.SavFollowupSaveRequest, userID int64) (*models.SavFollowup, error)
}

type SavFollowupHandler struct {
	repo savFollowupRepository
}

func NewSavFollowupHandler(repo savFollowupRepository) *SavFollowupHandler {
	return &SavFollowupHandler{repo: repo}
}

// List renvoie tous les suivis SAV.
// GET /api/v1/inventory/sav/followups
func (h *SavFollowupHandler) List(c *gin.Context) {
	followups, err := h.repo.List()
	if err != nil {
		shared.InternalError(c, "Impossible de récupérer les suivis SAV")
		return
	}
	shared.Success(c, http.StatusOK, gin.H{"followups": followups})
}

// Save crée ou met à jour le suivi d'une proforma. Les champs absents du corps ne sont
// pas touchés : l'écran coche « appelé » sans réécrire les observations.
// PUT /api/v1/inventory/sav/followups/:proformaId
func (h *SavFollowupHandler) Save(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	proformaID, err := strconv.ParseInt(c.Param("proformaId"), 10, 64)
	if err != nil || proformaID <= 0 {
		shared.BadRequest(c, "proformaId invalide")
		return
	}

	var req models.SavFollowupSaveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.BadRequest(c, "Données invalides: "+err.Error())
		return
	}

	followup, saveErr := h.repo.Save(proformaID, req, userID)
	if saveErr != nil {
		shared.BadRequest(c, saveErr.Error())
		return
	}

	shared.Success(c, http.StatusOK, gin.H{"followup": followup})
}
