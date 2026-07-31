package services

import (
	"fmt"
	"log"
	"strings"

	authRepositories "github.com/lunetterie/backend/internal/auth/repositories"
	"github.com/lunetterie/backend/internal/inventory/models"
	"github.com/lunetterie/backend/internal/inventory/repositories"
)

// DisplayService gère la mise en présentoir des montures (le code-barres ne change jamais)
type DisplayService struct {
	glassRepo    *repositories.GlassRepository
	movementRepo *repositories.MovementRepository
	allocation   *AllocationService
	stationRepo  *authRepositories.StationRepository
	transferRepo *repositories.TransferRepository
}

// NewDisplayService crée une nouvelle instance
func NewDisplayService(glassRepo *repositories.GlassRepository, movementRepo *repositories.MovementRepository, allocation *AllocationService, stationRepo *authRepositories.StationRepository, transferRepo *repositories.TransferRepository) *DisplayService {
	return &DisplayService{glassRepo: glassRepo, movementRepo: movementRepo, allocation: allocation, stationRepo: stationRepo, transferRepo: transferRepo}
}

// placeableStatuses liste les statuts à partir desquels une monture peut être placée sur le présentoir ou en laboratoire
var placeableStatuses = map[models.GlassStatus]bool{
	models.StatusEnStockSousStation: true,
	models.StatusEnStockGeneral:     true,
	// Une monture déjà exposée (présentoir ou labo) ailleurs reste déplaçable : la rescanner à un
	// nouveau poste vaut confirmation qu'elle a physiquement changé d'endroit.
	models.StatusEnPresentoir:  true,
	models.StatusEnLaboratoire: true,
}

// PlaceOnDisplay est déclenché par la recherche d'un code-barres sur presentoir.html, quel que
// soit le poste (Présentoir, Laboratoire, ou un magasin "station" comme Station Pointe-Noire).
//
// Cas EN_TRANSIT (monture envoyée par transfert, ex: depuis scan.html) : le scan vaut
// confirmation d'arrivée dans la grande majorité des cas — SAUF si un transfert actif existe
// mais vise explicitement un AUTRE poste, auquel cas le scan est refusé (avec une note), pour ne
// pas laisser un poste récupérer une monture destinée ailleurs. Une monture EN_TRANSIT sans
// aucun transfert actif retrouvé (jamais créé via le vrai circuit, données de test...) n'est
// PAS bloquée : mieux vaut recevoir la monture que la laisser bloquée sans recours dans l'app.
//   - Vers le Laboratoire : le scan vaut à la fois réception et mise en labo, en une seule action
//     (pas de notion de "stock" intermédiaire pour ce poste).
//   - Vers un magasin normal : le scan clôture le transfert (et le transfert entier si c'était la
//     dernière monture) et fait atterrir la monture en stock local (EN_STOCK_SOUS_STATION) avec
//     un emplacement de zone STOCK — PAS en présentoir.
//
// Pour tout autre statut (déjà en stock, déjà exposée ailleurs...) : la mise en présentoir/labo
// automatique ne se déclenche QUE si le poste scanné est lui-même le poste dédié Présentoir ou
// Laboratoire. À un poste "station" (magasin), rechercher une monture reste une simple
// consultation — aucun changement de statut. Faire passer une monture de "en stock local" à "sur
// le présentoir" depuis un magasin se fait explicitement via le bouton "Envoyer" (transfert réel
// vers le poste Présentoir), pas par une simple recherche.
//
// Le retour (string, error) renvoie en 2e position une erreur réelle (ex: monture introuvable),
// et en 1re position une note explicative facultative quand l'appel n'a rien changé (ex: "en
// transit vers un autre poste") — utile pour comprendre depuis l'UI pourquoi un scan n'a aucun
// effet, plutôt que d'échouer en silence.
func (s *DisplayService) PlaceOnDisplay(barcode string, stationID, userID int64) (string, error) {
	glass, err := s.glassRepo.GetByBarcode(barcode)
	if err != nil {
		return "", err
	}
	if glass.StationID == stationID && (glass.Status == models.StatusEnPresentoir || glass.Status == models.StatusEnLaboratoire) {
		return "", nil
	}

	if glass.Status == models.StatusEnTransit {
		if s.isLaboratoireStation(stationID) {
			return "", s.placeGlass(glass, stationID, userID)
		}
		note, blocking := s.completeTransferReception(glass.ID, stationID, userID)
		if blocking {
			return note, nil
		}
		return "", s.receiveIntoLocalStock(glass, stationID, userID)
	}

	// La mise en présentoir/labo automatique au scan ne s'applique qu'aux postes dédiés
	// (Présentoir, Laboratoire). À un poste "station" (magasin, ex: Station Pointe-Noire),
	// rechercher une monture déjà en stock local reste une simple consultation — le passage au
	// présentoir se fait explicitement via le bouton "Envoyer" (transfert réel vers Présentoir).
	if !s.isPresentoirStation(stationID) && !s.isLaboratoireStation(stationID) {
		return "", nil
	}

	if !placeableStatuses[glass.Status] {
		return fmt.Sprintf("Statut actuel « %s » : cette monture ne peut pas être placée sur le présentoir depuis ce statut.", glass.Status), nil
	}
	return "", s.placeGlass(glass, stationID, userID)
}

