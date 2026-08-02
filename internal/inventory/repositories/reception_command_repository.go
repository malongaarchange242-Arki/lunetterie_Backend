package repositories

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/lunetterie/backend/internal/inventory/models"
)

type ReceptionCommandRepository struct {
	db *sqlx.DB
}

func NewReceptionCommandRepository(db *sqlx.DB) *ReceptionCommandRepository {
	return &ReceptionCommandRepository{db: db}
}

func (r *ReceptionCommandRepository) Create(command *models.ReceptionCommand) error {
	query := `
        INSERT INTO reception_commands (code, target_count, registered_count, status, created_by)
        VALUES ($1, $2, $3, $4, $5)
        RETURNING id, created_at, updated_at
    `
	return r.db.QueryRowx(query, command.Code, command.TargetCount, command.RegisteredCount, command.Status, command.CreatedBy).
		Scan(&command.ID, &command.CreatedAt, &command.UpdatedAt)
}

func (r *ReceptionCommandRepository) GetByCode(code string) (*models.ReceptionCommand, error) {
	var command models.ReceptionCommand
	err := r.db.Get(&command, `
        SELECT id, code, target_count, registered_count, status, created_by, created_at, updated_at
        FROM reception_commands
        WHERE code = $1
    `, strings.ToUpper(strings.TrimSpace(code)))
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("impossible de récupérer la commande: %w", err)
	}
	return &command, nil
}

func (r *ReceptionCommandRepository) Increment(code string) (*models.ReceptionCommand, error) {
	var command models.ReceptionCommand
	err := r.db.Get(&command, `
        SELECT id, code, target_count, registered_count, status, created_by, created_at, updated_at
        FROM reception_commands
        WHERE code = $1
    `, strings.ToUpper(strings.TrimSpace(code)))
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("impossible de récupérer la commande: %w", err)
	}
	if command.Status != "active" {
		return &command, nil
	}
	if command.RegisteredCount >= command.TargetCount {
		command.Status = "completed"
		_, err = r.db.Exec(`
            UPDATE reception_commands
            SET registered_count = $1, status = $2, updated_at = NOW()
            WHERE id = $3
        `, command.RegisteredCount, command.Status, command.ID)
		return &command, err
	}
	command.RegisteredCount++
	if command.RegisteredCount >= command.TargetCount {
		command.Status = "completed"
	}
	_, err = r.db.Exec(`
        UPDATE reception_commands
        SET registered_count = $1, status = $2, updated_at = NOW()
        WHERE id = $3
    `, command.RegisteredCount, command.Status, command.ID)
	if err != nil {
		return nil, fmt.Errorf("impossible de mettre à jour la commande: %w", err)
	}
	return &command, nil
}
