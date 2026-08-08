package repositories

import (
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/lunetterie/backend/internal/inventory/models"
)

type DemandBasketRepository struct {
	db *sqlx.DB
}

func NewDemandBasketRepository(db *sqlx.DB) *DemandBasketRepository {
	return &DemandBasketRepository{db: db}
}

func (r *DemandBasketRepository) Create(item *models.DemandBasketItem) error {
	query := `
        INSERT INTO demand_baskets (city, genre, forme, gamme, taille, source, created_by)
        VALUES ($1, $2, $3, $4, $5, $6, $7)
        RETURNING id, status, created_at, updated_at
    `
	if err := r.db.QueryRowx(query, item.City, item.Genre, item.Forme, item.Gamme, item.Taille, item.Source, item.CreatedBy).
		Scan(&item.ID, &item.Status, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return fmt.Errorf("impossible d'ajouter la demande au panier: %w", err)
	}
	return nil
}

// CountsByCity ne compte que les lignes encore ouvertes : une demande déjà adressée au stock
// principal ne doit plus gonfler le compteur du panier.
func (r *DemandBasketRepository) CountsByCity() ([]models.DemandBasketCount, error) {
	counts := []models.DemandBasketCount{}
	query := `
        SELECT city, COUNT(*) AS count
        FROM demand_baskets
        WHERE status = 'OUVERTE'
        GROUP BY city
        ORDER BY city`
	if err := r.db.Select(&counts, query); err != nil {
		return nil, fmt.Errorf("impossible de compter les paniers: %w", err)
	}
	return counts, nil
}

func (r *DemandBasketRepository) ListByCity(city string) ([]models.DemandBasketItem, error) {
	items := []models.DemandBasketItem{}
	query := `
        SELECT id, city, genre, forme, gamme, taille, source, status, created_by, created_at, updated_at
        FROM demand_baskets
        WHERE city = $1 AND status = 'OUVERTE'
        ORDER BY created_at DESC`
	if err := r.db.Select(&items, query, city); err != nil {
		return nil, fmt.Errorf("impossible de récupérer le panier de %s: %w", city, err)
	}
	return items, nil
}

// MarkSent clôt les demandes reprises dans une demande au stock principal. Le filtre sur
// 'OUVERTE' rend l'appel idempotent : rejouer la même liste ne recompte rien.
func (r *DemandBasketRepository) MarkSent(ids []int64) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	query := `
        UPDATE demand_baskets
        SET status = 'ENVOYEE', updated_at = NOW()
        WHERE id = ANY($1) AND status = 'OUVERTE'`
	result, err := r.db.Exec(query, pq.Array(ids))
	if err != nil {
		return 0, fmt.Errorf("impossible de marquer les demandes comme envoyées: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("impossible de vérifier la mise à jour: %w", err)
	}
	return rows, nil
}
