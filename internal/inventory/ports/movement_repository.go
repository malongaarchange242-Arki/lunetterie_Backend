package ports

import "github.com/lunetterie/backend/internal/inventory/models"

// MovementRepository enregistre l'historique des mutations physiques.
type MovementRepository interface {
	Create(movement *models.Movement) error
}

type TransactionalMovementRepository interface {
	CreateTx(tx Transaction, movement *models.Movement) error
}
