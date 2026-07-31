package repositories

import (
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/lunetterie/backend/internal/inventory/models"
)

type SaleRepository struct {
	db *sqlx.DB
}

func NewSaleRepository(db *sqlx.DB) *SaleRepository {
	return &SaleRepository{db: db}
}

func (r *SaleRepository) Create(sale *models.Sale) error {
	query := `
        INSERT INTO sales (station_id, user_id, notes)
        VALUES (:station_id, :user_id, :notes)
        RETURNING id, created_at`

	rows, err := r.db.NamedQuery(query, sale)
	if err != nil {
		return fmt.Errorf("erreur création sale: %w", err)
	}
	defer rows.Close()

	if rows.Next() {
		return rows.Scan(&sale.ID, &sale.CreatedAt)
	}
	return fmt.Errorf("aucun ID retourné après création sale")
}

func (r *SaleRepository) AddItem(item *models.SaleItem) error {
	query := `
        INSERT INTO sale_items (sale_id, glass_id)
        VALUES (:sale_id, :glass_id)
        RETURNING id`

	rows, err := r.db.NamedQuery(query, item)
	if err != nil {
		return fmt.Errorf("erreur ajout sale item: %w", err)
	}
	defer rows.Close()

	if rows.Next() {
		return rows.Scan(&item.ID)
	}
	return fmt.Errorf("aucun ID retourné après ajout sale item")
}

type ReserveRepository struct {
	db *sqlx.DB
}

func NewReserveRepository(db *sqlx.DB) *ReserveRepository {
	return &ReserveRepository{db: db}
}

func (r *ReserveRepository) Create(reserve *models.Reserve) error {
	query := `
        INSERT INTO reserves (station_id, user_id, notes)
        VALUES (:station_id, :user_id, :notes)
        RETURNING id, created_at`

	rows, err := r.db.NamedQuery(query, reserve)
	if err != nil {
		return fmt.Errorf("erreur création reserve: %w", err)
	}
	defer rows.Close()

	if rows.Next() {
		return rows.Scan(&reserve.ID, &reserve.CreatedAt)
	}
	return fmt.Errorf("aucun ID retourné après création reserve")
}

func (r *ReserveRepository) AddItem(item *models.ReserveItem) error {
	query := `
        INSERT INTO reserve_items (reserve_id, glass_id)
        VALUES (:reserve_id, :glass_id)
        RETURNING id`

	rows, err := r.db.NamedQuery(query, item)
	if err != nil {
		return fmt.Errorf("erreur ajout reserve item: %w", err)
	}
	defer rows.Close()

	if rows.Next() {
		return rows.Scan(&item.ID)
	}
	return fmt.Errorf("aucun ID retourné après ajout reserve item")
}
