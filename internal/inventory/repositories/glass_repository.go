package repositories

import (
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/lunetterie/backend/internal/inventory/models"
)

// GlassRepository gère les opérations sur les montures
type GlassRepository struct {
	db *sqlx.DB
}

// NewGlassRepository crée une nouvelle instance
func NewGlassRepository(db *sqlx.DB) *GlassRepository {
	return &GlassRepository{db: db}
}

// Create crée une nouvelle monture
func (r *GlassRepository) Create(glass *models.Glass) error {
	query := `
		INSERT INTO glasses (
			barcode, serial_number, frame_model_id, station_id,
			location_id, supplier_id, delivery_id, analysis_id,
			status, is_reserved, reserved_for_order, price,
			photo_monture_url, photo_branche_url, notes
		) VALUES (
			:barcode, :serial_number, :frame_model_id, :station_id,
			:location_id, :supplier_id, :delivery_id, :analysis_id,
			:status, :is_reserved, :reserved_for_order, :price,
			:photo_monture_url, :photo_branche_url, :notes
		) RETURNING id, created_at, updated_at`

	rows, err := r.db.NamedQuery(query, glass)
	if err != nil {
		return fmt.Errorf("erreur création glass: %w", err)
	}
	defer rows.Close()

	if rows.Next() {
		return rows.Scan(&glass.ID, &glass.CreatedAt, &glass.UpdatedAt)
	}

	return fmt.Errorf("aucun ID retourné après création")
}

// GetByID récupère une monture par ID
func (r *GlassRepository) GetByID(id int64) (*models.Glass, error) {
	var glass models.Glass
	query := `SELECT * FROM glasses WHERE id = $1`
	err := r.db.Get(&glass, query, id)
	if err != nil {
		return nil, fmt.Errorf("glass introuvable: %w", err)
	}
	return &glass, nil
}

// GetByBarcode récupère une monture par code-barres
func (r *GlassRepository) GetByBarcode(barcode string) (*models.Glass, error) {
	var glass models.Glass
	query := `SELECT * FROM glasses WHERE barcode = $1`
	err := r.db.Get(&glass, query, barcode)
	if err != nil {
		return nil, fmt.Errorf("glass introuvable: %w", err)
	}
	return &glass, nil
}

// UpdateStatus met à jour le statut d'une monture
func (r *GlassRepository) UpdateStatus(glassID int64, status models.GlassStatus) error {
	query := `
		UPDATE glasses 
		SET status = $1, updated_at = NOW() 
		WHERE id = $2`
	_, err := r.db.Exec(query, status, glassID)
	return err
}

// UpdateLocation met à jour l'emplacement d'une monture
func (r *GlassRepository) UpdateLocation(glassID int64, locationID int64) error {
	query := `
		UPDATE glasses 
		SET location_id = $1, updated_at = NOW() 
		WHERE id = $2`
	_, err := r.db.Exec(query, locationID, glassID)
	return err
}

// ClearLocation vide l'emplacement d'une monture (ex: départ en transit)
func (r *GlassRepository) ClearLocation(glassID int64) error {
	query := `
		UPDATE glasses
		SET location_id = NULL, updated_at = NOW()
		WHERE id = $1`
	_, err := r.db.Exec(query, glassID)
	return err
}

// UpdateStationAndLocation change la station et l'emplacement d'une monture (ex: réception d'un transfert)
func (r *GlassRepository) UpdateStationAndLocation(glassID, stationID, locationID int64) error {
	query := `
		UPDATE glasses
		SET station_id = $1, location_id = $2, updated_at = NOW()
		WHERE id = $3`
	_, err := r.db.Exec(query, stationID, locationID, glassID)
	return err
}

// UpdateAnalysis met à jour l'analyse liée d'une monture
func (r *GlassRepository) UpdateAnalysis(glassID int64, analysisID int64) error {
	query := `
		UPDATE glasses 
		SET analysis_id = $1, updated_at = NOW() 
		WHERE id = $2`
	_, err := r.db.Exec(query, analysisID, glassID)
	return err
}
