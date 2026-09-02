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
// CreateDispatchBox fige le carton et son contenu.
//
// `items` ne doit contenir que les lignes réellement parties, pas toute la liste : c'est ce
// contenu que le magasinier pointera à l'arrivée, et lui faire chercher une monture restée au
// stock général le laisserait constater un manque qui n'en est pas un.
//
// `transferID` relie le carton au transfert qu'il transporte. C'est ce transfert qui porte
// l'état de réception monture par monture — le carton n'en garde aucune copie.
func (r *SendListRepository) CreateDispatchBox(list *models.SendList, items []models.SendListItem, transferID *int64) (*models.SendBox, error) {
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
		Status:      models.SendBoxStatusCreated,
		TransferID:  transferID,
	}

	tx, err := r.db.Beginx()
	if err != nil {
		return nil, fmt.Errorf("impossible d'ouvrir la transaction carton: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	header := `
        INSERT INTO send_boxes (list_id, code, reference, city, session_code, item_count, status, transfer_id, created_by)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
        RETURNING id, created_at, updated_at`
	if err := tx.QueryRowx(header,
		list.ID,
		box.Code,
		box.Reference,
		list.City,
		list.SessionCode,
		len(items),
		box.Status,
		transferID,
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
        transfer_id, created_by, opened_at, opened_by, opened_station_id,
        closed_at, closed_by, missing_count, created_at, updated_at`

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

// ListBoxes liste les cartons, filtrés par statut et/ou ville si fournis. Sert la vue
// Expédition, qui suit tous les colis partis vers tous les magasins — là où
// FindPendingBoxesByCity ne répond qu'à un poste sur sa propre ville.
func (r *SendListRepository) ListBoxes(status, city string) ([]models.SendBox, error) {
	boxes := []models.SendBox{}
	query := `SELECT ` + sendBoxColumns + ` FROM send_boxes WHERE 1 = 1`
	args := []interface{}{}
	if status != "" {
		args = append(args, status)
		query += fmt.Sprintf(" AND status = $%d", len(args))
	}
	if city != "" {
		args = append(args, city)
		query += fmt.Sprintf(" AND LOWER(TRIM(city)) = LOWER(TRIM($%d))", len(args))
	}
	query += ` ORDER BY created_at DESC`
	if err := r.db.Select(&boxes, query, args...); err != nil {
		return nil, fmt.Errorf("impossible de récupérer les cartons: %w", err)
	}
	return boxes, nil
}

// UnavailableBarcodes trie les codes-barres demandés en deux : ceux qu'on peut réellement
// expédier, et ceux qui ne sont plus disponibles, avec leur motif.
//
// Deux causes distinctes, deux messages différents :
//   - la monture n'est plus au stock général (vendue, déjà partie, en laboratoire…)
//   - elle est déjà promise sur une liste non traitée, donc un autre magasin l'attend
//
// Sans ce second contrôle, deux listes pourraient désigner la même monture et le magasinier
// la chercherait en vain pour la seconde.
func (r *SendListRepository) SplitAvailableBarcodes(barcodes []string) (available []string, rejected map[string]string, err error) {
	rejected = map[string]string{}
	if len(barcodes) == 0 {
		return []string{}, rejected, nil
	}

	type row struct {
		Barcode string  `db:"barcode"`
		Status  *string `db:"status"`
		OnList  *string `db:"on_list"`
	}
	rows := []row{}
	query := `
        SELECT
            b.barcode,
            g.status,
            (SELECT sl.session_code
               FROM send_list_items sli
               JOIN send_lists sl ON sl.id = sli.list_id
              WHERE sli.barcode = b.barcode AND sl.status NOT IN ('TRAITEE', 'ANNULEE')
              LIMIT 1) AS on_list
        FROM UNNEST($1::text[]) AS b(barcode)
        LEFT JOIN glasses g ON g.barcode = b.barcode`
	if err := r.db.Select(&rows, query, pq.Array(barcodes)); err != nil {
		return nil, nil, fmt.Errorf("impossible de vérifier la disponibilité des montures: %w", err)
	}

	for _, item := range rows {
		switch {
		case item.Status == nil:
			rejected[item.Barcode] = "monture introuvable"
		case *item.Status != string(models.StatusEnStockGeneral):
			rejected[item.Barcode] = "n'est plus au stock général (" + *item.Status + ")"
		case item.OnList != nil:
			rejected[item.Barcode] = "déjà sur la liste " + *item.OnList
		default:
			available = append(available, item.Barcode)
		}
	}
	if available == nil {
		available = []string{}
	}
	return available, rejected, nil
}

// NextStockListCode numérote les listes composées depuis le stock existant. Le préfixe les
// distingue des arrivages fournisseur : le magasinier voit d'un coup d'œil, dans « Listes
// reçues », s'il prépare une livraison neuve ou un réapprovisionnement de rayon.
func (r *SendListRepository) NextStockListCode() (string, error) {
	var seq int64
	if err := r.db.QueryRowx(`SELECT nextval(pg_get_serial_sequence('send_lists', 'id'))`).Scan(&seq); err != nil {
		return "", fmt.Errorf("impossible de réserver un numéro de liste: %w", err)
	}
	return fmt.Sprintf("STK-%d-%04d", time.Now().Year(), seq), nil
}

// RestockSuggestions confronte, pour chaque ville déjà livrée, la taille de son dernier
// carton au stock qu'il lui reste. Les villes jamais livrées sont absentes du résultat : sans
// carton de référence il n'y a pas de pourcentage à calculer, et les signaler reviendrait à
// alerter en permanence sur un magasin qui n'a jamais rien reçu.
//
// DISTINCT ON retient le carton le plus récent par ville — la comparaison se fait sur la
// ville normalisée, sinon « Pointe-Noire » et « pointe-noire » compteraient pour deux.
func (r *SendListRepository) RestockSuggestions() ([]models.RestockSuggestion, error) {
	suggestions := []models.RestockSuggestion{}
	query := `
        WITH last_box AS (
            SELECT DISTINCT ON (LOWER(TRIM(city)))
                LOWER(TRIM(city)) AS city_key,
                city,
                item_count,
                created_at
            FROM send_boxes
            ORDER BY LOWER(TRIM(city)), created_at DESC
        ),
        local_stock AS (
            SELECT LOWER(TRIM(s.city)) AS city_key, COUNT(*) AS qty
            FROM glasses g
            JOIN stations s ON s.id = g.station_id
            WHERE g.status = $1 AND s.city IS NOT NULL
            GROUP BY LOWER(TRIM(s.city))
        )
        SELECT
            lb.city,
            lb.item_count AS last_box_qty,
            lb.created_at AS last_box_at,
            COALESCE(ls.qty, 0) AS current_stock
        FROM last_box lb
        LEFT JOIN local_stock ls ON ls.city_key = lb.city_key
        ORDER BY lb.city ASC`
	if err := r.db.Select(&suggestions, query, string(models.StatusEnStockSousStation)); err != nil {
		return nil, fmt.Errorf("impossible de calculer les besoins de réapprovisionnement: %w", err)
	}

	// Le calcul reste en Go plutôt qu'en SQL : le seuil et la quantité sont des règles
	// métier, elles doivent se lire à côté de la constante qui les définit.
	for i := range suggestions {
		s := &suggestions[i]
		s.ToSend = s.LastBoxQty - s.CurrentStock
		if s.ToSend < 0 {
			s.ToSend = 0
		}
		s.Alert = s.LastBoxQty > 0 && float64(s.CurrentStock) <= models.RestockAlertRatio*float64(s.LastBoxQty)
	}
	return suggestions, nil
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

// FindBoxItems renvoie le contenu annoncé du carton, tel que figé au départ, augmenté de
// l'emplacement courant de chaque monture.
//
// La jointure est ce qui rend le pointage utilisable : `location_code` fige la case du stock
// général d'où la monture est partie, alors que le magasinier, monture en main, cherche où la
// ranger chez lui. Cet emplacement-là n'est attribué qu'à la réception, et il vit sur la
// monture — pas dans le carton, qui ne connaît que le passé.
//
// Une monture encore en transit n'a aucun emplacement : l'expédition libère sa case de départ
// sans lui en donner d'autre. `stock_location_code` est alors nul, ce qui est exact.
func (r *SendListRepository) FindBoxItems(boxID int64) ([]models.SendBoxItem, error) {
	items := []models.SendBoxItem{}
	// Les attributs de la monture voyagent avec la ligne : le carton n'en fige que la
	// référence, or le magasinier pointe une monture qu'il a en main — il la reconnaît à sa
	// photo et à sa marque bien avant son code-barres.
	query := `SELECT i.id, i.box_id, i.list_item_id, i.glass_id, i.barcode, i.reference,
                 i.location_code, i.created_at,
                 l.code AS stock_location_code,
                 g.photo_monture_url,
                 g.price,
                 g.created_at AS glass_created_at,
                 ga.brand, ga.shape, ga.color, ga.gender
        FROM send_box_items i
        LEFT JOIN glasses g ON g.id = i.glass_id
        LEFT JOIN glass_analysis ga ON ga.id = g.analysis_id
        LEFT JOIN storage_locations l ON l.id = g.location_id
        WHERE i.box_id = $1
        ORDER BY i.id ASC`
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

// MarkBoxClosed clôt le pointage. `missing` fige ce qui manquait à cet instant : les montures
// jamais scannées restent EN_TRANSIT, donc hors du stock du magasin, et leur ligne de transfert
// reste ouverte — un scan plus tard les recevra encore, sans rouvrir le carton.
//
// La condition sur le statut rend l'appel idempotent : zéro ligne touchée signale que le carton
// était déjà clos.
func (r *SendListRepository) MarkBoxClosed(boxID, userID int64, missing int) (int64, error) {
	query := `UPDATE send_boxes
        SET status = $1, closed_at = NOW(), closed_by = $2, missing_count = $3, updated_at = NOW()
        WHERE id = $4 AND status <> $1`
	result, err := r.db.Exec(query, models.SendBoxStatusClosed, nullIfZero(userID), missing, boxID)
	if err != nil {
		return 0, fmt.Errorf("impossible de clôturer le carton %d: %w", boxID, err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("impossible de lire le résultat de clôture du carton %d: %w", boxID, err)
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
		hasItemDetails := r.hasSendListItemDetailColumns()
		line := `
            INSERT INTO send_list_items (list_id, glass_id, barcode, reference, brand, shape, color, location_code)
            VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
		if !hasItemDetails {
			line = `
            INSERT INTO send_list_items (list_id, glass_id, barcode, reference, brand, location_code)
            VALUES ($1, $2, $3, $4, $5, $6)`
		}
		// Réserve la monture dans la même transaction que la ligne qui la désigne : la liste
		// et l'état des montures qu'elle contient ne doivent jamais diverger. Restreint à
		// EN_STOCK_GENERAL par précaution — une monture déjà réservée ou partie ailleurs entre
		// la sélection à l'écran et cet enregistrement ne doit pas être écrasée en silence.
		reserve := `
            UPDATE glasses SET status = 'RESERVEE_ENVOI', updated_at = NOW()
            WHERE id = $1 AND status = 'EN_STOCK_GENERAL'`
		for _, item := range items {
			var err error
			if hasItemDetails {
				_, err = tx.Exec(line, list.ID, item.GlassID,
					nullIfEmpty(item.Barcode), nullIfEmpty(item.Reference),
					nullIfEmpty(item.Brand), nullIfEmpty(item.Shape), nullIfEmpty(item.Color),
					nullIfEmpty(item.LocationCode))
			} else {
				_, err = tx.Exec(line, list.ID, item.GlassID,
					nullIfEmpty(item.Barcode), nullIfEmpty(item.Reference),
					nullIfEmpty(item.Brand), nullIfEmpty(item.LocationCode))
			}
			if err != nil {
				return fmt.Errorf("impossible d'enregistrer une ligne de la liste: %w", err)
			}
			if item.GlassID != nil {
				result, err := tx.Exec(reserve, *item.GlassID)
				if err != nil {
					return fmt.Errorf("impossible de réserver la monture #%d: %w", *item.GlassID, err)
				}
				reserved, err := result.RowsAffected()
				if err != nil {
					return fmt.Errorf("impossible de vérifier la réservation de la monture #%d: %w", *item.GlassID, err)
				}
				if reserved != 1 {
					return fmt.Errorf("la monture #%d n'est plus disponible au stock général", *item.GlassID)
				}
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

func (r *SendListRepository) hasSendListItemDetailColumns() bool {
	var count int
	err := r.db.Get(&count, `
        SELECT COUNT(*)
        FROM information_schema.columns
        WHERE table_schema = current_schema()
          AND table_name = 'send_list_items'
          AND column_name IN ('shape', 'color')`)
	return err != nil || count == 2
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
	SELECT sli.id, sli.list_id, sli.glass_id, sli.barcode, sli.reference, sli.brand, sli.shape, sli.color, sli.location_code,
	       g.photo_monture_url,
	       source.valise_code,
	       source.carton_code,
	       sli.created_at
	FROM send_list_items sli
	LEFT JOIN glasses g ON g.id = sli.glass_id
	LEFT JOIN LATERAL (
	       SELECT pc.code AS valise_code, pb.code AS carton_code
	       FROM pre_registration_cases pc
	       LEFT JOIN pre_registration_boxes pb ON pb.case_id = pc.id
	       WHERE pc.reception_command_id = g.reception_command_id
	         AND (pc.code = sli.location_code OR pb.code = sli.location_code OR sli.location_code IS NULL)
	       ORDER BY (pc.code = sli.location_code) DESC, (pb.code = sli.location_code) DESC, pb.id
	       LIMIT 1
	) source ON TRUE
		WHERE sli.list_id = $1`
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

// Cancel annule une liste qui n'a pas encore été dispatchée et libère les montures qu'elle
// avait réservées au stock général. La liste reste en base pour l'historique Direction.
func (r *SendListRepository) Cancel(id int64) (*models.SendList, error) {
	tx, err := r.db.Beginx()
	if err != nil {
		return nil, fmt.Errorf("impossible d'ouvrir la transaction d'annulation: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var list models.SendList
	query := `
        SELECT id, session_code, city, item_count, status, sent_count, destination_station_name, created_by, created_at, updated_at
        FROM send_lists
        WHERE id = $1
        FOR UPDATE`
	if err := tx.Get(&list, query, id); err != nil {
		return nil, fmt.Errorf("liste introuvable: %w", err)
	}

	switch list.Status {
	case models.SendListStatusAnnulee:
		return &list, nil
	case models.SendListStatusTraitee:
		return nil, fmt.Errorf("cette liste est déjà traitée et ne peut plus être annulée")
	case models.SendListStatusNouvelle, models.SendListStatusVue:
	default:
		return nil, fmt.Errorf("statut de liste non annulable: %s", list.Status)
	}
	if list.SentCount > 0 {
		return nil, fmt.Errorf("cette liste a déjà commencé à être expédiée")
	}

	release := `
        UPDATE glasses g
        SET status = 'EN_STOCK_GENERAL', updated_at = NOW()
        FROM send_list_items sli
        WHERE sli.list_id = $1
          AND sli.glass_id = g.id
          AND g.status = 'RESERVEE_ENVOI'`
	if _, err := tx.Exec(release, id); err != nil {
		return nil, fmt.Errorf("impossible de libérer les montures réservées: %w", err)
	}

	update := `
        UPDATE send_lists
        SET status = $2, updated_at = NOW()
        WHERE id = $1
        RETURNING id, session_code, city, item_count, status, sent_count, destination_station_name, created_by, created_at, updated_at`
	if err := tx.Get(&list, update, id, models.SendListStatusAnnulee); err != nil {
		return nil, fmt.Errorf("impossible d'annuler la liste: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("impossible de valider l'annulation: %w", err)
	}
	return &list, nil
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
        WHERE id = ANY($1) AND status NOT IN ('TRAITEE', 'ANNULEE')`, pq.Array(ids))
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
            status = CASE WHEN status NOT IN ('TRAITEE', 'ANNULEE') THEN 'TRAITEE' ELSE status END,
            updated_at = NOW()
        WHERE id = ANY($1) AND status <> 'ANNULEE'`, pq.Array(ids), sentCount, nullIfEmptyForDB(stationName))
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
