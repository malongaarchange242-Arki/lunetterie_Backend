package repositories

import (
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/lunetterie/backend/internal/inventory/ports"
)

// TransactionManager demarre les transactions PostgreSQL du module inventory.
type TransactionManager struct {
	db *sqlx.DB
}

func NewTransactionManager(db *sqlx.DB) *TransactionManager {
	return &TransactionManager{db: db}
}

func (m *TransactionManager) Begin() (ports.Transaction, error) {
	tx, err := m.db.Beginx()
	if err != nil {
		return nil, fmt.Errorf("impossible de d?marrer la transaction: %w", err)
	}
	return postgresTransaction{tx: tx}, nil
}

type postgresTransaction struct{ tx *sqlx.Tx }

func (t postgresTransaction) Commit() error   { return t.tx.Commit() }
func (t postgresTransaction) Rollback() error { return t.tx.Rollback() }
func (t postgresTransaction) Unwrap() any     { return t.tx }

func sqlTransaction(tx ports.Transaction) (*sqlx.Tx, error) {
	sqlTx, ok := tx.Unwrap().(*sqlx.Tx)
	if !ok || sqlTx == nil {
		return nil, fmt.Errorf("transaction PostgreSQL inventory invalide")
	}
	return sqlTx, nil
}
