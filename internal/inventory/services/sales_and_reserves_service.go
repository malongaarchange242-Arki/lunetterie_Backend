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
func (s *SaleService) CreateSale(stationID int64, barcodes []string, userID int64) (*models.Sale, error) {
	if len(barcodes) == 0 {
		return nil, fmt.Errorf("aucune monture sélectionnée")
	}

	sale := &models.Sale{StationID: stationID, UserID: userID}
	if err := s.saleRepo.Create(sale); err != nil {
		return nil, err
	}

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
			continue
		}
		if err := s.glassRepo.UpdateStatus(glass.ID, models.StatusEnTransit); err != nil {
			log.Printf("erreur passage en transit vers le laboratoire glass #%d: %v", glass.ID, err)
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

	return sale, nil
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
