package repositories

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/lunetterie/backend/internal/inventory/models"
)

const notificationColumns = `id, user_id, type, message, station_id, stock_type,
        current_stock, reference_stock, threshold, read_at, created_at`

type NotificationRepository struct {
	db *sqlx.DB
}

func NewNotificationRepository(db *sqlx.DB) *NotificationRepository {
	return &NotificationRepository{db: db}
}

// StationReference rend la quantité de référence de la station pour ce type de stock, et le
// nom de la station pour le message. Une référence absente sort à zéro : la station n'a pas
// de normale connue, il n'y a pas de 10 % à calculer et aucune alerte ne doit partir.
func (r *NotificationRepository) StationReference(stationID int64, stockType models.StockType) (int, string, error) {
	column := "local_reference_qty"
	if stockType == models.StockTypePresentoir {
		column = "presentoir_reference_qty"
	}

	var row struct {
		Reference sql.NullInt64 `db:"reference"`
		Name      string        `db:"name"`
	}
	query := `SELECT COALESCE(` + column + `, 0) AS reference, name FROM stations WHERE id = $1`
	if err := r.db.Get(&row, query, stationID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, "", nil
		}
		return 0, "", fmt.Errorf("impossible de lire la référence de stock: %w", err)
	}
	return int(row.Reference.Int64), row.Name, nil
}

