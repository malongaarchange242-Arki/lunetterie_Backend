package repositories

import (
	"fmt"

	"github.com/lunetterie/backend/internal/inventory/models"
	"github.com/lunetterie/backend/internal/inventory/ports"
)

func (r *GlassRepository) CreateTx(tx ports.Transaction, glass *models.Glass) error {
	sqlTx, err := sqlTransaction(tx)
	if err != nil {
		return err
	}
	rows, err := sqlTx.NamedQuery(`INSERT INTO glasses (barcode, serial_number, frame_model_id, station_id, location_id, supplier_id, delivery_id, analysis_id, status, is_reserved, reserved_for_order, price, photo_monture_url, reception_command_id, notes) VALUES (:barcode, :serial_number, :frame_model_id, :station_id, :location_id, :supplier_id, :delivery_id, :analysis_id, :status, :is_reserved, :reserved_for_order, :price, :photo_monture_url, :reception_command_id, :notes) RETURNING id, created_at, updated_at`, glass)
	if err != nil {
		return fmt.Errorf("erreur création glass: %w", err)
	}
	defer rows.Close()
	if rows.Next() {
		return rows.Scan(&glass.ID, &glass.CreatedAt, &glass.UpdatedAt)
	}
	return fmt.Errorf("aucun ID retourné après création")
}

func (r *GlassRepository) GetByIDTx(tx ports.Transaction, id int64) (*models.Glass, error) {
	sqlTx, err := sqlTransaction(tx)
	if err != nil {
		return nil, err
	}
	var glass models.Glass
	if err := sqlTx.Get(&glass, `SELECT * FROM glasses WHERE id = $1`, id); err != nil {
		return nil, fmt.Errorf("glass introuvable: %w", err)
	}
	return &glass, nil
}

func (r *GlassRepository) UpdateLocationTx(tx ports.Transaction, glassID, locationID int64) error {
	sqlTx, err := sqlTransaction(tx)
	if err != nil {
		return err
	}
	_, err = sqlTx.Exec(`UPDATE glasses SET location_id = $1, updated_at = NOW() WHERE id = $2`, locationID, glassID)
	return err
}

func (r *GlassRepository) UpdateStatusTx(tx ports.Transaction, glassID int64, status models.GlassStatus) error {
	sqlTx, err := sqlTransaction(tx)
	if err != nil {
		return err
	}
	_, err = sqlTx.Exec(`UPDATE glasses SET status = $1, updated_at = NOW() WHERE id = $2`, status, glassID)
	return err
}

func (r *GlassRepository) UpdateReservedStateTx(tx ports.Transaction, glassID int64, reserved bool) error {
	sqlTx, err := sqlTransaction(tx)
	if err != nil {
		return err
	}
	_, err = sqlTx.Exec(`UPDATE glasses SET is_reserved = $1, updated_at = NOW() WHERE id = $2`, reserved, glassID)
	return err
}
