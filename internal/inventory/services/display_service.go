package services

import (
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
}

// PlaceOnDisplay est déclenché par la recherche d'un code-barres au poste Présentoir : la
// monture passe automatiquement au statut EN_PRESENTOIR et reçoit un emplacement dédié à la
// zone présentoir de ce poste (le code-barres est conservé tel quel). Si elle se trouvait sur
// un autre poste, elle y est rattachée directement — le scan vaut confirmation physique de sa
// présence sur le présentoir. Si elle est déjà exposée à ce poste, ou dans un statut non
// éligible (réservée, vendue, en laboratoire...), l'appel est un no-op silencieux : la recherche
// reste alors une simple consultation.
//
// Cas EN_TRANSIT : le scan à la station de destination du transfert vaut à la fois réception
// (clôture la ligne de transfert, et le transfert entier si c'était la dernière monture) et
// mise en présentoir, en une seule action. Scanner à une autre station reste un no-op.
func (s *DisplayService) PlaceOnDisplay(barcode string, stationID, userID int64) error {
	glass, err := s.glassRepo.GetByBarcode(barcode)
	if err != nil {
		return err
	}
	if glass.StationID == stationID && (glass.Status == models.StatusEnPresentoir || glass.Status == models.StatusEnLaboratoire) {
		return nil
	}

	if glass.Status == models.StatusEnTransit {
		if !s.isLaboratoireStation(stationID) && !s.completeTransferReception(glass.ID, stationID, userID) {
			return nil
		}
	} else if !placeableStatuses[glass.Status] {
		return nil
	}

	location, err := s.allocation.FindOrCreatePresentoirLocation(stationID, barcode)
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

// completeTransferReception clôture la réception du transfert actif d'une monture, si la
// station scannée correspond bien à la destination du transfert. Renvoie false sans effet
// si la monture n'a pas de transfert actif vers cette station.
func (s *DisplayService) completeTransferReception(glassID, stationID, userID int64) bool {
	if s.transferRepo == nil {
		return false
	}

	item, err := s.transferRepo.GetActiveItemByGlassID(glassID)
	if err != nil {
		return false
	}
	transfer, err := s.transferRepo.GetByID(item.TransferID)
	if err != nil || transfer.ToStationID != stationID {
		return false
	}

	if err := s.transferRepo.MarkItemReceived(item.ID); err != nil {
		log.Printf("⚠️  Erreur clôture ligne de transfert (item #%d): %v", item.ID, err)
		return false
	}

	remaining, err := s.transferRepo.CountItemsNotReceived(transfer.ID)
	if err != nil {
		log.Printf("⚠️  Erreur comptage montures restantes (transfert #%d): %v", transfer.ID, err)
		return true
	}
	if remaining == 0 {
		if err := s.transferRepo.MarkReceived(transfer.ID, userID); err != nil {
			log.Printf("⚠️  Erreur clôture transfert #%d: %v", transfer.ID, err)
		}
	}
	return true
}

func (s *DisplayService) isLaboratoireStation(stationID int64) bool {
	if s.stationRepo == nil {
		return false
	}

	station, err := s.stationRepo.GetByID(stationID)
	if err != nil {
		log.Printf("⚠️  Impossible de récupérer la station #%d pour détection laboratoire: %v", stationID, err)
		return false
	}
	return strings.EqualFold(strings.TrimSpace(station.Name), "Laboratoire")
}
