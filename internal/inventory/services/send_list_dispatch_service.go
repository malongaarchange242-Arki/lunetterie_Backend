package services

import (
	"fmt"
	"strings"

	authModels "github.com/lunetterie/backend/internal/auth/models"
	authRepositories "github.com/lunetterie/backend/internal/auth/repositories"
	"github.com/lunetterie/backend/internal/inventory/models"
	"github.com/lunetterie/backend/internal/inventory/repositories"
)

// SendListDispatchService expédie une liste reçue (send_list) vers la station locale de sa
// ville, en une seule action déclenchée par le bouton « Envoyer » du poste de scan.
//
// Contrairement au transfert manuel, la destination n'est pas choisie dans un menu : elle se
// déduit de la ville portée par la liste. Mais le voyage, lui, est un vrai transfert — les
// montures partent EN_TRANSIT et n'entrent dans le stock du magasin qu'au pointage du carton
// à l'arrivée (SendBoxHandler.Receive).
//
// Il en allait autrement jusqu'ici : l'envoi valait arrivée, les montures passaient
// EN_STOCK_SOUS_STATION au moment même du départ. Un carton perdu en route laissait donc le
// magasin crédité de montures qu'il n'avait jamais reçues, sans aucun moyen de s'en apercevoir.
type SendListDispatchService struct {
	sendListRepo *repositories.SendListRepository
	glassRepo    *repositories.GlassRepository
	stationRepo  *authRepositories.StationRepository
	// Le circuit de transit est déjà écrit et éprouvé (préparation → expédition → réception
	// monture par monture) : on s'y branche au lieu d'en tenir une seconde version ici. C'est
	// lui qui détient désormais les emplacements et les mouvements — d'où la disparition des
	// dépendances `allocation` et `movementRepo`, que ce service manipulait en direct du temps
	// où il jouait l'arrivée lui-même.
	transfers *TransferService
}

func NewSendListDispatchService(
	sendListRepo *repositories.SendListRepository,
	glassRepo *repositories.GlassRepository,
	stationRepo *authRepositories.StationRepository,
	transfers *TransferService,
) *SendListDispatchService {
	return &SendListDispatchService{
		sendListRepo: sendListRepo,
		glassRepo:    glassRepo,
		stationRepo:  stationRepo,
		transfers:    transfers,
	}
}

// SendListSkippedItem : une monture de la liste qui n'a pas pu partir. Le colis part quand
// même avec les autres — mais le magasinier doit savoir laquelle manque et pourquoi.
type SendListSkippedItem struct {
	Reference string `json:"reference"`
	Reason    string `json:"reason"`
}

// SendListDispatchResult résume ce qui est réellement parti.
type SendListDispatchResult struct {
	StationID    int64                 `json:"station_id"`
	StationName  string                `json:"station_name"`
	City         string                `json:"city"`
	SentCount    int                   `json:"sent_count"`
	Status       string                `json:"status"`
	Skipped      []SendListSkippedItem `json:"skipped"`
	BoxID        int64                 `json:"box_id,omitempty"`
	BoxCode      string                `json:"box_code,omitempty"`
	BoxReference string                `json:"box_reference,omitempty"`
	BoxItemCount int                   `json:"box_item_count,omitempty"`
	TransferID   int64                 `json:"transfer_id,omitempty"`
}

