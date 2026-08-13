package repositories

import (
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/lunetterie/backend/internal/inventory/models"
)

type SocieteRepository struct {
	db *sqlx.DB
}

func NewSocieteRepository(db *sqlx.DB) *SocieteRepository {
	return &SocieteRepository{db: db}
}

const societeColumns = `id, name, contact, phone, active, created_by, created_at, updated_at`

// List renvoie les sociétés par ordre alphabétique. `includeInactive` n'est vrai que pour
// l'écran de gestion : la vendeuse ne doit pas pouvoir rattacher une proforma à une
// convention terminée.
func (r *SocieteRepository) List(includeInactive bool) ([]models.Societe, error) {
	// Initialisée non-nulle : une table vide doit sortir en `[]` et non en `null`.
	societes := []models.Societe{}

	query := `SELECT ` + societeColumns + ` FROM societes`
	if !includeInactive {
		query += ` WHERE active`
	}
	query += ` ORDER BY name ASC`

	if err := r.db.Select(&societes, query); err != nil {
		return nil, fmt.Errorf("impossible de récupérer les sociétés: %w", err)
	}
	return societes, nil
}

func (r *SocieteRepository) GetByID(id int64) (*models.Societe, error) {
	var societe models.Societe
	query := `SELECT ` + societeColumns + ` FROM societes WHERE id = $1`
	if err := r.db.Get(&societe, query, id); err != nil {
		return nil, fmt.Errorf("société introuvable: %w", err)
	}
	return &societe, nil
}

func (r *SocieteRepository) Create(societe *models.Societe) error {
	if societe == nil {
		return fmt.Errorf("société vide")
	}
	societe.Name = strings.TrimSpace(societe.Name)
	if societe.Name == "" {
		return fmt.Errorf("le nom de la société est requis")
	}

	query := `
        INSERT INTO societes (name, contact, phone, created_by)
        VALUES ($1, $2, $3, $4)
        RETURNING id, active, created_at, updated_at`

	if err := r.db.QueryRowx(query,
		societe.Name,
		societe.Contact,
		societe.Phone,
		societe.CreatedBy,
	).Scan(&societe.ID, &societe.Active, &societe.CreatedAt, &societe.UpdatedAt); err != nil {
		// L'index unique porte sur LOWER(TRIM(name)) : le rejet vient d'un doublon
		// d'orthographe, exactement ce que la table existe pour empêcher. Le dire ainsi
		// évite à la Direction de chercher une panne là où il y a déjà une fiche.
		if isUniqueViolation(err) {
			return fmt.Errorf("une société porte déjà ce nom: %s", societe.Name)
		}
		return fmt.Errorf("impossible d'enregistrer la société: %w", err)
	}

	return nil
}

// Update n'applique que les champs fournis : la Direction corrige un téléphone sans avoir à
// renvoyer le reste de la fiche.
func (r *SocieteRepository) Update(id int64, req models.SocieteUpdateRequest) (*models.Societe, error) {
	assignments := []string{}
	args := []interface{}{}

	if name := strings.TrimSpace(req.Name); name != "" {
		args = append(args, name)
		assignments = append(assignments, fmt.Sprintf("name = $%d", len(args)))
	}
	if contact := strings.TrimSpace(req.Contact); contact != "" {
		args = append(args, contact)
		assignments = append(assignments, fmt.Sprintf("contact = $%d", len(args)))
	}
	if phone := strings.TrimSpace(req.Phone); phone != "" {
		args = append(args, phone)
		assignments = append(assignments, fmt.Sprintf("phone = $%d", len(args)))
	}
	if req.Active != nil {
		args = append(args, *req.Active)
		assignments = append(assignments, fmt.Sprintf("active = $%d", len(args)))
	}

	if len(assignments) == 0 {
		return nil, fmt.Errorf("aucun champ à modifier")
	}

	args = append(args, id)
	query := `UPDATE societes SET ` + strings.Join(assignments, ", ") +
		`, updated_at = NOW() WHERE id = $` + fmt.Sprint(len(args)) +
		` RETURNING ` + societeColumns

	var societe models.Societe
	if err := r.db.QueryRowx(query, args...).StructScan(&societe); err != nil {
		if isUniqueViolation(err) {
			return nil, fmt.Errorf("une société porte déjà ce nom: %s", strings.TrimSpace(req.Name))
		}
		return nil, fmt.Errorf("impossible de modifier la société: %w", err)
	}
	return &societe, nil
}

// isUniqueViolation reconnaît le code SQLSTATE 23505 sans importer le pilote : le message
// d'erreur de pq le porte en toutes lettres, et le dépôt n'a pas d'autre raison de dépendre
// d'un type concret de driver.
func isUniqueViolation(err error) bool {
	return err != nil && strings.Contains(strings.ToLower(err.Error()), "duplicate key value")
}
