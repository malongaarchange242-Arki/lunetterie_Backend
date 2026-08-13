package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/lunetterie/backend/internal/inventory/dto"
	"github.com/lunetterie/backend/internal/inventory/repositories"
	"github.com/lunetterie/backend/internal/inventory/services"
	"github.com/lunetterie/backend/internal/shared"
)

// DeliveryHandler gère la création de delivery depuis le poste Laboratoire
type DeliveryHandler struct {
	service      *services.DeliveryService
	repo         *repositories.GlassRepository
	deliveryRepo *repositories.DeliveryRepository
}

func NewDeliveryHandler(svc *services.DeliveryService, repo *repositories.GlassRepository, deliveryRepo *repositories.DeliveryRepository) *DeliveryHandler {
	return &DeliveryHandler{service: svc, repo: repo, deliveryRepo: deliveryRepo}
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

// Handover acte la remise au client : la monture quitte le magasin.
//
// Distinct de CreateDelivery, qui est le bouton du laboratoire (« montage terminé ») :
// celui-ci est celui de la vendeuse (« le client est reparti avec »).
// POST /api/v1/inventory/deliveries/handover
// body: { station_id, barcodes: [...] }
func (h *DeliveryHandler) Handover(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	var req dto.CreateDeliveryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.BadRequest(c, "Données invalides: "+err.Error())
		return
	}

	remises, skipped, err := h.service.HandoverToClient(req.StationID, req.Barcodes, userID)
	if err != nil {
		shared.BadRequest(c, err.Error())
		return
	}

	// `skipped` accompagne la réussite au lieu de la remplacer : une remise de trois paires
	// dont une seule est refusée reste une remise réussie, et le poste doit pouvoir dire
	// laquelle n'est pas passée.
	shared.Success(c, http.StatusOK, gin.H{"handed_over": remises, "skipped": skipped})
}

// List rend les lignes de livraison, avec leur monture.
//
// `?station_id=` filtre sur le magasin, `?pending=true` ne garde que ce qui attend encore
// son client. Sans cette route, les tables deliveries / delivery_items se remplissaient
// sans qu'aucun écran ne puisse les relire.
// GET /api/v1/inventory/deliveries
func (h *DeliveryHandler) List(c *gin.Context) {
	var stationID int64
	if raw := strings.TrimSpace(c.Query("station_id")); raw != "" {
		parsed, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			shared.BadRequest(c, "Identifiant de station invalide")
			return
		}
		stationID = parsed
	}

	pending := strings.EqualFold(strings.TrimSpace(c.Query("pending")), "true")

	lines, err := h.deliveryRepo.List(stationID, pending)
	if err != nil {
		shared.InternalError(c, err.Error())
		return
	}

	shared.Success(c, http.StatusOK, gin.H{"deliveries": lines})
}
