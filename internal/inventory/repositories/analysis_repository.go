package repositories

import (
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/lunetterie/backend/internal/inventory/models"
)

// AnalysisRepository gère les analyses IA
type AnalysisRepository struct {
	db *sqlx.DB
}

// NewAnalysisRepository crée une nouvelle instance
func NewAnalysisRepository(db *sqlx.DB) *AnalysisRepository {
	return &AnalysisRepository{db: db}
}

// Create crée une nouvelle analyse
func (r *AnalysisRepository) Create(analysis *models.GlassAnalysis) error {
	query := `
		INSERT INTO glass_analysis (
			glass_id, model_version, shape, shape_confidence,
			color, color_confidence, material, material_confidence,
			mount_type, mount_type_confidence, gender, gender_confidence,
			brand, brand_confidence, reference, reference_confidence,
			size, crop_image_path, processing_time_ms
		) VALUES (
			:glass_id, :model_version, :shape, :shape_confidence,
			:color, :color_confidence, :material, :material_confidence,
			:mount_type, :mount_type_confidence, :gender, :gender_confidence,
			:brand, :brand_confidence, :reference, :reference_confidence,
			:size, :crop_image_path, :processing_time_ms
		) RETURNING id, created_at`

	rows, err := r.db.NamedQuery(query, analysis)
	if err != nil {
		return fmt.Errorf("erreur création analyse: %w", err)
	}
	defer rows.Close()

	if rows.Next() {
		return rows.Scan(&analysis.ID, &analysis.CreatedAt)
	}

	return fmt.Errorf("aucun ID retourné après création analyse")
}

// GetByGlassID récupère l'analyse la plus récente d'une monture
func (r *AnalysisRepository) GetByGlassID(glassID int64) (*models.GlassAnalysis, error) {
	var analysis models.GlassAnalysis
	query := `
		SELECT * FROM glass_analysis 
		WHERE glass_id = $1 
		ORDER BY created_at DESC 
		LIMIT 1`
	err := r.db.Get(&analysis, query, glassID)
	if err != nil {
		return nil, fmt.Errorf("analyse introuvable: %w", err)
	}
	return &analysis, nil
}
