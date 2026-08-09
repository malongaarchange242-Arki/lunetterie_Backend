package repositories

import (
	"database/sql"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/lunetterie/backend/internal/inventory/models"
)

type ProformaRepository struct {
	db *sqlx.DB
}

func NewProformaRepository(db *sqlx.DB) *ProformaRepository {
	return &ProformaRepository{db: db}
}

const proformaColumns = `id, code, station_id, client_name, client_phone, total_amount, status,
        note, created_by, settled_by, settled_at, created_at, updated_at`

const proformaItemColumns = `id, proforma_id, glass_id, barcode, reference, brand, shape, color,
        unit_price, offerte, outcome, settled_at, is_pending, created_at`

const proformaPrescriptionColumns = `proforma_id, societe, foyer, teinte,
        od_sphere, od_cylindre, od_axe, od_addition, od_prix,
        og_sphere, og_cylindre, og_axe, og_addition, og_prix,
        accessoires_label, accessoires_prix, montage_prix, remise_pct, note_libre,
        created_at, updated_at`

// Create persiste l'en-tête et ses lignes dans une transaction : une proforma enregistrée
// sans son contenu bloquerait des montures sans dire lesquelles.
//
// Le code est bâti depuis la séquence de la table, réservée avant l'insertion : compter les
// proformas existantes pour numéroter la suivante donnerait deux fois le même code à deux
// vendeurs qui valident au même instant.
func (r *ProformaRepository) Create(proforma *models.Proforma, items []models.ProformaItem, prescription *models.ProformaPrescription) error {
	if len(items) == 0 {
		return fmt.Errorf("une proforma sans monture n'a pas de sens")
	}

	tx, err := r.db.Beginx()
	if err != nil {
		return fmt.Errorf("impossible d'ouvrir la transaction proforma: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var nextID int64
	if err := tx.QueryRowx(`SELECT nextval(pg_get_serial_sequence('proformas', 'id'))`).Scan(&nextID); err != nil {
		return fmt.Errorf("impossible de réserver un numéro de proforma: %w", err)
	}
	proforma.ID = nextID
	proforma.Code = fmt.Sprintf("PRO-%d-%04d", time.Now().Year(), nextID)

	// Le total ne comptait que les montures : les verres, les accessoires et le montage
	// n'y entraient pas, et la remise non plus. Le montant lu par la vendeuse et celui
	// stocké divergeaient donc dès qu'il y avait un verre au dossier — et c'est le second
	// que voit la comptabilité. Même formule que computeTotals() côté écran.
	var montures float64
	for _, item := range items {
		montures += item.UnitPrice
	}
	brut := montures
	remisePct := 0.0
	if prescription != nil {
		brut += prescription.ODPrix + prescription.OGPrix + prescription.AccessoiresPrix + prescription.MontagePrix
		remisePct = prescription.RemisePct
	}
	proforma.TotalAmount = brut - math.Round(brut*remisePct/100)
	proforma.Status = models.ProformaStatusEnAttente

	header := `
        INSERT INTO proformas (id, code, station_id, client_name, client_phone, total_amount, status, note, created_by)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
        RETURNING created_at, updated_at`
	if err := tx.QueryRowx(header,
		proforma.ID,
		proforma.Code,
		proforma.StationID,
		proforma.ClientName,
		proforma.ClientPhone,
		proforma.TotalAmount,
		proforma.Status,
		proforma.Note,
		proforma.CreatedBy,
	).Scan(&proforma.CreatedAt, &proforma.UpdatedAt); err != nil {
		return fmt.Errorf("impossible d'enregistrer la proforma: %w", err)
	}

	line := `
        INSERT INTO proforma_items (proforma_id, glass_id, barcode, reference, brand, shape, color, unit_price, offerte)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
        RETURNING id, created_at`
	for i := range items {
		items[i].ProformaID = proforma.ID
		items[i].IsPending = true
		if err := tx.QueryRowx(line,
			proforma.ID,
			items[i].GlassID,
			items[i].Barcode,
			items[i].Reference,
			items[i].Brand,
			items[i].Shape,
			items[i].Color,
			items[i].UnitPrice,
			items[i].Offerte,
		).Scan(&items[i].ID, &items[i].CreatedAt); err != nil {
			// L'index unique partiel sur (glass_id) WHERE is_pending remonte ici : la monture
			// est déjà engagée sur une autre proforma en attente. C'est le blocage demandé, on
			// le traduit en message utilisable plutôt qu'en erreur SQL brute.
			label := "cette monture"
			if items[i].Reference != nil && *items[i].Reference != "" {
				label = *items[i].Reference
			} else if items[i].Barcode != nil {
				label = *items[i].Barcode
			}
			return fmt.Errorf("%s est déjà engagée sur une autre proforma en attente: %w", label, err)
		}
	}

	// L'ordonnance entre dans la même transaction que l'en-tête : une proforma sans sa
	// grille laisserait la Caisse deviner ce que le client a commandé.
	if prescription != nil {
		prescription.ProformaID = proforma.ID
		rx := `
        INSERT INTO proforma_prescriptions (proforma_id, societe, foyer, teinte,
            od_sphere, od_cylindre, od_axe, od_addition, od_prix,
            og_sphere, og_cylindre, og_axe, og_addition, og_prix,
            accessoires_label, accessoires_prix, montage_prix, remise_pct, note_libre)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)
        RETURNING created_at, updated_at`
		if err := tx.QueryRowx(rx,
			prescription.ProformaID,
			prescription.Societe, prescription.Foyer, prescription.Teinte,
			prescription.ODSphere, prescription.ODCylindre, prescription.ODAxe, prescription.ODAddition, prescription.ODPrix,
			prescription.OGSphere, prescription.OGCylindre, prescription.OGAxe, prescription.OGAddition, prescription.OGPrix,
			prescription.AccessoiresLabel, prescription.AccessoiresPrix, prescription.MontagePrix, prescription.RemisePct, prescription.NoteLibre,
		).Scan(&prescription.CreatedAt, &prescription.UpdatedAt); err != nil {
			return fmt.Errorf("impossible d'enregistrer l'ordonnance: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("impossible de valider la proforma: %w", err)
	}
	proforma.Items = items
	proforma.Prescription = prescription
	return nil
}

// GetPrescription renvoie l'ordonnance d'une proforma, ou nil si elle n'en a pas — les
// proformas créées avant la 027 n'ont que leur note.
func (r *ProformaRepository) GetPrescription(proformaID int64) (*models.ProformaPrescription, error) {
	var prescription models.ProformaPrescription
	query := `SELECT ` + proformaPrescriptionColumns + ` FROM proforma_prescriptions WHERE proforma_id = $1`
	if err := r.db.Get(&prescription, query, proformaID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("impossible de récupérer l'ordonnance: %w", err)
	}
	return &prescription, nil
}

// List renvoie les proformas, filtrées par statut si celui-ci est fourni. Les plus récentes
// d'abord : la Caisse traite ce qui vient d'arriver.
func (r *ProformaRepository) List(status string) ([]models.Proforma, error) {
	proformas := []models.Proforma{}
	query := `SELECT ` + proformaColumns + ` FROM proformas`
	args := []interface{}{}
	if status != "" {
		query += ` WHERE status = $1`
		args = append(args, status)
	}
	query += ` ORDER BY created_at DESC`
	if err := r.db.Select(&proformas, query, args...); err != nil {
		return nil, fmt.Errorf("impossible de récupérer les proformas: %w", err)
	}
	return proformas, nil
}

func (r *ProformaRepository) GetByID(id int64) (*models.Proforma, error) {
	var proforma models.Proforma
	query := `SELECT ` + proformaColumns + ` FROM proformas WHERE id = $1`
	if err := r.db.Get(&proforma, query, id); err != nil {
		return nil, fmt.Errorf("proforma introuvable: %w", err)
	}
	items, err := r.ListItems(id)
	if err != nil {
		return nil, err
	}
	proforma.Items = items
	return &proforma, nil
}

func (r *ProformaRepository) ListItems(proformaID int64) ([]models.ProformaItem, error) {
	items := []models.ProformaItem{}
	query := `SELECT ` + proformaItemColumns + ` FROM proforma_items WHERE proforma_id = $1 ORDER BY id ASC`
	if err := r.db.Select(&items, query, proformaID); err != nil {
		return nil, fmt.Errorf("impossible de récupérer les lignes de la proforma %d: %w", proformaID, err)
	}
	return items, nil
}

// SettleItem tranche une ligne. La condition sur is_pending rend l'appel idempotent et
// signale, par zéro ligne touchée, qu'un autre caissier vient de trancher celle-ci.
func (r *ProformaRepository) SettleItem(proformaID, itemID int64, outcome string) (int64, error) {
	query := `UPDATE proforma_items
        SET outcome = $1, settled_at = NOW(), is_pending = FALSE
        WHERE id = $2 AND proforma_id = $3 AND is_pending`
	result, err := r.db.Exec(query, outcome, itemID, proformaID)
	if err != nil {
		return 0, fmt.Errorf("impossible d'arbitrer la ligne %d: %w", itemID, err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("impossible de lire le résultat d'arbitrage de la ligne %d: %w", itemID, err)
	}
	return affected, nil
}

// CloseIfComplete ferme l'en-tête quand plus aucune ligne n'attend. Le statut se déduit des
// décisions : réglée dès qu'une monture a été vendue, annulée si le client a tout rendu.
// Renvoie le statut final, ou une chaîne vide si la proforma reste ouverte.
func (r *ProformaRepository) CloseIfComplete(proformaID, userID int64) (string, error) {
	var pending, sold int
	counts := `SELECT
            COUNT(*) FILTER (WHERE is_pending) AS pending,
            COUNT(*) FILTER (WHERE outcome = $2) AS sold
        FROM proforma_items WHERE proforma_id = $1`
	if err := r.db.QueryRowx(counts, proformaID, models.ProformaOutcomeVendue).Scan(&pending, &sold); err != nil {
		return "", fmt.Errorf("impossible de compter les lignes de la proforma %d: %w", proformaID, err)
	}
	if pending > 0 {
		return "", nil
	}

	status := models.ProformaStatusAnnulee
	if sold > 0 {
		status = models.ProformaStatusReglee
	}
	update := `UPDATE proformas
        SET status = $1, settled_by = $2, settled_at = NOW(), updated_at = NOW()
        WHERE id = $3`
	if _, err := r.db.Exec(update, status, nullIfZero(userID), proformaID); err != nil {
		return "", fmt.Errorf("impossible de clôturer la proforma %d: %w", proformaID, err)
	}
	return status, nil
}
