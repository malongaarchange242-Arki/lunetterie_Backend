package services

import (
	"fmt"
	"mime/multipart"

	"github.com/jmoiron/sqlx"
	"github.com/lunetterie/backend/internal/inventory/dto"
	inventoryModels "github.com/lunetterie/backend/internal/inventory/models"
	receptionPorts "github.com/lunetterie/backend/internal/reception/ports"
	"github.com/lunetterie/backend/internal/workflows"
)

// InventoryGateway décrit le contrat que Reception utilise pour vérifier les cartons déjà préparés.
// Ce contrat interdit la création automatique d’un carton au niveau Reception.
type InventoryGateway interface {
	HasAvailableCarton(stationID int64) (bool, error)
	FindFreeCarton(stationID int64) (*inventoryModels.StorageLocation, error)
}

// inventoryGatewayAdapter est la version concrète du contrat Inventory utilisé par Reception.
type inventoryGatewayAdapter struct {
	db *sqlx.DB
}

func NewInventoryGateway(db *sqlx.DB) receptionPorts.InventoryGateway {
	return &inventoryGatewayAdapter{db: db}
}

func (a *inventoryGatewayAdapter) HasAvailableCarton(stationID int64) (bool, error) {
	if a.db == nil {
		return false, fmt.Errorf("connexion DB unavailable")
	}
	var exists bool
	if err := a.db.Get(&exists, `
		SELECT EXISTS (
			SELECT 1
			FROM storage_locations sl
			WHERE sl.station_id = $1
			  AND sl.type = 'CARTON'
			  AND sl.status = 'LIBRE'
			  AND (sl.capacity IS NULL OR (
				SELECT COUNT(*) FROM glasses g WHERE g.location_id = sl.id
			  ) < sl.capacity)
		)`, stationID); err != nil {
		return false, fmt.Errorf("impossible de vérifier un carton disponible: %w", err)
	}
	return exists, nil
}

func (a *inventoryGatewayAdapter) FindFreeCarton(stationID int64) (*inventoryModels.StorageLocation, error) {
	if a.db == nil {
		return nil, fmt.Errorf("connexion DB unavailable")
	}
	var location inventoryModels.StorageLocation
	if err := a.db.Get(&location, `
		SELECT sl.id, sl.station_id, sl.code, sl.name, sl.barcode, sl.type, sl.zone, sl.capacity, sl.status
		FROM storage_locations sl
		WHERE sl.station_id = $1
		  AND sl.type = 'CARTON'
		  AND sl.status = 'LIBRE'
		  AND (sl.capacity IS NULL OR (
			SELECT COUNT(*) FROM glasses g WHERE g.location_id = sl.id
		  ) < sl.capacity)
		ORDER BY sl.code
		LIMIT 1`, stationID); err != nil {
		return nil, fmt.Errorf("aucun carton disponible: veuillez créer/préparer un carton")
	}
	return &location, nil
}

// ReceptionService adapte le workflow historique de réception au module Reception.
type ReceptionService struct {
	workflow *workflows.ReceptionWorkflow
	inventory InventoryGateway
}

func NewReceptionService(workflow *workflows.ReceptionWorkflow, inventory InventoryGateway) *ReceptionService {
	return &ReceptionService{workflow: workflow, inventory: inventory}
}

func (s *ReceptionService) Execute(req dto.ReceptionRequest, montureImage multipart.File, brancheImage multipart.File, userID int64) (*dto.ReceptionResponse, error) {
	if s.workflow == nil {
		return nil, fmt.Errorf("workflow de réception non initialisé")
	}
	if s.inventory != nil {
		available, err := s.inventory.HasAvailableCarton(req.StationID)
		if err != nil {
			return nil, err
		}
		if !available {
			return nil, fmt.Errorf("Veuillez créer/préparer un carton avant de réceptionner")
		}
	}
	return s.workflow.Execute(req, montureImage, brancheImage, userID)
}

func (s *ReceptionService) CreateReceptionSession(stationID int64, userID int64) (string, error) {
	if stationID <= 0 {
		return "", fmt.Errorf("station_id invalide")
	}
	if s.workflow == nil {
		return "", fmt.Errorf("workflow de réception non initialisé")
	}
	_ = userID
	return "reception-session-pending", nil
}

func (s *ReceptionService) CompleteReception(sessionID string, payload any) error {
	if sessionID == "" {
		return fmt.Errorf("session_id requis")
	}
	if s.workflow == nil {
		return fmt.Errorf("workflow de réception non initialisé")
	}
	_ = payload
	return nil
}
