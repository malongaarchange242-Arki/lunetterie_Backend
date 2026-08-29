package repositories

import (
	"fmt"

	"github.com/lunetterie/backend/internal/inventory/models"
	"github.com/lunetterie/backend/internal/inventory/ports"
)

func (r *MovementRepository) CreateTx(tx ports.Transaction, movement *models.Movement) error {
	sqlTx, err := sqlTransaction(tx)
	if err != nil {
		return err
	}
	rows, err := sqlTx.NamedQuery(`INSERT INTO movements (glass_id, from_station_id, to_station_id, from_location_id, to_location_id, action, user_id, notes) VALUES (:glass_id, :from_station_id, :to_station_id, :from_location_id, :to_location_id, :action, :user_id, :notes) RETURNING id, created_at`, movement)
	if err != nil {
		return fmt.Errorf("erreur création mouvement: %w", err)
	}
	defer rows.Close()
	if rows.Next() {
		return rows.Scan(&movement.ID, &movement.CreatedAt)
	}
	return fmt.Errorf("aucun ID retourné après création mouvement")
}
