package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/lunetterie/backend/internal/inventory/repositories"
	"github.com/lunetterie/backend/internal/inventory/services"
	"github.com/lunetterie/backend/internal/shared"
)

// GlassHandler gère la consultation des montures en stock
type GlassHandler struct {
	repo       *repositories.GlassRepository
	display    *services.DisplayService
	similarity *services.SimilarityService
}

// NewGlassHandler crée une nouvelle instance
func NewGlassHandler(repo *repositories.GlassRepository, display *services.DisplayService, similarity *services.SimilarityService) *GlassHandler {
	return &GlassHandler{repo: repo, display: display, similarity: similarity}
}

// ListGlasses liste les montures filtrées par statut, pour une station donnée,
// ou toutes stations confondues si station_id est omis.
// GET /api/v1/inventory/glasses?station_id=1&status=EN_STOCK_GENERAL,EN_STOCK_SOUS_STATION
// GET /api/v1/inventory/glasses?status=PRETE_A_LIVRER (toutes stations)
func (h *GlassHandler) ListGlasses(c *gin.Context) {
	statuses := []string{"EN_STOCK_GENERAL", "EN_STOCK_SOUS_STATION"}
	if raw := c.Query("status"); raw != "" {
		statuses = strings.Split(raw, ",")
	}

	var glasses interface{}
	var err error
	if raw := c.Query("station_id"); raw != "" {
		stationID, parseErr := strconv.ParseInt(raw, 10, 64)
		if parseErr != nil {
			shared.BadRequest(c, "station_id invalide")
			return
		}
		glasses, err = h.repo.FindByStationAndStatuses(stationID, statuses)
	} else {
		glasses, err = h.repo.FindByStatuses(statuses)
	}
	if err != nil {
		shared.InternalError(c, "Impossible de récupérer les montures")
		return
	}

	shared.Success(c, http.StatusOK, gin.H{"glasses": glasses})
}

// GetStockSummary renvoie le stock actif agrégé par référence, réparti entre
// Stock Général, Stock Local et Présentoir.
// GET /api/v1/inventory/stock-summary
func (h *GlassHandler) GetStockSummary(c *gin.Context) {
	summary, err := h.repo.GetStockSummaryByReference()
	if err != nil {
		shared.InternalError(c, "Impossible de calculer le résumé du stock")
		return
	}

	shared.Success(c, http.StatusOK, gin.H{"items": summary})
}

// GetGlassByBarcode recherche une monture par code-barres (toutes stations confondues).
// Si station_id est fourni et que la monture est à ce poste, sa recherche vaut confirmation de
// sa présence physique sur le présentoir : son statut et son emplacement sont mis à jour
// automatiquement (le code-barres, lui, ne change jamais).
// GET /api/v1/inventory/glasses/:barcode?station_id=1
func (h *GlassHandler) GetGlassByBarcode(c *gin.Context) {
	barcode := c.Param("barcode")

	var placementNote string
	if stationIDStr := c.Query("station_id"); stationIDStr != "" {
		stationID, err := strconv.ParseInt(stationIDStr, 10, 64)
		if err != nil {
			shared.BadRequest(c, "station_id invalide")
			return
		}
		userID, ok := currentUserID(c)
		if !ok {
			return
		}
		note, err := h.display.PlaceOnDisplay(barcode, stationID, userID)
		if err != nil {
			shared.NotFound(c, "Aucune monture ne correspond à ce code-barres")
			return
		}
		placementNote = note
	}

	glass, err := h.repo.FindDetailByBarcode(barcode)
	if err != nil {
		shared.NotFound(c, "Aucune monture ne correspond à ce code-barres")
		return
	}

	response := gin.H{"glass": glass}
	if placementNote != "" {
		response["placement_note"] = placementNote
	}
	shared.Success(c, http.StatusOK, response)
}

// GetSimilarGlasses classe les montures disponibles par ressemblance (genre, forme, prix) avec
// la monture identifiée par code-barres — pour proposer une alternative quand elle est
// indisponible, ou simplement suggérer des montures proches.
// GET /api/v1/inventory/glasses/:barcode/similar?limit=10
func (h *GlassHandler) GetSimilarGlasses(c *gin.Context) {
	barcode := c.Param("barcode")

	reference, err := h.repo.FindDetailByBarcode(barcode)
	if err != nil {
		shared.NotFound(c, "Aucune monture ne correspond à ce code-barres")
		return
	}

	limit := 10
	if raw := c.Query("limit"); raw != "" {
		parsed, parseErr := strconv.Atoi(raw)
		if parseErr != nil || parsed <= 0 {
			shared.BadRequest(c, "limit invalide")
			return
		}
		limit = parsed
	}

	similar, err := h.similarity.FindSimilar(reference, limit)
	if err != nil {
		shared.BadRequest(c, err.Error())
		return
	}

	shared.Success(c, http.StatusOK, gin.H{"glasses": similar})
}
