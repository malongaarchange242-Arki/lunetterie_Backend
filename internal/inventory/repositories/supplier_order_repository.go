package repositories

import (
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/lunetterie/backend/internal/inventory/models"
)

type SupplierOrderRepository struct {
	db *sqlx.DB
}

func NewSupplierOrderRepository(db *sqlx.DB) *SupplierOrderRepository {
	return &SupplierOrderRepository{db: db}
}

func (r *SupplierOrderRepository) Create(order *models.SupplierOrder) error {
	query := `
        INSERT INTO supplier_orders (supplier, quantity, order_date, note, created_by)
        VALUES ($1, $2, $3, $4, $5)
        RETURNING id, created_at, updated_at
    `
	return r.db.QueryRowx(query, order.Supplier, order.Quantity, order.OrderDate, order.Note, order.CreatedBy).
		Scan(&order.ID, &order.CreatedAt, &order.UpdatedAt)
}

func (r *SupplierOrderRepository) List() ([]models.SupplierOrder, error) {
	orders := []models.SupplierOrder{}
	err := r.db.Select(&orders, `
        SELECT id, supplier, quantity, order_date, note, created_by, created_at, updated_at
        FROM supplier_orders
        ORDER BY order_date DESC, created_at DESC
    `)
	if err != nil {
		return nil, fmt.Errorf("impossible de récupérer les commandes fournisseur: %w", err)
	}
	return orders, nil
}

func (r *SupplierOrderRepository) Delete(id int64) error {
	result, err := r.db.Exec(`DELETE FROM supplier_orders WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("impossible de supprimer la commande fournisseur: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("impossible de vérifier la suppression: %w", err)
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}
