package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/lunetterie/backend/internal/inventory/models"
	"github.com/lunetterie/backend/internal/inventory/repositories"
	"github.com/lunetterie/backend/internal/shared"
)

type PreRegistrationHandler struct {
	repo *repositories.PreRegistrationRepository
}

func NewPreRegistrationHandler(repo *repositories.PreRegistrationRepository) *PreRegistrationHandler {
	return &PreRegistrationHandler{repo: repo}
}

func (h *PreRegistrationHandler) ListCases(c *gin.Context) {
	cases, err := h.repo.ListCases(c.Param("code"))
	if err != nil {
		shared.NotFound(c, err.Error())
		return
	}
	shared.Success(c, http.StatusOK, gin.H{"cases": cases})
}

func mustGenerateCaseCode(repo *repositories.PreRegistrationRepository) string {
	code, err := repo.NextCaseCode()
	if err != nil {
		panic(err)
	}
	return code
}
func mustGenerateBoxCode(repo *repositories.PreRegistrationRepository) string {
	code, err := repo.NextBoxCode()
	if err != nil {
		panic(err)
	}
	return code
}
func (h *PreRegistrationHandler) CreateCase(c *gin.Context) {
	var req models.PreRegistrationCaseRequest
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Couleur) == "" || req.Montures < 1 {
		shared.BadRequest(c, "couleur, gamme, genre et montures valides sont requis")
		return
	}
	item := models.PreRegistrationCase{
		Code: mustGenerateCaseCode(h.repo), Couleur: strings.TrimSpace(req.Couleur),
		Hex: strings.TrimSpace(req.Hex), Gamme: strings.TrimSpace(req.Gamme), Genre: strings.TrimSpace(req.Genre), Montures: req.Montures,
	}
	if err := h.repo.CreateCase(c.Param("code"), item); err != nil {
		shared.InternalError(c, "Impossible de créer la valise: "+err.Error())
		return
	}
	created, err := h.repo.ListCases(c.Param("code"))
	if err != nil || len(created) == 0 {
		shared.InternalError(c, "Impossible de récupérer la valise créée")
		return
	}
	shared.Created(c, gin.H{"case": created[len(created)-1], "barcode_image_url": "/api/v1/inventory/labels/" + created[len(created)-1].Code + ".png"})
}

func (h *PreRegistrationHandler) CreateBox(c *gin.Context) {
	caseID, err := strconv.ParseInt(c.Param("caseId"), 10, 64)
	if err != nil {
		shared.BadRequest(c, "identifiant de valise invalide")
		return
	}
	var req models.PreRegistrationBoxRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Quantity < 1 || len(req.Formes) == 0 {
		shared.BadRequest(c, "code, quantité et formes valides sont requis")
		return
	}
	item := models.PreRegistrationBox{
		Code: mustGenerateBoxCode(h.repo), Quantity: req.Quantity, Formes: req.Formes,
		Marques: req.Marques, Couleurs: req.Couleurs, Matieres: req.Matieres,
		Gamme: strings.TrimSpace(req.Gamme), Type: strings.TrimSpace(req.Type), Prix: req.Prix,
	}
	if err := h.repo.CreateBox(caseID, item); err != nil {
		shared.InternalError(c, "Impossible de créer le carton: "+err.Error())
		return
	}
	created, err := h.repo.GetCase(caseID)
	if err != nil {
		shared.InternalError(c, "Impossible de récupérer le carton créé")
		return
	}
	shared.Created(c, gin.H{"case": created, "barcode_image_url": "/api/v1/inventory/labels/" + created.Code + ".png"})
}

func (h *PreRegistrationHandler) ScanBox(c *gin.Context) {
	caseID, err := strconv.ParseInt(c.Param("caseId"), 10, 64)
	if err != nil {
		shared.BadRequest(c, "identifiant de valise invalide")
		return
	}
	boxID, err := strconv.ParseInt(c.Param("boxId"), 10, 64)
	if err != nil {
		shared.BadRequest(c, "identifiant de carton invalide")
		return
	}
	box, err := h.repo.GetBox(caseID, boxID)
	if err != nil {
		shared.NotFound(c, "Carton introuvable dans cette valise")
		return
	}
	shared.Success(c, http.StatusOK, gin.H{"carton": box})
}

func (h *PreRegistrationHandler) DeleteCase(c *gin.Context) {
	caseID, err := strconv.ParseInt(c.Param("caseId"), 10, 64)
	if err != nil {
		shared.BadRequest(c, "identifiant de valise invalide")
		return
	}
	if err := h.repo.DeleteCase(caseID); err != nil {
		shared.BadRequest(c, err.Error())
		return
	}
	shared.Success(c, http.StatusOK, gin.H{"deleted": true, "case_id": caseID})
}

func (h *PreRegistrationHandler) DeleteBox(c *gin.Context) {
	boxID, err := strconv.ParseInt(c.Param("boxId"), 10, 64)
	if err != nil {
		shared.BadRequest(c, "identifiant de carton invalide")
		return
	}
	if err := h.repo.DeleteBox(boxID); err != nil {
		shared.BadRequest(c, err.Error())
		return
	}
	shared.Success(c, http.StatusOK, gin.H{"deleted": true, "box_id": boxID})
}
