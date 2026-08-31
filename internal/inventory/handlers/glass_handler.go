package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/lunetterie/backend/internal/inventory/models"
	"github.com/lunetterie/backend/internal/inventory/repositories"
	"github.com/lunetterie/backend/internal/inventory/services"
	"github.com/lunetterie/backend/internal/shared"
)

// GlassHandler gère la consultation des montures en stock
type GlassHandler struct {
	repo       *repositories.GlassRepository
	display    *services.DisplayService
	similarity *services.SimilarityService
	mutations  glassMutationService
}

type glassMutationService interface {
	CreateGlass(glass *models.Glass) error
	AssignGlass(glassID, cartonID, userID int64) error
	MoveGlass(glassID, cartonID, userID int64) error
	ReserveGlass(glassID, reservationID, userID int64) error
}

// NewGlassHandler crée une nouvelle instance
func NewGlassHandler(repo *repositories.GlassRepository, display *services.DisplayService, similarity *services.SimilarityService, mutations glassMutationService) *GlassHandler {
	return &GlassHandler{repo: repo, display: display, similarity: similarity, mutations: mutations}
}

// ListGlasses liste les montures filtrées par statut, pour une station donnée,
// ou toutes stations confondues si station_id est omis.
// GET /api/v1/inventory/glasses?station_id=1&status=EN_STOCK_GENERAL,EN_STOCK_SOUS_STATION
// GET /api/v1/inventory/glasses?status=PRETE_A_LIVRER (toutes stations)
func (h *GlassHandler) ListGlasses(c *gin.Context) {
	// RESERVEE_ENVOI reste listée par défaut : sans elle, une monture réservée par une liste
	// d'envoi disparaîtrait purement et simplement du Stock Général au lieu d'y rester visible
	// (grisée côté front) le temps que le magasinier la dispatche réellement.
	statuses := []string{"EN_STOCK_GENERAL", "EN_STOCK_SOUS_STATION", "RESERVEE_ENVOI"}
	if raw := c.Query("status"); raw != "" {
		statuses = strings.Split(raw, ",")
	}

	var glasses interface{}
	var err error
	if raw := c.Query("reception_command_id"); raw != "" {
		commandID, parseErr := strconv.ParseInt(raw, 10, 64)
		if parseErr != nil {
			shared.BadRequest(c, "reception_command_id invalide")
			return
		}
		glasses, err = h.repo.FindByReceptionCommand(commandID)
	} else if raw := c.Query("station_id"); raw != "" {
		stationID, parseErr := strconv.ParseInt(raw, 10, 64)
		if parseErr != nil {
			shared.BadRequest(c, "station_id invalide")
			return
		}
		glasses, err = h.repo.FindByStationAndStatuses(stationID, statuses)
	} else {
		registeredOnly := c.Query("registered_only") == "1" || strings.EqualFold(c.Query("registered_only"), "true")
		glasses, err = h.repo.FindByStatusesFiltered(statuses, registeredOnly)
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

// GetStockThresholdConfig renvoie la configuration locale de seuils utilisée pour
// la santé du stock et le besoin de réassort.
func (h *GlassHandler) GetStockThresholdConfig(c *gin.Context) {
	shared.Success(c, http.StatusOK, services.DefaultStockThresholdConfig())
}

// GetStockNeeds renvoie les écarts de stock à commander pour les axes métier.
func (h *GlassHandler) GetStockNeeds(c *gin.Context) {
	needs, err := h.repo.GetStockNeeds()
	if err != nil {
		shared.InternalError(c, "Impossible de calculer les besoins de stock")
		return
	}
	shared.Success(c, http.StatusOK, needs)
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

func parseOptionalString(v any) *string {
	if v == nil {
		return nil
	}
	switch value := v.(type) {
	case string:
		if value == "" {
			return nil
		}
		return &value
	case fmt.Stringer:
		text := value.String()
		if text == "" {
			return nil
		}
		return &text
	default:
		text := fmt.Sprint(value)
		if text == "" || text == "<nil>" {
			return nil
		}
		return &text
	}
}

func parseOptionalFloat64(v any) (*float64, error) {
	if v == nil {
		return nil, nil
	}
	switch value := v.(type) {
	case float64:
		return &value, nil
	case float32:
		floatValue := float64(value)
		return &floatValue, nil
	case int:
		floatValue := float64(value)
		return &floatValue, nil
	case int32:
		floatValue := float64(value)
		return &floatValue, nil
	case int64:
		floatValue := float64(value)
		return &floatValue, nil
	case string:
		trimmed := strings.TrimSpace(value)
		if trimmed == "" || trimmed == "null" {
			return nil, nil
		}
		floatValue, err := strconv.ParseFloat(trimmed, 64)
		if err != nil {
			return nil, err
		}
		return &floatValue, nil
	default:
		text := strings.TrimSpace(fmt.Sprint(value))
		if text == "" || text == "<nil>" || text == "null" {
			return nil, nil
		}
		floatValue, err := strconv.ParseFloat(text, 64)
		if err != nil {
			return nil, err
		}
		return &floatValue, nil
	}
}

func parseOptionalInt64(v any) (*int64, error) {
	if v == nil {
		return nil, nil
	}
	switch value := v.(type) {
	case int:
		intValue := int64(value)
		return &intValue, nil
	case int32:
		intValue := int64(value)
		return &intValue, nil
	case int64:
		return &value, nil
	case float64:
		intValue := int64(value)
		return &intValue, nil
	case string:
		trimmed := strings.TrimSpace(value)
		if trimmed == "" || trimmed == "null" {
			return nil, nil
		}
		intValue, err := strconv.ParseInt(trimmed, 10, 64)
		if err != nil {
			return nil, err
		}
		return &intValue, nil
	default:
		text := strings.TrimSpace(fmt.Sprint(value))
		if text == "" || text == "<nil>" || text == "null" {
			return nil, nil
		}
		intValue, err := strconv.ParseInt(text, 10, 64)
		if err != nil {
			return nil, err
		}
		return &intValue, nil
	}
}

func parseCreateGlassRequest(rawBody []byte) (*models.Glass, error) {
	var payload map[string]any
	if err := json.Unmarshal(rawBody, &payload); err != nil {
		return nil, err
	}

	barcode, _ := payload["barcode"].(string)
	barcode = strings.TrimSpace(barcode)
	if barcode == "" {
		barcode = ""
	}

	price, err := parseOptionalFloat64(payload["price"])
	if err != nil {
		return nil, fmt.Errorf("price invalide: %w", err)
	}

	receptionCommandID, err := parseOptionalInt64(payload["reception_command_id"])
	if err != nil {
		return nil, fmt.Errorf("reception_command_id invalide: %w", err)
	}

	glass := &models.Glass{
		Barcode:            barcode,
		StationID:          1,
		Price:              price,
		PhotoMontureURL:    parseOptionalString(payload["photo_monture_url"]),
		PhotoBrancheURL:    parseOptionalString(payload["photo_branche_url"]),
		PhotoArriereURL:    parseOptionalString(payload["photo_arriere_url"]),
		ReceptionCommandID: receptionCommandID,
		Status:             models.StatusEnStockGeneral,
		IsReserved:         false,
	}

	if notes := parseOptionalString(payload["notes"]); notes != nil && *notes != "" {
		glass.Notes = notes
	}

	return glass, nil
}

// CreateGlass crée une nouvelle monture au stock général à partir du pré-enregistrement
// POST /api/v1/inventory/glasses
func (h *GlassHandler) CreateGlass(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		shared.BadRequest(c, "Données invalides")
		return
	}

	glass, err := parseCreateGlassRequest(body)
	if err != nil {
		shared.BadRequest(c, "Données invalides: "+err.Error())
		return
	}

	if err := h.mutations.CreateGlass(glass); err != nil {
		shared.InternalError(c, "Impossible de créer la monture: "+err.Error())
		return
	}

	shared.Created(c, gin.H{"glass": glass})
}

// RelocateGlass réattribue un emplacement libre à une monture, dans la même zone de la même
// station. Appelé quand on réimprime l'étiquette et qu'on repose la monture ailleurs.
// POST /api/v1/inventory/glasses/:barcode/relocate
func (h *GlassHandler) RelocateGlass(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	location, err := h.display.RelocateGlass(c.Param("barcode"), userID)
	if err != nil {
		shared.BadRequest(c, err.Error())
		return
	}

	shared.Success(c, http.StatusOK, gin.H{
		"location_id": location.ID,
		"code":        location.Code,
	})
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
