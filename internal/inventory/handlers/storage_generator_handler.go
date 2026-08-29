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

func (h *StorageGeneratorHandler) CreateLocation(c *gin.Context) {
	var req dto.CreateStorageLocationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.BadRequest(c, "Données invalides: "+err.Error())
		return
	}
	location, err := h.service.CreateLocation(req.StationID, req.ParentLocationID, req.Type, req.Capacity)
	if err != nil {
		shared.BadRequest(c, err.Error())
		return
	}
	shared.Success(c, 201, location)
}

// NewStorageGeneratorHandler crée une nouvelle instance
func NewStorageGeneratorHandler(service *services.StorageGeneratorService) *StorageGeneratorHandler {
	return &StorageGeneratorHandler{service: service}
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

	locationID, path, code, err := h.service.PeekFreeLocation(stationID, models.ZoneType(zone))
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
