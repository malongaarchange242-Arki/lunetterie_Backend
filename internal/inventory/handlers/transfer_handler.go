package handlers

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	authRepositories "github.com/lunetterie/backend/internal/auth/repositories"
	"github.com/lunetterie/backend/internal/inventory/dto"
	"github.com/lunetterie/backend/internal/inventory/models"
	"github.com/lunetterie/backend/internal/inventory/repositories"
	"github.com/lunetterie/backend/internal/inventory/services"
	"github.com/lunetterie/backend/internal/shared"
)

// TransferHandler gère les endpoints de transfert inter-station
type TransferHandler struct {
	service    *services.TransferService
	glassRepo  *repositories.GlassRepository
	stationRepo *authRepositories.StationRepository
}

// NewTransferHandler crée une nouvelle instance
func NewTransferHandler(service *services.TransferService, glassRepo *repositories.GlassRepository, stationRepo *authRepositories.StationRepository) *TransferHandler {
	return &TransferHandler{service: service, glassRepo: glassRepo, stationRepo: stationRepo}
}

func mapTransferStatusForFront(status models.TransferStatus) string {
	switch status {
	case models.TransferStatusPreparation:
		return "valide"
	case models.TransferStatusInTransit:
		return "valide"
	case models.TransferStatusReceived:
		return "valide"
	case models.TransferStatusCancelled:
		return "refuse"
	default:
		return "attente"
	}
}

func toTransferFrontPayload(t *models.Transfer, items []models.TransferItem, glassRepo *repositories.GlassRepository, stationRepo *authRepositories.StationRepository) gin.H {
	uids := make([]string, 0, len(items))
	for _, item := range items {
		if glass, err := glassRepo.GetByID(item.GlassID); err == nil && strings.TrimSpace(glass.Barcode) != "" {
			uids = append(uids, glass.Barcode)
		}
	}

	fromStation := "Stock général"
	toStation := "Station"
	if stationRepo != nil {
		if s, err := stationRepo.GetByID(t.FromStationID); err == nil && strings.TrimSpace(s.Name) != "" {
			fromStation = s.Name
		}
		if s, err := stationRepo.GetByID(t.ToStationID); err == nil && strings.TrimSpace(s.Name) != "" {
			toStation = s.Name
		}
	}

	motif := "Transfert interne"
	if t.Notes != nil && strings.TrimSpace(*t.Notes) != "" {
		motif = strings.TrimSpace(*t.Notes)
	}
	trimmedToStation := strings.TrimSpace(toStation)
	codeBase := fmt.Sprintf("INT-%04d", t.ID)
	if len(trimmedToStation) >= 3 {
		codeBase = fmt.Sprintf("INT-%s-%04d", strings.ToUpper(trimmedToStation[:3]), t.ID)
	}

	return gin.H{
		"id":         t.ID,
		"ref":        fmt.Sprintf("INT-%d", t.ID),
		"magasin":    toStation,
		"source":     fromStation,
		"origine":    "admin",
		"date":       t.CreatedAt.Format("02/01/2006"),
		"motif":      motif,
		"besoin":     len(items),
		"statut":     mapTransferStatusForFront(t.Status),
		"uids":       uids,
		"cartonCode": "",
		"barcodeNum": codeBase,
		"expedie":    t.Status == models.TransferStatusInTransit || t.Status == models.TransferStatusReceived,
		"type":       "Tous",
		"gamme":      "Toutes",
		"genre":      "Tous",
		"couleur":    "Toutes",
		"urgence":    "Normale",
		"sourceLabel": fromStation,
	}
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

	payload := toTransferFrontPayload(transfer, nil, h.glassRepo, h.stationRepo)
	shared.Created(c, gin.H{
		"transfer": toTransferResponse(transfer, nil, h.glassRepo),
		"demande":  payload,
		"demandes": []gin.H{payload},
	})
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
	compat := make([]gin.H, 0, len(transfers))
	for _, t := range transfers {
		items, _ := h.service.ListItems(t.ID)
		resp = append(resp, toTransferResponse(&t, items, h.glassRepo))
		compat = append(compat, toTransferFrontPayload(&t, items, h.glassRepo, h.stationRepo))
	}
	shared.Success(c, 200, gin.H{
		"transfers": resp,
		"demandes":  compat,
		"items":     compat,
	})
}
