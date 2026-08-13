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
	Create(proforma *models.Proforma, items []models.ProformaItem, prescription *models.ProformaPrescription) error
	List(status string) ([]models.Proforma, error)
	GetByID(id int64) (*models.Proforma, error)
	GetPrescription(proformaID int64) (*models.ProformaPrescription, error)
	SettleItem(proformaID, itemID int64, outcome string) (int64, error)
	CloseIfComplete(proformaID, userID int64) (string, error)
}

// Les valeurs d'ordonnance arrivent telles que la vendeuse les a tapées : « +1.00 »,
// « 60° », « -0,50 » avec une virgule, ou rien. On convertit ce qui se laisse convertir et
// on renvoie nil pour le reste — la note en clair, toujours enregistrée à côté, garde de
// toute façon la saisie d'origine. Refuser la proforma pour un caractère parasite serait
// la pire des réponses au comptoir.
func optionalDecimal(raw string) *float64 {
	cleaned := strings.TrimSpace(raw)
	cleaned = strings.NewReplacer(",", ".", "°", "", "+", "", " ", "").Replace(cleaned)
	if cleaned == "" {
		return nil
	}
	value, err := strconv.ParseFloat(cleaned, 64)
	if err != nil {
		return nil
	}
	return &value
}

// L'axe est un entier de 0 à 180 : hors bornes, la contrainte de la table rejetterait
// toute la transaction, donc on écarte la valeur ici plutôt que de perdre la proforma.
func optionalAxis(raw string) *int {
	value := optionalDecimal(raw)
	if value == nil {
		return nil
	}
	axis := int(*value)
	if axis < 0 || axis > 180 {
		return nil
	}
	return &axis
}

func optionalText(raw string) *string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

// optionalID traduit « aucun choix » en NULL. Zéro est ce que renvoie un champ non transmis,
// et l'écrire tel quel dans une clé étrangère ferait échouer l'enregistrement entier.
func optionalID(raw int64) *int64 {
	if raw <= 0 {
		return nil
	}
	return &raw
}

func buildPrescription(req *models.ProformaPrescriptionRequest) *models.ProformaPrescription {
	if req == nil {
		return nil
	}
	remise := req.RemisePct
	// La table borne la remise à 0–100 ; on la ramène plutôt que de faire échouer
	// l'enregistrement sur une saisie aberrante.
	if remise < 0 {
		remise = 0
	} else if remise > 100 {
		remise = 100
	}
	return &models.ProformaPrescription{
		Societe: optionalText(req.Societe),
		// Zéro n'est pas un identifiant : il signifie « aucune société choisie », et la
		// colonne est une clé étrangère — y écrire 0 ferait échouer toute la proforma.
		SocieteID: optionalID(req.SocieteID),
		Foyer:     optionalText(req.Foyer),
		Teinte:    optionalText(req.Teinte),

		ODSphere:   optionalDecimal(req.ODSphere),
		ODCylindre: optionalDecimal(req.ODCylindre),
		ODAxe:      optionalAxis(req.ODAxe),
		ODAddition: optionalDecimal(req.ODAddition),
		ODPrix:     req.ODPrix,

		OGSphere:   optionalDecimal(req.OGSphere),
		OGCylindre: optionalDecimal(req.OGCylindre),
		OGAxe:      optionalAxis(req.OGAxe),
		OGAddition: optionalDecimal(req.OGAddition),
		OGPrix:     req.OGPrix,

		AccessoiresLabel: optionalText(req.AccessoiresLabel),
		AccessoiresPrix:  req.AccessoiresPrix,
		MontagePrix:      req.MontagePrix,
		RemisePct:        remise,
		NoteLibre:        optionalText(req.NoteLibre),
	}
}

type proformaGlassRepository interface {
	GetByBarcode(barcode string) (*models.Glass, error)
	// La référence, la marque, la forme et la couleur ne sont pas sur glasses : elles
	// viennent de glass_analysis, que seule cette requête joint.
	FindDetailByBarcode(barcode string) (*models.GlassListItem, error)
}

