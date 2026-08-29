package repositories

import (
	"fmt"

	"github.com/lunetterie/backend/internal/inventory/models"
	"github.com/lunetterie/backend/internal/inventory/ports"
)

func (r *LocationRepository) GetByIDTx(tx ports.Transaction, id int64) (*models.StorageLocation, error) {
	sqlTx, err := sqlTransaction(tx)
	if err != nil {
		return nil, err
	}
	var location models.StorageLocation
	if err := sqlTx.Get(&location, `SELECT * FROM storage_locations WHERE id = $1`, id); err != nil {
		return nil, fmt.Errorf("emplacement introuvable: %w", err)
	}
	return &location, nil
}

func (r *LocationRepository) CountGlassesAtLocationTx(tx ports.Transaction, locationID int64) (int, error) {
	sqlTx, err := sqlTransaction(tx)
	if err != nil {
		return 0, err
	}
	var count int
	if err := sqlTx.Get(&count, `SELECT COUNT(*) FROM glasses WHERE location_id = $1`, locationID); err != nil {
		return 0, fmt.Errorf("impossible de compter les montures de l'emplacement: %w", err)
	}
	return count, nil
}

func (r *LocationRepository) UpdateStatusTx(tx ports.Transaction, locationID int64, status string) error {
	sqlTx, err := sqlTransaction(tx)
	if err != nil {
		return err
	}
	_, err = sqlTx.Exec(`UPDATE storage_locations SET status = $1 WHERE id = $2`, status, locationID)
	return err
}
