package repositories

import (
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/lunetterie/backend/internal/inventory/models"
)

// MovementRepository gère les mouvements de montures
type MovementRepository struct {
	db *sqlx.DB
}

// NewMovementRepository crée une nouvelle instance
func NewMovementRepository(db *sqlx.DB) *MovementRepository {
	return &MovementRepository{db: db}
}

// Create crée un nouveau mouvement
func (r *MovementRepository) Create(movement *models.Movement) error {
	query := `
		INSERT INTO movements (
			glass_id, from_station_id, to_station_id,
			from_location_id, to_location_id, action, user_id, notes
		) VALUES (
			:glass_id, :from_station_id, :to_station_id,
			:from_location_id, :to_location_id, :action, :user_id, :notes
		) RETURNING id, created_at`

	rows, err := r.db.NamedQuery(query, movement)
	if err != nil {
		return fmt.Errorf("erreur création mouvement: %w", err)
	}
	defer rows.Close()

	if rows.Next() {
		return rows.Scan(&movement.ID, &movement.CreatedAt)
	}

	return fmt.Errorf("aucun ID retourné après création mouvement")
}

// List liste les mouvements enrichis (monture, postes, emplacements, utilisateur), les plus
// récents d'abord, avec filtres optionnels — pour l'historique/traffic des montures.
func (r *MovementRepository) List(filters models.MovementFilters) ([]models.MovementListItem, int, error) {
	base := `
		FROM movements m
		JOIN glasses g ON g.id = m.glass_id
		LEFT JOIN glass_analysis ga ON ga.id = g.analysis_id
		LEFT JOIN stations fs ON fs.id = m.from_station_id
		LEFT JOIN stations ts ON ts.id = m.to_station_id
		LEFT JOIN storage_locations fl ON fl.id = m.from_location_id
		LEFT JOIN storage_locations tl ON tl.id = m.to_location_id
		LEFT JOIN users u ON u.id = m.user_id
		WHERE 1=1`

	args := []interface{}{}
	if filters.StationID != nil {
		args = append(args, *filters.StationID)
		base += fmt.Sprintf(" AND (m.from_station_id = $%d OR m.to_station_id = $%d)", len(args), len(args))
	}
	if filters.Action != nil && *filters.Action != "" {
		args = append(args, *filters.Action)
		base += fmt.Sprintf(" AND m.action = $%d", len(args))
	}
	if filters.Barcode != nil && *filters.Barcode != "" {
		args = append(args, "%"+*filters.Barcode+"%")
		base += fmt.Sprintf(" AND g.barcode ILIKE $%d", len(args))
	}
	if filters.DateFrom != nil && *filters.DateFrom != "" {
		args = append(args, *filters.DateFrom)
		base += fmt.Sprintf(" AND m.created_at >= $%d", len(args))
	}
	if filters.DateTo != nil && *filters.DateTo != "" {
		args = append(args, *filters.DateTo)
		base += fmt.Sprintf(" AND m.created_at < ($%d::date + INTERVAL '1 day')", len(args))
	}

	var total int
	if err := r.db.Get(&total, "SELECT COUNT(*) "+base, args...); err != nil {
		return nil, 0, fmt.Errorf("impossible de compter les mouvements: %w", err)
	}

	limit := filters.Limit
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	args = append(args, limit)
	limitClause := fmt.Sprintf(" ORDER BY m.created_at DESC LIMIT $%d", len(args))
	args = append(args, filters.Offset)
	limitClause += fmt.Sprintf(" OFFSET $%d", len(args))

	query := `
		SELECT
			m.id, m.glass_id, g.barcode, ga.reference, ga.brand,
			m.from_station_id, fs.name AS from_station_name,
			m.to_station_id, ts.name AS to_station_name,
			fl.code AS from_location_code,
			tl.code AS to_location_code,
			m.action,
			u.first_name AS user_first_name, u.last_name AS user_last_name,
			m.notes, m.created_at ` + base + limitClause

	items := []models.MovementListItem{}
	if err := r.db.Select(&items, query, args...); err != nil {
		return nil, 0, fmt.Errorf("impossible de récupérer les mouvements: %w", err)
	}
	return items, total, nil
}
