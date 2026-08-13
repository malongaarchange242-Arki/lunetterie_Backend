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

type SaleHandler struct {
	service *services.SaleService
}

type ReserveHandler struct {
	service *services.ReserveService
	repo    *repositories.ReserveRepository
}

func NewSaleHandler(service *services.SaleService) *SaleHandler {
	return &SaleHandler{service: service}
}

func NewReserveHandler(service *services.ReserveService, repo *repositories.ReserveRepository) *ReserveHandler {
	return &ReserveHandler{service: service, repo: repo}
}

func (h *SaleHandler) CreateSale(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	var req dto.CreateSaleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.BadRequest(c, "Données invalides: "+err.Error())
		return
	}

	sale, notShipped, err := h.service.CreateSale(req.StationID, req.Barcodes, userID)
	if err != nil {
		shared.BadRequest(c, err.Error())
		return
	}

	// Les montures restées VENDUE faute d'envoi au Laboratoire partent avec la réponse :
	// la vente est enregistrée, mais elles n'arriveront pas au montage toutes seules.
	shared.Success(c, http.StatusCreated, gin.H{"sale": sale, "lab_failures": notShipped})
}

func (h *ReserveHandler) CreateReserve(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	var req dto.CreateReserveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.BadRequest(c, "Données invalides: "+err.Error())
		return
	}

	reserve, err := h.service.CreateReserve(req.StationID, req.Barcodes, userID)
	if err != nil {
		shared.BadRequest(c, err.Error())
		return
	}

	shared.Success(c, http.StatusCreated, gin.H{"reserve": reserve})
}

// List rend les mises en réserve avec leur monture et leur vraie date.
//
// `?station_id=` filtre sur le magasin, `?active=true` ne garde que les montures encore
// réservées. Sans cette route, la réserve ne se lisait qu'à travers le statut RESERVEE — ce
// qui efface toute réservation passée et empêche de dater correctement le délai de dix
// jours, faute de connaître le jour de la mise de côté.
// GET /api/v1/inventory/reserves
func (h *ReserveHandler) List(c *gin.Context) {
	var stationID int64
	if raw := strings.TrimSpace(c.Query("station_id")); raw != "" {
		parsed, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			shared.BadRequest(c, "Identifiant de station invalide")
			return
		}
		stationID = parsed
	}

	active := strings.EqualFold(strings.TrimSpace(c.Query("active")), "true")

	lines, err := h.repo.List(stationID, active)
	if err != nil {
		shared.InternalError(c, err.Error())
		return
	}

	shared.Success(c, http.StatusOK, gin.H{"reserves": lines})
}
