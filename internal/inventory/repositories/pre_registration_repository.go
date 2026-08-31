package repositories

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/lunetterie/backend/internal/inventory/models"
)

type PreRegistrationRepository struct {
	db *sqlx.DB
}

func NewPreRegistrationRepository(db *sqlx.DB) *PreRegistrationRepository {
	return &PreRegistrationRepository{db: db}
}

func (r *PreRegistrationRepository) commandID(code string) (int64, error) {
	var id int64
	err := r.db.Get(&id, `SELECT id FROM reception_commands WHERE code = $1`, strings.TrimSpace(code))
	if err == sql.ErrNoRows {
		return 0, fmt.Errorf("session de réception introuvable")
	}
	return id, err
}

func (r *PreRegistrationRepository) ListCases(commandCode string) ([]models.PreRegistrationCase, error) {
	commandID, err := r.commandID(commandCode)
	if err != nil {
		return nil, err
	}
	cases := make([]models.PreRegistrationCase, 0)
	err = r.db.Select(&cases, `
		SELECT id, reception_command_id, code, couleur, COALESCE(hex, '') AS hex, gamme, genre,
		       montures, validated, shipment_scanned, shipment_scanned_at, created_at, updated_at
		FROM pre_registration_cases
		WHERE reception_command_id = $1
		ORDER BY created_at ASC`, commandID)
	if err != nil {
		return nil, fmt.Errorf("impossible de lister les valises: %w", err)
	}
	for index := range cases {
		cases[index].Cartons, err = r.listBoxes(cases[index].ID)
		if err != nil {
			return nil, err
		}
	}
	return cases, nil
}

func (r *PreRegistrationRepository) NextCaseCode() (string, error) {
	var sequence int64
	if err := r.db.Get(&sequence, `SELECT nextval('valise_code_seq')`); err != nil {
		return "", err
	}
	return fmt.Sprintf("VAL-%03d", sequence), nil
}

func (r *PreRegistrationRepository) NextBoxCode() (string, error) {
	var sequence int64
	if err := r.db.Get(&sequence, `SELECT nextval('carton_code_seq')`); err != nil {
		return "", err
	}
	return fmt.Sprintf("CTN-%04d", sequence), nil
}
func (r *PreRegistrationRepository) CreateCase(commandCode string, input models.PreRegistrationCase) error {
	commandID, err := r.commandID(commandCode)
	if err != nil {
		return err
	}
	return r.db.QueryRowx(`
		INSERT INTO pre_registration_cases (reception_command_id, code, couleur, hex, gamme, genre, montures)
		VALUES ($1, $2, $3, NULLIF($4, ''), $5, $6, $7)
		RETURNING id, reception_command_id, created_at, updated_at`,
		commandID, input.Code, input.Couleur, input.Hex, input.Gamme, input.Genre, input.Montures,
	).Scan(&input.ID, &input.ReceptionCommandID, &input.CreatedAt, &input.UpdatedAt)
}

func (r *PreRegistrationRepository) CreateBox(caseID int64, input models.PreRegistrationBox) error {
	formes, err := json.Marshal(input.Formes)
	if err != nil {
		return fmt.Errorf("formes invalides: %w", err)
	}
	photosList := input.Photos
	if photosList == nil {
		photosList = []models.PreRegistrationPhoto{}
	}
	photos, err := json.Marshal(photosList)
	if err != nil {
		return fmt.Errorf("photos invalides: %w", err)
	}
	return r.db.QueryRowx(`
		INSERT INTO pre_registration_boxes
			(case_id, code, quantity, formes, marques, couleurs, matieres, photos, gamme, type_lunette, prix)
		VALUES ($1, $2, $3, $4::jsonb, $5, $6, $7, $8::jsonb, $9, $10, $11)
		ON CONFLICT (code) DO UPDATE SET
			case_id = EXCLUDED.case_id,
			quantity = EXCLUDED.quantity,
			formes = EXCLUDED.formes,
			marques = EXCLUDED.marques,
			couleurs = EXCLUDED.couleurs,
			matieres = EXCLUDED.matieres,
			photos = EXCLUDED.photos,
			gamme = EXCLUDED.gamme,
			type_lunette = EXCLUDED.type_lunette,
			prix = EXCLUDED.prix,
			updated_at = NOW()
		RETURNING id, created_at, updated_at`,
		caseID, input.Code, input.Quantity, formes, pq.Array(input.Marques), pq.Array(input.Couleurs),
		pq.Array(input.Matieres), photos, input.Gamme, input.Type, input.Prix,
	).Scan(&input.ID, &input.CreatedAt, &input.UpdatedAt)
}

