package services

import (
	"fmt"

	"github.com/lunetterie/backend/internal/inventory/models"
	"github.com/lunetterie/backend/internal/inventory/repositories"
)

// AnalysisService gère les analyses IA
type AnalysisService struct {
	repo *repositories.AnalysisRepository
}

// NewAnalysisService crée une nouvelle instance
func NewAnalysisService(repo *repositories.AnalysisRepository) *AnalysisService {
	return &AnalysisService{repo: repo}
}

// SaveAnalysis sauvegarde une nouvelle analyse
func (s *AnalysisService) SaveAnalysis(analysis *models.GlassAnalysis) error {
	if err := s.repo.Create(analysis); err != nil {
		return fmt.Errorf("erreur sauvegarde analyse: %w", err)
	}
	return nil
}

// GetByGlassID récupère l'analyse d'une monture
func (s *AnalysisService) GetByGlassID(glassID int64) (*models.GlassAnalysis, error) {
	return s.repo.GetByGlassID(glassID)
}
