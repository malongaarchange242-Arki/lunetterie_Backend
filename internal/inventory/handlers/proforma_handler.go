package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/lunetterie/backend/internal/inventory/models"
	"github.com/lunetterie/backend/internal/inventory/services"
	"github.com/lunetterie/backend/internal/shared"
)

type proformaRepository interface {
	Create(proforma *models.Proforma, items []models.ProformaItem) error
	List(status string) ([]models.Proforma, error)
	GetByID(id int64) (*models.Proforma, error)
	SettleItem(proformaID, itemID int64, outcome string) (int64, error)
	CloseIfComplete(proformaID, userID int64) (string, error)
}

type proformaGlassRepository interface {
	GetByBarcode(barcode string) (*models.Glass, error)
}

// proformaDisplayService : les deux mouvements physiques qu'une proforma déclenche.
// SendToCaisse à l'émission, PlaceOnDisplay quand le client renonce.
type proformaDisplayService interface {
	SendToCaisse(stationID int64, barcodes []string, userID int64) ([]string, []services.SkippedBarcode, error)
	PlaceOnDisplay(barcode string, stationID, userID int64) (string, error)
}

type proformaSaleService interface {
	CreateSale(stationID int64, barcodes []string, userID int64) (*models.Sale, error)
}

// ProformaHandler expose le document émis au Présentoir quand un client choisit des
// montures, puis arbitré ligne par ligne à la Caisse.
type ProformaHandler struct {
	repo       proformaRepository
	glassRepo  proformaGlassRepository
	displaySvc proformaDisplayService
	saleSvc    proformaSaleService
}

func NewProformaHandler(repo proformaRepository, glassRepo proformaGlassRepository, displaySvc proformaDisplayService, saleSvc proformaSaleService) *ProformaHandler {
	return &ProformaHandler{repo: repo, glassRepo: glassRepo, displaySvc: displaySvc, saleSvc: saleSvc}
}

// Create émet la proforma et envoie les montures à la Caisse.
// POST /api/v1/inventory/proformas
func (h *ProformaHandler) Create(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	var req models.ProformaCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.BadRequest(c, "Données invalides: "+err.Error())
		return
	}

	clientName := strings.TrimSpace(req.ClientName)
	if clientName == "" {
		shared.BadRequest(c, "Le nom du client est requis")
		return
	}
	if len(req.Barcodes) == 0 {
		shared.BadRequest(c, "Sélectionnez au moins une monture")
		return
	}

	// Les attributs sont figés ici, à l'émission : la proforma doit rester lisible même si
	// la monture change de statut ou d'emplacement par la suite.
	items := make([]models.ProformaItem, 0, len(req.Barcodes))
	barcodes := make([]string, 0, len(req.Barcodes))
	for _, raw := range req.Barcodes {
		barcode := strings.TrimSpace(raw)
		if barcode == "" {
			continue
		}
		glass, err := h.glassRepo.GetByBarcode(barcode)
		if err != nil {
			shared.BadRequest(c, "Monture introuvable : "+barcode)
			return
		}
		item := models.ProformaItem{Barcode: &barcode}
		glassID := glass.ID
		item.GlassID = &glassID
		if glass.Price != nil {
			item.UnitPrice = *glass.Price
		}
		items = append(items, item)
		barcodes = append(barcodes, barcode)
	}
	if len(items) == 0 {
		shared.BadRequest(c, "Sélectionnez au moins une monture")
		return
	}

	proforma := &models.Proforma{
		StationID:  req.StationID,
		ClientName: clientName,
		CreatedBy:  &userID,
	}
	if phone := strings.TrimSpace(req.ClientPhone); phone != "" {
		proforma.ClientPhone = &phone
	}
	if note := strings.TrimSpace(req.Note); note != "" {
		proforma.Note = &note
	}

	// L'enregistrement passe avant le déplacement : c'est l'index unique sur les lignes en
	// attente qui garantit qu'une monture n'est pas promise à deux clients. Déplacer d'abord
	// puis échouer ici laisserait des montures en caisse sans document.
	if err := h.repo.Create(proforma, items); err != nil {
		shared.BadRequest(c, err.Error())
		return
	}

	sent, skipped, err := h.displaySvc.SendToCaisse(req.StationID, barcodes, userID)
	if err != nil {
		// La proforma existe et les montures sont bloquées : la Caisse peut arbitrer même si
		// le déplacement a échoué. On le signale sans annuler le document.
		shared.Success(c, http.StatusCreated, gin.H{
			"proforma": proforma,
			"sent":     sent,
			"skipped":  skipped,
			"warning":  "Proforma enregistrée, mais l'envoi en caisse a échoué : " + err.Error(),
		})
		return
	}

	shared.Success(c, http.StatusCreated, gin.H{"proforma": proforma, "sent": sent, "skipped": skipped})
}

