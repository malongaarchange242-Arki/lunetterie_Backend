package repositories

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/lunetterie/backend/internal/inventory/models"
)

const savFollowupColumns = `id, proforma_id, called, called_at, no_answer, relance_at,
	observations, message, updated_by, created_at, updated_at`

type SavFollowupRepository struct {
	db *sqlx.DB
}

func NewSavFollowupRepository(db *sqlx.DB) *SavFollowupRepository {
	return &SavFollowupRepository{db: db}
}

// List renvoie tous les suivis. L'écran SAV les indexe ensuite par proforma_id : il
// charge de toute façon la liste des proformas, un filtre par station ici l'obligerait
// à recouper deux périmètres différents.
func (r *SavFollowupRepository) List() ([]models.SavFollowup, error) {
	followups := []models.SavFollowup{}
	query := `SELECT ` + savFollowupColumns + ` FROM sav_followups ORDER BY updated_at DESC`
	if err := r.db.Select(&followups, query); err != nil {
		return nil, fmt.Errorf("impossible de récupérer les suivis SAV: %w", err)
	}
	return followups, nil
}

func (r *SavFollowupRepository) GetByProforma(proformaID int64) (*models.SavFollowup, error) {
	var followup models.SavFollowup
	query := `SELECT ` + savFollowupColumns + ` FROM sav_followups WHERE proforma_id = $1`
	if err := r.db.Get(&followup, query, proformaID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("impossible de lire le suivi SAV: %w", err)
	}
	return &followup, nil
}

// Save applique les champs fournis sur le suivi existant, ou en crée un.
//
// Lecture puis écriture plutôt qu'un COALESCE dans l'ON CONFLICT : la requête doit
// pouvoir *effacer* une observation (chaîne vide) aussi bien que la laisser intacte
// (champ absent), et COALESCE ne sait pas distinguer les deux.
func (r *SavFollowupRepository) Save(proformaID int64, req models.SavFollowupSaveRequest, userID int64) (*models.SavFollowup, error) {
	if proformaID <= 0 {
		return nil, fmt.Errorf("proforma_id invalide")
	}

	var exists bool
	if err := r.db.Get(&exists, `SELECT EXISTS (SELECT 1 FROM proformas WHERE id = $1)`, proformaID); err != nil {
		return nil, fmt.Errorf("impossible de vérifier la proforma: %w", err)
	}
	if !exists {
		return nil, fmt.Errorf("proforma introuvable")
	}

	current, err := r.GetByProforma(proformaID)
	if err != nil {
		return nil, err
	}
	if current == nil {
		current = &models.SavFollowup{ProformaID: proformaID}
	}

	now := time.Now().UTC()

	if req.Called != nil {
		current.Called = *req.Called
		// La date d'appel suit la case : décocher un appel enregistré par erreur doit
		// aussi effacer son horodatage, sinon la fiche garde une trace fausse.
		if *req.Called {
			current.CalledAt = &now
		} else {
			current.CalledAt = nil
		}
	}
	if req.NoAnswer != nil {
		current.NoAnswer = *req.NoAnswer
	}
	if req.Observations != nil {
		current.Observations = emptyToNil(*req.Observations)
	}
	if req.Message != nil {
		current.Message = emptyToNil(*req.Message)
	}
	if req.RelanceAt != nil {
		raw := strings.TrimSpace(*req.RelanceAt)
		if raw == "" {
			current.RelanceAt = nil
		} else {
			parsed, parseErr := time.Parse("2006-01-02", raw)
			if parseErr != nil {
				return nil, fmt.Errorf("date de relance invalide, attendu AAAA-MM-JJ")
			}
			current.RelanceAt = &parsed
		}
	}

	current.UpdatedBy = &userID
	current.UpdatedAt = now

	query := `
		INSERT INTO sav_followups
			(proforma_id, called, called_at, no_answer, relance_at, observations, message, updated_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $9)
		ON CONFLICT (proforma_id) DO UPDATE SET
			called = EXCLUDED.called,
			called_at = EXCLUDED.called_at,
			no_answer = EXCLUDED.no_answer,
			relance_at = EXCLUDED.relance_at,
			observations = EXCLUDED.observations,
			message = EXCLUDED.message,
			updated_by = EXCLUDED.updated_by,
			updated_at = EXCLUDED.updated_at
		RETURNING ` + savFollowupColumns

	var saved models.SavFollowup
	if err := r.db.QueryRowx(query,
		current.ProformaID,
		current.Called,
		current.CalledAt,
		current.NoAnswer,
		current.RelanceAt,
		current.Observations,
		current.Message,
		current.UpdatedBy,
		now,
	).StructScan(&saved); err != nil {
		return nil, fmt.Errorf("impossible d'enregistrer le suivi SAV: %w", err)
	}

	return &saved, nil
}

func emptyToNil(value string) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}