// receiveIntoLocalStock fait atterrir une monture reçue par transfert dans le stock local de la
// station (zone STOCK), sans la mettre sur le présentoir. Reflète TransferService.ReceiveItem,
// mais déclenché ici automatiquement par la simple recherche du code-barres.
func (s *DisplayService) receiveIntoLocalStock(glass *models.Glass, stationID, userID int64) error {
	location, err := s.allocation.FindOrCreateStockLocation(stationID)
	if err != nil {
		log.Printf("⚠️  Impossible d'assigner un emplacement de stock pour la réception de la monture #%d: %v", glass.ID, err)
		return nil
	}

	oldStationID := glass.StationID
	oldLocationID := glass.LocationID
	if oldLocationID != nil {
		if err := s.allocation.FreeLocation(*oldLocationID); err != nil {
			log.Printf("⚠️  Erreur libération ancien emplacement (glass #%d): %v", glass.ID, err)
		}
	}
	if err := s.glassRepo.UpdateStationAndLocation(glass.ID, stationID, location.ID); err != nil {
		return err
	}
	if err := s.glassRepo.UpdateStatus(glass.ID, models.StatusEnStockSousStation); err != nil {
		return err
	}

	movement := &models.Movement{
		GlassID:        glass.ID,
		FromStationID:  &oldStationID,
		ToStationID:    &stationID,
		FromLocationID: oldLocationID,
		ToLocationID:   &location.ID,
		Action:         models.ActionReceptionStation,
		UserID:         userID,
	}
	if err := s.movementRepo.Create(movement); err != nil {
		log.Printf("⚠️  Erreur création mouvement réception station (glass #%d): %v", glass.ID, err)
	}
	return nil
}

// placeGlass assigne un emplacement présentoir (ou laboratoire) et bascule le statut en
// conséquence (EN_PRESENTOIR, ou EN_LABORATOIRE si la station scannée est le Laboratoire).
func (s *DisplayService) placeGlass(glass *models.Glass, stationID, userID int64) error {
	location, err := s.allocation.FindOrCreatePresentoirLocation(stationID, glass.Barcode)
	if err != nil {
		log.Printf("⚠️  Impossible d'assigner un emplacement présentoir pour la monture #%d: %v", glass.ID, err)
		return nil
	}

	oldStationID := glass.StationID
	oldLocationID := glass.LocationID
	if oldLocationID != nil {
		if err := s.allocation.FreeLocation(*oldLocationID); err != nil {
			log.Printf("⚠️  Erreur libération ancien emplacement (glass #%d): %v", glass.ID, err)
		}
	}
	if err := s.glassRepo.UpdateStationAndLocation(glass.ID, stationID, location.ID); err != nil {
		return err
	}

	status := models.StatusEnPresentoir
	action := models.ActionMiseEnPresentoir
	if s.isLaboratoireStation(stationID) {
		status = models.StatusEnLaboratoire
		action = models.ActionEnvoiLaboratoire
	}

	if err := s.glassRepo.UpdateStatus(glass.ID, status); err != nil {
		return err
	}

	movement := &models.Movement{
		GlassID:        glass.ID,
		FromStationID:  &oldStationID,
		ToStationID:    &stationID,
		FromLocationID: oldLocationID,
		ToLocationID:   &location.ID,
		Action:         action,
		UserID:         userID,
	}
	if err := s.movementRepo.Create(movement); err != nil {
		log.Printf("⚠️  Erreur création mouvement mise en présentoir (glass #%d): %v", glass.ID, err)
	}
	return nil
}

