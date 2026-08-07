package handlers

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/lunetterie/backend/internal/inventory/dto"
	"github.com/lunetterie/backend/internal/inventory/models"
	"github.com/lunetterie/backend/internal/inventory/services"
	"github.com/lunetterie/backend/internal/shared"
)

// StorageGeneratorHandler gère les endpoints d'assistance de génération de stockage
type StorageGeneratorHandler struct {
	service *services.StorageGeneratorService
}

// NewStorageGeneratorHandler crée une nouvelle instance
func NewStorageGeneratorHandler(service *services.StorageGeneratorService) *StorageGeneratorHandler {
	return &StorageGeneratorHandler{service: service}
}

// GenerateLocations génère automatiquement une arborescence d'emplacements pour une station
// POST /api/v1/inventory/storage/generate
func (h *StorageGeneratorHandler) GenerateLocations(c *gin.Context) {
	var req dto.GenerateLocationsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.BadRequest(c, "Données invalides: "+err.Error())
		return
	}

	if req.Template.NumRayons <= 0 || req.Template.EtageresParRayon <= 0 || req.Template.BacsParEtagere <= 0 || req.Template.PositionsParBac <= 0 {
		shared.BadRequest(c, "Les valeurs du template doivent être supérieures à 0")
		return
	}

	template := services.StationTemplate{
		Name:             req.Template.Name,
		NumRayons:        req.Template.NumRayons,
		EtageresParRayon: req.Template.EtageresParRayon,
		BacsParEtagere:   req.Template.BacsParEtagere,
		PositionsParBac:  req.Template.PositionsParBac,
	}

	total, err := h.service.GenerateLocations(req.StationID, template)
	if err != nil {
		shared.InternalError(c, "Erreur lors de la génération des emplacements: "+err.Error())
		return
	}

	shared.Success(c, 201, gin.H{"total": total})
}

// FindFreeLocation retourne un emplacement libre et son chemin lisible
// POST /api/v1/inventory/storage/find-free
func (h *StorageGeneratorHandler) FindFreeLocation(c *gin.Context) {
	var req dto.FindFreeLocationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.BadRequest(c, "Données invalides: "+err.Error())
		return
	}

	if req.StationID == 0 {
		shared.BadRequest(c, "station_id est requis")
		return
	}

	if req.Zone == "" {
		req.Zone = string(models.ZoneStock)
	}

	locationID, path, code, err := h.service.FindFreeLocation(req.StationID, models.ZoneType(req.Zone))
	if err != nil {
		shared.InternalError(c, "Erreur lors de la recherche d'un emplacement libre: "+err.Error())
		return
	}

	shared.Success(c, 200, gin.H{
		"location_id": locationID,
		"path":        path,
		"code":        code,
	})
}

// PreviewFreeLocation retourne le prochain emplacement libre SANS le réserver, pour un aperçu
// avant l'enregistrement réel (l'emplacement réellement attribué peut différer si un autre
// enregistrement concurrent le prend entre-temps).
// GET /api/v1/inventory/storage/next-free?station_id=1&zone=STOCK
func (h *StorageGeneratorHandler) PreviewFreeLocation(c *gin.Context) {
	stationID, err := strconv.ParseInt(c.Query("station_id"), 10, 64)
	if err != nil {
		shared.BadRequest(c, "station_id requis")
		return
	}

	zone := c.Query("zone")
	if zone == "" {
		zone = string(models.ZoneStock)
	}

	priceParam := c.Query("price")
	var price *float64
	if priceParam != "" {
		parsed, err := strconv.ParseFloat(priceParam, 64)
		if err != nil {
			shared.BadRequest(c, "price invalide")
			return
		}
		price = &parsed
	}
	gamme := c.Query("gamme")

	locationID, path, code, err := h.service.PeekFreeLocationForPrice(stationID, models.ZoneType(zone), price, gamme)
	if err != nil {
		shared.NotFound(c, "Aucun emplacement libre trouvé")
		return
	}

	shared.Success(c, 200, gin.H{
		"location_id": locationID,
		"path":        path,
		"code":        code,
	})
}
