package repositories

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

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
        INSERT INTO reception_commands (code, target_count, registered_count, status, supplier_order_id, created_by)
        VALUES ($1, $2, $3, $4, $5, $6)
        RETURNING id, created_at, updated_at
    `
	return r.db.QueryRowx(query, command.Code, command.TargetCount, command.RegisteredCount, command.Status, command.SupplierOrderID, command.CreatedBy).
		Scan(&command.ID, &command.CreatedAt, &command.UpdatedAt)
}

func (r *ReceptionCommandRepository) GetLatestCodeWithPrefix(prefix string) (string, error) {
	var code string
	err := r.db.Get(&code, `
        SELECT code
        FROM reception_commands
        WHERE code LIKE $1
        ORDER BY code DESC
        LIMIT 1
    `, prefix+"%")
	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
		return "", err
	}
	return code, nil
}

func (r *ReceptionCommandRepository) List(status string) ([]models.ReceptionCommand, error) {
	commands := []models.ReceptionCommand{}
	query := `
        SELECT id, code, target_count, registered_count, status, supplier_order_id, created_by, activated_at, created_at, updated_at
        FROM reception_commands
    `
	args := []interface{}{}
	if status != "" {
		query += " WHERE status = $1"
		args = append(args, status)
	}
	query += " ORDER BY created_at DESC"
	if err := r.db.Select(&commands, query, args...); err != nil {
		return nil, fmt.Errorf("impossible de lister les commandes: %w", err)
	}
	return commands, nil
}

func (r *ReceptionCommandRepository) GetByCode(code string) (*models.ReceptionCommand, error) {
	var command models.ReceptionCommand
	err := r.db.Get(&command, `
        SELECT id, code, target_count, registered_count, status, supplier_order_id, created_by, activated_at, created_at, updated_at
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

func (r *ReceptionCommandRepository) Activate(code string) (*models.ReceptionCommand, error) {
	command, err := r.GetByCode(code)
	if err != nil {
		return nil, err
	}
	if command == nil {
		return nil, nil
	}
	if command.Status != "active" || command.ActivatedAt != nil {
		return command, nil
	}

	now := time.Now()
	if _, err := r.db.Exec(`
        UPDATE reception_commands
        SET activated_at = $1, updated_at = NOW()
        WHERE id = $2
    `, now, command.ID); err != nil {
		return nil, fmt.Errorf("impossible d'activer la commande: %w", err)
	}
	command.ActivatedAt = &now
	return command, nil
}

func (r *ReceptionCommandRepository) Increment(code string) (*models.ReceptionCommand, error) {
	var command models.ReceptionCommand
	err := r.db.Get(&command, `
        SELECT id, code, target_count, registered_count, status, supplier_order_id, created_by, activated_at, created_at, updated_at
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
