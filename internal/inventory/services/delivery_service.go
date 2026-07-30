package services

import (
	"fmt"
	"log"

	"github.com/lunetterie/backend/internal/inventory/models"
	"github.com/lunetterie/backend/internal/inventory/repositories"
)

// DeliveryService orchestre la création de delivery depuis le poste Laboratoire
type DeliveryService struct {
	deliveryRepo *repositories.DeliveryRepository
	glassRepo    *repositories.GlassRepository
	movementSvc  *MovementService
}

func NewDeliveryService(deliveryRepo *repositories.DeliveryRepository, glassRepo *repositories.GlassRepository, movementSvc *MovementService) *DeliveryService {
	return &DeliveryService{deliveryRepo: deliveryRepo, glassRepo: glassRepo, movementSvc: movementSvc}
}

// CreateDelivery crée une delivery et associe les montures sélectionnées (par barcode)
func (s *DeliveryService) CreateDelivery(stationID int64, barcodes []string, userID int64) (*models.Delivery, error) {
	if len(barcodes) == 0 {
		return nil, fmt.Errorf("aucune monture sélectionnée")
	}

	// s'assure d'un supplier par défaut
	supplierID, err := s.deliveryRepo.FindOrCreateDefaultSupplier()
	if err != nil {
		return nil, err
	}

	delivery := &models.Delivery{
		SupplierID: supplierID,
		StationID:  stationID,
		Notes:      nil,
	}
	if err := s.deliveryRepo.Create(delivery); err != nil {
		return nil, err
	}

	for _, bc := range barcodes {
		glass, err := s.glassRepo.GetByBarcode(bc)
		if err != nil {
			log.Printf("monture introuvable pour livraison: %s", bc)
			continue
		}
		// lier la monture à la delivery
		item := &models.DeliveryItem{DeliveryID: delivery.ID, GlassID: glass.ID}
		if err := s.deliveryRepo.AddItem(item); err != nil {
			log.Printf("erreur ajout delivery item: %v", err)
			continue
		}

		// mettre à jour le statut de la monture en PRETE_A_LIVRER
		if err := s.glassRepo.UpdateStatus(glass.ID, models.StatusPreteALivrer); err != nil {
			log.Printf("erreur mise à jour statut glass #%d: %v", glass.ID, err)
		}

		// créer un mouvement 'LIVRAISON' (réutilise ActionLivraison si défini)
		movement := &models.Movement{
			GlassID:       glass.ID,
			FromStationID: &glass.StationID,
			ToStationID:   &stationID,
			Action:        models.ActionLivraison,
			UserID:        userID,
		}
		if err := s.movementSvc.CreateMovement(movement); err != nil {
			log.Printf("erreur création mouvement livraison: %v", err)
		}
	}

	return delivery, nil
}
