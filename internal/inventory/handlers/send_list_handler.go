package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/lunetterie/backend/internal/inventory/models"
	"github.com/lunetterie/backend/internal/inventory/services"
	"github.com/lunetterie/backend/internal/shared"
)

type sendListRepository interface {
	Create(list *models.SendList, items []models.SendListItemRequest) error
	NextStockListCode() (string, error)
	SplitAvailableBarcodes(barcodes []string) ([]string, map[string]string, error)
	List(status string) ([]models.SendList, error)
	ListItems(listID int64, query string) ([]models.SendListItem, error)
	Cancel(id int64) (*models.SendList, error)
	MarkSeen(ids []int64) (int64, error)
	MarkProcessed(ids []int64) (int64, error)
}

// sendListDispatcher expédie une liste vers la station locale de sa ville.
type sendListDispatcher interface {
	Dispatch(listID, fromStationID, userID int64) (*services.SendListDispatchResult, error)
}

// SendListHandler expose les listes d'envoi : la direction les crée depuis une session de
// réception terminée, le poste de scan les scrute pour prévenir le magasinier.
type SendListHandler struct {
	repo       sendListRepository
	dispatcher sendListDispatcher
}

func NewSendListHandler(repo sendListRepository, dispatcher sendListDispatcher) *SendListHandler {
	return &SendListHandler{repo: repo, dispatcher: dispatcher}
}

// Create archive une liste envoyée vers un magasin.
// POST /api/v1/inventory/send-lists
func (h *SendListHandler) Create(c *gin.Context) {
	var req models.SendListCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.BadRequest(c, "Données invalides")
		return
	}

	sessionCode := strings.TrimSpace(req.SessionCode)
	city := strings.TrimSpace(req.City)
	if city == "" {
		shared.BadRequest(c, "city est requis")
		return
	}
	if len(req.Items) == 0 {
		shared.BadRequest(c, "une liste vide ne peut pas être envoyée")
		return
	}

	// Pas de session de réception : la liste est composée depuis le stock existant. On lui
	// donne un code à part pour que le magasinier distingue, dans « Listes reçues », un
	// arrivage fournisseur d'un réapprovisionnement de rayon.
	fromStock := sessionCode == ""
	if fromStock {
		generated, err := h.repo.NextStockListCode()
		if err != nil {
			shared.InternalError(c, err.Error())
			return
		}
		sessionCode = generated
	}

	items := req.Items
	rejected := map[string]string{}
	// La vérification ne porte que sur les listes issues du stock : celles d'un arrivage
	// décrivent des montures qui viennent d'être réceptionnées, le contrôle n'aurait rien
	// à dire de plus.
	if fromStock {
		barcodes := make([]string, 0, len(req.Items))
		for _, item := range req.Items {
			if code := strings.TrimSpace(item.Barcode); code != "" {
				barcodes = append(barcodes, code)
			}
		}

		available, unavailable, err := h.repo.SplitAvailableBarcodes(barcodes)
		if err != nil {
			shared.InternalError(c, err.Error())
			return
		}
		rejected = unavailable

		keep := make(map[string]bool, len(available))
		for _, code := range available {
			keep[code] = true
		}
		filtered := make([]models.SendListItemRequest, 0, len(available))
		for _, item := range req.Items {
			if keep[strings.TrimSpace(item.Barcode)] {
				filtered = append(filtered, item)
			}
		}
		items = filtered

		// Tout est indisponible : on ne crée pas une liste vide que le magasinier ouvrirait
		// pour rien. Les motifs partent quand même, c'est ce qui explique le refus.
		if len(items) == 0 {
			shared.BadRequestWithData(c, "Aucune des montures sélectionnées n'est disponible.", gin.H{"rejected": rejected})
			return
		}
	}

	list := &models.SendList{SessionCode: sessionCode, City: city}
	if userID, exists := c.Get("user_id"); exists {
		if id, err := strconv.ParseInt(fmt.Sprintf("%v", userID), 10, 64); err == nil {
			list.CreatedBy = &id
		}
	}

	if err := h.repo.Create(list, items); err != nil {
		shared.InternalError(c, "Erreur lors de l'enregistrement de la liste")
		return
	}

	shared.Created(c, gin.H{"list": list, "rejected": rejected, "sent": len(items)})
}

// List renvoie les listes, filtrables par statut.
// GET /api/v1/inventory/send-lists?status=NOUVELLE
func (h *SendListHandler) List(c *gin.Context) {
	status := strings.ToUpper(strings.TrimSpace(c.Query("status")))
	if status != "" && status != "NOUVELLE" && status != "VUE" && status != "TRAITEE" && status != "ANNULEE" {
		shared.BadRequest(c, "status invalide")
		return
	}

	lists, err := h.repo.List(status)
	if err != nil {
		shared.InternalError(c, err.Error())
		return
	}
	shared.Success(c, http.StatusOK, gin.H{"lists": lists})
}

func mapValidationStatus(status string) string {
	switch strings.ToUpper(strings.TrimSpace(status)) {
	case "NOUVELLE", "VUE":
		return "attente"
	case "TRAITEE":
		return "valide"
	case "ANNULEE":
		return "refuse"
	default:
		return "attente"
	}
}

