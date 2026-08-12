package repositories

import (
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/lunetterie/backend/internal/inventory/models"
)

// Les colonnes sont énumérées plutôt que `*` pour le COALESCE sur `barcode` : la colonne
// est NULL-able (026_claims) alors que le modèle la lit dans un `string`. Sans lui, une
// seule réclamation saisie sans code-barres ferait échouer la lecture de toute la liste.
const claimColumns = `id, station_id, client_name, COALESCE(barcode, '') AS barcode, motif,
        detail, status, created_by, created_at, updated_at`

type ClaimRepository struct {
	db *sqlx.DB
}

func NewClaimRepository(db *sqlx.DB) *ClaimRepository {
	return &ClaimRepository{db: db}
}

func (r *ClaimRepository) Create(claim *models.Claim) error {
	if claim == nil {
		return fmt.Errorf("réclamation vide")
	}
	if claim.Status == "" {
		claim.Status = models.ClaimStatusOuverte
	}
	if claim.CreatedAt.IsZero() {
		claim.CreatedAt = time.Now().UTC()
	}
	claim.UpdatedAt = claim.CreatedAt

	query := `
		INSERT INTO claims (station_id, client_name, barcode, motif, detail, status, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id`

	if err := r.db.QueryRowx(query,
		claim.StationID,
		claim.ClientName,
		claim.Barcode,
		claim.Motif,
		claim.Detail,
		claim.Status,
		claim.CreatedBy,
		claim.CreatedAt,
		claim.UpdatedAt,
	).Scan(&claim.ID); err != nil {
		return fmt.Errorf("impossible d'enregistrer la réclamation: %w", err)
	}

	return nil
}

// List rend les réclamations de la plus récente à la plus ancienne. Les deux filtres sont
// facultatifs : `stationID` à zéro et `status` vide couvrent toute la table. Le tri et les
// filtres suivent les index posés par 026_claims, qui les attendaient déjà.
func (r *ClaimRepository) List(stationID int64, status string) ([]models.Claim, error) {
	// Initialisée non-nulle : une table vide doit sortir en `[]` et non en `null`, sinon
	// le front reçoit autre chose qu'une liste sur son premier appel.
	claims := []models.Claim{}

	query := `SELECT ` + claimColumns + ` FROM claims`
	args := []interface{}{}
	conditions := []string{}
	if stationID > 0 {
		args = append(args, stationID)
		conditions = append(conditions, fmt.Sprintf("station_id = $%d", len(args)))
	}
	if status != "" {
		args = append(args, status)
		conditions = append(conditions, fmt.Sprintf("status = $%d", len(args)))
	}
	if len(conditions) > 0 {
		query += ` WHERE ` + strings.Join(conditions, " AND ")
	}
	query += ` ORDER BY created_at DESC`

	if err := r.db.Select(&claims, query, args...); err != nil {
		return nil, fmt.Errorf("impossible de récupérer les réclamations: %w", err)
	}
	return claims, nil
}
