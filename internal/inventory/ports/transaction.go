package ports

// Transaction encapsule une transaction PostgreSQL du module inventory.
type Transaction interface {
	Commit() error
	Rollback() error
	Unwrap() any
}

// TransactionManager demarre une transaction appartenant au module inventory.
type TransactionManager interface {
	Begin() (Transaction, error)
}