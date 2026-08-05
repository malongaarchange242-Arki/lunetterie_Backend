package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lunetterie/backend/internal/inventory/models"
	"github.com/lunetterie/backend/internal/inventory/repositories"
	"github.com/lunetterie/backend/internal/shared"
)

type SupplierOrderHandler struct {
	repo *repositories.SupplierOrderRepository
}

func NewSupplierOrderHandler(repo *repositories.SupplierOrderRepository) *SupplierOrderHandler {
	return &SupplierOrderHandler{repo: repo}
}

func (h *SupplierOrderHandler) Create(c *gin.Context) {
	var req models.SupplierOrderCreateRequest
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

	order := &models.SupplierOrder{
		Supplier:  supplier,
		Quantity:  req.Quantity,
		OrderDate: orderDate,
	}
	note := strings.TrimSpace(req.Note)
	if note != "" {
		order.Note = &note
	}
	if userID, exists := c.Get("user_id"); exists {
		if id, err := strconv.ParseInt(fmt.Sprintf("%v", userID), 10, 64); err == nil {
			order.CreatedBy = &id
		}
	}

	if err := h.repo.Create(order); err != nil {
		shared.InternalError(c, "Erreur lors de la création de la commande fournisseur")
		return
	}

	shared.Created(c, gin.H{"order": order})
}

func (h *SupplierOrderHandler) List(c *gin.Context) {
	orders, err := h.repo.List()
	if err != nil {
		shared.InternalError(c, err.Error())
		return
	}
	shared.Success(c, http.StatusOK, gin.H{"orders": orders})
}

func (h *SupplierOrderHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		shared.BadRequest(c, "id invalide")
		return
	}
	if err := h.repo.Delete(id); err != nil {
		if err == sql.ErrNoRows {
			shared.NotFound(c, "commande fournisseur introuvable")
			return
		}
		shared.InternalError(c, err.Error())
		return
	}
	shared.Success(c, http.StatusOK, gin.H{"deleted": true})
}
