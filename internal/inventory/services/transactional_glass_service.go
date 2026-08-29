package services

import (
	"fmt"

	"github.com/lunetterie/backend/internal/inventory/models"
	"github.com/lunetterie/backend/internal/inventory/ports"
)

// TransactionalGlassService est la façade de mutation utilisée avant l'extraction du service.
type TransactionalGlassService struct {
	transactions ports.TransactionManager
	glasses      ports.TransactionalGlassRepository
	storage      ports.TransactionalStorageRepository
	movements    ports.TransactionalMovementRepository
	barcodes     ports.BarcodeGenerator
}

func NewTransactionalGlassService(
	transactions ports.TransactionManager,
	glasses ports.TransactionalGlassRepository,
	storage ports.TransactionalStorageRepository,
	movements ports.TransactionalMovementRepository,
	barcodes ports.BarcodeGenerator,
) *TransactionalGlassService {
	return &TransactionalGlassService{transactions: transactions, glasses: glasses, storage: storage, movements: movements, barcodes: barcodes}
}

func (s *TransactionalGlassService) CreateGlass(glass *models.Glass) (err error) {
	if glass == nil {
		return fmt.Errorf("monture obligatoire")
	}
	if glass.Barcode == "" {
		glass.Barcode, err = s.barcodes.GenerateBarcode()
		if err != nil {
			return fmt.Errorf("impossible de générer le barcode: %w", err)
		}
	}
	if glass.Status == "" {
		glass.Status = models.StatusRecuFournisseur
	}
	tx, err := s.transactions.Begin()
	if err != nil {
		return fmt.Errorf("impossible de démarrer la transaction: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()
	if err = s.glasses.CreateTx(tx, glass); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *TransactionalGlassService) AssignGlass(glassID, cartonID, userID int64) (err error) {
	tx, err := s.transactions.Begin()
	if err != nil {
		return fmt.Errorf("impossible de démarrer la transaction: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()
	glass, err := s.validateAssignment(tx, glassID, cartonID)
	if err != nil {
		return err
	}
	if glass.LocationID != nil && *glass.LocationID == cartonID {
		return tx.Commit()
	}
	if err = s.glasses.UpdateLocationTx(tx, glassID, cartonID); err != nil {
		return err
	}
	if err = s.storage.UpdateStatusTx(tx, cartonID, "OCCUPE"); err != nil {
		return err
	}
	if err = s.movements.CreateTx(tx, &models.Movement{GlassID: glassID, FromStationID: &glass.StationID, ToStationID: &glass.StationID, FromLocationID: glass.LocationID, ToLocationID: &cartonID, Action: models.ActionRangement, UserID: userID}); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *TransactionalGlassService) MoveGlass(glassID, cartonID, userID int64) (err error) {

	tx, err := s.transactions.Begin()
	if err != nil {
		return fmt.Errorf("impossible de démarrer la transaction: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()
	glass, err := s.validateAssignment(tx, glassID, cartonID)
	if err != nil {
		return err
	}
	if glass.LocationID != nil && *glass.LocationID == cartonID {
		return tx.Commit()
	}
	if glass.LocationID != nil {
		if err = s.storage.UpdateStatusTx(tx, *glass.LocationID, "LIBRE"); err != nil {
			return err
		}
	}
	if err = s.glasses.UpdateLocationTx(tx, glassID, cartonID); err != nil {
		return err
	}
	if err = s.storage.UpdateStatusTx(tx, cartonID, "OCCUPE"); err != nil {
		return err
	}
	if err = s.movements.CreateTx(tx, &models.Movement{GlassID: glassID, FromStationID: &glass.StationID, ToStationID: &glass.StationID, FromLocationID: glass.LocationID, ToLocationID: &cartonID, Action: models.ActionRangement, UserID: userID}); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *TransactionalGlassService) ReserveGlass(glassID, reservationID, userID int64) (err error) {
	tx, err := s.transactions.Begin()
	if err != nil {
		return fmt.Errorf("impossible de démarrer la transaction: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()
	glass, err := s.glasses.GetByIDTx(tx, glassID)
	if err != nil {
		return err
	}
	if !models.TracksStock(glass.Status) || glass.IsReserved {
		return fmt.Errorf("monture indisponible pour réservation")
	}
	if err = s.glasses.UpdateReservedStateTx(tx, glassID, true); err != nil {
		return err
	}
	if err = s.glasses.UpdateStatusTx(tx, glassID, models.StatusReservee); err != nil {
		return err
	}
	note := fmt.Sprintf("reservation:%d", reservationID)
	if err = s.movements.CreateTx(tx, &models.Movement{GlassID: glassID, FromStationID: &glass.StationID, FromLocationID: glass.LocationID, Action: models.ActionReservation, UserID: userID, Notes: &note}); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *TransactionalGlassService) validateAssignment(tx ports.Transaction, glassID, cartonID int64) (*models.Glass, error) {
	glass, err := s.glasses.GetByIDTx(tx, glassID)
	if err != nil {
		return nil, fmt.Errorf("glass introuvable: %w", err)
	}
	carton, err := s.storage.GetByIDTx(tx, cartonID)
	if err != nil {
		return nil, fmt.Errorf("carton introuvable: %w", err)
	}
	if carton.Type != "CARTON" {
		return nil, fmt.Errorf("emplacement cible invalide: un carton est requis")
	}
	if carton.Status == "MAINTENANCE" {
		return nil, fmt.Errorf("carton indisponible: maintenance")
	}
	if carton.Capacity != nil {
		count, err := s.storage.CountGlassesAtLocationTx(tx, cartonID)
		if err != nil {
			return nil, fmt.Errorf("impossible de vérifier la capacité du carton: %w", err)
		}
		alreadyAssigned := glass.LocationID != nil && *glass.LocationID == cartonID
		if count >= *carton.Capacity && !alreadyAssigned {
			return nil, fmt.Errorf("capacité du carton atteinte")
		}
	}
	return glass, nil
}
