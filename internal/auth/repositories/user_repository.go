package repositories

import (
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
		u.setup_token, u.setup_token_expires_at,
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

func (r *UserRepository) Create(user *models.User) error {
	query := `INSERT INTO users (first_name, last_name, email, phone, gender, role_id, station_id, is_active, password_hash, password_hash_deprecated, setup_token, setup_token_expires_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, '', $10, $11, NOW(), NOW()) RETURNING id, created_at, updated_at`
	return r.db.QueryRowx(query, user.FirstName, user.LastName, user.Email, user.Phone, user.Gender, user.RoleID, user.StationID, user.IsActive, user.PasswordHash, user.SetupToken, user.SetupTokenExpiresAt).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)
}

// SetPassword enregistre le mot de passe et invalide le jeton de configuration initiale
// (à usage unique : une fois le mot de passe défini, le jeton ne doit plus être réutilisable).
func (r *UserRepository) SetPassword(userID int64, passwordHash string) error {
	query := `UPDATE users SET password_hash = $1, setup_token = NULL, setup_token_expires_at = NULL, updated_at = NOW() WHERE id = $2`
	_, err := r.db.Exec(query, passwordHash, userID)
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
