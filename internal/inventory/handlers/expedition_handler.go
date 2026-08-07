package handlers

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lunetterie/backend/internal/inventory/models"
	"github.com/lunetterie/backend/internal/inventory/repositories"
	"github.com/lunetterie/backend/internal/shared"
)

type ExpeditionHandler struct {
	repo *repositories.SupplierOrderRepository
}

func NewExpeditionHandler(repo *repositories.SupplierOrderRepository) *ExpeditionHandler {
	return &ExpeditionHandler{repo: repo}
}

func (h *ExpeditionHandler) Create(c *gin.Context) {
	var req models.ExpeditionCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.BadRequest(c, "Données invalides")
		return
	}

	supplier := strings.TrimSpace(req.Supplier)
	if supplier == "" {
		shared.BadRequest(c, "supplier est requis")
		return
	}
	if req.Quantity < 1 {
		shared.BadRequest(c, "quantity doit être supérieur à 0")
		return
	}
	orderDate, err := time.Parse("2006-01-02", req.OrderDate)
	if err != nil {
		shared.BadRequest(c, "order_date invalide (format attendu AAAA-MM-JJ)")
		return
	}

	noteParts := []string{}
	if trimmedNote := strings.TrimSpace(req.Note); trimmedNote != "" {
		noteParts = append(noteParts, trimmedNote)
	}
	if trimmedCountry := strings.TrimSpace(req.Country); trimmedCountry != "" {
		noteParts = append(noteParts, fmt.Sprintf("Pays: %s", trimmedCountry))
	}
	if trimmedCity := strings.TrimSpace(req.City); trimmedCity != "" {
		noteParts = append(noteParts, fmt.Sprintf("Ville: %s", trimmedCity))
	}
	note := strings.Join(noteParts, " | ")

	order := &models.SupplierOrder{
		Supplier:  supplier,
		Quantity:  req.Quantity,
		OrderDate: orderDate,
	}
	if note != "" {
		order.Note = &note
	}
	if userID, exists := c.Get("user_id"); exists {
		if id, err := strconv.ParseInt(fmt.Sprintf("%v", userID), 10, 64); err == nil {
			order.CreatedBy = &id
		}
	}

	if err := h.repo.Create(order); err != nil {
		shared.InternalError(c, "Erreur lors de l'enregistrement de l'expédition")
		return
	}

	shared.Created(c, gin.H{"order": order})
}
