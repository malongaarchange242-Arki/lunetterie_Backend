package services

import (
	"fmt"
	"log"

	authRepositories "github.com/lunetterie/backend/internal/auth/repositories"
	"github.com/lunetterie/backend/internal/inventory/models"
	"github.com/lunetterie/backend/internal/inventory/repositories"
)

type SaleService struct {
	saleRepo     *repositories.SaleRepository
	glassRepo    *repositories.GlassRepository
	movementRepo *repositories.MovementRepository
	allocation   *AllocationService
	stationRepo  *authRepositories.StationRepository
}

type ReserveService struct {
	reserveRepo  *repositories.ReserveRepository
	glassRepo    *repositories.GlassRepository
	movementRepo *repositories.MovementRepository
	allocation   *AllocationService
}

func NewSaleService(saleRepo *repositories.SaleRepository, glassRepo *repositories.GlassRepository, movementRepo *repositories.MovementRepository, allocation *AllocationService, stationRepo *authRepositories.StationRepository) *SaleService {
	return &SaleService{saleRepo: saleRepo, glassRepo: glassRepo, movementRepo: movementRepo, allocation: allocation, stationRepo: stationRepo}
}

func NewReserveService(reserveRepo *repositories.ReserveRepository, glassRepo *repositories.GlassRepository, movementRepo *repositories.MovementRepository, allocation *AllocationService) *ReserveService {
	return &ReserveService{reserveRepo: reserveRepo, glassRepo: glassRepo, movementRepo: movementRepo, allocation: allocation}
}

// CreateSale enregistre la vente (table sales/sale_items, statut VENDUE), puis envoie
// automatiquement la monture vers le Laboratoire (EN_TRANSIT) pour préparation/montage avant
// livraison. Le passage EN_TRANSIT -> EN_LABORATOIRE se fait ensuite normalement au scan sur
// presentoir.html au poste Laboratoire (DisplayService.PlaceOnDisplay, déjà en place).
//
// Le second retour nomme les montures encaissées que l'envoi n'a pas pu emporter : elles
// restent VENDUE, sans emplacement, et n'arriveront jamais au montage. L'appelant DOIT le
// remonter à l'écran. Cet échec était muet — un simple avertissement dans les logs — et une
// station « Laboratoire » absente suffisait à figer chaque vente sans que personne le voie.
func (s *SaleService) CreateSale(stationID int64, barcodes []string, userID int64) (*models.Sale, []string, error) {
	if len(barcodes) == 0 {
		return nil, nil, fmt.Errorf("aucune monture sélectionnée")
	}

	sale := &models.Sale{StationID: stationID, UserID: userID}
	if err := s.saleRepo.Create(sale); err != nil {
		return nil, nil, err
	}

	// Les montures encaissées que le laboratoire ne recevra pas.
	notShipped := []string{}

	var laboratoireStationID int64
	if s.stationRepo != nil {
		if station, err := s.stationRepo.FindByName("Laboratoire"); err == nil {
			laboratoireStationID = station.ID
		} else {
			log.Printf("⚠️  Station Laboratoire introuvable, la monture vendue restera en statut VENDUE sans envoi automatique: %v", err)
		}
	}

	for _, barcode := range barcodes {
		glass, err := s.glassRepo.GetByBarcode(barcode)
		if err != nil {
			log.Printf("monture introuvable pour vente: %s", barcode)
			continue
		}

		if glass.Status == models.StatusVendue || glass.Status == models.StatusEnTransit || glass.Status == models.StatusEnLaboratoire {
			log.Printf("monture déjà vendue ou en cours d'envoi au laboratoire: %s", barcode)
			continue
		}

		oldLocationID := glass.LocationID
		if oldLocationID != nil {
			if err := s.allocation.FreeLocation(*oldLocationID); err != nil {
				log.Printf("erreur libération emplacement glass #%d: %v", glass.ID, err)
			}
		}
		if err := s.glassRepo.ClearLocation(glass.ID); err != nil {
			log.Printf("erreur vidage emplacement glass #%d: %v", glass.ID, err)
		}
		if err := s.glassRepo.UpdateStatus(glass.ID, models.StatusVendue); err != nil {
			log.Printf("erreur mise à jour statut vendue pour glass #%d: %v", glass.ID, err)
		}

		saleMovement := &models.Movement{
			GlassID:        glass.ID,
			FromStationID:  &stationID,
			FromLocationID: oldLocationID,
			Action:         models.ActionRetraitPresentoir,
			UserID:         userID,
		}
		if err := s.movementRepo.Create(saleMovement); err != nil {
			log.Printf("erreur création mouvement vente glass #%d: %v", glass.ID, err)
		}

		item := &models.SaleItem{SaleID: sale.ID, GlassID: glass.ID}
		if err := s.saleRepo.AddItem(item); err != nil {
			log.Printf("erreur ajout sale item: %v", err)
		}

		if laboratoireStationID == 0 {
			notShipped = append(notShipped, barcode)
			continue
		}
		if err := s.glassRepo.UpdateStatus(glass.ID, models.StatusEnTransit); err != nil {
			log.Printf("erreur passage en transit vers le laboratoire glass #%d: %v", glass.ID, err)
			notShipped = append(notShipped, barcode)
			continue
		}
		expeditionMovement := &models.Movement{
			GlassID:       glass.ID,
			FromStationID: &stationID,
			ToStationID:   &laboratoireStationID,
			Action:        models.ActionExpedition,
			UserID:        userID,
		}
		if err := s.movementRepo.Create(expeditionMovement); err != nil {
			log.Printf("erreur création mouvement expédition vers laboratoire glass #%d: %v", glass.ID, err)
		}
	}

	return sale, notShipped, nil
}

