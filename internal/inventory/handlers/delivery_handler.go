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

// DeliveryHandler gère la création de delivery depuis le poste Laboratoire
type DeliveryHandler struct {
	service *services.DeliveryService
	repo    *repositories.GlassRepository
}

func NewDeliveryHandler(svc *services.DeliveryService, repo *repositories.GlassRepository) *DeliveryHandler {
	return &DeliveryHandler{service: svc, repo: repo}
}

func (h *DeliveryHandler) CreateDelivery(c *gin.Context) {
	userIDVal, ok := c.Get("user_id")
	if !ok {
		shared.Unauthorized(c, "Utilisateur non authentifié")
		return
	}
	userIDStr, _ := userIDVal.(string)
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		shared.BadRequest(c, "ID utilisateur invalide")
		return
	}

	var req dto.CreateDeliveryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.BadRequest(c, "Données invalides: "+err.Error())
		return
	}

	// Convert selected barcodes from request
	delivery, err := h.service.CreateDelivery(req.StationID, req.Barcodes, userID)
	if err != nil {
		shared.BadRequest(c, err.Error())
		return
	}
	shared.Success(c, http.StatusCreated, gin.H{"delivery": delivery})
}

func parseInt64(s string) int64 {
	var v int64
	// best-effort: ignore errors
	return v
}
