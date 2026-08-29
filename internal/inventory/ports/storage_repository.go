package ports

import "github.com/lunetterie/backend/internal/inventory/models"

// StorageRepository expose la validation necessaire aux affectations inventory.
type StorageRepository interface {
	GetByID(id int64) (*models.StorageLocation, error)
	CountGlassesAtLocation(locationID int64) (int, error)
	UpdateStatus(locationID int64, status string) error
}

type TransactionalStorageRepository interface {
	GetByIDTx(tx Transaction, id int64) (*models.StorageLocation, error)
	CountGlassesAtLocationTx(tx Transaction, locationID int64) (int, error)
	UpdateStatusTx(tx Transaction, locationID int64, status string) error
}
