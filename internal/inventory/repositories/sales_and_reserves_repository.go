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

// List rend les mises en réserve, de la plus récente à la plus ancienne, avec la monture.
//
// Le pendant en lecture manquait : les tables reserves / reserve_items se remplissaient
// sans qu'aucune route ne permette de les relire. La réserve ne se voyait donc qu'à travers
// le statut RESERVEE de la monture — ce qui efface toute réservation passée dès que la
// monture repart, et interdit de savoir qui avait mis quoi de côté, et quand.
//
// `activeOnly` ne garde que les montures encore réservées aujourd'hui.
func (r *ReserveRepository) List(stationID int64, activeOnly bool) ([]models.ReserveLine, error) {
	lines := []models.ReserveLine{}

	query := `
        SELECT ri.id, ri.reserve_id, ri.glass_id,
            res.station_id, res.user_id, res.created_at AS reserved_at,
            g.barcode, g.status, g.price,
            ga.reference, ga.brand, ga.shape, ga.color
        FROM reserve_items ri
        JOIN reserves res ON res.id = ri.reserve_id
        JOIN glasses g ON g.id = ri.glass_id
        LEFT JOIN glass_analysis ga ON ga.id = g.analysis_id
        WHERE 1=1`

	args := []interface{}{}
	if stationID > 0 {
		args = append(args, stationID)
		query += fmt.Sprintf(" AND res.station_id = $%d", len(args))
	}
	if activeOnly {
		args = append(args, models.StatusReservee)
		query += fmt.Sprintf(" AND g.status = $%d", len(args))
	}
	query += " ORDER BY res.created_at DESC, ri.id DESC"

	if err := r.db.Select(&lines, query, args...); err != nil {
		return nil, fmt.Errorf("impossible de récupérer les réserves: %w", err)
	}
	return lines, nil
}

// FindExpired liste les montures encore réservées dont la mise de côté remonte à plus de
// `days` jours. La date de référence est celle de la réservation, jamais celle de la monture.
//
// LATERAL et non un simple JOIN : une monture peut avoir été réservée plusieurs fois (client
// qui renonce, puis un autre qui réserve). Seule la dernière réservation fait courir le délai —
// un JOIN plat ferait ressortir la monture sur sa plus vieille réservation et la renverrait au
// présentoir alors qu'elle vient d'être remise de côté.
func (r *ReserveRepository) FindExpired(days int) ([]models.ExpiredReserve, error) {
	items := []models.ExpiredReserve{}
	query := `
        SELECT g.id AS glass_id, g.barcode, g.station_id, g.location_id,
               last_reserve.user_id, last_reserve.created_at AS reserved_at
        FROM glasses g
        JOIN LATERAL (
            SELECT res.user_id, res.created_at
            FROM reserve_items ri
            JOIN reserves res ON res.id = ri.reserve_id
            WHERE ri.glass_id = g.id
            ORDER BY res.created_at DESC
            LIMIT 1
        ) last_reserve ON TRUE
        WHERE g.status = $1
          AND last_reserve.created_at < NOW() - make_interval(days => $2::int)`
	if err := r.db.Select(&items, query, models.StatusReservee, days); err != nil {
		return nil, fmt.Errorf("impossible de lister les réservations expirées: %w", err)
	}
	return items, nil
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
