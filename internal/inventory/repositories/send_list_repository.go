package repositories

import (
	"fmt"
	"strings"
	"time"

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

// CreateDispatchBox persiste un carton associé à l'envoi complet d'une liste de réception.
func (r *SendListRepository) CreateDispatchBox(list *models.SendList, items []models.SendListItem) (*models.SendBox, error) {
	if list == nil {
		return nil, fmt.Errorf("liste invalide pour carton")
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("liste vide pour carton")
	}

	code := buildSendBoxCode(list.SessionCode)
	reference := buildSendBoxReference(list.SessionCode)
	box := &models.SendBox{
		ListID:      list.ID,
		Code:        code,
		Reference:   reference,
		City:        list.City,
		SessionCode: list.SessionCode,
		ItemCount:   len(items),
		Status:      "CREATED",
	}

	tx, err := r.db.Beginx()
	if err != nil {
		return nil, fmt.Errorf("impossible d'ouvrir la transaction carton: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	header := `
        INSERT INTO send_boxes (list_id, code, reference, city, session_code, item_count, status, created_by)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
        RETURNING id, created_at, updated_at`
	if err := tx.QueryRowx(header,
		list.ID,
		box.Code,
		box.Reference,
		list.City,
		list.SessionCode,
		len(items),
		box.Status,
		list.CreatedBy,
	).Scan(&box.ID, &box.CreatedAt, &box.UpdatedAt); err != nil {
		return nil, fmt.Errorf("impossible d'enregistrer le carton: %w", err)
	}

	line := `
        INSERT INTO send_box_items (box_id, list_item_id, glass_id, barcode, reference, location_code)
        VALUES ($1, $2, $3, $4, $5, $6)`
	for _, item := range items {
		if _, err := tx.Exec(line,
			box.ID,
			item.ID,
			item.GlassID,
			nullIfEmptyFromString(item.Barcode),
			nullIfEmptyFromString(item.Reference),
			nullIfEmptyFromString(item.LocationCode),
		); err != nil {
			return nil, fmt.Errorf("impossible d'enregistrer la ligne carton: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("impossible de valider le carton: %w", err)
	}
	return box, nil
}

const sendBoxColumns = `id, list_id, code, reference, city, session_code, item_count, status,
        created_by, opened_at, opened_by, opened_station_id, created_at, updated_at`

// FindPendingBoxesByCity liste les cartons partis vers une ville et pas encore ouverts. Le
// poste de magasin s'en sert pour savoir s'il doit réclamer un code-barres avant de travailler.
// Le plus ancien d'abord : un colis en attente depuis trois jours passe avant celui d'hier.
func (r *SendListRepository) FindPendingBoxesByCity(city string) ([]models.SendBox, error) {
	boxes := []models.SendBox{}
	query := `SELECT ` + sendBoxColumns + `
        FROM send_boxes
        WHERE status = $1
          AND LOWER(TRIM(city)) = LOWER(TRIM($2))
        ORDER BY created_at ASC`
	if err := r.db.Select(&boxes, query, models.SendBoxStatusCreated, city); err != nil {
		return nil, fmt.Errorf("impossible de récupérer les cartons en attente pour %q: %w", city, err)
	}
	return boxes, nil
}

// FindBoxByCode retrouve un carton par le code imprimé sur son étiquette. La comparaison
// ignore la casse et les espaces : une douchette peut ajouter un blanc en fin de trame.
func (r *SendListRepository) FindBoxByCode(code string) (*models.SendBox, error) {
	var box models.SendBox
	query := `SELECT ` + sendBoxColumns + `
        FROM send_boxes
        WHERE UPPER(TRIM(code)) = UPPER(TRIM($1))
           OR UPPER(TRIM(reference)) = UPPER(TRIM($1))`
	if err := r.db.Get(&box, query, code); err != nil {
		return nil, fmt.Errorf("carton introuvable pour le code %q: %w", code, err)
	}
	return &box, nil
}

// FindBoxItems renvoie le contenu annoncé du carton, tel que figé au départ.
func (r *SendListRepository) FindBoxItems(boxID int64) ([]models.SendBoxItem, error) {
	items := []models.SendBoxItem{}
	query := `SELECT id, box_id, list_item_id, glass_id, barcode, reference, location_code, created_at
        FROM send_box_items
        WHERE box_id = $1
        ORDER BY id ASC`
	if err := r.db.Select(&items, query, boxID); err != nil {
		return nil, fmt.Errorf("impossible de récupérer le contenu du carton %d: %w", boxID, err)
	}
	return items, nil
}

// MarkBoxOpened clôt l'attente d'un carton. La condition sur le statut rend l'appel idempotent
// et évite qu'un second scan écrase l'identité du premier réceptionnaire : zéro ligne touchée
// signale à l'appelant que le carton était déjà ouvert.
func (r *SendListRepository) MarkBoxOpened(boxID, userID, stationID int64) (int64, error) {
	query := `UPDATE send_boxes
        SET status = $1, opened_at = NOW(), opened_by = $2, opened_station_id = $3, updated_at = NOW()
        WHERE id = $4 AND status <> $1`
	result, err := r.db.Exec(query, models.SendBoxStatusOpened, nullIfZero(userID), nullIfZero(stationID), boxID)
	if err != nil {
		return 0, fmt.Errorf("impossible d'ouvrir le carton %d: %w", boxID, err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("impossible de lire le résultat d'ouverture du carton %d: %w", boxID, err)
	}
	return affected, nil
}

func nullIfZero(value int64) interface{} {
	if value == 0 {
		return nil
	}
	return value
}

func nullIfEmptyFromString(value *string) interface{} {
	if value == nil || strings.TrimSpace(*value) == "" {
		return nil
	}
	return *value
}

func buildSendBoxCode(sessionCode string) string {
	clean := strings.ToUpper(strings.TrimSpace(sessionCode))
	if clean == "" {
		clean = "LISTE"
	}
	return fmt.Sprintf("CB-%s-%d", clean, time.Now().UnixNano()%100000)
}

func buildSendBoxReference(sessionCode string) string {
	clean := strings.ToUpper(strings.TrimSpace(sessionCode))
	if clean == "" {
		clean = "LISTE"
	}
	return fmt.Sprintf("COL-%s-%d", clean, time.Now().UnixNano()%1000000)
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
        SELECT id, session_code, city, item_count, status, sent_count, destination_station_name, created_by, created_at, updated_at
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
        SELECT id, session_code, city, item_count, status, sent_count, destination_station_name, created_by, created_at, updated_at
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
        SET status = 'TRAITEE', sent_count = item_count, updated_at = NOW()
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

func (r *SendListRepository) MarkSentCount(ids []int64, sentCount int64, stationName string) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	result, err := r.db.Exec(`
        UPDATE send_lists
        SET sent_count = $2,
            destination_station_name = $3,
            status = CASE WHEN status <> 'TRAITEE' THEN 'TRAITEE' ELSE status END,
            updated_at = NOW()
        WHERE id = ANY($1)`, pq.Array(ids), sentCount, nullIfEmptyForDB(stationName))
	if err != nil {
		return 0, fmt.Errorf("impossible de mémoriser le résultat d'envoi: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("impossible de vérifier la mise à jour: %w", err)
	}
	return rows, nil
}

func nullIfEmptyForDB(value string) interface{} {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
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