func (r *PreRegistrationRepository) listBoxes(caseID int64) ([]models.PreRegistrationBox, error) {
	type row struct {
		ID        int64          `db:"id"`
		CaseID    int64          `db:"case_id"`
		Code      string         `db:"code"`
		Quantity  int            `db:"quantity"`
		Formes    []byte         `db:"formes"`
		Marques   pq.StringArray `db:"marques"`
		Couleurs  pq.StringArray `db:"couleurs"`
		Matieres  pq.StringArray `db:"matieres"`
		Photos    []byte         `db:"photos"`
		Gamme     string         `db:"gamme"`
		Type      string         `db:"type_lunette"`
		Prix      float64        `db:"prix"`
		CreatedAt time.Time      `db:"created_at"`
		UpdatedAt time.Time      `db:"updated_at"`
	}
	rows := make([]row, 0)
	if err := r.db.Select(&rows, `
		SELECT id, case_id, code, quantity, formes, marques, couleurs, matieres, photos,
		       gamme, type_lunette, prix, created_at, updated_at
		FROM pre_registration_boxes
		WHERE case_id = $1
		ORDER BY created_at ASC`, caseID); err != nil {
		return nil, fmt.Errorf("impossible de lister les cartons: %w", err)
	}
	boxes := make([]models.PreRegistrationBox, 0, len(rows))
	for _, item := range rows {
		formes := map[string]int{}
		if len(item.Formes) > 0 {
			if err := json.Unmarshal(item.Formes, &formes); err != nil {
				return nil, fmt.Errorf("formes de carton invalides: %w", err)
			}
		}
		photos := make([]models.PreRegistrationPhoto, 0)
		if len(item.Photos) > 0 && string(item.Photos) != "null" {
			if err := json.Unmarshal(item.Photos, &photos); err != nil {
				return nil, fmt.Errorf("photos de carton invalides: %w", err)
			}
		}
		boxes = append(boxes, models.PreRegistrationBox{
			ID: item.ID, CaseID: item.CaseID, Code: item.Code, Quantity: item.Quantity, Formes: formes,
			Marques: []string(item.Marques), Couleurs: []string(item.Couleurs), Matieres: []string(item.Matieres), Photos: photos,
			Gamme: item.Gamme, Type: item.Type, Prix: item.Prix, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt,
		})
	}
	return boxes, nil
}

func (r *PreRegistrationRepository) GetCase(caseID int64) (*models.PreRegistrationCase, error) {
	var item models.PreRegistrationCase
	err := r.db.Get(&item, `
		SELECT id, reception_command_id, code, couleur, COALESCE(hex, '') AS hex, gamme, genre,
		       montures, validated, shipment_scanned, shipment_scanned_at, created_at, updated_at
		FROM pre_registration_cases WHERE id = $1`, caseID)
	if err != nil {
		return nil, err
	}
	item.Cartons, err = r.listBoxes(item.ID)
	return &item, err
}

func (r *PreRegistrationRepository) GetBox(caseID, boxID int64) (*models.PreRegistrationBox, error) {
	item, err := r.GetCase(caseID)
	if err != nil {
		return nil, err
	}
	for index := range item.Cartons {
		if item.Cartons[index].ID == boxID {
			return &item.Cartons[index], nil
		}
	}
	return nil, sql.ErrNoRows
}

func (r *PreRegistrationRepository) DeleteCase(caseID int64) error {
	result, err := r.db.Exec(`
		DELETE FROM pre_registration_cases
		WHERE id = $1 AND validated = false AND shipment_scanned = false`, caseID)
	if err != nil {
		return fmt.Errorf("impossible de supprimer la valise: %w", err)
	}
	count, err := result.RowsAffected()
	if err != nil || count == 0 {
		return fmt.Errorf("valise introuvable ou verrouillée")
	}
	return nil
}

func (r *PreRegistrationRepository) DeleteBox(boxID int64) error {
	result, err := r.db.Exec(`
		DELETE FROM pre_registration_boxes AS b
		WHERE b.id = $1
		  AND EXISTS (
			SELECT 1 FROM pre_registration_cases AS c
			WHERE c.id = b.case_id AND c.validated = false AND c.shipment_scanned = false
		  )`, boxID)
	if err != nil {
		return fmt.Errorf("impossible de supprimer le carton: %w", err)
	}
	count, err := result.RowsAffected()
	if err != nil || count == 0 {
		return fmt.Errorf("carton introuvable ou verrouillé")
	}
	return nil
}

