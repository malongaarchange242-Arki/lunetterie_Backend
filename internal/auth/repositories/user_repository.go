package repositories

import (
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/lunetterie/backend/internal/auth/models"
)

type UserRepository struct {
	db *sqlx.DB
}

func NewUserRepository(db *sqlx.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) FindByFingerprint(fingerprintHash string) (*models.User, error) {
	user := models.User{}
	query := `SELECT u.id, u.first_name, u.last_name, u.email, u.fingerprint_hash, u.gender, u.phone, u.role_id, r.name AS role_name, u.station_id, s.name AS station_name, u.is_active, u.last_login, u.created_at, u.updated_at,
		EXISTS(SELECT 1 FROM webauthn_credentials wc WHERE wc.user_id = u.id) AS webauthn_registered
		FROM users u
		LEFT JOIN roles r ON u.role_id = r.id
		LEFT JOIN stations s ON u.station_id = s.id
		WHERE u.fingerprint_hash = $1 AND u.is_active = true`
	if err := r.db.Get(&user, query, fingerprintHash); err != nil {
		return nil, fmt.Errorf("utilisateur non trouvé: %w", err)
	}
	return &user, nil
}
func (r *UserRepository) CreateFingerprintUser(user *models.User) error {
	query := `INSERT INTO users (first_name, last_name, email, fingerprint_hash, gender, role_id, station_id, is_active, created_at, updated_at)
              VALUES ($1, $2, $3, $4, $5, $6, $7, true, NOW(), NOW()) RETURNING id`
	return r.db.QueryRowx(query, user.FirstName, user.LastName, user.Email, user.FingerprintHash, user.Gender, user.RoleID, user.StationID).Scan(&user.ID)
}

func (r *UserRepository) FindByID(id int64) (*models.User, error) {
	user := models.User{}
	query := `SELECT u.id, u.first_name, u.last_name, u.email, u.fingerprint_hash, u.gender, u.phone, u.role_id, r.name AS role_name, u.station_id, s.name AS station_name, u.is_active, u.last_login, u.created_at, u.updated_at,
		EXISTS(SELECT 1 FROM webauthn_credentials wc WHERE wc.user_id = u.id) AS webauthn_registered
		FROM users u
		LEFT JOIN roles r ON u.role_id = r.id
		LEFT JOIN stations s ON u.station_id = s.id
		WHERE u.id = $1`
	if err := r.db.Get(&user, query, id); err != nil {
		return nil, fmt.Errorf("utilisateur non trouvé: %w", err)
	}
	return &user, nil
}

func (r *UserRepository) FindByEmail(email string) (*models.User, error) {
	user := models.User{}
	query := `SELECT u.id, u.first_name, u.last_name, u.email, u.password_hash, u.fingerprint_hash, u.gender, u.phone, u.role_id, r.name AS role_name, u.station_id, s.name AS station_name, u.is_active, u.last_login, u.created_at, u.updated_at,
		(u.password_hash IS NOT NULL) AS has_password,
		EXISTS(SELECT 1 FROM webauthn_credentials wc WHERE wc.user_id = u.id) AS webauthn_registered
		FROM users u
		LEFT JOIN roles r ON u.role_id = r.id
		LEFT JOIN stations s ON u.station_id = s.id
		WHERE u.email = $1`
	if err := r.db.Get(&user, query, email); err != nil {
		return nil, fmt.Errorf("utilisateur non trouvé: %w", err)
	}
	return &user, nil
}

// ErrAmbiguousName signale que plusieurs comptes portent le nom saisi. Distinct de
// « introuvable » : l'appelant doit pouvoir l'expliquer à l'employé plutôt que de lui
// répondre que son compte n'existe pas.
var ErrAmbiguousName = errors.New("plusieurs employés portent ce nom")

// FindByName retrouve un employé par son nom complet.
//
// Contrairement à l'e-mail, la table users n'impose AUCUNE unicité sur first_name/last_name :
// deux homonymes sont possibles. On refuse alors la connexion au lieu d'en choisir un
// arbitrairement — authentifier le mauvais employé serait bien pire que de bloquer.
//
// Les deux ordres sont acceptés (« Archange MALONGA » comme « MALONGA Archange ») et les
// espaces multiples réduits : personne ne retient dans quel sens son nom a été saisi.
func (r *UserRepository) FindByName(name string) (*models.User, error) {
	users := []models.User{}
	query := `SELECT u.id, u.first_name, u.last_name, u.email, u.password_hash, u.fingerprint_hash, u.gender, u.phone, u.role_id, r.name AS role_name, u.station_id, s.name AS station_name, u.is_active, u.last_login, u.created_at, u.updated_at,
		(u.password_hash IS NOT NULL) AS has_password,
		EXISTS(SELECT 1 FROM webauthn_credentials wc WHERE wc.user_id = u.id) AS webauthn_registered
		FROM users u
		LEFT JOIN roles r ON u.role_id = r.id
		LEFT JOIN stations s ON u.station_id = s.id
		WHERE LOWER(TRIM(REGEXP_REPLACE(u.first_name || ' ' || u.last_name, '\s+', ' ', 'g'))) = LOWER(TRIM($1))
		   OR LOWER(TRIM(REGEXP_REPLACE(u.last_name || ' ' || u.first_name, '\s+', ' ', 'g'))) = LOWER(TRIM($1))`
	if err := r.db.Select(&users, query, name); err != nil {
		return nil, fmt.Errorf("impossible de rechercher l'employé: %w", err)
	}
	if len(users) == 0 {
		return nil, fmt.Errorf("utilisateur non trouvé")
	}
	if len(users) > 1 {
		return nil, ErrAmbiguousName
	}
	return &users[0], nil
}

func (r *UserRepository) Create(user *models.User) error {
	query := `INSERT INTO users (first_name, last_name, email, phone, gender, role_id, station_id, is_active, password_hash, password_hash_deprecated, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, '', NOW(), NOW()) RETURNING id, created_at, updated_at`
	return r.db.QueryRowx(query, user.FirstName, user.LastName, user.Email, user.Phone, user.Gender, user.RoleID, user.StationID, user.IsActive, user.PasswordHash).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)
}

func (r *UserRepository) SetPassword(userID int64, passwordHash string) error {
	query := `UPDATE users SET password_hash = $1, updated_at = NOW() WHERE id = $2`
	_, err := r.db.Exec(query, passwordHash, userID)
	return err
}

func (r *UserRepository) SetActive(userID int64, isActive bool) error {
	query := `UPDATE users SET is_active = $1, updated_at = NOW() WHERE id = $2`
	_, err := r.db.Exec(query, isActive, userID)
	return err
}

func (r *UserRepository) FindAll() ([]models.User, error) {
	users := []models.User{}
	query := `SELECT u.id, u.first_name, u.last_name, u.email, u.fingerprint_hash, u.gender, u.phone, u.role_id, r.name AS role_name, u.station_id, s.name AS station_name, u.is_active, u.last_login, u.created_at, u.updated_at,
		(u.password_hash IS NOT NULL) AS has_password,
		EXISTS(SELECT 1 FROM webauthn_credentials wc WHERE wc.user_id = u.id) AS webauthn_registered
		FROM users u
		LEFT JOIN roles r ON u.role_id = r.id
		LEFT JOIN stations s ON u.station_id = s.id
		ORDER BY u.created_at DESC`
	if err := r.db.Select(&users, query); err != nil {
		return nil, fmt.Errorf("impossible de récupérer les utilisateurs: %w", err)
	}
	return users, nil
}
