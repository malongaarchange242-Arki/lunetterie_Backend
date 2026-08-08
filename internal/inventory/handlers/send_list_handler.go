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

type sendListRepository interface {
	Create(list *models.SendList, items []models.SendListItemRequest) error
	List(status string) ([]models.SendList, error)
	ListItems(listID int64, query string) ([]models.SendListItem, error)
	MarkSeen(ids []int64) (int64, error)
}

// SendListHandler expose les listes d'envoi : la direction les crée depuis une session de
// réception terminée, le poste de scan les scrute pour prévenir le magasinier.
type SendListHandler struct {
	repo sendListRepository
}

func NewSendListHandler(repo sendListRepository) *SendListHandler {
	return &SendListHandler{repo: repo}
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
	if sessionCode == "" || city == "" {
		shared.BadRequest(c, "session_code et city sont requis")
		return
	}
	if len(req.Items) == 0 {
		shared.BadRequest(c, "une liste vide ne peut pas être envoyée")
		return
	}

	list := &models.SendList{SessionCode: sessionCode, City: city}
	if userID, exists := c.Get("user_id"); exists {
		if id, err := strconv.ParseInt(fmt.Sprintf("%v", userID), 10, 64); err == nil {
			list.CreatedBy = &id
		}
	}

	if err := h.repo.Create(list, req.Items); err != nil {
		shared.InternalError(c, "Erreur lors de l'enregistrement de la liste")
		return
	}

	shared.Created(c, gin.H{"list": list})
}

// List renvoie les listes, filtrables par statut.
// GET /api/v1/inventory/send-lists?status=NOUVELLE
func (h *SendListHandler) List(c *gin.Context) {
	status := strings.ToUpper(strings.TrimSpace(c.Query("status")))
	if status != "" && status != "NOUVELLE" && status != "VUE" && status != "TRAITEE" {
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
