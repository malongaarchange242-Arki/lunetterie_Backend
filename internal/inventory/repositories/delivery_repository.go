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

// MarkHandedOver horodate la remise sur la ligne de livraison de cette monture.
//
// La plus récente seulement : une monture peut avoir été montée, remise, puis revenue en
// SAV et remontée. Sans ce garde-fou, une nouvelle remise réécrirait la date de l'ancienne.
func (r *DeliveryRepository) MarkHandedOver(glassID int64) error {
	query := `
        UPDATE delivery_items
        SET handed_over_at = NOW()
        WHERE id = (
            SELECT id FROM delivery_items
            WHERE glass_id = $1 AND handed_over_at IS NULL
            ORDER BY id DESC
            LIMIT 1
        )`
	if _, err := r.db.Exec(query, glassID); err != nil {
		return fmt.Errorf("erreur horodatage de la remise: %w", err)
	}
	return nil
}

// List rend les lignes de livraison, de la plus récente à la plus ancienne.
//
// Le pendant en lecture manquait : les tables deliveries / delivery_items se remplissaient
// depuis le poste Laboratoire sans qu'aucune route ne permette de les relire. Le montage
// terminé, la remise et son horodatage n'existaient donc que dans la base.
//
// `pendingOnly` ne garde que ce qui attend encore son client — la question que pose le
// poste Vendeuse à chaque ouverture.
func (r *DeliveryRepository) List(stationID int64, pendingOnly bool) ([]models.DeliveryLine, error) {
	lines := []models.DeliveryLine{}

	query := `
        SELECT di.id, di.delivery_id, di.glass_id, di.handed_over_at,
            g.barcode, g.status, g.price,
            ga.reference, ga.brand, ga.shape, ga.color,
            d.station_id, d.received_at
        FROM delivery_items di
        JOIN deliveries d ON d.id = di.delivery_id
        JOIN glasses g ON g.id = di.glass_id
        LEFT JOIN glass_analysis ga ON ga.id = g.analysis_id
        WHERE 1=1`

	args := []interface{}{}
	if stationID > 0 {
		args = append(args, stationID)
		query += fmt.Sprintf(" AND d.station_id = $%d", len(args))
	}
	if pendingOnly {
		query += " AND di.handed_over_at IS NULL"
	}
	query += " ORDER BY di.id DESC"

	if err := r.db.Select(&lines, query, args...); err != nil {
		return nil, fmt.Errorf("impossible de récupérer les livraisons: %w", err)
	}
	return lines, nil
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
