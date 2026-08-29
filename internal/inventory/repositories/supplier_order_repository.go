package repositories

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/lunetterie/backend/internal/inventory/models"
)

type SupplierOrderRepository struct{ db *sqlx.DB }

func NewSupplierOrderRepository(db *sqlx.DB) *SupplierOrderRepository {
	return &SupplierOrderRepository{db: db}
}

func supplierOrderColumnNames(db *sqlx.DB) ([]string, error) {
	rows, err := db.Queryx(`
		SELECT column_name
		FROM information_schema.columns
		WHERE table_name = 'supplier_orders'
		ORDER BY ordinal_position
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cols := []string{}
	for rows.Next() {
		var col string
		if err := rows.Scan(&col); err != nil {
			return nil, err
		}
		cols = append(cols, col)
	}
	return cols, rows.Err()
}

func hasSupplierOrderColumn(cols []string, want string) bool {
	for _, col := range cols {
		if col == want {
			return true
		}
	}
	return false
}

func supplierOrderInsertSpec(cols []string, order *models.SupplierOrder) ([]string, []interface{}) {
	fields := []string{"supplier", "quantity", "order_date"}
	values := []interface{}{order.Supplier, order.Quantity, order.OrderDate}

	if hasSupplierOrderColumn(cols, "reference") {
		fields = append(fields, "reference")
		values = append(values, order.Reference)
	}
	if hasSupplierOrderColumn(cols, "provenance") {
		fields = append(fields, "provenance")
		values = append(values, order.Provenance)
	}
	if hasSupplierOrderColumn(cols, "destination") {
		fields = append(fields, "destination")
		values = append(values, order.Destination)
	}
	if hasSupplierOrderColumn(cols, "gender") {
		fields = append(fields, "gender")
		values = append(values, order.Gender)
	}
	if hasSupplierOrderColumn(cols, "gamme") {
		fields = append(fields, "gamme")
		values = append(values, order.Gamme)
	}
	if hasSupplierOrderColumn(cols, "transport") {
		fields = append(fields, "transport")
		values = append(values, order.Transport)
	}
	if hasSupplierOrderColumn(cols, "barcode_num") {
		fields = append(fields, "barcode_num")
		values = append(values, order.BarcodeNum)
	}
	if hasSupplierOrderColumn(cols, "status") {
		fields = append(fields, "status")
		values = append(values, order.Status)
	}
	if hasSupplierOrderColumn(cols, "note") {
		fields = append(fields, "note")
		values = append(values, order.Note)
	}
	if hasSupplierOrderColumn(cols, "created_by") {
		fields = append(fields, "created_by")
		values = append(values, order.CreatedBy)
	}

	return fields, values
}

func supplierOrderSelectColumns(cols []string) []string {
	base := []string{"id", "supplier", "quantity", "order_date", "created_at", "updated_at"}
	if hasSupplierOrderColumn(cols, "reference") {
		base = append(base, "reference")
	}
	if hasSupplierOrderColumn(cols, "provenance") {
		base = append(base, "provenance")
	}
	if hasSupplierOrderColumn(cols, "destination") {
		base = append(base, "destination")
	}
	if hasSupplierOrderColumn(cols, "gender") {
		base = append(base, "gender")
	}
	if hasSupplierOrderColumn(cols, "gamme") {
		base = append(base, "gamme")
	}
	if hasSupplierOrderColumn(cols, "transport") {
		base = append(base, "transport")
	}
	if hasSupplierOrderColumn(cols, "barcode_num") {
		base = append(base, "barcode_num")
	}
	if hasSupplierOrderColumn(cols, "status") {
		base = append(base, "status")
	}
	if hasSupplierOrderColumn(cols, "note") {
		base = append(base, "note")
	}
	if hasSupplierOrderColumn(cols, "created_by") {
		base = append(base, "created_by")
	}
	return base
}

func (r *SupplierOrderRepository) Create(order *models.SupplierOrder) error {
	cols, err := supplierOrderColumnNames(r.db)
	if err != nil {
		return fmt.Errorf("impossible de lire le schéma supplier_orders: %w", err)
	}
	insertCols, values := supplierOrderInsertSpec(cols, order)
	placeholders := make([]string, len(insertCols))
	for i := range placeholders {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
	}

	query := fmt.Sprintf(
		`INSERT INTO supplier_orders (%s) VALUES (%s) RETURNING id, created_at, updated_at`,
		strings.Join(insertCols, ", "),
		strings.Join(placeholders, ", "),
	)
	return r.db.QueryRowx(query, values...).Scan(&order.ID, &order.CreatedAt, &order.UpdatedAt)
}

func (r *SupplierOrderRepository) List() ([]models.SupplierOrder, error) {
	cols, err := supplierOrderColumnNames(r.db)
	if err != nil {
		return nil, fmt.Errorf("impossible de lire le schéma supplier_orders: %w", err)
	}
	selectCols := supplierOrderSelectColumns(cols)
	if len(selectCols) == 0 {
		selectCols = []string{"id", "supplier", "quantity", "order_date", "created_at", "updated_at"}
	}

	tmpl := fmt.Sprintf(`
		SELECT %s
		FROM supplier_orders
		ORDER BY order_date DESC, created_at DESC
	`, strings.Join(selectCols, ", "))

	orders := []models.SupplierOrder{}
	if err := r.db.Select(&orders, tmpl); err != nil {
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
