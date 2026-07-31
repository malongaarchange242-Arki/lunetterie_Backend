package services

import (
	"fmt"
	"log"

	authRepositories "github.com/lunetterie/backend/internal/auth/repositories"
	"github.com/lunetterie/backend/internal/inventory/models"
	"github.com/lunetterie/backend/internal/inventory/repositories"
)

// TransferService orchestre le workflow de transfert inter-station
type TransferService struct {
	transferRepo *repositories.TransferRepository
	glassRepo    *repositories.GlassRepository
	movementRepo *repositories.MovementRepository
	allocation   *AllocationService
	stationRepo  *authRepositories.StationRepository
}

var transferableStatuses = map[models.GlassStatus]bool{
	models.StatusEnStockGeneral:     true,
	models.StatusEnStockSousStation: true,
	models.StatusEnPresentoir:       true,
}

func NewTransferService(
	transferRepo *repositories.TransferRepository,
	glassRepo *repositories.GlassRepository,
	movementRepo *repositories.MovementRepository,
	allocation *AllocationService,
	stationRepo *authRepositories.StationRepository,
) *TransferService {
	return &TransferService{
		transferRepo: transferRepo,
		glassRepo:    glassRepo,
		movementRepo: movementRepo,
		allocation:   allocation,
		stationRepo:  stationRepo,
	}
}

// CreateTransfer crée un nouveau transfert en préparation
func (s *TransferService) CreateTransfer(fromStationID, toStationID int64, notes *string, userID int64) (*models.Transfer, error) {
	if fromStationID == toStationID {
		return nil, fmt.Errorf("la station de départ et d'arrivée doivent être différentes")
	}

	transfer := &models.Transfer{
		FromStationID: fromStationID,
		ToStationID:   toStationID,
		CreatedBy:     userID,
		Status:        models.TransferStatusPreparation,
		Notes:         notes,
	}
	if err := s.transferRepo.Create(transfer); err != nil {
		return nil, err
	}
	return transfer, nil
}

// AddItem ajoute une monture (scannée) à un transfert en préparation
func (s *TransferService) AddItem(transferID int64, barcode string) (*models.TransferItem, *models.Glass, error) {
	transfer, err := s.transferRepo.GetByID(transferID)
	if err != nil {
		return nil, nil, err
	}
	if transfer.Status != models.TransferStatusPreparation {
		return nil, nil, fmt.Errorf("ce transfert n'est plus en préparation (statut: %s)", transfer.Status)
	}

	glass, err := s.glassRepo.GetByBarcode(barcode)
	if err != nil {
		return nil, nil, fmt.Errorf("monture introuvable: %w", err)
	}
	if glass.StationID != transfer.FromStationID {
		return nil, nil, fmt.Errorf("cette monture ne se trouve pas dans la station de départ du transfert")
	}
	if !transferableStatuses[glass.Status] {
		return nil, nil, fmt.Errorf("cette monture ne peut pas être transférée (statut actuel: %s)", glass.Status)
	}
	if _, err := s.transferRepo.GetItemByGlassID(transferID, glass.ID); err == nil {
		return nil, nil, fmt.Errorf("cette monture est déjà dans ce transfert")
	}

	item := &models.TransferItem{
		TransferID: transferID,
		GlassID:    glass.ID,
		Status:     models.TransferItemStatusPending,
	}
	if err := s.transferRepo.AddItem(item); err != nil {
		return nil, nil, err
	}
	return item, glass, nil
}

