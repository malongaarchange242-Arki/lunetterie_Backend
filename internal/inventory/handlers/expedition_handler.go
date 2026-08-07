package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lunetterie/backend/internal/inventory/models"
	"github.com/lunetterie/backend/internal/shared"
)

type expeditionRepository interface {
	Create(order *models.SupplierOrder) error
	List() ([]models.SupplierOrder, error)
}

type ExpeditionHandler struct {
	repo expeditionRepository
}

func NewExpeditionHandler(repo interface {
	Create(*models.SupplierOrder) error
	List() ([]models.SupplierOrder, error)
}) *ExpeditionHandler {
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
	country := strings.TrimSpace(req.Country)
	city := strings.TrimSpace(req.City)
	if country != "" || city != "" {
		locationParts := []string{}
		if country != "" {
			locationParts = append(locationParts, fmt.Sprintf("Pays: %s", country))
		}
		if city != "" {
			locationParts = append(locationParts, fmt.Sprintf("Ville: %s", city))
		}
		locationText := strings.Join(locationParts, " | ")
		if !containsLocation(noteParts, locationText) {
			noteParts = append(noteParts, locationText)
		}
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

func containsLocation(noteParts []string, locationText string) bool {
	for _, part := range noteParts {
		if strings.Contains(strings.ToLower(part), strings.ToLower(locationText)) {
			return true
		}
	}
	return false
}

func (h *ExpeditionHandler) List(c *gin.Context) {
	orders, err := h.repo.List()
	if err != nil {
		shared.InternalError(c, err.Error())
		return
	}
	shared.Success(c, http.StatusOK, gin.H{"orders": orders})
}
