package ports

import "github.com/lunetterie/backend/internal/inventory/models"

// GlassRepository expose uniquement les mutations dont le domaine inventory a besoin.
type GlassRepository interface {
	Create(glass *models.Glass) error
	GetByID(id int64) (*models.Glass, error)
	UpdateLocation(glassID, locationID int64) error
	UpdateStatus(glassID int64, status models.GlassStatus) error
	UpdateReservedState(glassID int64, reserved bool) error
}

// TransactionalGlassRepository expose les memes mutations sur une transaction ouverte.
type TransactionalGlassRepository interface {
	GetByIDTx(tx Transaction, id int64) (*models.Glass, error)
	CreateTx(tx Transaction, glass *models.Glass) error
	UpdateLocationTx(tx Transaction, glassID, locationID int64) error
	UpdateStatusTx(tx Transaction, glassID int64, status models.GlassStatus) error
	UpdateReservedStateTx(tx Transaction, glassID int64, reserved bool) error
}
