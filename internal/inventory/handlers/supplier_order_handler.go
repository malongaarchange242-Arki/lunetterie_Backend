package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"regexp"
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

// generateNextSequentialCode génère le prochain code séquentiel pour un bon de commande
// Format: BC-YYYY-NNN où NNN est un numéro séquentiel (001, 002, etc.)
func (h *SupplierOrderHandler) generateNextSequentialCode(year int) (string, error) {
	prefix := fmt.Sprintf("BC-%d-", year)
	latestCode, err := h.repo.GetLatestCodeWithPrefix(prefix)
	if err != nil {
		return "", err
	}

	nextNum := 1
	if latestCode != "" {
		// Extraire le numéro séquentiel du dernier code (ex: "BC-2026-042" -> 42)
		re := regexp.MustCompile(`BC-\d+-(\d+)$`)
		matches := re.FindStringSubmatch(latestCode)
		if len(matches) > 1 {
			if num, err := strconv.Atoi(matches[1]); err == nil {
				nextNum = num + 1
			}
		}
	}

	return fmt.Sprintf("%s%03d", prefix, nextNum), nil
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
	gender := strings.ToUpper(strings.TrimSpace(req.Gender))
	if gender == "" {
		gender = "UNISEXE"
	}
	if gender != "HOMME" && gender != "FEMME" && gender != "ENFANT" && gender != "UNISEXE" {
		shared.BadRequest(c, "gender invalide")
		return
	}
	gamme := strings.ToLower(strings.TrimSpace(req.Gamme))
	if gamme == "" {
		gamme = "classique"
	}
	reference := strings.TrimSpace(req.Reference)
	if reference == "" {
		// Générer un code séquentiel automatique
		generatedCode, err := h.generateNextSequentialCode(orderDate.Year())
		if err != nil {
			shared.InternalError(c, "Erreur lors de la génération de la référence")
			return
		}
		reference = generatedCode
	}
	barcodeNum := strings.TrimSpace(req.BarcodeNum)
	if barcodeNum == "" {
		barcodeNum = reference
	}
	destination := strings.TrimSpace(req.Destination)
	if destination == "" {
		destination = "Stock général"
	}
	order := &models.SupplierOrder{Reference: reference, Supplier: supplier, Provenance: strings.TrimSpace(req.Provenance), Destination: destination, Quantity: req.Quantity, Gender: gender, Gamme: gamme, OrderDate: orderDate, Transport: strings.TrimSpace(req.Transport), BarcodeNum: barcodeNum, Status: "attente"}
	if note := strings.TrimSpace(req.Note); note != "" {
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
