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
	query := `SELECT u.id, u.first_name, u.last_name, u.email, u.fingerprint_hash, u.gender, u.phone, u.role_id, r.name AS role_name, u.station_id, u.is_active, u.last_login, u.created_at, u.updated_at,
		EXISTS(SELECT 1 FROM webauthn_credentials wc WHERE wc.user_id = u.id) AS webauthn_registered
		FROM users u
		LEFT JOIN roles r ON u.role_id = r.id
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
	query := `SELECT u.id, u.first_name, u.last_name, u.email, u.fingerprint_hash, u.gender, u.phone, u.role_id, r.name AS role_name, u.station_id, u.is_active, u.last_login, u.created_at, u.updated_at,
		EXISTS(SELECT 1 FROM webauthn_credentials wc WHERE wc.user_id = u.id) AS webauthn_registered
		FROM users u
		LEFT JOIN roles r ON u.role_id = r.id
		WHERE u.id = $1`
	if err := r.db.Get(&user, query, id); err != nil {
		return nil, fmt.Errorf("utilisateur non trouvé: %w", err)
	}
	return &user, nil
}

func (r *UserRepository) FindByEmail(email string) (*models.User, error) {
	user := models.User{}
	query := `SELECT u.id, u.first_name, u.last_name, u.email, u.fingerprint_hash, u.gender, u.phone, u.role_id, r.name AS role_name, u.station_id, u.is_active, u.last_login, u.created_at, u.updated_at,
		EXISTS(SELECT 1 FROM webauthn_credentials wc WHERE wc.user_id = u.id) AS webauthn_registered
		FROM users u
		LEFT JOIN roles r ON u.role_id = r.id
		WHERE u.email = $1`
	if err := r.db.Get(&user, query, email); err != nil {
		return nil, fmt.Errorf("utilisateur non trouvé: %w", err)
	}
	return &user, nil
}

func (r *UserRepository) Create(user *models.User) error {
	query := `INSERT INTO users (first_name, last_name, email, phone, gender, role_id, station_id, is_active, password_hash_deprecated, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, '', NOW(), NOW()) RETURNING id, created_at, updated_at`
	return r.db.QueryRowx(query, user.FirstName, user.LastName, user.Email, user.Phone, user.Gender, user.RoleID, user.StationID, user.IsActive).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)
}

func (r *UserRepository) FindAll() ([]models.User, error) {
	users := []models.User{}
	query := `SELECT u.id, u.first_name, u.last_name, u.email, u.fingerprint_hash, u.gender, u.phone, u.role_id, r.name AS role_name, u.station_id, u.is_active, u.last_login, u.created_at, u.updated_at,
		EXISTS(SELECT 1 FROM webauthn_credentials wc WHERE wc.user_id = u.id) AS webauthn_registered
		FROM users u
		LEFT JOIN roles r ON u.role_id = r.id
		ORDER BY u.created_at DESC`
	if err := r.db.Select(&users, query); err != nil {
		return nil, fmt.Errorf("impossible de récupérer les utilisateurs: %w", err)
	}
	return users, nil
}
