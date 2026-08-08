package repositories

import (
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
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
			photo_monture_url, photo_branche_url, reception_command_id, notes
		) VALUES (
			:barcode, :serial_number, :frame_model_id, :station_id,
			:location_id, :supplier_id, :delivery_id, :analysis_id,
			:status, :is_reserved, :reserved_for_order, :price,
			:photo_monture_url, :photo_branche_url, :reception_command_id, :notes
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

// FindByStationAndStatuses liste les montures d'une station filtrées par statut,
// avec leurs attributs (référence, forme, couleur...) issus de l'analyse IA/vérification.
func (r *GlassRepository) FindByStationAndStatuses(stationID int64, statuses []string) ([]models.GlassListItem, error) {
	items := []models.GlassListItem{}
	query := `
		SELECT g.id, g.barcode, g.station_id, g.status, g.price,
			g.photo_monture_url, g.photo_branche_url,
			ga.reference, ga.brand, ga.gender, ga.shape, ga.color, ga.size, ga.material,
			sl.code AS location_code,
			lm.author AS last_action_by, lm.action AS last_action
		FROM glasses g
		LEFT JOIN glass_analysis ga ON ga.id = g.analysis_id
		LEFT JOIN storage_locations sl ON sl.id = g.location_id` + lastMovementJoin + `
		WHERE g.station_id = $1 AND g.status = ANY($2)
		ORDER BY g.created_at DESC`
	if err := r.db.Select(&items, query, stationID, pq.Array(statuses)); err != nil {
		return nil, fmt.Errorf("impossible de récupérer les montures: %w", err)
	}
	return items, nil
}

// stockCriticalThreshold : une référence est jugée en stock critique quand sa quantité
// totale active (Stock Général + Stock Local + Présentoir) descend à ce seuil ou en dessous.
const stockCriticalThreshold = 2

// GetStockSummaryByReference agrège le stock actif par référence, réparti entre
// Stock Général (station "Stock Principal"), Stock Local (station "Station Pointe-Noire")
// et Présentoir. Exclut les montures vendues, perdues, cassées ou retournées.
func (r *GlassRepository) GetStockSummaryByReference() ([]models.StockSummaryItem, error) {
	items := []models.StockSummaryItem{}
	query := `
		SELECT
			ga.reference, ga.brand,
			COUNT(*) FILTER (WHERE g.status = 'EN_STOCK_GENERAL') AS qty_general,
			COUNT(*) FILTER (WHERE g.status = 'EN_STOCK_SOUS_STATION') AS qty_local,
			COUNT(*) FILTER (WHERE g.status = 'EN_PRESENTOIR') AS qty_presentoir,
			COUNT(*) AS qty_total,
			(COUNT(*) <= $1) AS is_critical
		FROM glasses g
		LEFT JOIN glass_analysis ga ON ga.id = g.analysis_id
		WHERE g.status NOT IN ('VENDUE', 'PERDUE', 'CASSEE', 'RETOURNEE', 'RESERVEE', 'EN_TRANSIT', 'EN_LABORATOIRE', 'PRETE_A_LIVRER')
		GROUP BY ga.reference, ga.brand
		ORDER BY ga.reference NULLS LAST`
	if err := r.db.Select(&items, query, stockCriticalThreshold); err != nil {
		return nil, fmt.Errorf("impossible de calculer le résumé du stock: %w", err)
	}
	return items, nil
}

// lastMovementJoin rattache à chaque monture l'auteur et l'action de son dernier mouvement.
// LEFT JOIN LATERAL et non une sous-requête corrélée par colonne : une seule lecture de
// movements pour les deux champs, et une monture sans mouvement reste dans le résultat.
const lastMovementJoin = `
		LEFT JOIN LATERAL (
			SELECT m.action,
				NULLIF(TRIM(CONCAT(u.first_name, ' ', u.last_name)), '') AS author
			FROM movements m
			LEFT JOIN users u ON u.id = m.user_id
			WHERE m.glass_id = g.id
			ORDER BY m.created_at DESC, m.id DESC
			LIMIT 1
		) lm ON TRUE`

// FindByStatuses liste les montures filtrées par statut, toutes stations confondues.
func (r *GlassRepository) FindByStatuses(statuses []string) ([]models.GlassListItem, error) {
	items := []models.GlassListItem{}
	query := `
		SELECT g.id, g.barcode, g.station_id, s.name AS station_name, g.status, g.price, g.created_at,
			g.photo_monture_url, g.photo_branche_url,
			ga.reference, ga.brand, ga.gender, ga.shape, ga.color, ga.size, ga.material,
			sl.code AS location_code,
			lm.author AS last_action_by, lm.action AS last_action
		FROM glasses g
		LEFT JOIN glass_analysis ga ON ga.id = g.analysis_id
		LEFT JOIN storage_locations sl ON sl.id = g.location_id
		LEFT JOIN stations s ON s.id = g.station_id` + lastMovementJoin + `
		WHERE g.status = ANY($1)
		ORDER BY g.created_at DESC`
	if err := r.db.Select(&items, query, pq.Array(statuses)); err != nil {
		return nil, fmt.Errorf("impossible de récupérer les montures: %w", err)
	}
	return items, nil
}

func (r *GlassRepository) FindByReceptionCommand(commandID int64) ([]models.GlassListItem, error) {
	items := []models.GlassListItem{}
	query := `
		SELECT g.id, g.barcode, g.station_id, s.name AS station_name, g.status, g.price, g.created_at,
			g.photo_monture_url, g.photo_branche_url,
			ga.reference, ga.brand, ga.gender, ga.shape, ga.color, ga.size, ga.material,
			sl.code AS location_code
		FROM glasses g
		LEFT JOIN glass_analysis ga ON ga.id = g.analysis_id
		LEFT JOIN storage_locations sl ON sl.id = g.location_id
		LEFT JOIN stations s ON s.id = g.station_id
		WHERE g.reception_command_id = $1
		ORDER BY g.created_at DESC`
	if err := r.db.Select(&items, query, commandID); err != nil {
		return nil, fmt.Errorf("impossible de récupérer les montures de la session: %w", err)
	}
	return items, nil
}

// FindDetailByBarcode recherche une monture par code-barres (toutes stations confondues),
// avec ses attributs issus de l'analyse IA/vérification et son emplacement actuel.
func (r *GlassRepository) FindDetailByBarcode(barcode string) (*models.GlassListItem, error) {
	var item models.GlassListItem
	query := `
		SELECT g.id, g.barcode, g.station_id, s.name AS station_name, g.status, g.price,
			g.photo_monture_url, g.photo_branche_url,
			ga.reference, ga.brand, ga.gender, ga.shape, ga.color, ga.size, ga.material,
			sl.code AS location_code
		FROM glasses g
		LEFT JOIN glass_analysis ga ON ga.id = g.analysis_id
		LEFT JOIN storage_locations sl ON sl.id = g.location_id
		LEFT JOIN stations s ON s.id = g.station_id
		WHERE g.barcode = $1`
	if err := r.db.Get(&item, query, barcode); err != nil {
		return nil, fmt.Errorf("monture introuvable: %w", err)
	}
	return &item, nil
}

// availableStatuses liste les statuts d'une monture pouvant être proposée comme alternative à
// un client (en stock ou exposée) — à l'exclusion des montures réservées, en transit, en
// laboratoire ou déjà sorties du stock actif (vendue, perdue, cassée, retournée).
var availableStatuses = []string{
	string(models.StatusEnStockGeneral),
	string(models.StatusEnStockSousStation),
	string(models.StatusEnPresentoir),
}

// FindAvailableExcluding liste les montures disponibles (hors réservées/vendues/etc.), à
// l'exclusion d'une monture donnée — utilisé pour chercher des alternatives similaires à une
// monture de référence.
func (r *GlassRepository) FindAvailableExcluding(excludeID int64) ([]models.GlassListItem, error) {
	items := []models.GlassListItem{}
	query := `
		SELECT g.id, g.barcode, g.station_id, s.name AS station_name, g.status, g.price,
			g.photo_monture_url, g.photo_branche_url,
			ga.reference, ga.brand, ga.gender, ga.shape, ga.color, ga.size, ga.material,
			sl.code AS location_code
		FROM glasses g
		LEFT JOIN glass_analysis ga ON ga.id = g.analysis_id
		LEFT JOIN storage_locations sl ON sl.id = g.location_id
		LEFT JOIN stations s ON s.id = g.station_id
		WHERE g.status = ANY($1) AND g.id != $2
		ORDER BY g.created_at DESC`
	if err := r.db.Select(&items, query, pq.Array(availableStatuses), excludeID); err != nil {
		return nil, fmt.Errorf("impossible de récupérer les montures disponibles: %w", err)
	}
	return items, nil
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

// UpdateReservedState met à jour l'état réservé d'une monture
func (r *GlassRepository) UpdateReservedState(glassID int64, reserved bool) error {
	query := `
        UPDATE glasses 
        SET is_reserved = $1, updated_at = NOW() 
        WHERE id = $2`
	_, err := r.db.Exec(query, reserved, glassID)
	return err
}
