package repositories

import (
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/lunetterie/backend/internal/inventory/models"
)

type DeliveryRepository struct {
	db *sqlx.DB
}

func NewDeliveryRepository(db *sqlx.DB) *DeliveryRepository {
	return &DeliveryRepository{db: db}
}

func (r *DeliveryRepository) Create(delivery *models.Delivery) error {
	query := `
        INSERT INTO deliveries (supplier_id, reference, received_by, station_id, notes)
        VALUES (:supplier_id, :reference, :received_by, :station_id, :notes)
        RETURNING id, received_at, created_at`

	rows, err := r.db.NamedQuery(query, delivery)
	if err != nil {
		return fmt.Errorf("erreur création delivery: %w", err)
	}
	defer rows.Close()

	if rows.Next() {
		return rows.Scan(&delivery.ID, &delivery.ReceivedAt, &delivery.CreatedAt)
	}
	return fmt.Errorf("aucun ID retourné après création delivery")
}

func (r *DeliveryRepository) AddItem(item *models.DeliveryItem) error {
	query := `
        INSERT INTO delivery_items (delivery_id, glass_id)
        VALUES (:delivery_id, :glass_id)
        RETURNING id`

	rows, err := r.db.NamedQuery(query, item)
	if err != nil {
		return fmt.Errorf("erreur ajout delivery item: %w", err)
	}
	defer rows.Close()

	if rows.Next() {
		return rows.Scan(&item.ID)
	}
	return fmt.Errorf("aucun ID retourné après ajout delivery item")
}

func (r *DeliveryRepository) FindOrCreateDefaultSupplier() (int64, error) {
	const defaultName = "Livraison laboratoire"
	var supplierID int64
	query := `SELECT id FROM suppliers WHERE name = $1 LIMIT 1`
	if err := r.db.Get(&supplierID, query, defaultName); err == nil {
		return supplierID, nil
	}

	insert := `INSERT INTO suppliers (name) VALUES ($1) RETURNING id`
	if err := r.db.Get(&supplierID, insert, defaultName); err != nil {
		return 0, fmt.Errorf("erreur création fournisseur par défaut: %w", err)
	}
	return supplierID, nil
}
