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
}

type sendBoxStationRepository interface {
	GetByID(id int64) (*authModels.Station, error)
}

// SendBoxHandler expose les cartons expédiés vers un magasin. Le poste de stock interroge
// /pending à son ouverture pour savoir s'il attend un colis, puis /open avec le code-barres
// imprimé sur l'étiquette : c'est ce scan qui démarre la session de réception.
type SendBoxHandler struct {
	repo        sendBoxRepository
	stationRepo sendBoxStationRepository
}

func NewSendBoxHandler(repo sendBoxRepository, stationRepo sendBoxStationRepository) *SendBoxHandler {
	return &SendBoxHandler{repo: repo, stationRepo: stationRepo}
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

// Open démarre la session de réception : le magasinier scanne l'étiquette du carton, on
// vérifie qu'il est bien destiné à sa ville, puis on renvoie le contenu annoncé pour qu'il
// puisse le pointer monture par monture.
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

	station, city, ok := h.resolveStation(c, strconv.FormatInt(req.StationID, 10))
	if !ok {
		return
	}

	box, err := h.repo.FindBoxByCode(code)
	if err != nil {
		shared.NotFound(c, "Aucun carton ne correspond au code "+code+".")
		return
	}

	// Un carton scanné au mauvais magasin est une erreur d'acheminement, pas un problème
	// technique : on le dit explicitement plutôt que de laisser ouvrir le colis d'une autre ville.
	if !strings.EqualFold(strings.TrimSpace(box.City), city) {
		shared.BadRequest(c, "Ce carton est destiné à "+box.City+", pas à "+city+".")
		return
	}

	if box.Status == models.SendBoxStatusOpened {
		shared.BadRequest(c, "Ce carton a déjà été ouvert : la session correspondante est terminée.")
		return
	}

	affected, err := h.repo.MarkBoxOpened(box.ID, userID, station.ID)
	if err != nil {
		shared.InternalError(c, err.Error())
		return
	}
	// Course entre deux postes sur le même carton : le second scan ne doit pas croire qu'il a
	// ouvert la session.
	if affected == 0 {
		shared.BadRequest(c, "Ce carton vient d'être ouvert par un autre poste.")
		return
	}

	items, err := h.repo.FindBoxItems(box.ID)
	if err != nil {
		shared.InternalError(c, err.Error())
		return
	}

	box.Status = models.SendBoxStatusOpened
	shared.Success(c, http.StatusOK, gin.H{
		"box":   box,
		"items": items,
	})
}
