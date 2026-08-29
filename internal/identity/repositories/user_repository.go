package repositories

import (
	"fmt"

	"github.com/jmoiron/sqlx"
	authModels "github.com/lunetterie/backend/internal/auth/models"
	authRepos "github.com/lunetterie/backend/internal/auth/repositories"
	identityModels "github.com/lunetterie/backend/internal/identity/models"
)

// UserRepository est l’adaptateur Identity vers le dépôt utilisateur legacy.
type UserRepository struct {
	delegate *authRepos.UserRepository
}

func NewUserRepository(db *sqlx.DB) *UserRepository {
	return &UserRepository{delegate: authRepos.NewUserRepository(db)}
}

func (r *UserRepository) GetByID(id int64) (*identityModels.User, error) {
	if r.delegate == nil {
		return nil, fmt.Errorf("repository Identity non initialisé")
	}
	user, err := r.delegate.FindByID(id)
	if err != nil {
		return nil, err
	}
	return authUserToIdentity(user), nil
}

func (r *UserRepository) GetByEmail(email string) (*identityModels.User, error) {
	if r.delegate == nil {
		return nil, fmt.Errorf("repository Identity non initialisé")
	}
	user, err := r.delegate.FindByEmail(email)
	if err != nil {
		return nil, err
	}
	return authUserToIdentity(user), nil
}

func authUserToIdentity(user *authModels.User) *identityModels.User {
	if user == nil {
		return nil
	}
	return &identityModels.User{
		ID:                     user.ID,
		FirstName:              user.FirstName,
		LastName:               user.LastName,
		Email:                  user.Email,
		PasswordHashDeprecated: user.PasswordHashDeprecated,
		PasswordHash:           user.PasswordHash,
		HasPassword:            user.HasPassword,
		FingerprintHash:        user.FingerprintHash,
		WebAuthnRegistered:     user.WebAuthnRegistered,
		Gender:                 user.Gender,
		Phone:                  user.Phone,
		City:                   user.City,
		RoleID:                 user.RoleID,
		RoleName:               user.RoleName,
		StationID:              user.StationID,
		StationName:            user.StationName,
		IsActive:               user.IsActive,
		LastLogin:              user.LastLogin,
		CreatedAt:              user.CreatedAt,
		UpdatedAt:              user.UpdatedAt,
	}
}