// proformaDisplayService : les deux mouvements physiques qu'une proforma déclenche.
// SendToCaisse à l'émission, PlaceOnDisplay quand le client renonce.
type proformaDisplayService interface {
	SendToCaisse(stationID int64, barcodes []string, userID int64) ([]string, []services.SkippedBarcode, error)
	PlaceOnDisplay(barcode string, stationID, userID int64) (string, error)
}

type proformaSaleService interface {
	// Le second retour nomme les montures encaissées que l'envoi au Laboratoire n'a pas
	// emportées : elles restent VENDUE et n'arriveront jamais au montage.
	CreateSale(stationID int64, barcodes []string, userID int64) (*models.Sale, []string, error)
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

	// Deux formes acceptées : `lines` porte le geste « offerte », `barcodes` est l'ancien
	// envoi que les clients pas encore migrés utilisent toujours.
	lines := req.Lines
	if len(lines) == 0 {
		for _, raw := range req.Barcodes {
			lines = append(lines, models.ProformaLineRequest{Barcode: raw})
		}
	}
	if len(lines) == 0 {
		shared.BadRequest(c, "Sélectionnez au moins une monture")
		return
	}

	// Les attributs sont figés ici, à l'émission : la proforma doit rester lisible même si
	// la monture change de statut ou d'emplacement par la suite. Ils n'étaient en réalité
	// pas recopiés — seul le code-barres l'était — et une monture supprimée laissait une
	// ligne muette, à rebours de ce que la table promettait.
	items := make([]models.ProformaItem, 0, len(lines))
	barcodes := make([]string, 0, len(lines))
	for _, line := range lines {
		barcode := strings.TrimSpace(line.Barcode)
		if barcode == "" {
			continue
		}
		// FindDetailByBarcode et non GetByBarcode : la table glasses ne porte ni référence
		// ni marque ni forme ni couleur, elles viennent de glass_analysis par jointure.
		// C'est ce détour manquant qui laissait ces colonnes à NULL.
		glass, err := h.glassRepo.FindDetailByBarcode(barcode)
		if err != nil || glass == nil {
			shared.BadRequest(c, "Monture introuvable : "+barcode)
			return
		}
		item := models.ProformaItem{Barcode: &barcode, Offerte: line.Offerte}
		glassID := glass.ID
		item.GlassID = &glassID
		item.Reference = glass.Reference
		item.Brand = glass.Brand
		item.Shape = glass.Shape
		item.Color = glass.Color
		// Offerte : le prix tombe à zéro et la colonne dit pourquoi. Auparavant la case
		// n'était pas transmise du tout et la monture repartait à plein tarif en base.
		if glass.Price != nil && !line.Offerte {
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
	if err := h.repo.Create(proforma, items, buildPrescription(req.Prescription)); err != nil {
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
	// Les proformas antérieures à la 027 n'ont pas d'ordonnance en colonnes : nil, et le
	// document se relit alors depuis la note comme avant. Une erreur de lecture ne doit
	// pas cacher la proforma elle-même.
	if prescription, err := h.repo.GetPrescription(id); err == nil {
		proforma.Prescription = prescription
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
	//
	// `labFailures` nomme celles que l'expédition n'a pas emportées — station « Laboratoire »
	// absente, le plus souvent. La vente est bonne, mais ces montures resteront VENDUE sans
	// jamais arriver au montage : le caissier doit le lire, pas le découvrir trois jours après.
	var labFailures []string
	if len(soldBarcodes) > 0 {
		_, notShipped, err := h.saleSvc.CreateSale(proforma.StationID, soldBarcodes, userID)
		if err != nil {
			shared.InternalError(c, "Encaissement impossible : "+err.Error())
			return
		}
		labFailures = notShipped
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
		"lab_failures":    labFailures,
		"already_settled": alreadySettled,
	})
}
