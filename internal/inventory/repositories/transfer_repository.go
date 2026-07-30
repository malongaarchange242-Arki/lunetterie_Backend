package repositories

import (
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/lunetterie/backend/internal/inventory/models"
)

// TransferRepository gère les transferts inter-stations
type TransferRepository struct {
	db *sqlx.DB
}

// NewTransferRepository crée une nouvelle instance
func NewTransferRepository(db *sqlx.DB) *TransferRepository {
	return &TransferRepository{db: db}
}

// Create crée un nouveau transfert
func (r *TransferRepository) Create(transfer *models.Transfer) error {
	query := `
		INSERT INTO transfers (from_station_id, to_station_id, created_by, status, notes)
		VALUES (:from_station_id, :to_station_id, :created_by, :status, :notes)
		RETURNING id, created_at`

	rows, err := r.db.NamedQuery(query, transfer)
	if err != nil {
		return fmt.Errorf("erreur création transfert: %w", err)
	}
	defer rows.Close()

	if rows.Next() {
		return rows.Scan(&transfer.ID, &transfer.CreatedAt)
	}
	return fmt.Errorf("aucun ID retourné après création transfert")
}

// GetByID récupère un transfert par ID
func (r *TransferRepository) GetByID(id int64) (*models.Transfer, error) {
	var transfer models.Transfer
	if err := r.db.Get(&transfer, `SELECT * FROM transfers WHERE id = $1`, id); err != nil {
		return nil, fmt.Errorf("transfert introuvable: %w", err)
	}
	return &transfer, nil
}

// List liste les transferts, avec filtres optionnels
func (r *TransferRepository) List(stationID *int64, status *string) ([]models.Transfer, error) {
	query := `SELECT * FROM transfers WHERE 1=1`
	args := []interface{}{}

	if stationID != nil {
		args = append(args, *stationID)
		query += fmt.Sprintf(" AND (from_station_id = $%d OR to_station_id = $%d)", len(args), len(args))
	}
	if status != nil {
		args = append(args, *status)
		query += fmt.Sprintf(" AND status = $%d", len(args))
	}
	query += " ORDER BY created_at DESC"

	var transfers []models.Transfer
	if err := r.db.Select(&transfers, query, args...); err != nil {
		return nil, fmt.Errorf("impossible de récupérer les transferts: %w", err)
	}
	return transfers, nil
}

// UpdateStatus met à jour le statut d'un transfert
func (r *TransferRepository) UpdateStatus(id int64, status models.TransferStatus) error {
	_, err := r.db.Exec(`UPDATE transfers SET status = $1 WHERE id = $2`, status, id)
	return err
}

// MarkReceived clôture un transfert (réception terminée)
func (r *TransferRepository) MarkReceived(id int64, receivedBy int64) error {
	_, err := r.db.Exec(
		`UPDATE transfers SET status = $1, received_by = $2, received_at = NOW() WHERE id = $3`,
		models.TransferStatusReceived, receivedBy, id,
	)
	return err
}

// AddItem ajoute une monture à un transfert
func (r *TransferRepository) AddItem(item *models.TransferItem) error {
	query := `
		INSERT INTO transfer_items (transfer_id, glass_id, status)
		VALUES (:transfer_id, :glass_id, :status)
		RETURNING id`

	rows, err := r.db.NamedQuery(query, item)
	if err != nil {
		return fmt.Errorf("erreur ajout monture au transfert: %w", err)
	}
	defer rows.Close()

	if rows.Next() {
		return rows.Scan(&item.ID)
	}
	return fmt.Errorf("aucun ID retourné après ajout monture")
}

// ListItems liste les montures d'un transfert
func (r *TransferRepository) ListItems(transferID int64) ([]models.TransferItem, error) {
	var items []models.TransferItem
	query := `SELECT * FROM transfer_items WHERE transfer_id = $1 ORDER BY id`
	if err := r.db.Select(&items, query, transferID); err != nil {
		return nil, fmt.Errorf("impossible de récupérer les montures du transfert: %w", err)
	}
	return items, nil
}

// GetItemByGlassID récupère la ligne de transfert associée à une monture
func (r *TransferRepository) GetItemByGlassID(transferID, glassID int64) (*models.TransferItem, error) {
	var item models.TransferItem
	query := `SELECT * FROM transfer_items WHERE transfer_id = $1 AND glass_id = $2`
	if err := r.db.Get(&item, query, transferID, glassID); err != nil {
		return nil, fmt.Errorf("monture introuvable dans ce transfert: %w", err)
	}
	return &item, nil
}

// GetActiveItemByGlassID récupère la ligne de transfert en cours (IN_TRANSIT) d'une monture,
// quel que soit le transfert. Renvoie une erreur si la monture n'a pas de transfert actif.
func (r *TransferRepository) GetActiveItemByGlassID(glassID int64) (*models.TransferItem, error) {
	var item models.TransferItem
	query := `SELECT * FROM transfer_items WHERE glass_id = $1 AND status = $2`
	if err := r.db.Get(&item, query, glassID, models.TransferItemStatusInTransit); err != nil {
		return nil, fmt.Errorf("aucun transfert actif pour cette monture: %w", err)
	}
	return &item, nil
}

// MarkItemsInTransit passe toutes les montures PENDING d'un transfert en IN_TRANSIT
func (r *TransferRepository) MarkItemsInTransit(transferID int64) error {
	_, err := r.db.Exec(
		`UPDATE transfer_items SET status = $1 WHERE transfer_id = $2 AND status = $3`,
		models.TransferItemStatusInTransit, transferID, models.TransferItemStatusPending,
	)
	return err
}

// MarkItemReceived marque une monture comme reçue
func (r *TransferRepository) MarkItemReceived(itemID int64) error {
	_, err := r.db.Exec(
		`UPDATE transfer_items SET status = $1, scanned_at = NOW() WHERE id = $2`,
		models.TransferItemStatusReceived, itemID,
	)
	return err
}

// CountItemsNotReceived compte les montures pas encore reçues dans un transfert
func (r *TransferRepository) CountItemsNotReceived(transferID int64) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM transfer_items WHERE transfer_id = $1 AND status != $2`
	err := r.db.Get(&count, query, transferID, models.TransferItemStatusReceived)
	return count, err
}

// CountItems compte le nombre de montures dans un transfert
func (r *TransferRepository) CountItems(transferID int64) (int, error) {
	var count int
	err := r.db.Get(&count, `SELECT COUNT(*) FROM transfer_items WHERE transfer_id = $1`, transferID)
	return count, err
}
