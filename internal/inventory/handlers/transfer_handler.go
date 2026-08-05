package handlers

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/lunetterie/backend/internal/inventory/dto"
	"github.com/lunetterie/backend/internal/inventory/models"
	"github.com/lunetterie/backend/internal/inventory/repositories"
	"github.com/lunetterie/backend/internal/inventory/services"
	"github.com/lunetterie/backend/internal/shared"
)

// TransferHandler gère les endpoints de transfert inter-station
type TransferHandler struct {
	service   *services.TransferService
	glassRepo *repositories.GlassRepository
}

// NewTransferHandler crée une nouvelle instance
func NewTransferHandler(service *services.TransferService, glassRepo *repositories.GlassRepository) *TransferHandler {
	return &TransferHandler{service: service, glassRepo: glassRepo}
}

func toTransferResponse(t *models.Transfer, items []models.TransferItem, glassRepo *repositories.GlassRepository) dto.TransferResponse {
	resp := dto.TransferResponse{
		ID:            t.ID,
		FromStationID: t.FromStationID,
		ToStationID:   t.ToStationID,
		Status:        string(t.Status),
		CreatedBy:     t.CreatedBy,
		ReceivedBy:    t.ReceivedBy,
		Notes:         t.Notes,
		CreatedAt:     t.CreatedAt.String(),
	}
	if t.ReceivedAt != nil {
		s := t.ReceivedAt.String()
		resp.ReceivedAt = &s
	}
	for _, item := range items {
		resp.Items = append(resp.Items, toTransferItemResponse(&item, glassRepo))
	}
	return resp
}

func toTransferItemResponse(item *models.TransferItem, glassRepo *repositories.GlassRepository) dto.TransferItemResponse {
	barcode := ""
	if glass, err := glassRepo.GetByID(item.GlassID); err == nil {
		barcode = glass.Barcode
	}
	r := dto.TransferItemResponse{
		ID:      item.ID,
		GlassID: item.GlassID,
		Barcode: barcode,
		Status:  string(item.Status),
	}
	if item.ScannedAt != nil {
		s := item.ScannedAt.String()
		r.ScannedAt = &s
	}
	return r
}

// CreateTransfer crée un nouveau transfert
// POST /api/v1/inventory/transfers
func (h *TransferHandler) CreateTransfer(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	var req dto.CreateTransferRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.BadRequest(c, "Données invalides: "+err.Error())
		return
	}

	transfer, err := h.service.CreateTransfer(req.FromStationID, req.ToStationID, req.Notes, userID)
	if err != nil {
		shared.BadRequest(c, err.Error())
		return
	}

	shared.Created(c, toTransferResponse(transfer, nil, h.glassRepo))
}

// AddItem ajoute une monture scannée à un transfert en préparation
// POST /api/v1/inventory/transfers/:id/items
func (h *TransferHandler) AddItem(c *gin.Context) {
	transferID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		shared.BadRequest(c, "ID de transfert invalide")
		return
	}

	var req dto.TransferItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.BadRequest(c, "Données invalides: "+err.Error())
		return
	}

	item, _, err := h.service.AddItem(transferID, req.Barcode)
	if err != nil {
		shared.BadRequest(c, err.Error())
		return
	}

	shared.Created(c, toTransferItemResponse(item, h.glassRepo))
}

// Dispatch finalise la préparation d'un transfert (départ en transit)
// POST /api/v1/inventory/transfers/:id/dispatch
func (h *TransferHandler) Dispatch(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	transferID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		shared.BadRequest(c, "ID de transfert invalide")
		return
	}

	transfer, err := h.service.Dispatch(transferID, userID)
	if err != nil {
		shared.BadRequest(c, err.Error())
		return
	}

	_, items, _ := h.service.GetTransfer(transferID)
	shared.Success(c, 200, toTransferResponse(transfer, items, h.glassRepo))
}

// ReceiveItem confirme la réception d'une monture scannée à la station de destination
// POST /api/v1/inventory/transfers/:id/receive
func (h *TransferHandler) ReceiveItem(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	transferID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		shared.BadRequest(c, "ID de transfert invalide")
		return
	}

	var req dto.TransferItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.BadRequest(c, "Données invalides: "+err.Error())
		return
	}

	item, glass, location, transfer, err := h.service.ReceiveItem(transferID, req.Barcode, userID)
	if err != nil {
		shared.BadRequest(c, err.Error())
		return
	}

	shared.Success(c, 200, dto.ReceiveTransferItemResponse{
		TransferItem:   toTransferItemResponse(item, h.glassRepo),
		NewLocation:    location.Code,
		GlassStatus:    string(glass.Status),
		TransferStatus: string(transfer.Status),
	})
}

// GetTransfer récupère le détail d'un transfert
// GET /api/v1/inventory/transfers/:id
func (h *TransferHandler) GetTransfer(c *gin.Context) {
	transferID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		shared.BadRequest(c, "ID de transfert invalide")
		return
	}

	transfer, items, err := h.service.GetTransfer(transferID)
	if err != nil {
		shared.NotFound(c, "Transfert introuvable")
		return
	}

	shared.Success(c, 200, toTransferResponse(transfer, items, h.glassRepo))
}

// ListTransfers liste les transferts (filtres optionnels: station_id, status)
// GET /api/v1/inventory/transfers
func (h *TransferHandler) ListTransfers(c *gin.Context) {
	var stationID *int64
	if s := c.Query("station_id"); s != "" {
		if id, err := strconv.ParseInt(s, 10, 64); err == nil {
			stationID = &id
		}
	}
	var status *string
	if s := c.Query("status"); s != "" {
		status = &s
	}

	transfers, err := h.service.ListTransfers(stationID, status)
	if err != nil {
		shared.InternalError(c, err.Error())
		return
	}

	resp := make([]dto.TransferResponse, 0, len(transfers))
	for _, t := range transfers {
		items, _ := h.service.ListItems(t.ID)
		resp = append(resp, toTransferResponse(&t, items, h.glassRepo))
	}
	shared.Success(c, 200, resp)
}
