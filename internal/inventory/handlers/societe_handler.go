package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/lunetterie/backend/internal/inventory/models"
	"github.com/lunetterie/backend/internal/shared"
)

type societeRepository interface {
	List(includeInactive bool) ([]models.Societe, error)
	Create(societe *models.Societe) error
	Update(id int64, req models.SocieteUpdateRequest) (*models.Societe, error)
}

type SocieteHandler struct {
	repo societeRepository
}

func NewSocieteHandler(repo societeRepository) *SocieteHandler {
	return &SocieteHandler{repo: repo}
}

// List sert la liste déroulante du champ « Société » d'une proforma. Ouverte à tout compte
// authentifié : c'est la vendeuse qui la consulte, et elle n'a que le droit de lire.
//
// `?include_inactive=true` ajoute les conventions terminées — réservé à l'écran de gestion,
// qui doit pouvoir les réactiver.
// GET /api/v1/inventory/societes
func (h *SocieteHandler) List(c *gin.Context) {
	includeInactive := strings.EqualFold(strings.TrimSpace(c.Query("include_inactive")), "true")

	societes, err := h.repo.List(includeInactive)
	if err != nil {
		shared.InternalError(c, err.Error())
		return
	}

	shared.Success(c, http.StatusOK, gin.H{"societes": societes})
}

// POST /api/v1/inventory/societes
func (h *SocieteHandler) Create(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	var req models.SocieteCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.BadRequest(c, "Données invalides: "+err.Error())
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		shared.BadRequest(c, "Le nom de la société est requis")
		return
	}

	societe := &models.Societe{Name: req.Name, CreatedBy: &userID}
	if contact := strings.TrimSpace(req.Contact); contact != "" {
		societe.Contact = &contact
	}
	if phone := strings.TrimSpace(req.Phone); phone != "" {
		societe.Phone = &phone
	}

	if err := h.repo.Create(societe); err != nil {
		shared.BadRequest(c, err.Error())
		return
	}

	shared.Success(c, http.StatusCreated, gin.H{"societe": societe})
}

// PUT /api/v1/inventory/societes/:id
func (h *SocieteHandler) Update(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		shared.BadRequest(c, "Identifiant de société invalide")
		return
	}

	var req models.SocieteUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.BadRequest(c, "Données invalides: "+err.Error())
		return
	}

	societe, err := h.repo.Update(id, req)
	if err != nil {
		shared.BadRequest(c, err.Error())
		return
	}

	shared.Success(c, http.StatusOK, gin.H{"societe": societe})
}
