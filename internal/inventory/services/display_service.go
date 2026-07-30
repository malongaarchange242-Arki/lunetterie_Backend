package services

import (
	"log"

	"github.com/lunetterie/backend/internal/inventory/models"
	"github.com/lunetterie/backend/internal/inventory/repositories"
)

// DisplayService gère la mise en présentoir des montures (le code-barres ne change jamais)
type DisplayService struct {
	glassRepo    *repositories.GlassRepository
	movementRepo *repositories.MovementRepository
	allocation   *AllocationService
}

// NewDisplayService crée une nouvelle instance
func NewDisplayService(glassRepo *repositories.GlassRepository, movementRepo *repositories.MovementRepository, allocation *AllocationService) *DisplayService {
	return &DisplayService{glassRepo: glassRepo, movementRepo: movementRepo, allocation: allocation}
}

// placeableStatuses liste les statuts à partir desquels une monture peut être placée sur le présentoir
var placeableStatuses = map[models.GlassStatus]bool{
	models.StatusEnStockSousStation: true,
	models.StatusEnStockGeneral:     true,
}

// PlaceOnDisplay est déclenché par la recherche d'un code-barres au poste Présentoir : si la
// monture est bien à ce poste et pas encore exposée, elle passe automatiquement au statut
// EN_PRESENTOIR et reçoit un emplacement dédié à la zone présentoir (le code-barres est conservé
// tel quel). Si la monture appartient à un autre poste, est déjà exposée, ou n'est pas dans un
// statut éligible, l'appel est un no-op silencieux (la recherche reste une simple consultation).
func (s *DisplayService) PlaceOnDisplay(barcode string, stationID, userID int64) error {
	glass, err := s.glassRepo.GetByBarcode(barcode)
	if err != nil {
		return err
	}
	if glass.StationID != stationID || glass.Status == models.StatusEnPresentoir || !placeableStatuses[glass.Status] {
		return nil
	}

	location, err := s.allocation.FindFreeLocation(stationID, models.ZonePresentoir)
	if err != nil {
		log.Printf("⚠️  Aucun emplacement présentoir libre pour la monture #%d: %v", glass.ID, err)
		return nil
	}

	oldLocationID := glass.LocationID
	if oldLocationID != nil {
		if err := s.allocation.FreeLocation(*oldLocationID); err != nil {
			log.Printf("⚠️  Erreur libération ancien emplacement (glass #%d): %v", glass.ID, err)
		}
	}
	if err := s.glassRepo.UpdateLocation(glass.ID, location.ID); err != nil {
		return err
	}
	if err := s.glassRepo.UpdateStatus(glass.ID, models.StatusEnPresentoir); err != nil {
		return err
	}

	movement := &models.Movement{
		GlassID:        glass.ID,
		FromStationID:  &stationID,
		ToStationID:    &stationID,
		FromLocationID: oldLocationID,
		ToLocationID:   &location.ID,
		Action:         models.ActionMiseEnPresentoir,
		UserID:         userID,
	}
	if err := s.movementRepo.Create(movement); err != nil {
		log.Printf("⚠️  Erreur création mouvement mise en présentoir (glass #%d): %v", glass.ID, err)
	}
	return nil
}