// Dispatch envoie toutes les montures d'une liste vers la station locale de sa ville.
func (s *SendListDispatchService) Dispatch(listID, fromStationID, userID int64) (*SendListDispatchResult, error) {
	list, err := s.sendListRepo.GetByID(listID)
	if err != nil {
		return nil, err
	}
	if list.Status == "TRAITEE" {
		return nil, fmt.Errorf("cette liste a déjà été envoyée")
	}
	if list.Status == models.SendListStatusAnnulee {
		return nil, fmt.Errorf("cette liste a été annulée")
	}

	station, err := s.resolveLocalStation(list.City)
	if err != nil {
		return nil, err
	}
	if station.ID == fromStationID {
		return nil, fmt.Errorf("la station locale de « %s » est le poste courant : rien à expédier", list.City)
	}

	items, err := s.sendListRepo.ListItems(listID, "")
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("cette liste ne contient aucune monture")
	}

	// Les montures sont résolues avant tout déplacement : une liste entièrement injouable
	// (déjà partie, montures introuvables...) doit échouer sans rien avoir bougé.
	shipments, skipped := s.resolveGlasses(items, fromStationID, station.ID)
	if len(shipments) == 0 {
		return nil, fmt.Errorf("aucune monture de cette liste ne peut être envoyée : %s", summarizeSkipped(skipped))
	}

	// Le voyage est un vrai transfert : préparation, puis expédition. C'est `Dispatch` qui
	// libère les emplacements de départ, bascule les montures EN_TRANSIT et écrit le mouvement
	// d'expédition. Aucune réception n'est créée ici — elle appartient au pointage du carton.
	notes := fmt.Sprintf("Carton d'expédition — liste %s vers %s", list.SessionCode, list.City)
	transfer, err := s.transfers.CreateTransfer(fromStationID, station.ID, &notes, userID)
	if err != nil {
		return nil, err
	}

	sentItems := make([]models.SendListItem, 0, len(shipments))
	for _, shipment := range shipments {
		if _, _, err := s.transfers.AddItem(transfer.ID, shipment.glass.Barcode); err != nil {
			// Une monture refusée par le transfert ne retient pas le carton : elle rejoint les
			// écartées, nommée avec son motif, et les autres partent quand même.
			skipped = append(skipped, SendListSkippedItem{
				Reference: sendListItemLabel(shipment.item),
				Reason:    err.Error(),
			})
			continue
		}
		sentItems = append(sentItems, shipment.item)
	}
	if len(sentItems) == 0 {
		return nil, fmt.Errorf("aucune monture de cette liste ne peut être envoyée : %s", summarizeSkipped(skipped))
	}

	if _, err := s.transfers.Dispatch(transfer.ID, userID); err != nil {
		return nil, fmt.Errorf("impossible d'expédier le carton: %w", err)
	}
	sent := len(sentItems)

	// Le carton ne fige que ce qui est réellement parti : faire chercher au magasinier une
	// monture restée au stock général lui ferait constater un manque qui n'en est pas un.
	box, err := s.sendListRepo.CreateDispatchBox(list, sentItems, &transfer.ID)
	if err != nil {
		return nil, err
	}

	// La liste n'est clôturée qu'une fois les montures réellement parties : si l'envoi échoue
	// en route, elle reste ouverte et le magasinier peut le relancer — les montures déjà
	// déplacées seront alors ignorées (déjà en stock à destination).
	if _, err := s.sendListRepo.MarkProcessed([]int64{listID}); err != nil {
		return nil, err
	}

	if _, err := s.sendListRepo.MarkSentCount([]int64{listID}, int64(sent), station.Name); err != nil {
		return nil, err
	}

	return &SendListDispatchResult{
		StationID:   station.ID,
		StationName: station.Name,
		City:        list.City,
		SentCount:   sent,
		// EN_TRANSIT, et non plus EN_STOCK_SOUS_STATION : les montures sont parties, elles ne
		// sont pas arrivées. C'est le pointage du carton qui les fera entrer en stock.
		Status:       string(models.StatusEnTransit),
		Skipped:      skipped,
		BoxID:        box.ID,
		BoxCode:      box.Code,
		BoxReference: box.Reference,
		BoxItemCount: box.ItemCount,
		TransferID:   transfer.ID,
	}, nil
}

