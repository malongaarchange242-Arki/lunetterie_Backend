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
	SELECT rc.id, rc.code, rc.target_count, rc.registered_count, rc.status, rc.supplier_order_id, rc.created_by, rc.activated_at, rc.created_at, rc.updated_at,
	       so.gender AS order_gender, so.gamme AS order_gamme
	FROM reception_commands rc
	LEFT JOIN supplier_orders so ON so.id = rc.supplier_order_id
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
	SELECT rc.id, rc.code, rc.target_count, rc.registered_count, rc.status, rc.supplier_order_id, rc.created_by, rc.activated_at, rc.created_at, rc.updated_at,
	       so.gender AS order_gender, so.gamme AS order_gamme
	FROM reception_commands rc
	LEFT JOIN supplier_orders so ON so.id = rc.supplier_order_id
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
		UPDATE reception_commands rc
		SET registered_count = LEAST(rc.target_count, rc.registered_count + 1),
		    status = CASE WHEN rc.registered_count + 1 >= rc.target_count THEN 'completed' ELSE rc.status END,
		    updated_at = NOW()
		WHERE rc.code = $1
		  AND rc.status = 'active'
		  AND rc.registered_count < rc.target_count
		RETURNING rc.id, rc.code, rc.target_count, rc.registered_count, rc.status,
		          rc.supplier_order_id, rc.created_by, rc.activated_at, rc.created_at, rc.updated_at,
		          ''::varchar AS order_gender, ''::varchar AS order_gamme
	`, strings.ToUpper(strings.TrimSpace(code)))
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("session introuvable, fermée ou quota atteint")
		}
		return nil, fmt.Errorf("impossible de récupérer la commande: %w", err)
	}
	return &command, nil
}
