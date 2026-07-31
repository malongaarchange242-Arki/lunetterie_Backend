package services

import (
	"fmt"
	"log"

	"github.com/lunetterie/backend/internal/inventory/models"
	"github.com/lunetterie/backend/internal/inventory/repositories"
)

type SaleService struct {
	saleRepo     *repositories.SaleRepository
	glassRepo    *repositories.GlassRepository
	movementRepo *repositories.MovementRepository
	allocation   *AllocationService
}

type ReserveService struct {
	reserveRepo  *repositories.ReserveRepository
	glassRepo    *repositories.GlassRepository
	movementRepo *repositories.MovementRepository
	allocation   *AllocationService
}

func NewSaleService(saleRepo *repositories.SaleRepository, glassRepo *repositories.GlassRepository, movementRepo *repositories.MovementRepository, allocation *AllocationService) *SaleService {
	return &SaleService{saleRepo: saleRepo, glassRepo: glassRepo, movementRepo: movementRepo, allocation: allocation}
}

func NewReserveService(reserveRepo *repositories.ReserveRepository, glassRepo *repositories.GlassRepository, movementRepo *repositories.MovementRepository, allocation *AllocationService) *ReserveService {
	return &ReserveService{reserveRepo: reserveRepo, glassRepo: glassRepo, movementRepo: movementRepo, allocation: allocation}
}

func (s *SaleService) CreateSale(stationID int64, barcodes []string, userID int64) (*models.Sale, error) {
	if len(barcodes) == 0 {
		return nil, fmt.Errorf("aucune monture sélectionnée")
	}

	sale := &models.Sale{
		StationID: stationID,
		UserID:    userID,
	}
	if err := s.saleRepo.Create(sale); err != nil {
		return nil, err
	}

	for _, barcode := range barcodes {
		glass, err := s.glassRepo.GetByBarcode(barcode)
		if err != nil {
			log.Printf("monture introuvable pour vente: %s", barcode)
			continue
		}

		if glass.Status == models.StatusVendue {
			log.Printf("monture déjà vendue: %s", barcode)
			continue
		}

		oldLocationID := glass.LocationID
		if oldLocationID != nil {
			if err := s.allocation.FreeLocation(*oldLocationID); err != nil {
				log.Printf("erreur libération emplacement glass #%d: %v", glass.ID, err)
			}
		}
		if err := s.glassRepo.UpdateStatus(glass.ID, models.StatusVendue); err != nil {
			log.Printf("erreur mise à jour statut vendue pour glass #%d: %v", glass.ID, err)
		}
		if err := s.glassRepo.ClearLocation(glass.ID); err != nil {
			log.Printf("erreur vidage emplacement glass #%d: %v", glass.ID, err)
		}
		movement := &models.Movement{
			GlassID:        glass.ID,
			FromStationID:  &stationID,
			FromLocationID: oldLocationID,
			Action:         models.ActionRetraitPresentoir,
			UserID:         userID,
		}
		if err := s.movementRepo.Create(movement); err != nil {
			log.Printf("erreur création mouvement vente glass #%d: %v", glass.ID, err)
		}

		item := &models.SaleItem{SaleID: sale.ID, GlassID: glass.ID}
		if err := s.saleRepo.AddItem(item); err != nil {
			log.Printf("erreur ajout sale item: %v", err)
		}
	}

	return sale, nil
}

func (s *ReserveService) CreateReserve(stationID int64, barcodes []string, userID int64) (*models.Reserve, error) {
	if len(barcodes) == 0 {
		return nil, fmt.Errorf("aucune monture sélectionnée")
	}

	reserve := &models.Reserve{
		StationID: stationID,
		UserID:    userID,
	}
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
