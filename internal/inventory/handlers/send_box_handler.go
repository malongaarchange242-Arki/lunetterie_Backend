package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	authModels "github.com/lunetterie/backend/internal/auth/models"
	"github.com/lunetterie/backend/internal/inventory/models"
	"github.com/lunetterie/backend/internal/shared"
)

type sendBoxRepository interface {
	ListBoxes(status, city string) ([]models.SendBox, error)
	RestockSuggestions() ([]models.RestockSuggestion, error)
	FindPendingBoxesByCity(city string) ([]models.SendBox, error)
	FindBoxByCode(code string) (*models.SendBox, error)
	FindBoxItems(boxID int64) ([]models.SendBoxItem, error)
	MarkBoxOpened(boxID, userID, stationID int64) (int64, error)
	MarkBoxClosed(boxID, userID int64, missing int) (int64, error)
}

type sendBoxStationRepository interface {
	GetByID(id int64) (*authModels.Station, error)
}

// sendBoxTransferService : le carton ne tient pas son propre état de réception, il le lit dans
// le transfert qu'il transporte. Une monture reçue est une ligne de transfert passée RECEIVED.
type sendBoxTransferService interface {
	ListItems(transferID int64) ([]models.TransferItem, error)
	ReceiveItem(transferID int64, barcode string, userID int64) (*models.TransferItem, *models.Glass, *models.StorageLocation, *models.Transfer, error)
}

// sendBoxItemView : une ligne du carton, augmentée de ce que le pointage en sait. `Received`
// vient du transfert, pas du navigateur : deux postes peuvent pointer le même carton, et une
// reprise après rechargement doit retrouver l'avancement réel.
type sendBoxItemView struct {
	models.SendBoxItem
	Received bool `json:"received"`
}

// SendBoxHandler expose les cartons expédiés vers un magasin. Le poste de stock interroge
// /pending à son ouverture pour savoir s'il attend un colis, puis /open avec le code-barres
// imprimé sur l'étiquette : c'est ce scan qui démarre la session de réception.
type SendBoxHandler struct {
	repo        sendBoxRepository
	stationRepo sendBoxStationRepository
	transfers   sendBoxTransferService
}

func NewSendBoxHandler(repo sendBoxRepository, stationRepo sendBoxStationRepository, transfers sendBoxTransferService) *SendBoxHandler {
	return &SendBoxHandler{repo: repo, stationRepo: stationRepo, transfers: transfers}
}

// boxItemViews rapproche le contenu figé du carton de l'état réel du transfert : quelles
// montures sont déjà entrées en stock, et combien il en reste à pointer.
//
// Un carton antérieur au suivi de transit n'a pas de transfert : son contenu est déjà en stock
// depuis l'expédition. On le rend alors intégralement « reçu », faute de quoi le magasinier
// chercherait à pointer des montures que plus rien n'attend.
func (h *SendBoxHandler) boxItemViews(box *models.SendBox) ([]sendBoxItemView, int, error) {
	items, err := h.repo.FindBoxItems(box.ID)
	if err != nil {
		return nil, 0, err
	}

	legacy := box.TransferID == nil
	received := map[int64]bool{}
	if !legacy {
		lines, err := h.transfers.ListItems(*box.TransferID)
		if err != nil {
			return nil, 0, err
		}
		for _, line := range lines {
			if line.Status == models.TransferItemStatusReceived {
				received[line.GlassID] = true
			}
		}
	}

	views := make([]sendBoxItemView, 0, len(items))
	count := 0
	for _, item := range items {
		done := legacy || (item.GlassID != nil && received[*item.GlassID])
		if done {
			count++
		}
		views = append(views, sendBoxItemView{SendBoxItem: item, Received: done})
	}
	return views, count, nil
}

// requireBoxForStation applique les deux gardes communes à toute action sur un carton :
// le code désigne bien un carton, et ce carton est destiné à la ville du poste.
func (h *SendBoxHandler) requireBoxForStation(c *gin.Context, code, rawStationID string) (*models.SendBox, *authModels.Station, bool) {
	station, city, ok := h.resolveStation(c, rawStationID)
	if !ok {
		return nil, nil, false
	}

	box, err := h.repo.FindBoxByCode(code)
	if err != nil {
		shared.NotFound(c, "Aucun carton ne correspond au code "+code+".")
		return nil, nil, false
	}

	// Un carton scanné au mauvais magasin est une erreur d'acheminement, pas un problème
	// technique : on le dit explicitement plutôt que de laisser ouvrir le colis d'une autre ville.
	if !strings.EqualFold(strings.TrimSpace(box.City), city) {
		shared.BadRequest(c, "Ce carton est destiné à "+box.City+", pas à "+city+".")
		return nil, nil, false
	}
	return box, station, true
}

