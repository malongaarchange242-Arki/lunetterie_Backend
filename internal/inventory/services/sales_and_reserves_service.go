package services

import (
	"fmt"
	"log"

	"github.com/lunetterie/backend/internal/inventory/models"
	"github.com/lunetterie/backend/internal/inventory/repositories"
)

type SaleService struct {
	saleRepo  *repositories.SaleRepository
	glassRepo *repositories.GlassRepository
}

type ReserveService struct {
	reserveRepo *repositories.ReserveRepository
	glassRepo   *repositories.GlassRepository
}

func NewSaleService(saleRepo *repositories.SaleRepository, glassRepo *repositories.GlassRepository) *SaleService {
	return &SaleService{saleRepo: saleRepo, glassRepo: glassRepo}
}

func NewReserveService(reserveRepo *repositories.ReserveRepository, glassRepo *repositories.GlassRepository) *ReserveService {
	return &ReserveService{reserveRepo: reserveRepo, glassRepo: glassRepo}
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

		// For a "Vendre" action from Présentoir we mark the glass as en laboratoire
		// so the backend keeps it available for lab processing rather than marking
		// it immediately as sold. This sets the status to EN_LABORATOIRE.
		if err := s.glassRepo.UpdateStatus(glass.ID, models.StatusEnLaboratoire); err != nil {
			log.Printf("erreur mise à jour statut en laboratoire pour glass #%d: %v", glass.ID, err)
		}
		if err := s.glassRepo.ClearLocation(glass.ID); err != nil {
			log.Printf("erreur vidage emplacement glass #%d: %v", glass.ID, err)
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

		if err := s.glassRepo.UpdateStatus(glass.ID, models.StatusReservee); err != nil {
			log.Printf("erreur mise à jour statut réservée pour glass #%d: %v", glass.ID, err)
		}
		if err := s.glassRepo.UpdateReservedState(glass.ID, true); err != nil {
			log.Printf("erreur marquage réservation glass #%d: %v", glass.ID, err)
		}

		item := &models.ReserveItem{ReserveID: reserve.ID, GlassID: glass.ID}
		if err := s.reserveRepo.AddItem(item); err != nil {
			log.Printf("erreur ajout reserve item: %v", err)
		}
	}

	return reserve, nil
}
