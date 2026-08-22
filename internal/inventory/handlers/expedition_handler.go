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
	gender := strings.ToUpper(strings.TrimSpace(req.Gender))
	if gender != "HOMME" && gender != "FEMME" && gender != "ENFANT" && gender != "UNISEXE" {
		shared.BadRequest(c, "gender invalide")
		return
	}
	orderDate, err := time.Parse("2006-01-02", req.OrderDate)
	if err != nil {
		shared.BadRequest(c, "order_date invalide (format attendu AAAA-MM-JJ)")
		return
	}

	note := buildExpeditionNote(req.Note, req.Country, req.City)

	order := &models.SupplierOrder{
		Supplier:  supplier,
		Quantity:  req.Quantity,
		Gender:    gender,
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

func buildExpeditionNote(rawNote string, country string, city string) string {
	noteParts := []string{}
	if trimmedNote := strings.TrimSpace(rawNote); trimmedNote != "" {
		for _, part := range strings.Split(trimmedNote, "|") {
			trimmedPart := strings.TrimSpace(part)
			if trimmedPart != "" && !containsNotePart(noteParts, trimmedPart) {
				noteParts = append(noteParts, trimmedPart)
			}
		}
	}

	country = strings.TrimSpace(country)
	city = strings.TrimSpace(city)
	if country != "" && !containsNotePart(noteParts, fmt.Sprintf("Pays: %s", country)) {
		noteParts = append(noteParts, fmt.Sprintf("Pays: %s", country))
	}
	if city != "" && !containsNotePart(noteParts, fmt.Sprintf("Ville: %s", city)) {
		noteParts = append(noteParts, fmt.Sprintf("Ville: %s", city))
	}
	return strings.Join(noteParts, " | ")
}

func containsNotePart(noteParts []string, candidate string) bool {
	candidate = strings.ToLower(strings.TrimSpace(candidate))
	for _, part := range noteParts {
		if strings.ToLower(strings.TrimSpace(part)) == candidate {
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
