package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/lunetterie/backend/internal/inventory/models"
	"github.com/lunetterie/backend/internal/shared"
)

type claimRepository interface {
	Create(claim *models.Claim) error
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
