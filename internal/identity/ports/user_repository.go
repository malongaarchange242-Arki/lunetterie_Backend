package ports

import "github.com/lunetterie/backend/internal/identity/models"

// UserRepository expose les opérations minimales de persistance pour l'identité.
type UserRepository interface {
	GetByID(id int64) (*models.User, error)
	GetByEmail(email string) (*models.User, error)
	Create(user *models.User) error
	Update(user *models.User) error
}
