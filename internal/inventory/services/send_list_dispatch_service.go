package services

import (
	"fmt"
	"log"
	"strings"

	authModels "github.com/lunetterie/backend/internal/auth/models"
	authRepositories "github.com/lunetterie/backend/internal/auth/repositories"
	"github.com/lunetterie/backend/internal/inventory/models"
	"github.com/lunetterie/backend/internal/inventory/repositories"
)

// SendListDispatchService expédie une liste reçue (send_list) vers la station locale de sa
// ville, en une seule action déclenchée par le bouton « Envoyer » du poste de scan.
//
// Contrairement au transfert manuel (TransferService), la destination n'est pas choisie dans
// un menu : elle se déduit de la ville portée par la liste. Et l'envoi vaut arrivée — les
// montures passent directement EN_STOCK_SOUS_STATION dans le stock de la station locale, sans
// étape EN_TRANSIT ni scan de réception à destination.
type SendListDispatchService struct {
	sendListRepo *repositories.SendListRepository
	glassRepo    *repositories.GlassRepository
	movementRepo *repositories.MovementRepository
	allocation   *AllocationService
	stationRepo  *authRepositories.StationRepository
}

func NewSendListDispatchService(
	sendListRepo *repositories.SendListRepository,
	glassRepo *repositories.GlassRepository,
	movementRepo *repositories.MovementRepository,
	allocation *AllocationService,
	stationRepo *authRepositories.StationRepository,
) *SendListDispatchService {
	return &SendListDispatchService{
		sendListRepo: sendListRepo,
		glassRepo:    glassRepo,
		movementRepo: movementRepo,
		allocation:   allocation,
		stationRepo:  stationRepo,
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
	glasses, skipped := s.resolveGlasses(items, fromStationID, station.ID)
	if len(glasses) == 0 {
		return nil, fmt.Errorf("aucune monture de cette liste ne peut être envoyée : %s", summarizeSkipped(skipped))
	}

	sent := 0
	for _, glass := range glasses {
		if err := s.sendGlassToStation(glass, station.ID, userID); err != nil {
			return nil, err
		}
		sent++
	}

	// On matérialise le carton de départ après la résolution et les mouvements effectifs.
	box, err := s.sendListRepo.CreateDispatchBox(list, items)
	if err != nil {
		return nil, err
	}

	// La liste n'est clôturée qu'une fois les montures réellement parties : si l'envoi échoue
	// en route, elle reste ouverte et le magasinier peut le relancer — les montures déjà
	// déplacées seront alors ignorées (déjà en stock à destination).
	if _, err := s.sendListRepo.MarkProcessed([]int64{listID}); err != nil {
		return nil, err
	}

	return &SendListDispatchResult{
		StationID:    station.ID,
		StationName:  station.Name,
		City:         list.City,
		SentCount:    sent,
		Status:       string(models.StatusEnStockSousStation),
		Skipped:      skipped,
		BoxID:        box.ID,
		BoxCode:      box.Code,
		BoxReference: box.Reference,
		BoxItemCount: box.ItemCount,
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

// resolveGlasses retrouve les montures physiques derrière les lignes de la liste. Le
// code-barres prime sur glass_id : c'est lui que le magasinier vient de scanner, et glass_id
// peut être nul (ON DELETE SET NULL sur send_list_items).
func (s *SendListDispatchService) resolveGlasses(items []models.SendListItem, fromStationID, toStationID int64) ([]*models.Glass, []SendListSkippedItem) {
	glasses := []*models.Glass{}
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
		glasses = append(glasses, glass)
	}
	return glasses, skipped
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

// sendGlassToStation libère l'emplacement d'origine, réserve un emplacement de zone STOCK à la
// station locale et bascule la monture EN_STOCK_SOUS_STATION.
//
// Deux mouvements sont tracés plutôt qu'un seul : le départ (EXPEDITION) et l'arrivée
// (RECEPTION_STATION). L'envoi les enchaîne dans la même seconde, mais l'historique du Stock
// Général et celui du magasin restent chacun complets, exactement comme si la réception avait
// été scannée à destination.
func (s *SendListDispatchService) sendGlassToStation(glass *models.Glass, toStationID, userID int64) error {
	location, err := s.allocation.FindOrCreateStockLocation(toStationID)
	if err != nil {
		return fmt.Errorf("impossible d'assigner un emplacement à la monture #%d dans la station locale: %w", glass.ID, err)
	}

	fromStationID := glass.StationID
	fromLocationID := glass.LocationID
	if fromLocationID != nil {
		if err := s.allocation.FreeLocation(*fromLocationID); err != nil {
			return fmt.Errorf("impossible de libérer l'emplacement de la monture #%d: %w", glass.ID, err)
		}
	}
	if err := s.glassRepo.UpdateStationAndLocation(glass.ID, toStationID, location.ID); err != nil {
		return fmt.Errorf("impossible de déplacer la monture #%d vers la station locale: %w", glass.ID, err)
	}
	if err := s.glassRepo.UpdateStatus(glass.ID, models.StatusEnStockSousStation); err != nil {
		return fmt.Errorf("impossible de mettre à jour le statut de la monture #%d: %w", glass.ID, err)
	}

	departure := &models.Movement{
		GlassID:        glass.ID,
		FromStationID:  &fromStationID,
		ToStationID:    &toStationID,
		FromLocationID: fromLocationID,
		Action:         models.ActionExpedition,
		UserID:         userID,
	}
	if err := s.movementRepo.Create(departure); err != nil {
		log.Printf("⚠️  Erreur création mouvement expédition liste (glass #%d): %v", glass.ID, err)
	}

	arrival := &models.Movement{
		GlassID:       glass.ID,
		FromStationID: &fromStationID,
		ToStationID:   &toStationID,
		ToLocationID:  &location.ID,
		Action:        models.ActionReceptionStation,
		UserID:        userID,
	}
	if err := s.movementRepo.Create(arrival); err != nil {
		log.Printf("⚠️  Erreur création mouvement réception station liste (glass #%d): %v", glass.ID, err)
	}
	return nil
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