// resolveLocalStation traduit la ville de la liste en station de destination. Zéro ou
// plusieurs candidats : on refuse d'envoyer plutôt que de deviner — un colis parti au mauvais
// magasin coûte bien plus cher qu'un envoi bloqué.
func (s *SendListDispatchService) resolveLocalStation(city string) (*authModels.Station, error) {
	stations, err := s.stationRepo.FindLocalStationsByCity(city)
	if err != nil {
		return nil, err
	}
	if len(stations) == 0 {
		return nil, fmt.Errorf("aucune station locale n'est enregistrée pour la ville « %s » : impossible d'envoyer cette liste", city)
	}
	if len(stations) > 1 {
		names := make([]string, 0, len(stations))
		for _, station := range stations {
			names = append(names, station.Name)
		}
		return nil, fmt.Errorf("plusieurs stations locales existent pour « %s » (%s) : impossible de choisir la destination automatiquement", city, strings.Join(names, ", "))
	}
	return &stations[0], nil
}

// sendListShipment garde ensemble la ligne de la liste et la monture qu'elle désigne : le
// transfert a besoin de la seconde, le carton de la première, et les dissocier ferait diverger
// ce qui part de ce que le magasinier pointera.
type sendListShipment struct {
	item  models.SendListItem
	glass *models.Glass
}

// resolveGlasses retrouve les montures physiques derrière les lignes de la liste. Le
// code-barres prime sur glass_id : c'est lui que le magasinier vient de scanner, et glass_id
// peut être nul (ON DELETE SET NULL sur send_list_items).
func (s *SendListDispatchService) resolveGlasses(items []models.SendListItem, fromStationID, toStationID int64) ([]sendListShipment, []SendListSkippedItem) {
	shipments := []sendListShipment{}
	skipped := []SendListSkippedItem{}
	seen := map[int64]bool{}

	for _, item := range items {
		label := sendListItemLabel(item)

		glass, err := s.findGlass(item)
		if err != nil {
			skipped = append(skipped, SendListSkippedItem{Reference: label, Reason: "introuvable en base"})
			continue
		}
		if seen[glass.ID] {
			continue
		}
		if glass.StationID == toStationID {
			skipped = append(skipped, SendListSkippedItem{Reference: label, Reason: "déjà présente dans cette station locale"})
			continue
		}
		if glass.StationID != fromStationID {
			skipped = append(skipped, SendListSkippedItem{Reference: label, Reason: "ne se trouve pas dans le stock de départ"})
			continue
		}
		if !transferableStatuses[glass.Status] {
			skipped = append(skipped, SendListSkippedItem{Reference: label, Reason: fmt.Sprintf("statut actuel « %s »", glass.Status)})
			continue
		}

		seen[glass.ID] = true
		shipments = append(shipments, sendListShipment{item: item, glass: glass})
	}
	return shipments, skipped
}

func (s *SendListDispatchService) findGlass(item models.SendListItem) (*models.Glass, error) {
	if item.Barcode != nil && strings.TrimSpace(*item.Barcode) != "" {
		if glass, err := s.glassRepo.GetByBarcode(strings.TrimSpace(*item.Barcode)); err == nil {
			return glass, nil
		}
	}
	if item.GlassID != nil {
		return s.glassRepo.GetByID(*item.GlassID)
	}
	return nil, fmt.Errorf("monture introuvable")
}

func sendListItemLabel(item models.SendListItem) string {
	if item.Reference != nil && strings.TrimSpace(*item.Reference) != "" {
		return strings.TrimSpace(*item.Reference)
	}
	if item.Barcode != nil && strings.TrimSpace(*item.Barcode) != "" {
		return strings.TrimSpace(*item.Barcode)
	}
	return fmt.Sprintf("ligne #%d", item.ID)
}

func summarizeSkipped(skipped []SendListSkippedItem) string {
	if len(skipped) == 0 {
		return "aucune monture exploitable"
	}
	parts := make([]string, 0, len(skipped))
	for _, item := range skipped {
		parts = append(parts, fmt.Sprintf("%s (%s)", item.Reference, item.Reason))
	}
	return strings.Join(parts, ", ")
}
