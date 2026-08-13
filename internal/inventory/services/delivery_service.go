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

// HandoverToClient acte la sortie du magasin : le client repart avec ses lunettes.
//
// C'est le dernier geste du cycle, et il manquait entièrement — PRETE_A_LIVRER était le
// dernier statut, si bien qu'une monture remise restait comptée en stock pour toujours et
// que le pointage de la remise ne vivait que dans le navigateur du poste.
//
// À ne pas confondre avec CreateDelivery au-dessus : celle-là est le bouton du laboratoire
// (« montage terminé »), celle-ci celui de la vendeuse (« le client est reparti avec »).
//
// Renvoie les montures écartées avec leur motif, plutôt que de les journaliser en silence :
// la vendeuse doit savoir laquelle des paires qu'elle tient n'a pas été enregistrée.
func (s *DeliveryService) HandoverToClient(stationID int64, barcodes []string, userID int64) ([]string, []SkippedBarcode, error) {
	if len(barcodes) == 0 {
		return nil, nil, fmt.Errorf("aucune monture sélectionnée")
	}

	remises := []string{}
	skipped := []SkippedBarcode{}

	for _, barcode := range barcodes {
		glass, err := s.glassRepo.GetByBarcode(barcode)
		if err != nil {
			skipped = append(skipped, SkippedBarcode{Barcode: barcode, Reason: "monture introuvable"})
			continue
		}

		// Seule une monture que le laboratoire a déclarée prête peut être remise : accepter
		// n'importe quel statut permettrait de sortir du magasin une paire encore en montage.
		if glass.Status != models.StatusPreteALivrer {
			skipped = append(skipped, SkippedBarcode{
				Barcode: barcode,
				Reason:  fmt.Sprintf("statut « %s » : elle n'est pas prête à être remise", glass.Status),
			})
			continue
		}

		if err := s.glassRepo.UpdateStatus(glass.ID, models.StatusLivree); err != nil {
			skipped = append(skipped, SkippedBarcode{Barcode: barcode, Reason: "statut non enregistré"})
			continue
		}

		// L'emplacement est rendu après le changement de statut, et son échec n'annule pas la
		// remise : la monture est partie, un casier resté marqué occupé se corrige.
		if glass.LocationID != nil {
			if err := s.glassRepo.ClearLocation(glass.ID); err != nil {
				log.Printf("⚠️  Emplacement non libéré pour la monture #%d: %v", glass.ID, err)
			}
		}

		movement := &models.Movement{
			GlassID:       glass.ID,
			FromStationID: &stationID,
			Action:        models.ActionRemiseClient,
			UserID:        userID,
		}
		if err := s.movementSvc.CreateMovement(movement); err != nil {
			log.Printf("⚠️  Mouvement de remise non créé pour la monture #%d: %v", glass.ID, err)
		}

		if err := s.deliveryRepo.MarkHandedOver(glass.ID); err != nil {
			log.Printf("⚠️  Ligne de livraison non horodatée pour la monture #%d: %v", glass.ID, err)
		}

		remises = append(remises, glass.Barcode)
	}

	return remises, skipped, nil
}