// completeTransferReception clôture, en best-effort, la réception du transfert actif d'une
// monture. Renvoie (note, blocking) : blocking=true UNIQUEMENT quand un transfert actif existe
// mais vise un AUTRE poste que celui scanné — dans ce cas, et uniquement celui-ci, la réception
// est refusée (pour ne pas laisser un poste récupérer une monture destinée ailleurs). Dans tous
// les autres cas (aucun transfert actif retrouvé, service indisponible, erreur de clôture),
// blocking est false : le scan vaut quand même confirmation d'arrivée, plutôt que de laisser la
// monture bloquée EN_TRANSIT indéfiniment sans recours possible depuis l'app.
func (s *DisplayService) completeTransferReception(glassID, stationID, userID int64) (string, bool) {
	if s.transferRepo == nil {
		return "", false
	}

	item, err := s.transferRepo.GetActiveItemByGlassID(glassID)
	if err != nil {
		// Aucun transfert actif retrouvé : rien à clôturer, mais on ne bloque pas la réception.
		return "", false
	}
	transfer, err := s.transferRepo.GetByID(item.TransferID)
	if err != nil {
		return "", false
	}
	if transfer.ToStationID != stationID {
		return fmt.Sprintf("Cette monture est en transit vers %s (station #%d), pas vers ce poste-ci : impossible de la réceptionner ici.", s.stationDisplayName(transfer.ToStationID), transfer.ToStationID), true
	}

	if err := s.transferRepo.MarkItemReceived(item.ID); err != nil {
		log.Printf("⚠️  Erreur clôture ligne de transfert (item #%d): %v", item.ID, err)
		return "", false
	}

	remaining, err := s.transferRepo.CountItemsNotReceived(transfer.ID)
	if err != nil {
		log.Printf("⚠️  Erreur comptage montures restantes (transfert #%d): %v", transfer.ID, err)
		return "", false
	}
	if remaining == 0 {
		if err := s.transferRepo.MarkReceived(transfer.ID, userID); err != nil {
			log.Printf("⚠️  Erreur clôture transfert #%d: %v", transfer.ID, err)
		}
	}
	return "", false
}

func (s *DisplayService) stationDisplayName(stationID int64) string {
	if s.stationRepo != nil {
		if station, err := s.stationRepo.GetByID(stationID); err == nil {
			return station.Name
		}
	}
	return fmt.Sprintf("la station #%d", stationID)
}

func (s *DisplayService) isLaboratoireStation(stationID int64) bool {
	return s.stationNameEquals(stationID, "Laboratoire")
}

func (s *DisplayService) isPresentoirStation(stationID int64) bool {
	return s.stationNameEquals(stationID, "Présentoir")
}

func (s *DisplayService) stationNameEquals(stationID int64, name string) bool {
	if s.stationRepo == nil {
		return false
	}

	station, err := s.stationRepo.GetByID(stationID)
	if err != nil {
		log.Printf("⚠️  Impossible de récupérer la station #%d: %v", stationID, err)
		return false
	}
	return strings.EqualFold(strings.TrimSpace(station.Name), name)
}