// ReserveExpiryDays : au-delà de ce délai, une monture mise de côté repart au présentoir.
// Une réserve n'est pas une vente — passé dix jours sans que le client revienne, la monture
// doit se revendre plutôt que rester immobilisée dans un tiroir.
const ReserveExpiryDays = 10

// ReleaseExpiredReserves rend au présentoir toutes les réservations dépassées. Renvoie le
// nombre de montures effectivement libérées.
//
// Chaque monture est traitée isolément : une qui échoue (plus d'emplacement libre au
// présentoir, par exemple) ne doit pas empêcher les suivantes de repartir. L'erreur est
// journalisée, la boucle continue, et la monture sera reprise au prochain passage puisqu'elle
// reste RESERVEE.
func (s *ReserveService) ReleaseExpiredReserves() (int, error) {
	expired, err := s.reserveRepo.FindExpired(ReserveExpiryDays)
	if err != nil {
		return 0, err
	}

	released := 0
	for _, item := range expired {
		if err := s.returnToDisplay(item); err != nil {
			log.Printf("⚠️  Réservation du %s non levée pour la monture #%d (%s): %v",
				item.ReservedAt.Format("2006-01-02"), item.GlassID, item.Barcode, err)
			continue
		}
		released++
	}
	return released, nil
}

// returnToDisplay repose une monture réservée sur le présentoir de sa propre station : elle
// n'a pas bougé physiquement pendant la réserve, elle retourne là où elle était exposée.
func (s *ReserveService) returnToDisplay(item models.ExpiredReserve) error {
	location, err := s.allocation.FindOrCreatePresentoirLocation(item.StationID, item.Barcode)
	if err != nil {
		return err
	}

	// L'emplacement avant le statut : une monture annoncée EN_PRESENTOIR sans casier serait
	// cherchée en rayon sans y être. Si l'attribution échoue, on rend l'emplacement réservé
	// et la monture reste RESERVEE — un état cohérent, retenté au prochain passage.
	if err := s.glassRepo.UpdateLocation(item.GlassID, location.ID); err != nil {
		if freeErr := s.allocation.FreeLocation(location.ID); freeErr != nil {
			log.Printf("⚠️  Emplacement #%d réservé mais non attribué et non libéré: %v", location.ID, freeErr)
		}
		return err
	}
	if err := s.glassRepo.UpdateStatus(item.GlassID, models.StatusEnPresentoir); err != nil {
		return err
	}
	// Le drapeau suit le statut : le laisser à true garderait la monture marquée réservée
	// partout où ce champ est lu.
	if err := s.glassRepo.UpdateReservedState(item.GlassID, false); err != nil {
		log.Printf("⚠️  Drapeau de réservation non levé pour la monture #%d: %v", item.GlassID, err)
	}

	movement := &models.Movement{
		GlassID:        item.GlassID,
		FromStationID:  &item.StationID,
		ToStationID:    &item.StationID,
		FromLocationID: item.LocationID,
		ToLocationID:   &location.ID,
		Action:         models.ActionMiseEnPresentoir,
		UserID:         item.UserID,
	}
	if err := s.movementRepo.Create(movement); err != nil {
		log.Printf("⚠️  Erreur création mouvement retour présentoir (glass #%d): %v", item.GlassID, err)
	}
	return nil
}

func (s *ReserveService) CreateReserve(stationID int64, barcodes []string, userID int64) (*models.Reserve, error) {
	if len(barcodes) == 0 {
		return nil, fmt.Errorf("aucune monture sélectionnée")
	}

	reserve := &models.Reserve{StationID: stationID, UserID: userID}
	if err := s.reserveRepo.Create(reserve); err != nil {
		return nil, err
	}

	for _, barcode := range barcodes {
		glass, err := s.glassRepo.GetByBarcode(barcode)
		if err != nil {
			log.Printf("monture introuvable pour réserve: %s", barcode)
			continue
		}

		if glass.Status == models.StatusReservee {
			log.Printf("monture déjà réservée: %s", barcode)
			continue
		}

		oldLocationID := glass.LocationID
		if oldLocationID != nil {
			if err := s.allocation.FreeLocation(*oldLocationID); err != nil {
				log.Printf("erreur libération emplacement glass #%d: %v", glass.ID, err)
			}
		}

		if err := s.glassRepo.UpdateStatus(glass.ID, models.StatusReservee); err != nil {
			log.Printf("erreur mise à jour statut réservée pour glass #%d: %v", glass.ID, err)
		}
		if err := s.glassRepo.UpdateReservedState(glass.ID, true); err != nil {
			log.Printf("erreur marquage réservation glass #%d: %v", glass.ID, err)
		}
		if err := s.glassRepo.ClearLocation(glass.ID); err != nil {
			log.Printf("erreur vidage emplacement glass #%d: %v", glass.ID, err)
		}

		movement := &models.Movement{
			GlassID:        glass.ID,
			FromStationID:  &stationID,
			FromLocationID: oldLocationID,
			Action:         models.ActionReservation,
			UserID:         userID,
		}
		if err := s.movementRepo.Create(movement); err != nil {
			log.Printf("erreur création mouvement réserve glass #%d: %v", glass.ID, err)
		}

		item := &models.ReserveItem{ReserveID: reserve.ID, GlassID: glass.ID}
		if err := s.reserveRepo.AddItem(item); err != nil {
			log.Printf("erreur ajout reserve item: %v", err)
		}
	}

	return reserve, nil
}
