package handlers

import (
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lunetterie/backend/internal/inventory/models"
	"github.com/lunetterie/backend/internal/inventory/repositories"
	"github.com/lunetterie/backend/internal/inventory/services"
	"github.com/lunetterie/backend/internal/shared"
)

type PreRegistrationHandler struct {
	repo       *repositories.PreRegistrationRepository
	storageSvc *services.StorageService
}

func NewPreRegistrationHandler(repo *repositories.PreRegistrationRepository, storageSvc *services.StorageService) *PreRegistrationHandler {
	return &PreRegistrationHandler{repo: repo, storageSvc: storageSvc}
}

func (h *PreRegistrationHandler) ListCases(c *gin.Context) {
	cases, err := h.repo.ListCases(c.Param("code"))
	if err != nil {
		shared.NotFound(c, err.Error())
		return
	}
	shared.Success(c, http.StatusOK, gin.H{"cases": cases})
}

func (h *PreRegistrationHandler) CreateCase(c *gin.Context) {
	var req models.PreRegistrationCaseRequest
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Couleur) == "" || req.Montures < 1 {
		shared.BadRequest(c, "couleur, gamme, genre et montures valides sont requis")
		return
	}
	caseCode, err := h.repo.NextCaseCode()
	if err != nil {
		shared.InternalError(c, "Impossible de generer le code valise: "+err.Error())
		return
	}
	item := models.PreRegistrationCase{
		Code: caseCode, Couleur: strings.TrimSpace(req.Couleur),
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
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.BadRequest(c, "payload carton invalide")
		return
	}
	if req.Quantity < 1 && req.QuantityAlt > 0 {
		req.Quantity = req.QuantityAlt
	}
	if req.Quantity < 1 || len(req.Formes) == 0 {
		shared.BadRequest(c, "quantite et formes valides sont requises")
		return
	}
	item := models.PreRegistrationBox{
		Code:     req.Code,
		Quantity: req.Quantity,
		Formes:   req.Formes,
		Marques:  req.Marques,
		Couleurs: req.Couleurs,
		Matieres: req.Matieres,
		Photos:   req.Photos,
		Gamme:    strings.TrimSpace(req.Gamme),
		Type:     strings.TrimSpace(req.Type),
		Prix:     req.Prix,
	}
	if strings.TrimSpace(item.Code) == "" {
		boxCode, err := h.repo.NextBoxCode()
		if err != nil {
			shared.InternalError(c, "Impossible de generer le code carton: "+err.Error())
			return
		}
		item.Code = boxCode
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

func sanitizeStorageToken(value string) string {
	replacer := strings.NewReplacer("/", "-", "\\", "-", " ", "-", ".", "-")
	clean := replacer.Replace(strings.TrimSpace(value))
	clean = strings.Trim(clean, "-_ ")
	if clean == "" {
		return "photo"
	}
	return clean
}

func buildPhotoStoragePath(caseCode, boxCode, kind, photoID, fileName string) string {
	kindToken := sanitizeStorageToken(kind)
	if kindToken == "" {
		kindToken = "photo"
	}
	fileBase := sanitizeStorageToken(fileName)
	if fileBase == "photo" || fileBase == "" {
		fileBase = "image"
	}
	seed := strings.TrimSpace(photoID)
	if seed == "" {
		seed = strconv.FormatInt(time.Now().UnixNano(), 10)
	}
	return "pre-registration/" + strings.TrimSpace(caseCode) + "/" + strings.TrimSpace(boxCode) + "/" + seed + "-" + kindToken + "-" + fileBase + ".jpg"
}

func (h *PreRegistrationHandler) UploadBoxPhoto(c *gin.Context) {
	if h.storageSvc == nil {
		shared.InternalError(c, "service de stockage non initialisé")
		return
	}
	caseCode := strings.TrimSpace(c.PostForm("caseCode"))
	boxCode := strings.TrimSpace(c.PostForm("boxCode"))
	kind := strings.TrimSpace(c.PostForm("kind"))
	photoID := strings.TrimSpace(c.PostForm("photoId"))
	fileName := strings.TrimSpace(c.PostForm("fileName"))
	if caseCode == "" || boxCode == "" || kind == "" {
		shared.BadRequest(c, "caseCode, boxCode et kind sont requis")
		return
	}
	file, header, err := c.Request.FormFile("image")
	if err != nil {
		shared.BadRequest(c, "image requise")
		return
	}
	defer file.Close()
	data, err := io.ReadAll(file)
	if err != nil {
		shared.BadRequest(c, "image illisible")
		return
	}
	path := buildPhotoStoragePath(caseCode, boxCode, kind, photoID, fileName)
	url, err := h.storageSvc.Upload(path, data, header.Header.Get("Content-Type"))
	if err != nil {
		shared.InternalError(c, "Impossible d'uploader la photo: "+err.Error())
		return
	}
	shared.Success(c, http.StatusOK, gin.H{"kind": kind, "url": url, "path": path})
}

func (h *PreRegistrationHandler) UpdateBoxPhotos(c *gin.Context) {
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
	var req struct {
		Photos []models.PreRegistrationPhoto `json:"photos"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.BadRequest(c, "payload invalide")
		return
	}
	if err := h.repo.UpdateBoxPhotos(caseID, boxID, req.Photos); err != nil {
		shared.InternalError(c, "Impossible de sauvegarder les photos du carton: "+err.Error())
		return
	}
	box, err := h.repo.GetBox(caseID, boxID)
	if err != nil {
		shared.InternalError(c, "Impossible de récupérer le carton mis à jour")
		return
	}
	shared.Success(c, http.StatusOK, gin.H{"box": box})
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

func (h *PreRegistrationHandler) OpenBox(c *gin.Context) {
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
	if err := h.repo.MarkCaseOpened(caseID); err != nil {
		shared.InternalError(c, "Impossible d'ouvrir le carton: "+err.Error())
		return
	}
	shared.Success(c, http.StatusOK, gin.H{"opened": true, "case_id": caseID, "box_id": boxID})
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
