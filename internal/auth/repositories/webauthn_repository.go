package repositories

import (
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/lunetterie/backend/internal/auth/models"
)

type WebAuthnRepository struct {
	db *sqlx.DB
}

func NewWebAuthnRepository(db *sqlx.DB) *WebAuthnRepository {
	return &WebAuthnRepository{db: db}
}

func (r *WebAuthnRepository) SaveCredential(cred *models.WebAuthnCredential) error {
	query := `
		INSERT INTO webauthn_credentials (user_id, credential_id, public_key, algorithm, aaguid, sign_count, created_at)
		VALUES (:user_id, :credential_id, :public_key, :algorithm, :aaguid, :sign_count, NOW())
		ON CONFLICT (credential_id) DO UPDATE SET
			user_id = :user_id,
			sign_count = :sign_count,
			updated_at = NOW()
		RETURNING id, created_at`

	rows, err := r.db.NamedQuery(query, cred)
	if err != nil {
		return fmt.Errorf("erreur sauvegarde credential: %w", err)
	}
	defer rows.Close()

	if rows.Next() {
		return rows.Scan(&cred.ID, &cred.CreatedAt)
	}
	return nil
}

func (r *WebAuthnRepository) SavePendingCredential(credentialID string, publicKey []byte, algorithm int) error {
	query := `
		INSERT INTO webauthn_credentials (user_id, credential_id, public_key, algorithm, aaguid, sign_count, created_at)
		VALUES (NULL, $1, $2, $3, '', 0, NOW())
		ON CONFLICT (credential_id) DO UPDATE SET
			user_id = NULL,
			public_key = EXCLUDED.public_key,
			algorithm = EXCLUDED.algorithm,
			updated_at = NOW()`
	_, err := r.db.Exec(query, credentialID, publicKey, algorithm)
	if err != nil {
		return fmt.Errorf("erreur sauvegarde credential temporaire: %w", err)
	}
	return nil
}

func (r *WebAuthnRepository) LinkCredentialToUser(credentialID string, userID int64) error {
	query := `UPDATE webauthn_credentials SET user_id = $1, updated_at = NOW() WHERE credential_id = $2 AND user_id IS NULL`
	result, err := r.db.Exec(query, userID, credentialID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return fmt.Errorf("aucune credential temporaire trouvee")
	}
	return nil
}

func (r *WebAuthnRepository) PendingCredentialExists(credentialID string) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM webauthn_credentials WHERE credential_id = $1 AND user_id IS NULL)`
	err := r.db.Get(&exists, query, credentialID)
	return exists, err
}

func (r *WebAuthnRepository) DeleteCredentialsByUser(userID int64) error {
	_, err := r.db.Exec("DELETE FROM webauthn_credentials WHERE user_id = $1", userID)
	return err
}

func (r *WebAuthnRepository) GetCredentialsByUser(userID int64) ([]models.WebAuthnCredential, error) {
	var creds []models.WebAuthnCredential
	query := `SELECT id, user_id, credential_id, public_key, algorithm, aaguid, sign_count, created_at FROM webauthn_credentials WHERE user_id = $1`
	err := r.db.Select(&creds, query, userID)
	return creds, err
}

func (r *WebAuthnRepository) GetCredentialByID(credentialID string) (*models.WebAuthnCredential, error) {
	var cred models.WebAuthnCredential
	query := `SELECT id, user_id, credential_id, public_key, algorithm, aaguid, sign_count, created_at FROM webauthn_credentials WHERE credential_id = $1`
	err := r.db.Get(&cred, query, credentialID)
	return &cred, err
}

func (r *WebAuthnRepository) SaveChallenge(challenge *models.WebAuthnChallenge) error {
	query := `
		INSERT INTO webauthn_challenges (user_id, challenge, expires_at, created_at)
		VALUES (:user_id, :challenge, :expires_at, NOW())
		RETURNING id, created_at`

	rows, err := r.db.NamedQuery(query, challenge)
	if err != nil {
		return fmt.Errorf("erreur sauvegarde challenge: %w", err)
	}
	defer rows.Close()

	if rows.Next() {
		return rows.Scan(&challenge.ID, &challenge.CreatedAt)
	}
	return nil
}

func (r *WebAuthnRepository) SaveRegistrationChallenge(challenge string, expiresAt time.Time) error {
	query := `
		INSERT INTO webauthn_challenges (user_id, challenge, expires_at, created_at)
		VALUES (NULL, $1, $2, NOW())`
	_, err := r.db.Exec(query, challenge, expiresAt)
	if err != nil {
		return fmt.Errorf("erreur sauvegarde challenge temporaire: %w", err)
	}
	return nil
}

func (r *WebAuthnRepository) GetRegistrationChallenge(challenge string) (*models.WebAuthnChallenge, error) {
	var c models.WebAuthnChallenge
	query := `SELECT id, 0 AS user_id, challenge, expires_at, created_at FROM webauthn_challenges WHERE user_id IS NULL AND challenge = $1 AND expires_at > $2`
	err := r.db.Get(&c, query, challenge, time.Now())
	return &c, err
}

func (r *WebAuthnRepository) GetChallenge(userID int64, challenge string) (*models.WebAuthnChallenge, error) {
	var c models.WebAuthnChallenge
	query := `SELECT id, user_id, challenge, expires_at, created_at FROM webauthn_challenges WHERE user_id = $1 AND challenge = $2 AND expires_at > $3`
	err := r.db.Get(&c, query, userID, challenge, time.Now())
	return &c, err
}

func (r *WebAuthnRepository) DeleteChallenge(id int64) error {
	_, err := r.db.Exec("DELETE FROM webauthn_challenges WHERE id = $1", id)
	return err
}

func (r *WebAuthnRepository) CleanExpiredChallenges() error {
	_, err := r.db.Exec("DELETE FROM webauthn_challenges WHERE expires_at < NOW()")
	return err
}