// resolveStation traduit le station_id de la requête en station réelle et en ville. La ville
// n'est jamais prise du client : un poste ne doit pouvoir ouvrir que les cartons destinés à
// la sienne. Renvoie la ville à part car elle est facultative en base (*string).
func (h *SendBoxHandler) resolveStation(c *gin.Context, rawStationID string) (*authModels.Station, string, bool) {
	stationID, err := strconv.ParseInt(strings.TrimSpace(rawStationID), 10, 64)
	if err != nil || stationID <= 0 {
		shared.BadRequest(c, "station_id est requis")
		return nil, "", false
	}
	station, err := h.stationRepo.GetByID(stationID)
	if err != nil {
		shared.NotFound(c, "Station introuvable")
		return nil, "", false
	}
	city := ""
	if station.City != nil {
		city = strings.TrimSpace(*station.City)
	}
	if city == "" {
		shared.BadRequest(c, "La station "+station.Name+" n'a pas de ville renseignée : impossible de savoir quels cartons lui sont destinés.")
		return nil, "", false
	}
	return station, city, true
}

// List suit tous les cartons, toutes villes confondues : c'est la vue Expédition, qui doit
// voir ce qui est parti et ce qui n'a pas encore été ouvert à l'arrivée.
// GET /api/v1/inventory/send-boxes?status=CREATED&city=Pointe-Noire
func (h *SendBoxHandler) List(c *gin.Context) {
	boxes, err := h.repo.ListBoxes(strings.TrimSpace(c.Query("status")), strings.TrimSpace(c.Query("city")))
	if err != nil {
		shared.InternalError(c, err.Error())
		return
	}
	shared.Success(c, http.StatusOK, gin.H{"boxes": boxes})
}

// Restock indique, pour chaque magasin déjà livré, combien de montures lui renvoyer pour le
// remettre au niveau de sa dernière livraison, et lesquels sont sous le seuil d'alerte.
// GET /api/v1/inventory/send-boxes/restock
func (h *SendBoxHandler) Restock(c *gin.Context) {
	suggestions, err := h.repo.RestockSuggestions()
	if err != nil {
		shared.InternalError(c, err.Error())
		return
	}
	shared.Success(c, http.StatusOK, gin.H{
		"suggestions": suggestions,
		"threshold":   models.RestockAlertRatio,
	})
}

// Pending liste les cartons partis vers la ville du poste et pas encore ouverts.
// GET /api/v1/inventory/send-boxes/pending?station_id=3
func (h *SendBoxHandler) Pending(c *gin.Context) {
	station, city, ok := h.resolveStation(c, c.Query("station_id"))
	if !ok {
		return
	}

	boxes, err := h.repo.FindPendingBoxesByCity(city)
	if err != nil {
		shared.InternalError(c, err.Error())
		return
	}

	shared.Success(c, http.StatusOK, gin.H{
		"boxes":        boxes,
		"pending":      len(boxes),
		"city":         city,
		"station_name": station.Name,
	})
}

type openSendBoxRequest struct {
	Code      string `json:"code"`
	StationID int64  `json:"station_id"`
}

// Open démarre — ou reprend — la session de réception : le magasinier scanne l'étiquette du
// carton, on vérifie qu'il est bien destiné à sa ville, puis on renvoie le contenu annoncé avec
// l'avancement du pointage.
//
// L'appel est volontairement reprenable. L'ouverture marquait autrefois le carton OPENED une
// fois pour toutes, et aucune route ne permettait de relire son contenu : une page fermée au
// milieu du pointage condamnait le carton. Un carton déjà ouvert se rouvre donc, sans écraser
// l'identité du premier réceptionnaire — MarkBoxOpened ne touche que les cartons non ouverts.
// Seule la clôture est définitive.
// POST /api/v1/inventory/send-boxes/open
func (h *SendBoxHandler) Open(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	var req openSendBoxRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.BadRequest(c, "Données invalides")
		return
	}

	code := strings.TrimSpace(req.Code)
	if code == "" {
		shared.BadRequest(c, "Le code du carton est requis")
		return
	}

	box, station, ok := h.requireBoxForStation(c, code, strconv.FormatInt(req.StationID, 10))
	if !ok {
		return
	}

	if box.Status == models.SendBoxStatusClosed {
		// La date n'est pas garantie : `closed_at` a été ajoutée après coup, et un carton clos
		// par une version antérieure n'en a pas. Le refus doit tenir sans elle.
		message := "Ce carton a été clôturé : son pointage est terminé."
		if box.ClosedAt != nil {
			message = "Ce carton a été clôturé le " + box.ClosedAt.Format("02/01/2006") + " : son pointage est terminé."
		}
		shared.BadRequest(c, message)
		return
	}

	if box.Status == models.SendBoxStatusCreated {
		if _, err := h.repo.MarkBoxOpened(box.ID, userID, station.ID); err != nil {
			shared.InternalError(c, err.Error())
			return
		}
		box.Status = models.SendBoxStatusOpened
	}

	views, receivedCount, err := h.boxItemViews(box)
	if err != nil {
		shared.InternalError(c, err.Error())
		return
	}

	shared.Success(c, http.StatusOK, gin.H{
		"box":            box,
		"items":          views,
		"received_count": receivedCount,
		"expected_count": len(views),
	})
}

