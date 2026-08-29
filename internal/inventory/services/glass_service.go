package services

import (
	"fmt"

	"github.com/lunetterie/backend/internal/inventory/models"
	"github.com/lunetterie/backend/internal/inventory/ports"
)

// GlassService porte les mutations metier propres a une monture.
type GlassService struct {
	glasses   ports.GlassRepository
	storage   ports.StorageRepository
	movements ports.MovementRepository
	barcodes  ports.BarcodeGenerator
}

func NewGlassService(
	glasses ports.GlassRepository,
	storage ports.StorageRepository,
	movements ports.MovementRepository,
	barcodes ports.BarcodeGenerator,
) *GlassService {
	return &GlassService{glasses: glasses, storage: storage, movements: movements, barcodes: barcodes}
}

// CreateGlass genere un barcode si necessaire puis persiste la monture.
func (s *GlassService) CreateGlass(glass *models.Glass) error {
	if glass == nil {
		return fmt.Errorf("monture obligatoire")
	}
	if glass.Barcode == "" {
		barcode, err := s.barcodes.GenerateBarcode()
		if err != nil {
			return fmt.Errorf("impossible de générer le barcode: %w", err)
		}
		glass.Barcode = barcode
	}
	if glass.Status == "" {
		glass.Status = models.StatusRecuFournisseur
	}
	return s.glasses.Create(glass)
}

// AssignGlass affecte une monture a un carton et conserve son emplacement courant.
func (s *GlassService) AssignGlass(glassID, cartonID, userID int64) error {
	glass, err := s.glasses.GetByID(glassID)
	if err != nil {
		return fmt.Errorf("glass introuvable: %w", err)
	}
	carton, err := s.storage.GetByID(cartonID)
	if err != nil {
		return fmt.Errorf("carton introuvable: %w", err)
	}
	if carton.Type != "CARTON" {
		return fmt.Errorf("emplacement cible invalide: un carton est requis")
	}
	if carton.Status == "MAINTENANCE" {
		return fmt.Errorf("carton indisponible: maintenance")
	}
	if carton.Capacity != nil {
		count, err := s.storage.CountGlassesAtLocation(cartonID)
		if err != nil {
			return fmt.Errorf("impossible de vérifier la capacité du carton: %w", err)
		}
		alreadyAssigned := glass.LocationID != nil && *glass.LocationID == cartonID
		if count >= *carton.Capacity && !alreadyAssigned {
			return fmt.Errorf("capacité du carton atteinte")
		}
	}
	if glass.LocationID != nil && *glass.LocationID == cartonID {
		return nil
	}
	if err := s.glasses.UpdateLocation(glassID, cartonID); err != nil {
		return fmt.Errorf("impossible d'affecter la monture: %w", err)
	}
	if err := s.storage.UpdateStatus(cartonID, "OCCUPE"); err != nil {
		return fmt.Errorf("impossible de mettre à jour le carton: %w", err)
	}
	return s.movements.Create(&models.Movement{
		GlassID: glassID, FromStationID: &glass.StationID, ToStationID: &glass.StationID,
		FromLocationID: glass.LocationID, ToLocationID: &cartonID,
		Action: models.ActionRangement, UserID: userID,
	})
}

// ReserveGlass reserve une monture sans exposer les tables de reservation commerciale.
func (s *GlassService) ReserveGlass(glassID, reservationID, userID int64) error {
	glass, err := s.glasses.GetByID(glassID)
	if err != nil {
		return fmt.Errorf("glass introuvable: %w", err)
	}
	if !models.TracksStock(glass.Status) || glass.IsReserved {
		return fmt.Errorf("monture indisponible pour réservation")
	}
	if err := s.glasses.UpdateReservedState(glassID, true); err != nil {
		return fmt.Errorf("impossible de réserver la monture: %w", err)
	}
	if err := s.glasses.UpdateStatus(glassID, models.StatusReservee); err != nil {
		return fmt.Errorf("impossible de mettre à jour le statut: %w", err)
	}
	return s.movements.Create(&models.Movement{
		GlassID: glassID, FromStationID: &glass.StationID, FromLocationID: glass.LocationID,
		Action: models.ActionReservation, UserID: userID,
		Notes: func() *string { value := fmt.Sprintf("reservation:%d", reservationID); return &value }(),
	})
}

// MoveGlass deplace une monture, libere son ancien emplacement et trace l'operation.
func (s *GlassService) MoveGlass(glassID, cartonID, userID int64) error {
	glass, err := s.glasses.GetByID(glassID)
	if err != nil {
		return fmt.Errorf("glass introuvable: %w", err)
	}
	oldLocationID := glass.LocationID
	if err := s.AssignGlass(glassID, cartonID, userID); err != nil {
		return err
	}
	if oldLocationID != nil && *oldLocationID != cartonID {
		if err := s.storage.UpdateStatus(*oldLocationID, "LIBRE"); err != nil {
			return fmt.Errorf("impossible de libérer l'ancien emplacement: %w", err)
		}
	}
	return nil
}
