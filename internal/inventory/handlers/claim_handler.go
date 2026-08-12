package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/lunetterie/backend/internal/inventory/models"
	"github.com/lunetterie/backend/internal/shared"
)

type claimRepository interface {
	Create(claim *models.Claim) error
	List(stationID int64, status string) ([]models.Claim, error)
}

type ClaimHandler struct {
	repo claimRepository
}

func NewClaimHandler(repo claimRepository) *ClaimHandler {
	return &ClaimHandler{repo: repo}
}

func (h *ClaimHandler) Create(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	var req models.ClaimCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.BadRequest(c, "Données invalides: "+err.Error())
		return
	}
	if req.ClientName == "" {
		shared.BadRequest(c, "Le nom du client est requis")
		return
	}
	if req.Motif == "" {
		shared.BadRequest(c, "Le motif est requis")
		return
	}

	claim := &models.Claim{
		StationID:  req.StationID,
		ClientName: req.ClientName,
		Barcode:    req.Barcode,
		Motif:      req.Motif,
		Detail:     nil,
		CreatedBy:  &userID,
	}
	if req.Detail != "" {
		claim.Detail = &req.Detail
	}

	if err := h.repo.Create(claim); err != nil {
		shared.BadRequest(c, err.Error())
		return
	}

	shared.Success(c, http.StatusCreated, gin.H{"claim": claim})
}

// List rend le pendant en lecture de Create, qui a longtemps manqué : la réclamation
// partait en base et n'en ressortait jamais, si bien que le suivi affichait zéro sur une
// table pleine. `station_id` et `status` sont facultatifs — le poste appelle sans filtre.
// GET /api/v1/inventory/claims
func (h *ClaimHandler) List(c *gin.Context) {
	var stationID int64
	if raw := strings.TrimSpace(c.Query("station_id")); raw != "" {
		parsed, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			shared.BadRequest(c, "Identifiant de station invalide")
			return
		}
		stationID = parsed
	}

	claims, err := h.repo.List(stationID, strings.TrimSpace(c.Query("status")))
	if err != nil {
		shared.InternalError(c, err.Error())
		return
	}

	shared.Success(c, http.StatusOK, gin.H{"claims": claims})
}
