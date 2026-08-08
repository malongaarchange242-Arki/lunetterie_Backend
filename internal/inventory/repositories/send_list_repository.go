package repositories

import (
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/lunetterie/backend/internal/inventory/models"
)

type SendListRepository struct {
	db *sqlx.DB
}

func NewSendListRepository(db *sqlx.DB) *SendListRepository {
	return &SendListRepository{db: db}
}

// Create insère l'en-tête et ses lignes dans une transaction : une liste enregistrée sans
// son contenu serait un ordre de préparation vide, pire qu'aucune liste.
func (r *SendListRepository) Create(list *models.SendList, items []models.SendListItemRequest) error {
	tx, err := r.db.Beginx()
	if err != nil {
		return fmt.Errorf("impossible d'ouvrir la transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	list.ItemCount = len(items)
	header := `
        INSERT INTO send_lists (session_code, city, item_count, created_by)
        VALUES ($1, $2, $3, $4)
        RETURNING id, status, created_at, updated_at`
	if err := tx.QueryRowx(header, list.SessionCode, list.City, list.ItemCount, list.CreatedBy).
		Scan(&list.ID, &list.Status, &list.CreatedAt, &list.UpdatedAt); err != nil {
		return fmt.Errorf("impossible d'enregistrer la liste: %w", err)
	}

	if len(items) > 0 {
		line := `
            INSERT INTO send_list_items (list_id, glass_id, barcode, reference, brand, location_code)
            VALUES ($1, $2, $3, $4, $5, $6)`
		for _, item := range items {
			if _, err := tx.Exec(line, list.ID, item.GlassID,
				nullIfEmpty(item.Barcode), nullIfEmpty(item.Reference),
				nullIfEmpty(item.Brand), nullIfEmpty(item.LocationCode)); err != nil {
				return fmt.Errorf("impossible d'enregistrer une ligne de la liste: %w", err)
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("impossible de valider la liste: %w", err)
	}
	return nil
}

func nullIfEmpty(value string) interface{} {
	if value == "" {
		return nil
	}
	return value
}

// List renvoie les listes, filtrées par statut si demandé, les plus récentes d'abord.
func (r *SendListRepository) List(status string) ([]models.SendList, error) {
	lists := []models.SendList{}
	query := `
        SELECT id, session_code, city, item_count, status, created_by, created_at, updated_at
        FROM send_lists
        WHERE ($1 = '' OR status = $1)
        ORDER BY created_at DESC`
	if err := r.db.Select(&lists, query, status); err != nil {
		return nil, fmt.Errorf("impossible de récupérer les listes: %w", err)
	}
	return lists, nil
}

// GetByID récupère l'en-tête d'une liste : c'est lui qui porte la ville, donc la station
// locale de destination au moment de l'expédition.
func (r *SendListRepository) GetByID(id int64) (*models.SendList, error) {
	var list models.SendList
	query := `
        SELECT id, session_code, city, item_count, status, created_by, created_at, updated_at
        FROM send_lists
        WHERE id = $1`
	if err := r.db.Get(&list, query, id); err != nil {
		return nil, fmt.Errorf("liste introuvable: %w", err)
	}
	return &list, nil
}

func (r *SendListRepository) ListItems(listID int64, query string) ([]models.SendListItem, error) {
	items := []models.SendListItem{}
	baseQuery := `
        SELECT id, list_id, glass_id, barcode, reference, brand, location_code, created_at
        FROM send_list_items
        WHERE list_id = $1`
	args := []interface{}{listID}
	if query != "" {
		baseQuery += ` AND (LOWER(barcode) LIKE LOWER($2) OR LOWER(reference) LIKE LOWER($2))`
		args = append(args, "%"+query+"%")
	}
	baseQuery += ` ORDER BY id`
	if err := r.db.Select(&items, baseQuery, args...); err != nil {
		return nil, fmt.Errorf("impossible de récupérer le contenu de la liste: %w", err)
	}
	return items, nil
}

// MarkProcessed clôt une liste dont toutes les montures ont été vérifiées et le colis
// préparé. Accepte NOUVELLE comme VUE : une liste peut être contrôlée sans que la
// notification ait été acquittée. Rejouer l'appel ne change rien.
func (r *SendListRepository) MarkProcessed(ids []int64) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	result, err := r.db.Exec(`
        UPDATE send_lists
        SET status = 'TRAITEE', updated_at = NOW()
        WHERE id = ANY($1) AND status <> 'TRAITEE'`, pq.Array(ids))
	if err != nil {
		return 0, fmt.Errorf("impossible de clôturer les listes: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("impossible de vérifier la mise à jour: %w", err)
	}
	return rows, nil
}

// MarkSeen fait passer les listes de NOUVELLE à VUE. Le filtre sur NOUVELLE rend l'appel
// idempotent et empêche de faire régresser une liste déjà traitée.
func (r *SendListRepository) MarkSeen(ids []int64) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	result, err := r.db.Exec(`
        UPDATE send_lists
        SET status = 'VUE', updated_at = NOW()
        WHERE id = ANY($1) AND status = 'NOUVELLE'`, pq.Array(ids))
	if err != nil {
		return 0, fmt.Errorf("impossible de marquer les listes comme vues: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("impossible de vérifier la mise à jour: %w", err)
	}
	return rows, nil
}
