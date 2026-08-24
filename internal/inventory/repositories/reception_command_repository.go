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

// Create crée une nouvelle session de réception.
func (r *ReceptionCommandRepository) Create(command *models.ReceptionCommand) error {
	query := `
		INSERT INTO reception_commands (
			code,
			target_count,
			registered_count,
			status,
			supplier_order_id,
			created_by
		)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at
	`

	return r.db.QueryRowx(
		query,
		command.Code,
		command.TargetCount,
		command.RegisteredCount,
		command.Status,
		command.SupplierOrderID,
		command.CreatedBy,
	).Scan(
		&command.ID,
		&command.CreatedAt,
		&command.UpdatedAt,
	)
}

// GetLatestCodeWithPrefix retourne le dernier code généré pour un préfixe.
func (r *ReceptionCommandRepository) GetLatestCodeWithPrefix(prefix string) (string, error) {
	var code string

	err := r.db.Get(
		&code,
		`
			SELECT code
			FROM reception_commands
			WHERE code LIKE $1
			ORDER BY code DESC
			LIMIT 1
		`,
		prefix+"%",
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}

		return "", fmt.Errorf(
			"impossible de récupérer le dernier code de réception: %w",
			err,
		)
	}

	return code, nil
}

// List retourne toutes les sessions de réception.
// COALESCE est utilisé pour éviter les erreurs sqlx lors du mapping
// des anciennes sessions dont supplier_order_id est NULL ou ne possède
// pas de commande fournisseur associée.
func (r *ReceptionCommandRepository) List(status string) ([]models.ReceptionCommand, error) {
	commands := make([]models.ReceptionCommand, 0)

	query := `
		SELECT
			rc.id,
			rc.code,
			rc.target_count,
			rc.registered_count,
			rc.status,
			rc.supplier_order_id,
			rc.created_by,
			rc.activated_at,
			rc.created_at,
			rc.updated_at,

			COALESCE(so.gender, '') AS order_gender,
			COALESCE(so.gamme, '') AS order_gamme

		FROM reception_commands rc

		LEFT JOIN supplier_orders so
			ON so.id = rc.supplier_order_id
	`

	args := make([]interface{}, 0)

	if strings.TrimSpace(status) != "" {
		query += `
			WHERE rc.status = $1
		`
		args = append(args, strings.TrimSpace(status))
	}

	query += `
		ORDER BY rc.created_at DESC
	`

	if err := r.db.Select(&commands, query, args...); err != nil {
		return nil, fmt.Errorf(
			"impossible de lister les commandes de réception: %w",
			err,
		)
	}

	return commands, nil
}

// GetByCode récupère une session à partir de son code.
func (r *ReceptionCommandRepository) GetByCode(code string) (*models.ReceptionCommand, error) {
	var command models.ReceptionCommand

	normalizedCode := strings.ToUpper(strings.TrimSpace(code))

	if normalizedCode == "" {
		return nil, fmt.Errorf("code de session vide")
	}

	err := r.db.Get(
		&command,
		`
			SELECT
				rc.id,
				rc.code,
				rc.target_count,
				rc.registered_count,
				rc.status,
				rc.supplier_order_id,
				rc.created_by,
				rc.activated_at,
				rc.created_at,
				rc.updated_at,

				COALESCE(so.gender, '') AS order_gender,
				COALESCE(so.gamme, '') AS order_gamme

			FROM reception_commands rc

			LEFT JOIN supplier_orders so
				ON so.id = rc.supplier_order_id

			WHERE rc.code = $1

			LIMIT 1
		`,
		normalizedCode,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}

		return nil, fmt.Errorf(
			"impossible de récupérer la commande de réception: %w",
			err,
		)
	}

	return &command, nil
}

// Activate active une session de réception.
func (r *ReceptionCommandRepository) Activate(code string) (*models.ReceptionCommand, error) {
	command, err := r.GetByCode(code)
	if err != nil {
		return nil, err
	}

	if command == nil {
		return nil, nil
	}

	// Déjà terminée ou déjà activée.
	if command.Status != "active" || command.ActivatedAt != nil {
		return command, nil
	}

	now := time.Now()

	result, err := r.db.Exec(
		`
			UPDATE reception_commands
			SET
				activated_at = $1,
				updated_at = NOW()
			WHERE id = $2
		`,
		now,
		command.ID,
	)

	if err != nil {
		return nil, fmt.Errorf(
			"impossible d'activer la commande de réception: %w",
			err,
		)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf(
			"impossible de vérifier l'activation de la commande: %w",
			err,
		)
	}

	if rowsAffected == 0 {
		return nil, fmt.Errorf(
			"aucune session n'a été activée",
		)
	}

	command.ActivatedAt = &now
	command.UpdatedAt = now

	return command, nil
}

// Increment incrémente le nombre de lunettes enregistrées dans une session.
func (r *ReceptionCommandRepository) Increment(code string) (*models.ReceptionCommand, error) {
	var command models.ReceptionCommand

	normalizedCode := strings.ToUpper(strings.TrimSpace(code))

	if normalizedCode == "" {
		return nil, fmt.Errorf("code de session vide")
	}

	err := r.db.Get(
		&command,
		`
			UPDATE reception_commands rc

			SET
				registered_count = LEAST(
					rc.target_count,
					rc.registered_count + 1
				),

				status = CASE
					WHEN rc.registered_count + 1 >= rc.target_count
						THEN 'completed'
					ELSE rc.status
				END,

				updated_at = NOW()

			WHERE rc.code = $1
			  AND rc.status = 'active'
			  AND rc.registered_count < rc.target_count

			RETURNING
				rc.id,
				rc.code,
				rc.target_count,
				rc.registered_count,
				rc.status,
				rc.supplier_order_id,
				rc.created_by,
				rc.activated_at,
				rc.created_at,
				rc.updated_at,

				COALESCE(
					(
						SELECT so.gender
						FROM supplier_orders so
						WHERE so.id = rc.supplier_order_id
					),
					''
				) AS order_gender,

				COALESCE(
					(
						SELECT so.gamme
						FROM supplier_orders so
						WHERE so.id = rc.supplier_order_id
					),
					''
				) AS order_gamme
		`,
		normalizedCode,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf(
				"session introuvable, fermée ou quota atteint",
			)
		}

		return nil, fmt.Errorf(
			"impossible d'incrémenter la commande de réception: %w",
			err,
		)
	}

	return &command, nil
}