// CurrentStock compte les montures que la station détient dans ce stock. Présentoir et stock
// local partagent le station_id et ne se distinguent que par le statut.
func (r *NotificationRepository) CurrentStock(stationID int64, stockType models.StockType) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM glasses WHERE station_id = $1 AND status = $2`
	if err := r.db.Get(&count, query, stationID, string(stockType.GlassStatus())); err != nil {
		return 0, fmt.Errorf("impossible de compter le stock de la station: %w", err)
	}
	return count, nil
}

// Arm passe l'alerte à l'état actif et dit si elle vient de basculer.
//
// C'est tout le mécanisme anti-spam : la clause WHERE ne laisse passer la mise à jour que
// si l'alerte était inactive, donc seul le franchissement du seuil rend `true`. Les
// mutations suivantes sous le seuil retrouvent une alerte déjà active et rendent `false`.
//
// L'atomicité vient de la contrainte UNIQUE(station_id, stock_type) : deux mutations
// concurrentes sur la même station se sérialisent sur la ligne, et la seconde ne voit plus
// active = false. Impossible d'émettre deux notifications pour un même franchissement.
func (r *NotificationRepository) Arm(stationID int64, stockType models.StockType) (bool, error) {
	query := `
        INSERT INTO stock_alert_states (station_id, stock_type, active, triggered_at, updated_at)
        VALUES ($1, $2, true, NOW(), NOW())
        ON CONFLICT (station_id, stock_type) DO UPDATE
            SET active = true, triggered_at = NOW(), updated_at = NOW()
            WHERE stock_alert_states.active = false
        RETURNING id`

	var id int64
	if err := r.db.QueryRowx(query, stationID, string(stockType)).Scan(&id); err != nil {
		// Pas de ligne rendue : la clause WHERE a écarté la mise à jour, l'alerte était
		// déjà active. Ce n'est pas une erreur, c'est le cas courant.
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("impossible d'armer l'alerte de stock: %w", err)
	}
	return true, nil
}

// Disarm réarme l'alerte quand le stock repasse au-dessus du seuil : le prochain
// franchissement vers le bas notifiera de nouveau.
func (r *NotificationRepository) Disarm(stationID int64, stockType models.StockType) error {
	query := `
        INSERT INTO stock_alert_states (station_id, stock_type, active, cleared_at, updated_at)
        VALUES ($1, $2, false, NOW(), NOW())
        ON CONFLICT (station_id, stock_type) DO UPDATE
            SET active = false, cleared_at = NOW(), updated_at = NOW()
            WHERE stock_alert_states.active = true`

	if _, err := r.db.Exec(query, stationID, string(stockType)); err != nil {
		return fmt.Errorf("impossible de réarmer l'alerte de stock: %w", err)
	}
	return nil
}

// Recipients rend qui doit recevoir l'alerte.
//
// Présentoir : le responsable de la station concernée, et lui seul — c'est lui qui regarnit
// le meuble. Stock local : les administrateurs, qui décident d'un réapprovisionnement.
//
// Le filtre porte sur roles.name et non sur un identifiant en dur : les migrations 025 et
// 028 posent des identifiants de rôle conditionnels, avec un repli qui laisse la séquence
// choisir. Un numéro écrit ici pourrait désigner un autre rôle selon l'ordre d'exécution.
func (r *NotificationRepository) Recipients(stationID int64, stockType models.StockType) ([]int64, error) {
	ids := []int64{}

	var query string
	var args []interface{}
	if stockType == models.StockTypePresentoir {
		query = `
            SELECT u.id FROM users u
            JOIN roles r ON r.id = u.role_id
            WHERE r.name = 'RESPONSABLE_STATION'
              AND u.station_id = $1
              AND COALESCE(u.is_active, true)
            ORDER BY u.id`
		args = []interface{}{stationID}
	} else {
		// Les comptes ADMIN semés par les migrations ont station_id à NULL : ils ne sont
		// rattachés à aucune station, un filtre par station ne rendrait personne.
		query = `
            SELECT u.id FROM users u
            JOIN roles r ON r.id = u.role_id
            WHERE r.name IN ('ADMIN', 'SUPER_ADMIN')
              AND COALESCE(u.is_active, true)
            ORDER BY u.id`
	}

	if err := r.db.Select(&ids, query, args...); err != nil {
		return nil, fmt.Errorf("impossible de déterminer les destinataires de l'alerte: %w", err)
	}
	return ids, nil
}

// CreateMany insère les notifications d'un même franchissement en une transaction : soit
// tous les destinataires sont prévenus, soit aucun ne l'est, plutôt qu'un responsable averti
// et son administrateur non.
func (r *NotificationRepository) CreateMany(notifications []*models.Notification) error {
	if len(notifications) == 0 {
		return nil
	}

	tx, err := r.db.Beginx()
	if err != nil {
		return fmt.Errorf("impossible d'ouvrir la transaction de notification: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	query := `
        INSERT INTO notifications (user_id, type, message, station_id, stock_type,
            current_stock, reference_stock, threshold)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
        RETURNING id, created_at`

	for _, n := range notifications {
		if err := tx.QueryRowx(query, n.UserID, n.Type, n.Message, n.StationID, n.StockType,
			n.CurrentStock, n.ReferenceStock, n.Threshold).Scan(&n.ID, &n.CreatedAt); err != nil {
			return fmt.Errorf("impossible d'enregistrer la notification: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("impossible de valider les notifications: %w", err)
	}
	return nil
}

// ListForUser rend les notifications d'un utilisateur, de la plus récente à la plus ancienne.
func (r *NotificationRepository) ListForUser(userID int64, unreadOnly bool) ([]models.Notification, error) {
	// Initialisée non-nulle : une boîte vide doit sortir en `[]` et non en `null`, sinon le
	// front reçoit autre chose qu'une liste sur son premier appel.
	notifications := []models.Notification{}

	query := `SELECT ` + notificationColumns + ` FROM notifications WHERE user_id = $1`
	if unreadOnly {
		query += ` AND read_at IS NULL`
	}
	query += ` ORDER BY created_at DESC`

	if err := r.db.Select(&notifications, query, userID); err != nil {
		return nil, fmt.Errorf("impossible de récupérer les notifications: %w", err)
	}
	return notifications, nil
}

// MarkRead marque une notification comme lue. Le filtre sur user_id est une garde et non un
// raffinement : sans lui, n'importe qui pourrait marquer lue la notification d'un autre en
// devinant un identifiant.
func (r *NotificationRepository) MarkRead(userID, notificationID int64) error {
	query := `UPDATE notifications SET read_at = NOW()
              WHERE id = $1 AND user_id = $2 AND read_at IS NULL`
	if _, err := r.db.Exec(query, notificationID, userID); err != nil {
		return fmt.Errorf("impossible de marquer la notification comme lue: %w", err)
	}
	return nil
}

func (r *NotificationRepository) MarkAllRead(userID int64) error {
	query := `UPDATE notifications SET read_at = NOW() WHERE user_id = $1 AND read_at IS NULL`
	if _, err := r.db.Exec(query, userID); err != nil {
		return fmt.Errorf("impossible de marquer les notifications comme lues: %w", err)
	}
	return nil
}
