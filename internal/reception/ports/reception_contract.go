package ports

import (
	inventoryModels "github.com/lunetterie/backend/internal/inventory/models"
)

// InventoryGateway décrit le contrat que le module Reception utilise pour vérifier le stock
// et affecter la monture dans un carton déjà préparé, sans créer de carton automatiquement.
type InventoryGateway interface {
	HasAvailableCarton(stationID int64) (bool, error)
	FindFreeCarton(stationID int64) (*inventoryModels.StorageLocation, error)
}

// ReceiptGateway regroupe les dépendances externes du workflow de réception.
type ReceiptGateway interface {
	CreateGlass(input any) error
	ReserveLocation(stationID int64) (string, error)
}

// ReceptionService décrit le contrat métier attendu par les handlers HTTP.
type ReceptionService interface {
	CreateReceptionSession(stationID int64, userID int64) (string, error)
	CompleteReception(sessionID string, payload any) error
}
