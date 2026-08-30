package repositories

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/lunetterie/backend/internal/inventory/models"
)

// GlassRepository gère les opérations sur les montures
type GlassRepository struct {
	db *sqlx.DB
	// onStatusChanged est prévenu après chaque changement de statut, avec la station de la
	// monture. Voir SetStatusObserver.
	onStatusChanged func(stationID int64)
}

// NewGlassRepository crée une nouvelle instance
func NewGlassRepository(db *sqlx.DB) *GlassRepository {
	return &GlassRepository{db: db}
}

// SetStatusObserver branche un observateur sur les changements de statut.
//
// Le contrôle du seuil de stock est posé ici plutôt qu'aux appelants parce que UpdateStatus
// est le passage obligé de toutes les mutations : neuf sites répartis dans cinq services
// (display, sales_and_reserves, transfer, delivery) y aboutissent. Les instrumenter un par
// un marcherait aujourd'hui et laisserait passer le dixième, écrit demain.
//
// L'observateur est appelé après que la mise à jour a pris, et son échec n'est pas remonté :
// le déplacement de la monture est acquis, une alerte manquée ne doit pas le défaire.
func (r *GlassRepository) SetStatusObserver(observer func(stationID int64)) {
	r.onStatusChanged = observer
}

// Create crée une nouvelle monture
func (r *GlassRepository) Create(glass *models.Glass) error {
	query := `
		INSERT INTO glasses (
			barcode, serial_number, frame_model_id, station_id,
			location_id, supplier_id, delivery_id, analysis_id,
			status, is_reserved, reserved_for_order, price,
			photo_monture_url, photo_branche_url, photo_arriere_url, reception_command_id, notes
		) VALUES (
			:barcode, :serial_number, :frame_model_id, :station_id,
			:location_id, :supplier_id, :delivery_id, :analysis_id,
			:status, :is_reserved, :reserved_for_order, :price,
			:photo_monture_url, :photo_branche_url, :photo_arriere_url, :reception_command_id, :notes
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
	// `g.created_at` fait partie du SELECT : sans lui, `GlassListItem.CreatedAt` reste à la
	// valeur zéro de Go et l'écran affiche « 0001-01-01 » — une date que personne ne
	// reconnaît comme une absence.
	query := `
		SELECT g.id, g.barcode, g.station_id, g.status, g.price, g.created_at, g.reception_command_id,
			g.photo_monture_url,
			ga.reference, ga.brand, ga.gender, ga.shape, ga.color, ga.size, ga.material, ga.mount_type,
			COALESCE(so.gamme, '') AS gamme,
			sl.code AS location_code,
			reg.author AS registered_by
		FROM glasses g
		LEFT JOIN LATERAL (
			SELECT ga.*
			FROM glass_analysis ga
			WHERE ga.id = g.analysis_id OR ga.glass_id = g.id
			ORDER BY (ga.id = g.analysis_id) DESC, ga.created_at DESC, ga.id DESC
			LIMIT 1
		) ga ON TRUE
		LEFT JOIN reception_commands rc ON rc.id = g.reception_command_id
		LEFT JOIN supplier_orders so ON so.id = rc.supplier_order_id
		LEFT JOIN storage_locations sl ON sl.id = g.location_id` + registrationJoin + `
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
		LEFT JOIN LATERAL (
			SELECT ga.*
			FROM glass_analysis ga
			WHERE ga.id = g.analysis_id OR ga.glass_id = g.id
			ORDER BY (ga.id = g.analysis_id) DESC, ga.created_at DESC, ga.id DESC
			LIMIT 1
		) ga ON TRUE
		WHERE g.status NOT IN ('VENDUE', 'PERDUE', 'CASSEE', 'RETOURNEE', 'RESERVEE', 'EN_TRANSIT', 'EN_LABORATOIRE', 'PRETE_A_LIVRER')
		GROUP BY ga.reference, ga.brand
		ORDER BY ga.reference NULLS LAST`
	if err := r.db.Select(&items, query, stockCriticalThreshold); err != nil {
		return nil, fmt.Errorf("impossible de calculer le résumé du stock: %w", err)
	}
	return items, nil
}

// registrationJoin rattache à chaque monture l'employé qui l'a enregistrée. L'enregistrement
// crée un mouvement RECEPTION_FOURNISSEUR (voir reception_workflow.go), c'est donc lui qu'on
// cherche en priorité. Le workflow n'échoue pas si la création du mouvement échoue : le tri
// retombe alors sur le plus ancien mouvement, faute de mieux.
//
// LEFT JOIN LATERAL et non sous-requête corrélée : une monture sans aucun mouvement reste
// dans le résultat, avec un auteur nul.
const registrationJoin = `
		LEFT JOIN LATERAL (
			SELECT NULLIF(TRIM(CONCAT(u.first_name, ' ', u.last_name)), '') AS author
			FROM movements m
			LEFT JOIN users u ON u.id = m.user_id
			WHERE m.glass_id = g.id
			ORDER BY (m.action = 'RECEPTION_FOURNISSEUR') DESC, m.created_at ASC, m.id ASC
			LIMIT 1
		) reg ON TRUE`

// FindByStatuses liste les montures filtrées par statut, toutes stations confondues.
func (r *GlassRepository) FindByStatuses(statuses []string) ([]models.GlassListItem, error) {
	return r.FindByStatusesFiltered(statuses, false)
}

func (r *GlassRepository) FindByStatusesFiltered(statuses []string, registeredOnly bool) ([]models.GlassListItem, error) {
	items := []models.GlassListItem{}
	query := `
		SELECT g.id, g.barcode, g.station_id, s.name AS station_name, g.status, g.price, g.created_at, g.reception_command_id,
			g.photo_monture_url,
			ga.reference, ga.brand, ga.gender, ga.shape, ga.color, ga.size, ga.material, ga.mount_type,
			COALESCE(so.gamme, '') AS gamme,
			sl.code AS location_code,
			reg.author AS registered_by,
			(SELECT sl2.city
			   FROM send_list_items sli2
			   JOIN send_lists sl2 ON sl2.id = sli2.list_id
			  WHERE sli2.barcode = g.barcode AND sl2.status NOT IN ('TRAITEE', 'ANNULEE')
			  ORDER BY sli2.id DESC LIMIT 1) AS reserved_for_city
		FROM glasses g
		LEFT JOIN LATERAL (
			SELECT ga.*
			FROM glass_analysis ga
			WHERE ga.id = g.analysis_id OR ga.glass_id = g.id
			ORDER BY (ga.id = g.analysis_id) DESC, ga.created_at DESC, ga.id DESC
			LIMIT 1
		) ga ON TRUE
		LEFT JOIN reception_commands rc ON rc.id = g.reception_command_id
		LEFT JOIN supplier_orders so ON so.id = rc.supplier_order_id
		LEFT JOIN storage_locations sl ON sl.id = g.location_id
		LEFT JOIN stations s ON s.id = g.station_id` + registrationJoin + `
		WHERE g.status = ANY($1)`
	if registeredOnly {
		query += `
		  AND g.reception_command_id IS NOT NULL`
	}
	query += `
		ORDER BY g.created_at DESC`
	if err := r.db.Select(&items, query, pq.Array(statuses)); err != nil {
		return nil, fmt.Errorf("impossible de récupérer les montures: %w", err)
	}
	return items, nil
}

func (r *GlassRepository) FindByReceptionCommand(commandID int64) ([]models.GlassListItem, error) {
	items := []models.GlassListItem{}
	query := `
		SELECT g.id, g.barcode, g.station_id, s.name AS station_name, g.status, g.price, g.created_at, g.reception_command_id,
			g.photo_monture_url,
			ga.reference, ga.brand, ga.gender, ga.shape, ga.color, ga.size, ga.material, ga.mount_type,
			COALESCE(so.gamme, '') AS gamme,
			sl.code AS location_code
		FROM glasses g
		LEFT JOIN LATERAL (
			SELECT ga.*
			FROM glass_analysis ga
			WHERE ga.id = g.analysis_id OR ga.glass_id = g.id
			ORDER BY (ga.id = g.analysis_id) DESC, ga.created_at DESC, ga.id DESC
			LIMIT 1
		) ga ON TRUE
		LEFT JOIN reception_commands rc ON rc.id = g.reception_command_id
		LEFT JOIN supplier_orders so ON so.id = rc.supplier_order_id
		LEFT JOIN storage_locations sl ON sl.id = g.location_id
		LEFT JOIN stations s ON s.id = g.station_id
		WHERE g.reception_command_id = $1
		ORDER BY g.created_at DESC`
	if err := r.db.Select(&items, query, commandID); err != nil {
		return nil, fmt.Errorf("impossible de récupérer les montures de la session: %w", err)
	}
	return items, nil
}

func (r *GlassRepository) FindLunCngAnalysisRepairCandidates(limit int) ([]models.GlassAnalysisRepairCandidate, error) {
	if limit < 1 || limit > 100 {
		limit = 25
	}

	items := []models.GlassAnalysisRepairCandidate{}
	query := `
		SELECT
			g.id,
			g.barcode,
			COALESCE(g.photo_monture_url, '') AS photo_monture_url,
			g.analysis_id,
			ga.reference,
			ga.brand,
			ga.shape,
			ga.gender
		FROM glasses g
		LEFT JOIN LATERAL (
			SELECT ga.*
			FROM glass_analysis ga
			WHERE ga.id = g.analysis_id OR ga.glass_id = g.id
			ORDER BY (ga.id = g.analysis_id) DESC, ga.created_at DESC, ga.id DESC
			LIMIT 1
		) ga ON TRUE
		WHERE g.barcode LIKE 'LUN-CNG-%'
		  AND g.reception_command_id IS NOT NULL
		  AND COALESCE(g.photo_monture_url, '') <> ''
		  AND COALESCE(ga.model_version, '') NOT IN ('repair-lun-cng-1.0.0', 'repair-img-err-1.0')
		  AND (
			NULLIF(TRIM(COALESCE(ga.reference, '')), '') IS NULL
			OR NULLIF(TRIM(COALESCE(ga.brand, '')), '') IS NULL
			OR NULLIF(TRIM(COALESCE(ga.shape, '')), '') IS NULL
			OR NULLIF(TRIM(COALESCE(ga.gender, '')), '') IS NULL
		  )
		ORDER BY g.created_at ASC, g.id ASC
		LIMIT $1`
	if err := r.db.Select(&items, query, limit); err != nil {
		return nil, fmt.Errorf("impossible de récupérer les montures à réparer: %w", err)
	}
	return items, nil
}

// FindDetailByBarcode recherche une monture par code-barres (toutes stations confondues),
// avec ses attributs issus de l'analyse IA/vérification et son emplacement actuel.
func (r *GlassRepository) FindDetailByBarcode(barcode string) (*models.GlassListItem, error) {
	var item models.GlassListItem
	query := `
		SELECT g.id, g.barcode, g.station_id, s.name AS station_name, g.status, g.price, g.created_at, g.reception_command_id,
			g.photo_monture_url,
			ga.reference, ga.brand, ga.gender, ga.shape, ga.color, ga.size, ga.material,
			sl.code AS location_code
		FROM glasses g
		LEFT JOIN LATERAL (
			SELECT ga.*
			FROM glass_analysis ga
			WHERE ga.id = g.analysis_id OR ga.glass_id = g.id
			ORDER BY (ga.id = g.analysis_id) DESC, ga.created_at DESC, ga.id DESC
			LIMIT 1
		) ga ON TRUE
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
		SELECT g.id, g.barcode, g.station_id, s.name AS station_name, g.status, g.price, g.created_at, g.reception_command_id,
			g.photo_monture_url,
			ga.reference, ga.brand, ga.gender, ga.shape, ga.color, ga.size, ga.material,
			sl.code AS location_code
		FROM glasses g
		LEFT JOIN LATERAL (
			SELECT ga.*
			FROM glass_analysis ga
			WHERE ga.id = g.analysis_id OR ga.glass_id = g.id
			ORDER BY (ga.id = g.analysis_id) DESC, ga.created_at DESC, ga.id DESC
			LIMIT 1
		) ga ON TRUE
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
//
// RETURNING plutôt qu'une seconde lecture : la station doit être celle sur laquelle la
// mutation vient de porter, et un SELECT séparé pourrait tomber après qu'un transfert
// concurrent l'a déjà déplacée.
//
// La jointure sur `previous` rend le statut d'avant : la clause FROM lit l'instantané
// antérieur à la mise à jour. Il sert à écarter les mutations sans effet sur les compteurs
// surveillés — sans ce filtre, l'expédition d'un carton de cinquante montures déclencherait
// cinquante contrôles de seuil pour des statuts qui ne comptent ni au présentoir ni au
// stock local.
func (r *GlassRepository) UpdateStatus(glassID int64, status models.GlassStatus) error {
	query := `
		UPDATE glasses g
		SET status = $1, updated_at = NOW()
		FROM glasses previous
		WHERE g.id = $2 AND previous.id = g.id
		RETURNING g.station_id, previous.status`

	var stationID sql.NullInt64
	var previousStatus string
	if err := r.db.QueryRowx(query, status, glassID).Scan(&stationID, &previousStatus); err != nil {
		// Aucune ligne : la monture n'existe pas. L'ancien Exec restait muet dans ce cas,
		// on garde ce silence pour ne pas transformer en erreur ce que les appelants
		// traitaient jusqu'ici comme un succès.
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return err
	}

	// Les deux sens comptent : une monture qui entre en présentoir le garnit, une qui en
	// sort le vide. Ne regarder que le statut d'arrivée manquerait la moitié des baisses.
	touchesStock := models.TracksStock(status) || models.TracksStock(models.GlassStatus(previousStatus))
	if r.onStatusChanged != nil && stationID.Valid && touchesStock {
		r.onStatusChanged(stationID.Int64)
	}
	return nil
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
