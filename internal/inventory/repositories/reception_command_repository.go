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
			rc.shipment_status,
			rc.dispatched_at,
			rc.arrived_at,
			rc.supplier_order_id,
			rc.created_by,
			rc.activated_at,
			rc.created_at,
			rc.updated_at,
			CASE
				WHEN rc.status = 'completed' THEN 'completed'
				WHEN EXISTS (SELECT 1 FROM pre_registration_cases prc WHERE prc.reception_command_id = rc.id) THEN 'in_progress'
				ELSE 'not_started'
			END AS pre_registration_status,

			COALESCE(so.gender, '') AS order_gender,
			COALESCE(so.provenance, so.supplier, '') AS order_provenance,
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

// Delete supprime une session de réception et les montures qui y ont été enregistrées.
// C'est une suppression administrative complète : on retire d'abord les tables qui
// référencent glasses en RESTRICT, puis les montures, puis la session.
func (r *ReceptionCommandRepository) Delete(id int64) error {
	tx, err := r.db.Beginx()
	if err != nil {
		return fmt.Errorf("impossible de démarrer la suppression de la session: %w", err)
	}
	defer tx.Rollback()

	var exists bool
	if err := tx.Get(&exists, `SELECT EXISTS (SELECT 1 FROM reception_commands WHERE id = $1)`, id); err != nil {
		return fmt.Errorf("impossible de vérifier la session de réception: %w", err)
	}
	if !exists {
		return sql.ErrNoRows
	}

	dependentDeletes := []string{
		`
			WITH deleted AS (
				DELETE FROM transfer_items
				WHERE glass_id IN (SELECT id FROM glasses WHERE reception_command_id = $1)
				RETURNING transfer_id
			)
			DELETE FROM transfers
			WHERE id IN (SELECT transfer_id FROM deleted)
			  AND NOT EXISTS (SELECT 1 FROM transfer_items WHERE transfer_items.transfer_id = transfers.id)
		`,
		`
			WITH deleted AS (
				DELETE FROM sale_items
				WHERE glass_id IN (SELECT id FROM glasses WHERE reception_command_id = $1)
				RETURNING sale_id
			)
			DELETE FROM sales
			WHERE id IN (SELECT sale_id FROM deleted)
			  AND NOT EXISTS (SELECT 1 FROM sale_items WHERE sale_items.sale_id = sales.id)
		`,
		`
			WITH deleted AS (
				DELETE FROM reserve_items
				WHERE glass_id IN (SELECT id FROM glasses WHERE reception_command_id = $1)
				RETURNING reserve_id
			)
			DELETE FROM reserves
			WHERE id IN (SELECT reserve_id FROM deleted)
			  AND NOT EXISTS (SELECT 1 FROM reserve_items WHERE reserve_items.reserve_id = reserves.id)
		`,
	}
	for _, query := range dependentDeletes {
		if _, err := tx.Exec(query, id); err != nil {
			return fmt.Errorf("impossible de supprimer les dépendances des montures: %w", err)
		}
	}

	if _, err := tx.Exec(`DELETE FROM glasses WHERE reception_command_id = $1`, id); err != nil {
		return fmt.Errorf("impossible de supprimer les montures de la session: %w", err)
	}

	if _, err := tx.Exec(`DELETE FROM reception_commands WHERE id = $1`, id); err != nil {
		return fmt.Errorf("impossible de supprimer la session de réception: %w", err)
	}

	return tx.Commit()
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
				CASE
					WHEN rc.status = 'completed' THEN 'completed'
					WHEN EXISTS (SELECT 1 FROM pre_registration_cases prc WHERE prc.reception_command_id = rc.id) THEN 'in_progress'
					ELSE 'not_started'
				END AS pre_registration_status,

				COALESCE(so.gender, '') AS order_gender,
				COALESCE(so.provenance, so.supplier, '') AS order_provenance,
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
					WHEN rc.registered_count + 1 >= rc.target_count THEN 'completed'
					ELSE 'active'
				END,
				updated_at = NOW()
			WHERE rc.code = $1
			RETURNING rc.id, rc.code, rc.target_count, rc.registered_count, rc.status, rc.supplier_order_id, rc.created_by, rc.activated_at, rc.created_at, rc.updated_at
		`,
		normalizedCode,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}

		return nil, fmt.Errorf(
			"impossible d'incrémenter la commande de réception: %w",
			err,
		)
	}

	return &command, nil
}

// ListArrivedCommands récupère toutes les commandes arrivées au stock général
func (r *ReceptionCommandRepository) ListArrivedCommands() ([]models.ReceptionCommand, error) {
	commands := make([]models.ReceptionCommand, 0)

	err := r.db.Select(&commands, `
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
			rc.shipment_status,
			rc.dispatched_at,
			rc.arrived_at,
			CASE
				WHEN rc.status = 'completed' THEN 'completed'
				WHEN EXISTS (SELECT 1 FROM pre_registration_cases prc WHERE prc.reception_command_id = rc.id) THEN 'in_progress'
				ELSE 'not_started'
			END AS pre_registration_status,
			COALESCE(so.gender, '') AS order_gender,
			COALESCE(so.provenance, so.supplier, '') AS order_provenance,
			COALESCE(so.gamme, '') AS order_gamme
		FROM reception_commands rc
		LEFT JOIN supplier_orders so ON so.id = rc.supplier_order_id
		WHERE rc.shipment_status = 'arrived'
		ORDER BY rc.arrived_at DESC NULLS LAST
	`)

	if err != nil {
		return nil, fmt.Errorf("erreur recuperation commandes arrivees: %w", err)
	}

	return commands, nil
}

// ListShipmentCommands retourne les commandes physiquement expédiées. Le poste de
// scan doit aussi voir celles en transit : c'est le scan des valises qui les fait
// précisément passer à l'état "arrived".
func (r *ReceptionCommandRepository) ListShipmentCommands() ([]models.ReceptionCommand, error) {
	commands := make([]models.ReceptionCommand, 0)
	err := r.db.Select(&commands, `
		SELECT
			rc.id, rc.code, rc.target_count, rc.registered_count, rc.status,
			rc.supplier_order_id, rc.created_by, rc.activated_at, rc.created_at, rc.updated_at,
			rc.shipment_status, rc.dispatched_at, rc.arrived_at,
			CASE
				WHEN rc.status = 'completed' THEN 'completed'
				WHEN EXISTS (SELECT 1 FROM pre_registration_cases prc WHERE prc.reception_command_id = rc.id) THEN 'in_progress'
				ELSE 'not_started'
			END AS pre_registration_status,
			COALESCE(so.gender, '') AS order_gender,
			COALESCE(so.provenance, so.supplier, '') AS order_provenance,
			COALESCE(so.gamme, '') AS order_gamme
		FROM reception_commands rc
		LEFT JOIN supplier_orders so ON so.id = rc.supplier_order_id
		WHERE rc.shipment_status IN ('in_transit', 'arrived')
		ORDER BY CASE rc.shipment_status WHEN 'in_transit' THEN 0 ELSE 1 END, rc.dispatched_at DESC NULLS LAST
	`)
	if err != nil {
		return nil, fmt.Errorf("erreur récupération commandes expédiées: %w", err)
	}
	return commands, nil
}