// Dispatch finalise la préparation : les montures partent en transit
func (s *TransferService) Dispatch(transferID int64, userID int64) (*models.Transfer, error) {
	transfer, err := s.transferRepo.GetByID(transferID)
	if err != nil {
		return nil, err
	}
	if transfer.Status != models.TransferStatusPreparation {
		return nil, fmt.Errorf("ce transfert n'est plus en préparation (statut: %s)", transfer.Status)
	}

	items, err := s.transferRepo.ListItems(transferID)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("aucune monture dans ce transfert")
	}

	for _, item := range items {
		glass, err := s.glassRepo.GetByID(item.GlassID)
		if err != nil {
			return nil, fmt.Errorf("monture #%d introuvable: %w", item.GlassID, err)
		}

		fromLocationID := glass.LocationID
		if fromLocationID != nil {
			if err := s.allocation.FreeLocation(*fromLocationID); err != nil {
				return nil, fmt.Errorf("impossible de libérer l'emplacement de la monture #%d: %w", glass.ID, err)
			}
		}
		if err := s.glassRepo.ClearLocation(glass.ID); err != nil {
			return nil, fmt.Errorf("impossible de vider l'emplacement de la monture #%d: %w", glass.ID, err)
		}
		if err := s.glassRepo.UpdateStatus(glass.ID, models.StatusEnTransit); err != nil {
			return nil, fmt.Errorf("impossible de mettre à jour le statut de la monture #%d: %w", glass.ID, err)
		}
		// Basculé tout de suite après le statut de la monture (et non en un seul lot après la
		// boucle) : si un item suivant du même transfert échoue, celui-ci reste cohérent
		// (monture EN_TRANSIT + ligne de transfert IN_TRANSIT ensemble), au lieu de laisser la
		// monture EN_TRANSIT avec une ligne de transfert restée PENDING pour toujours.
		if err := s.transferRepo.MarkItemInTransit(item.ID); err != nil {
			return nil, fmt.Errorf("impossible de marquer la monture #%d en transit: %w", glass.ID, err)
		}

		movement := &models.Movement{
			GlassID:        glass.ID,
			FromStationID:  &transfer.FromStationID,
			ToStationID:    &transfer.ToStationID,
			FromLocationID: fromLocationID,
			Action:         models.ActionExpedition,
			UserID:         userID,
		}
		if err := s.movementRepo.Create(movement); err != nil {
			log.Printf("⚠️  Erreur création mouvement expédition (glass #%d): %v", glass.ID, err)
		}
	}

	if err := s.transferRepo.UpdateStatus(transferID, models.TransferStatusInTransit); err != nil {
		return nil, err
	}
	transfer.Status = models.TransferStatusInTransit
	return transfer, nil
}

// ReceiveItem confirme la réception d'une monture scannée à la station de destination
func (s *TransferService) ReceiveItem(transferID int64, barcode string, userID int64) (*models.TransferItem, *models.Glass, *models.StorageLocation, *models.Transfer, error) {
	transfer, err := s.transferRepo.GetByID(transferID)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	if transfer.Status != models.TransferStatusInTransit {
		return nil, nil, nil, nil, fmt.Errorf("ce transfert n'est pas en transit (statut: %s)", transfer.Status)
	}

	glass, err := s.glassRepo.GetByBarcode(barcode)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("monture introuvable: %w", err)
	}

	item, err := s.transferRepo.GetItemByGlassID(transferID, glass.ID)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("cette monture ne fait pas partie de ce transfert")
	}
	if item.Status == models.TransferItemStatusReceived {
		return nil, nil, nil, nil, fmt.Errorf("cette monture a déjà été réceptionnée")
	}

	location, err := s.allocation.FindOrCreateStockLocation(transfer.ToStationID)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("erreur allocation emplacement: %w", err)
	}

	if err := s.glassRepo.UpdateStationAndLocation(glass.ID, transfer.ToStationID, location.ID); err != nil {
		return nil, nil, nil, nil, err
	}
	if err := s.glassRepo.UpdateStatus(glass.ID, models.StatusEnStockSousStation); err != nil {
		return nil, nil, nil, nil, err
	}

	movement := &models.Movement{
		GlassID:       glass.ID,
		FromStationID: &transfer.FromStationID,
		ToStationID:   &transfer.ToStationID,
		ToLocationID:  &location.ID,
		Action:        models.ActionReceptionStation,
		UserID:        userID,
	}
	if err := s.movementRepo.Create(movement); err != nil {
		log.Printf("⚠️  Erreur création mouvement réception (glass #%d): %v", glass.ID, err)
	}

	if err := s.transferRepo.MarkItemReceived(item.ID); err != nil {
		return nil, nil, nil, nil, err
	}
	item.Status = models.TransferItemStatusReceived

	remaining, err := s.transferRepo.CountItemsNotReceived(transferID)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	if remaining == 0 {
		if err := s.transferRepo.MarkReceived(transferID, userID); err != nil {
			return nil, nil, nil, nil, err
		}
		transfer.Status = models.TransferStatusReceived
	}

	glass.StationID = transfer.ToStationID
	glass.LocationID = &location.ID
	glass.Status = models.StatusEnStockSousStation

	return item, glass, location, transfer, nil
}

// GetTransfer récupère un transfert avec ses montures
func (s *TransferService) ListItems(transferID int64) ([]models.TransferItem, error) {
	return s.transferRepo.ListItems(transferID)
}

func (s *TransferService) GetTransfer(id int64) (*models.Transfer, []models.TransferItem, error) {
	transfer, err := s.transferRepo.GetByID(id)
	if err != nil {
		return nil, nil, err
	}
	items, err := s.transferRepo.ListItems(id)
	if err != nil {
		return nil, nil, err
	}
	return transfer, items, nil
}

// ListTransfers liste les transferts avec filtres optionnels
func (s *TransferService) ListTransfers(stationID *int64, status *string) ([]models.Transfer, error) {
	return s.transferRepo.List(stationID, status)
}