func (r *PreRegistrationRepository) UpdateBoxPhotos(caseID, boxID int64, photos []models.PreRegistrationPhoto) error {
	payload, err := json.Marshal(photos)
	if err != nil {
		return fmt.Errorf("photos invalides: %w", err)
	}
	result, err := r.db.Exec(`
		UPDATE pre_registration_boxes
		SET photos = $3::jsonb,
		    updated_at = NOW()
		WHERE case_id = $1 AND id = $2`, caseID, boxID, payload)
	if err != nil {
		return fmt.Errorf("impossible de sauvegarder les photos du carton: %w", err)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("impossible de vérifier le carton photo: %w", err)
	}
	if count == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func NewPreRegistrationCase(code, couleur, hex, gamme, genre string, montures int) models.PreRegistrationCase {
	return models.PreRegistrationCase{Code: code, Couleur: couleur, Hex: hex, Gamme: gamme, Genre: genre, Montures: montures, CreatedAt: time.Time{}}
}

// DispatchReceptionCommand marque une commande comme expédiée (en transit)
func (r *PreRegistrationRepository) DispatchReceptionCommand(commandCode string) error {
	now := time.Now()
	_, err := r.db.Exec(`
		UPDATE reception_commands 
		SET shipment_status = 'in_transit', dispatched_at = $1, updated_at = $1
		WHERE code = $2`,
		now, strings.TrimSpace(commandCode))
	if err != nil {
		return fmt.Errorf("erreur dispatcher commande: %w", err)
	}
	return nil
}

// ScanReceptionCase marque une valise comme scannée à l'arrivée
func (r *PreRegistrationRepository) ScanReceptionCase(caseID int64) error {
	now := time.Now()
	_, err := r.db.Exec(`
		UPDATE pre_registration_cases
		SET shipment_scanned = true, shipment_scanned_at = $1, updated_at = $1
		WHERE id = $2`,
		now, caseID)
	if err != nil {
		return fmt.Errorf("erreur scanner valise: %w", err)
	}
	return nil
}

// MarkCaseOpened réplique le début du traitement du carton au stock général.
// On l'utilise quand un carton est ouvert manuellement depuis l'écran de réception :
// cela fait passer la commande dans le statut d'arrivée visible dans l'historique.
func (r *PreRegistrationRepository) MarkCaseOpened(caseID int64) error {
	var commandID int64
	if err := r.db.Get(&commandID, `
		SELECT reception_command_id
		FROM pre_registration_cases
		WHERE id = $1`, caseID); err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("valise introuvable")
		}
		return fmt.Errorf("erreur récupération commande associée: %w", err)
	}
	if commandID == 0 {
		return fmt.Errorf("commande associée introuvable")
	}
	now := time.Now()
	_, err := r.db.Exec(`
		UPDATE reception_commands
		SET shipment_status = 'arrived', arrived_at = COALESCE(arrived_at, $1), updated_at = $1
		WHERE id = $2`,
		now, commandID)
	if err != nil {
		return fmt.Errorf("erreur marquer commande ouverte: %w", err)
	}
	return nil
}

// CheckAllCasesScanned vérifie si toutes les valises d'une commande sont scannées
// Retourne (allScanned, commandCode, error)
func (r *PreRegistrationRepository) CheckAllCasesScanned(caseID int64) (bool, string, error) {
	var commandID int64
	var commandCode string

	// Récupérer l'ID de la commande pour cette valise
	err := r.db.Get(&commandID, `
		SELECT reception_command_id FROM pre_registration_cases WHERE id = $1`, caseID)
	if err != nil {
		return false, "", fmt.Errorf("erreur recuperation commande: %w", err)
	}

	// Récupérer le code de la commande
	err = r.db.Get(&commandCode, `
		SELECT code FROM reception_commands WHERE id = $1`, commandID)
	if err != nil {
		return false, "", fmt.Errorf("erreur recuperation code: %w", err)
	}

	// Vérifier si toutes les valises sont scannées
	var totalCases, scannedCases int
	err = r.db.Get(&totalCases, `
		SELECT COUNT(*) FROM pre_registration_cases WHERE reception_command_id = $1`, commandID)
	if err != nil {
		return false, "", fmt.Errorf("erreur compte valises: %w", err)
	}

	err = r.db.Get(&scannedCases, `
		SELECT COUNT(*) FROM pre_registration_cases WHERE reception_command_id = $1 AND shipment_scanned = true`, commandID)
	if err != nil {
		return false, "", fmt.Errorf("erreur compte valises scannees: %w", err)
	}

	return totalCases > 0 && scannedCases == totalCases, commandCode, nil
}

// MarkCommandArrived marque une commande comme arrivée au stock général
func (r *PreRegistrationRepository) MarkCommandArrived(commandCode string) error {
	now := time.Now()
	_, err := r.db.Exec(`
		UPDATE reception_commands 
		SET shipment_status = 'arrived', arrived_at = $1, updated_at = $1
		WHERE code = $2`,
		now, strings.TrimSpace(commandCode))
	if err != nil {
		return fmt.Errorf("erreur marquer arrive: %w", err)
	}
	return nil
}