// List alimente l'écran de la Caisse.
// GET /api/v1/inventory/proformas?status=EN_ATTENTE
func (h *ProformaHandler) List(c *gin.Context) {
	proformas, err := h.repo.List(strings.TrimSpace(c.Query("status")))
	if err != nil {
		shared.InternalError(c, err.Error())
		return
	}
	shared.Success(c, http.StatusOK, gin.H{"proformas": proformas})
}

// GET /api/v1/inventory/proformas/:id
func (h *ProformaHandler) Get(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		shared.BadRequest(c, "Identifiant de proforma invalide")
		return
	}
	proforma, err := h.repo.GetByID(id)
	if err != nil {
		shared.NotFound(c, "Proforma introuvable")
		return
	}
	shared.Success(c, http.StatusOK, gin.H{"proforma": proforma})
}

// Settle applique les décisions de la Caisse, ligne par ligne : le client peut garder une
// paire et renoncer à l'autre.
// POST /api/v1/inventory/proformas/:id/settle
func (h *ProformaHandler) Settle(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		shared.BadRequest(c, "Identifiant de proforma invalide")
		return
	}

	var req models.ProformaSettleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.BadRequest(c, "Données invalides: "+err.Error())
		return
	}
	if len(req.Decisions) == 0 {
		shared.BadRequest(c, "Aucune décision transmise")
		return
	}

	proforma, err := h.repo.GetByID(id)
	if err != nil {
		shared.NotFound(c, "Proforma introuvable")
		return
	}

	itemsByID := make(map[int64]models.ProformaItem, len(proforma.Items))
	for _, item := range proforma.Items {
		itemsByID[item.ID] = item
	}

	var soldBarcodes, returnedBarcodes []string
	var alreadySettled []int64
	for _, decision := range req.Decisions {
		if decision.Outcome != models.ProformaOutcomeVendue && decision.Outcome != models.ProformaOutcomeRetour {
			shared.BadRequest(c, "Décision inconnue : "+decision.Outcome)
			return
		}
		item, exists := itemsByID[decision.ItemID]
		if !exists {
			shared.BadRequest(c, "Ligne absente de cette proforma")
			return
		}

		affected, err := h.repo.SettleItem(id, decision.ItemID, decision.Outcome)
		if err != nil {
			shared.InternalError(c, err.Error())
			return
		}
		// Zéro ligne touchée : un autre caissier vient de trancher celle-ci. On ne rejoue pas
		// le mouvement physique, sinon la monture partirait deux fois.
		if affected == 0 {
			alreadySettled = append(alreadySettled, decision.ItemID)
			continue
		}
		if item.Barcode == nil || *item.Barcode == "" {
			continue
		}
		if decision.Outcome == models.ProformaOutcomeVendue {
			soldBarcodes = append(soldBarcodes, *item.Barcode)
		} else {
			returnedBarcodes = append(returnedBarcodes, *item.Barcode)
		}
	}

	// Une seule vente pour toutes les lignes encaissées : CreateSale marque VENDUE puis
	// expédie au Laboratoire, c'est le chemin « la lunette va au laboratoire ».
	if len(soldBarcodes) > 0 {
		if _, err := h.saleSvc.CreateSale(proforma.StationID, soldBarcodes, userID); err != nil {
			shared.InternalError(c, "Encaissement impossible : "+err.Error())
			return
		}
	}

	// Le client a renoncé : la monture retourne au Présentoir qui l'a proposée et y reprend
	// un emplacement. Chaque monture est traitée isolément — une qui échoue (plus d'emplacement
	// libre) ne doit pas empêcher les suivantes de repartir.
	var returnFailures []string
	for _, barcode := range returnedBarcodes {
		if _, err := h.displaySvc.PlaceOnDisplay(barcode, proforma.StationID, userID); err != nil {
			returnFailures = append(returnFailures, barcode)
		}
	}

	status, err := h.repo.CloseIfComplete(id, userID)
	if err != nil {
		shared.InternalError(c, err.Error())
		return
	}

	updated, err := h.repo.GetByID(id)
	if err != nil {
		shared.InternalError(c, err.Error())
		return
	}

	shared.Success(c, http.StatusOK, gin.H{
		"proforma":        updated,
		"status":          status,
		"sold":            soldBarcodes,
		"returned":        returnedBarcodes,
		"return_failures": returnFailures,
		"already_settled": alreadySettled,
	})
}
