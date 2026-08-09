package repositories

import (
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/lunetterie/backend/internal/inventory/models"
)

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
