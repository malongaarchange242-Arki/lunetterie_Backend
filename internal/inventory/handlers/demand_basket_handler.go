package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/lunetterie/backend/internal/inventory/models"
	"github.com/lunetterie/backend/internal/shared"
)

type demandBasketRepository interface {
	Create(item *models.DemandBasketItem) error
	CountsByCity() ([]models.DemandBasketCount, error)
	ListByCity(city string) ([]models.DemandBasketItem, error)
	MarkSent(ids []int64) (int64, error)
}

// DemandBasketHandler expose les paniers de demande : un panier par magasin (ville),
// alimenté par les recherches du chatbot.
type DemandBasketHandler struct {
	repo demandBasketRepository
}

func NewDemandBasketHandler(repo demandBasketRepository) *DemandBasketHandler {
	return &DemandBasketHandler{repo: repo}
}

// optionalString renvoie nil pour une chaîne vide : les quatre critères sont facultatifs et
// une chaîne vide en base se confondrait avec « critère renseigné mais sans valeur ».
func optionalString(raw string) *string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

// Create dépose une ligne de demande dans le panier d'une ville.
// POST /api/v1/inventory/baskets
func (h *DemandBasketHandler) Create(c *gin.Context) {
	var req models.DemandBasketCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.BadRequest(c, "Données invalides")
		return
	}

	city := strings.TrimSpace(req.City)
	if city == "" {
		shared.BadRequest(c, "city est requis")
		return
	}

	// Une demande sans aucun critère n'apprend rien : elle gonflerait le compteur sans
	// jamais pouvoir être rapprochée du stock principal.
	if optionalString(req.Genre) == nil && optionalString(req.Forme) == nil &&
		optionalString(req.Gamme) == nil && optionalString(req.Taille) == nil {
		shared.BadRequest(c, "au moins un critère (genre, forme, gamme, taille) est requis")
		return
	}

	source := strings.ToUpper(strings.TrimSpace(req.Source))
	if source != "MANUEL" {
		source = "CHATBOT"
	}

	item := &models.DemandBasketItem{
		City:   city,
		Genre:  optionalString(req.Genre),
		Forme:  optionalString(req.Forme),
		Gamme:  optionalString(req.Gamme),
		Taille: optionalString(req.Taille),
		Source: source,
	}
	if userID, exists := c.Get("user_id"); exists {
		if id, err := strconv.ParseInt(fmt.Sprintf("%v", userID), 10, 64); err == nil {
			item.CreatedBy = &id
		}
	}

	if err := h.repo.Create(item); err != nil {
		shared.InternalError(c, "Erreur lors de l'ajout au panier")
		return
	}

	shared.Created(c, gin.H{"item": item})
}

// Counts renvoie le compteur de chaque panier, pour la rangée de paniers de l'écran stock.
// GET /api/v1/inventory/baskets/counts
func (h *DemandBasketHandler) Counts(c *gin.Context) {
	counts, err := h.repo.CountsByCity()
	if err != nil {
		shared.InternalError(c, err.Error())
		return
	}
	shared.Success(c, http.StatusOK, gin.H{"counts": counts})
}

// List renvoie le contenu du panier d'une ville.
// GET /api/v1/inventory/baskets?city=Pointe-Noire
func (h *DemandBasketHandler) List(c *gin.Context) {
	city := strings.TrimSpace(c.Query("city"))
	if city == "" {
		shared.BadRequest(c, "city est requis")
		return
	}

	items, err := h.repo.ListByCity(city)
	if err != nil {
		shared.InternalError(c, err.Error())
		return
	}
	shared.Success(c, http.StatusOK, gin.H{"items": items})
}

// MarkSent clôt les demandes reprises dans une demande adressée au stock principal.
// POST /api/v1/inventory/baskets/sent
func (h *DemandBasketHandler) MarkSent(c *gin.Context) {
	var req models.DemandBasketMarkSentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.BadRequest(c, "Données invalides")
		return
	}
	if len(req.IDs) == 0 {
		shared.BadRequest(c, "ids est requis")
		return
	}

	updated, err := h.repo.MarkSent(req.IDs)
	if err != nil {
		shared.InternalError(c, err.Error())
		return
	}
	shared.Success(c, http.StatusOK, gin.H{"updated": updated})
}