func toValidationPayload(list models.SendList, items []models.SendListItem) gin.H {
	uids := make([]string, 0, len(items))
	for _, item := range items {
		if item.Barcode != nil && strings.TrimSpace(*item.Barcode) != "" {
			uids = append(uids, strings.TrimSpace(*item.Barcode))
		}
	}

	date := list.CreatedAt.Format("2006-01-02")
	if !list.CreatedAt.IsZero() {
		date = list.CreatedAt.Format("02/01/2006")
	}
	return gin.H{
		"id":          list.ID,
		"ref":         list.SessionCode,
		"magasin":     list.City,
		"source":      func() string { if list.DestinationStationName != nil { return *list.DestinationStationName }; return "Stock général" }(),
		"origine":     "auto",
		"date":        date,
		"motif":       fmt.Sprintf("Liste %s vers %s", list.SessionCode, list.City),
		"besoin":      list.ItemCount,
		"statut":      mapValidationStatus(list.Status),
		"uids":        uids,
		"cartonCode":  "",
		"type":        "Tous",
		"gamme":       "Toutes",
		"genre":       "Tous",
		"couleur":     "Toutes",
		"urgence":     "Normale",
		"sourceLabel": func() string { if list.DestinationStationName != nil { return *list.DestinationStationName }; return "Stock général" }(),
	}
}

// ListValidations renvoie les listes dans le format attendu par la page de validation du front.
// GET /api/v1/inventory/send-lists/validation
func (h *SendListHandler) ListValidations(c *gin.Context) {
	status := strings.TrimSpace(c.Query("status"))
	if status != "" && status != "attente" && status != "valide" && status != "refuse" && status != "tous" {
		shared.BadRequest(c, "status invalide")
		return
	}

	backendStatus := ""
	if status == "attente" {
		backendStatus = "NOUVELLE"
	} else if status == "valide" {
		backendStatus = "TRAITEE"
	} else if status == "refuse" {
		backendStatus = "ANNULEE"
	}

	lists, err := h.repo.List(backendStatus)
	if err != nil {
		shared.InternalError(c, err.Error())
		return
	}

	result := make([]gin.H, 0, len(lists))
	for _, list := range lists {
		items, err := h.repo.ListItems(list.ID, "")
		if err != nil {
			shared.InternalError(c, err.Error())
			return
		}
		result = append(result, toValidationPayload(list, items))
	}
	shared.Success(c, http.StatusOK, gin.H{"demandes": result})
}

// GetItems renvoie le contenu d'une liste.
// GET /api/v1/inventory/send-lists/:id/items
func (h *SendListHandler) GetItems(c *gin.Context) {
	listID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		shared.BadRequest(c, "ID de liste invalide")
		return
	}
	query := strings.TrimSpace(c.Query("query"))

	items, err := h.repo.ListItems(listID, query)
	if err != nil {
		shared.InternalError(c, err.Error())
		return
	}
	shared.Success(c, http.StatusOK, gin.H{"items": items})
}

// Cancel annule une liste tant qu'elle n'a pas encore été dispatchée.
// DELETE /api/v1/inventory/send-lists/:id
func (h *SendListHandler) Cancel(c *gin.Context) {
	listID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || listID <= 0 {
		shared.BadRequest(c, "ID de liste invalide")
		return
	}

	list, err := h.repo.Cancel(listID)
	if err != nil {
		shared.BadRequest(c, err.Error())
		return
	}
	shared.Success(c, http.StatusOK, gin.H{"list": list})
}

// MarkProcessed clôt les listes dont le colis est préparé, une fois toutes les montures
// vérifiées au poste de scan.
// POST /api/v1/inventory/send-lists/processed
func (h *SendListHandler) MarkProcessed(c *gin.Context) {
	var req struct {
		IDs []int64 `json:"ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.BadRequest(c, "Données invalides")
		return
	}
	if len(req.IDs) == 0 {
		shared.BadRequest(c, "ids est requis")
		return
	}

	updated, err := h.repo.MarkProcessed(req.IDs)
	if err != nil {
		shared.InternalError(c, err.Error())
		return
	}
	shared.Success(c, http.StatusOK, gin.H{"updated": updated})
}

// Dispatch envoie les montures d'une liste vérifiée vers la station locale de sa ville : les
// montures atterrissent dans le stock de ce magasin (EN_STOCK_SOUS_STATION) et la liste est
// clôturée. Remplace l'appel à /processed, qui ne faisait que clore la liste.
//
// L'identifiant passe par le corps et non par l'URL, comme /seen et /processed : le groupe
// /send-lists a déjà des routes POST statiques, une route paramétrée au même niveau les
// mettrait en conflit dans l'arbre de routage.
// POST /api/v1/inventory/send-lists/dispatch
func (h *SendListHandler) Dispatch(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	var req struct {
		ID            int64 `json:"id"`
		FromStationID int64 `json:"from_station_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.BadRequest(c, "Données invalides")
		return
	}
	if req.ID <= 0 {
		shared.BadRequest(c, "id est requis")
		return
	}
	if req.FromStationID <= 0 {
		shared.BadRequest(c, "from_station_id est requis")
		return
	}

	result, err := h.dispatcher.Dispatch(req.ID, req.FromStationID, userID)
	if err != nil {
		shared.BadRequest(c, err.Error())
		return
	}

	shared.Success(c, http.StatusOK, gin.H{"dispatch": result})
}

// MarkSeen accuse réception côté poste de scan, pour que la notification ne se répète pas.
// POST /api/v1/inventory/send-lists/seen
func (h *SendListHandler) MarkSeen(c *gin.Context) {
	var req struct {
		IDs []int64 `json:"ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.BadRequest(c, "Données invalides")
		return
	}
	if len(req.IDs) == 0 {
		shared.BadRequest(c, "ids est requis")
		return
	}

	updated, err := h.repo.MarkSeen(req.IDs)
	if err != nil {
		shared.InternalError(c, err.Error())
		return
	}
	shared.Success(c, http.StatusOK, gin.H{"updated": updated})
}