type receiveSendBoxRequest struct {
	Code      string `json:"code"`
	Barcode   string `json:"barcode"`
	StationID int64  `json:"station_id"`
}

// Receive pointe une monture du carton : le scan de son code-barres clôt sa ligne de transfert,
// lui alloue un emplacement dans le magasin et la fait passer EN_STOCK_SOUS_STATION.
//
// C'est ici, et seulement ici, que la monture entre réellement au stock. Une monture jamais
// scannée reste EN_TRANSIT : elle n'est comptée nulle part, ce qui est exactement ce qu'on veut
// d'un colis incomplet.
// POST /api/v1/inventory/send-boxes/receive
func (h *SendBoxHandler) Receive(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	var req receiveSendBoxRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.BadRequest(c, "Données invalides")
		return
	}

	code := strings.TrimSpace(req.Code)
	barcode := strings.TrimSpace(req.Barcode)
	if code == "" || barcode == "" {
		shared.BadRequest(c, "Le code du carton et le code-barres de la monture sont requis")
		return
	}

	box, _, ok := h.requireBoxForStation(c, code, strconv.FormatInt(req.StationID, 10))
	if !ok {
		return
	}

	if box.Status != models.SendBoxStatusOpened {
		shared.BadRequest(c, "Scannez d'abord l'étiquette du carton pour ouvrir son pointage.")
		return
	}
	// Carton parti avant le suivi de transit : son contenu est entré en stock à l'expédition,
	// il n'y a plus rien à recevoir.
	if box.TransferID == nil {
		shared.BadRequest(c, "Ce carton est antérieur au suivi de transit : son contenu est déjà en stock.")
		return
	}

	item, glass, location, transfer, err := h.transfers.ReceiveItem(*box.TransferID, barcode, userID)
	if err != nil {
		// Monture absente du carton, déjà pointée, introuvable : autant de refus métier que le
		// magasinier doit lire tels quels, pas une erreur 500.
		shared.BadRequest(c, err.Error())
		return
	}

	_, receivedCount, err := h.boxItemViews(box)
	if err != nil {
		shared.InternalError(c, err.Error())
		return
	}

	shared.Success(c, http.StatusOK, gin.H{
		"item":           item,
		"glass":          glass,
		"location":       location,
		"transfer":       transfer,
		"received_count": receivedCount,
		"expected_count": box.ItemCount,
	})
}

type closeSendBoxRequest struct {
	Code      string `json:"code"`
	StationID int64  `json:"station_id"`
}

// Close clôt le pointage, y compris incomplet.
//
// Un carton peut arriver amputé — c'est le quotidien de la logistique. On préfère donc acter
// l'arrivée et consigner l'écart plutôt que de laisser un carton ouvert indéfiniment parce
// qu'une monture s'est perdue. Les manquantes restent EN_TRANSIT : elles n'entrent pas au stock
// du magasin, et `missing_count` fige l'écart pour le litige.
// POST /api/v1/inventory/send-boxes/close
func (h *SendBoxHandler) Close(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	var req closeSendBoxRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.BadRequest(c, "Données invalides")
		return
	}

	code := strings.TrimSpace(req.Code)
	if code == "" {
		shared.BadRequest(c, "Le code du carton est requis")
		return
	}

	box, _, ok := h.requireBoxForStation(c, code, strconv.FormatInt(req.StationID, 10))
	if !ok {
		return
	}
	if box.Status == models.SendBoxStatusClosed {
		shared.BadRequest(c, "Ce carton est déjà clôturé.")
		return
	}
	if box.Status != models.SendBoxStatusOpened {
		shared.BadRequest(c, "Scannez d'abord l'étiquette du carton pour ouvrir son pointage.")
		return
	}

	views, receivedCount, err := h.boxItemViews(box)
	if err != nil {
		shared.InternalError(c, err.Error())
		return
	}
	missing := len(views) - receivedCount
	if missing < 0 {
		missing = 0
	}

	if _, err := h.repo.MarkBoxClosed(box.ID, userID, missing); err != nil {
		shared.InternalError(c, err.Error())
		return
	}

	// La liste des manquantes part avec la réponse : c'est le seul moment où le magasinier peut
	// encore les nommer, et c'est ce qu'il transmettra au transporteur.
	missingItems := make([]sendBoxItemView, 0, missing)
	for _, view := range views {
		if !view.Received {
			missingItems = append(missingItems, view)
		}
	}

	box.Status = models.SendBoxStatusClosed
	box.MissingCount = missing
	shared.Success(c, http.StatusOK, gin.H{
		"box":            box,
		"received_count": receivedCount,
		"expected_count": len(views),
		"missing_count":  missing,
		"missing":        missingItems,
	})
}
