package handlers

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lunetterie/backend/internal/inventory/repositories"
	"github.com/lunetterie/backend/internal/shared"
)

type ShipmentHandler struct {
	preRegRepo *repositories.PreRegistrationRepository
	rcRepo     *repositories.ReceptionCommandRepository
}

func NewShipmentHandler(preRegRepo *repositories.PreRegistrationRepository, rcRepo *repositories.ReceptionCommandRepository) *ShipmentHandler {
	return &ShipmentHandler{
		preRegRepo: preRegRepo,
		rcRepo:     rcRepo,
	}
}

// Dispatch expédies une commande de réception (passe en transit)
func (h *ShipmentHandler) Dispatch(c *gin.Context) {
	code := strings.TrimSpace(c.Param("code"))
	if code == "" {
		shared.BadRequest(c, "code de commande requis")
		return
	}
	if err := h.preRegRepo.DispatchReceptionCommand(code); err != nil {
		shared.InternalError(c, "Impossible de dispatcher la commande: "+err.Error())
		return
	}
	command, err := h.rcRepo.GetByCode(code)
	if err != nil || command == nil {
		shared.InternalError(c, "Impossible de récupérer la commande après expédition")
		return
	}
	shared.Success(c, http.StatusOK, gin.H{"command": command})
}

// ScanCase marque une valise comme scannée à l'arrivée
func (h *ShipmentHandler) ScanCase(c *gin.Context) {
	caseID, err := strconv.ParseInt(c.Param("caseId"), 10, 64)
	if err != nil {
		shared.BadRequest(c, "identifiant de valise invalide")
		return
	}
	if err := h.preRegRepo.ScanReceptionCase(caseID); err != nil {
		shared.InternalError(c, "Impossible de scanner la valise: "+err.Error())
		return
	}

	// Vérifier si toutes les valises sont scannées et marquer la commande comme arrivée
	allScanned, commandCode, err := h.preRegRepo.CheckAllCasesScanned(caseID)
	if err == nil && allScanned && commandCode != "" {
		h.preRegRepo.MarkCommandArrived(commandCode)
	}

	shared.Success(c, http.StatusOK, gin.H{"scanned": true})
}

// GetArrivedCommands récupère les commandes arrivées au stock général
func (h *ShipmentHandler) GetArrivedCommands(c *gin.Context) {
	commands, err := h.rcRepo.ListArrivedCommands()
	if err != nil {
		shared.InternalError(c, "Impossible de récupérer les commandes arrivées: "+err.Error())
		return
	}
	
	enriched := make([]map[string]interface{}, 0, len(commands))
	for _, cmd := range commands {
		shipment, _ := h.preRegRepo.GetShipmentDataByCommandID(cmd.ID)
		if shipment != nil {
			shipment.Date = cmd.DispatchedAt
			shipment.ArriveDate = cmd.ArrivedAt
		}
		
		data := map[string]interface{}{
			"id": cmd.ID,
			"code": cmd.Code,
			"target_count": cmd.TargetCount,
			"registered_count": cmd.RegisteredCount,
			"status": cmd.Status,
			"pre_registration_status": cmd.PreRegistrationStatus,
			"shipment_status": cmd.ShipmentStatus,
			"dispatched_at": cmd.DispatchedAt,
			"arrived_at": cmd.ArrivedAt,
			"supplier_order_id": cmd.SupplierOrderID,
			"gender": cmd.OrderGender,
			"provenance": cmd.OrderProvenance,
			"gamme": cmd.OrderGamme,
			"created_by": cmd.CreatedBy,
			"activated_at": cmd.ActivatedAt,
			"created_at": cmd.CreatedAt,
			"updated_at": cmd.UpdatedAt,
			"ship": shipment,
		}
		enriched = append(enriched, data)
	}
	shared.Success(c, http.StatusOK, gin.H{"commands": enriched})
}

// ListShipmentCommands expose les expéditions en transit et celles déjà arrivées
// au poste de scan. Une commande en transit doit être visible pour pouvoir scanner
// ses valises et déclencher son arrivée.
func (h *ShipmentHandler) ListShipmentCommands(c *gin.Context) {
	commands, err := h.rcRepo.ListShipmentCommands()
	if err != nil {
		shared.InternalError(c, "Impossible de récupérer les expéditions: "+err.Error())
		return
	}
	
	enriched := make([]map[string]interface{}, 0, len(commands))
	for _, cmd := range commands {
		shipment, _ := h.preRegRepo.GetShipmentDataByCommandID(cmd.ID)
		if shipment != nil {
			shipment.Date = cmd.DispatchedAt
			shipment.ArriveDate = cmd.ArrivedAt
		}
		
		data := map[string]interface{}{
			"id": cmd.ID,
			"code": cmd.Code,
			"target_count": cmd.TargetCount,
			"registered_count": cmd.RegisteredCount,
			"status": cmd.Status,
			"pre_registration_status": cmd.PreRegistrationStatus,
			"shipment_status": cmd.ShipmentStatus,
			"dispatched_at": cmd.DispatchedAt,
			"arrived_at": cmd.ArrivedAt,
			"supplier_order_id": cmd.SupplierOrderID,
			"gender": cmd.OrderGender,
			"provenance": cmd.OrderProvenance,
			"gamme": cmd.OrderGamme,
			"created_by": cmd.CreatedBy,
			"activated_at": cmd.ActivatedAt,
			"created_at": cmd.CreatedAt,
			"updated_at": cmd.UpdatedAt,
			"ship": shipment,
		}
		enriched = append(enriched, data)
	}
	shared.Success(c, http.StatusOK, gin.H{"commands": enriched})
}

// MarkArrived marque une commande comme complètement arrivée au stock
func (h *ShipmentHandler) MarkArrived(c *gin.Context) {
	code := strings.TrimSpace(c.Param("code"))
	if code == "" {
		shared.BadRequest(c, "code de commande requis")
		return
	}
	command, err := h.rcRepo.GetByCode(code)
	if err != nil || command == nil {
		shared.NotFound(c, "Commande introuvable")
		return
	}
	if command.ShipmentStatus == "arrived" {
		shared.Success(c, http.StatusOK, gin.H{
			"arrived":    true,
			"arrivingAt": command.ArrivedAt.Format(time.RFC3339),
			"message":    "Commande déjà arrivée au stock",
		})
		return
	}
	if command.ShipmentStatus != "in_transit" {
		shared.BadRequest(c, "Seules les commandes en transit peuvent arriver")
		return
	}
	if err := h.preRegRepo.MarkCommandArrived(code); err != nil {
		shared.InternalError(c, "Impossible de marquer la commande comme arrivée: "+err.Error())
		return
	}
	now := time.Now()
	shared.Success(c, http.StatusOK, gin.H{
		"arrived":    true,
		"arrivingAt": now.Format(time.RFC3339),
		"message":    "Commande arrivée au stock",
	})
}
