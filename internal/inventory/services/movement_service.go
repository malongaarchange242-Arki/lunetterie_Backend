package services

import (
	"fmt"

	"github.com/lunetterie/backend/internal/inventory/models"
	"github.com/lunetterie/backend/internal/inventory/repositories"
)

// MovementService gère les mouvements de montures
type MovementService struct {
	repo *repositories.MovementRepository
}

// NewMovementService crée une nouvelle instance
func NewMovementService(repo *repositories.MovementRepository) *MovementService {
	return &MovementService{repo: repo}
}

// CreateMovement crée un nouveau mouvement
func (s *MovementService) CreateMovement(movement *models.Movement) error {
	if err := s.repo.Create(movement); err != nil {
		return fmt.Errorf("erreur création mouvement: %w", err)
	}
	return nil
}
