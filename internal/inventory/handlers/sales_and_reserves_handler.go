package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/lunetterie/backend/internal/inventory/dto"
	"github.com/lunetterie/backend/internal/inventory/services"
	"github.com/lunetterie/backend/internal/shared"
)

type SaleHandler struct {
	service *services.SaleService
}

type ReserveHandler struct {
	service *services.ReserveService
}

func NewSaleHandler(service *services.SaleService) *SaleHandler {
	return &SaleHandler{service: service}
}

func NewReserveHandler(service *services.ReserveService) *ReserveHandler {
	return &ReserveHandler{service: service}
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

	sale, err := h.service.CreateSale(req.StationID, req.Barcodes, userID)
	if err != nil {
		shared.BadRequest(c, err.Error())
		return
	}

	shared.Success(c, http.StatusCreated, gin.H{"sale": sale})
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
