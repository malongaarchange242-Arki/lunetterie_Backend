package workflows

import (
	"fmt"
	"io"
	"log"
	"mime/multipart"

	"github.com/lunetterie/backend/internal/inventory/dto"
	"github.com/lunetterie/backend/internal/inventory/models"
	"github.com/lunetterie/backend/internal/inventory/repositories"
	"github.com/lunetterie/backend/internal/inventory/services"
)

// ReceptionWorkflow orchestrate le workflow complet de réception
type ReceptionWorkflow struct {
	allocationService   *services.AllocationService
	movementService     *services.MovementService
	barcodeService      *services.BarcodeService
	analysisService     *services.AnalysisService
	storageService      *services.StorageService
	similarityService   *services.SimilarityService
	glassRepo           *repositories.GlassRepository
	locationRepo        *repositories.LocationRepository
	analysisRepo        *repositories.AnalysisRepository
	shapeCorrectionRepo *repositories.ShapeCorrectionRepository
}

// NewReceptionWorkflow crée une nouvelle instance
func NewReceptionWorkflow(
	allocationSvc *services.AllocationService,
	movementSvc *services.MovementService,
	barcodeSvc *services.BarcodeService,
	analysisSvc *services.AnalysisService,
	storageSvc *services.StorageService,
	similaritySvc *services.SimilarityService,
	glassRepo *repositories.GlassRepository,
	locationRepo *repositories.LocationRepository,
	analysisRepo *repositories.AnalysisRepository,
	shapeCorrectionRepo *repositories.ShapeCorrectionRepository,
) *ReceptionWorkflow {
	return &ReceptionWorkflow{
		allocationService:   allocationSvc,
		movementService:     movementSvc,
		barcodeService:      barcodeSvc,
		analysisService:     analysisSvc,
		storageService:      storageSvc,
		similarityService:   similaritySvc,
		glassRepo:           glassRepo,
		locationRepo:        locationRepo,
		analysisRepo:        analysisRepo,
		shapeCorrectionRepo: shapeCorrectionRepo,
	}
}

// Execute exécute le workflow complet de réception. Les caractéristiques (forme, couleur,
// matière, ...) proviennent de l'écran de vérification du scan : l'analyse IA a déjà tourné
// via POST /inventory/analyze et l'utilisateur les a validées/corrigées avant d'arriver ici.
func (w *ReceptionWorkflow) Execute(req dto.ReceptionRequest, montureImage multipart.File, brancheImage multipart.File, userID int64) (*dto.ReceptionResponse, error) {
	log.Printf("🚀 Démarrage workflow réception - User: %d, Station: %d", userID, req.StationID)

	montureBytes, err := io.ReadAll(montureImage)
	if err != nil {
		return nil, fmt.Errorf("erreur lecture image monture: %w", err)
	}
	var brancheBytes []byte
	if brancheImage != nil {
		brancheBytes, err = io.ReadAll(brancheImage)
		if err != nil {
			return nil, fmt.Errorf("erreur lecture image branche: %w", err)
		}
	}

	analysisResult := buildAnalysisResult(req)

	// Étape: Générer un code-barres unique
	log.Println("🏷️  Génération code-barres...")
	barcode, err := w.barcodeService.GenerateBarcode()
	if err != nil {
		return nil, fmt.Errorf("erreur code-barres: %w", err)
	}
	log.Printf("✅ Code-barres généré: %s", barcode)

	// Étape: Envoyer les photos dans le bucket Supabase Storage
	log.Println("📤 Upload des photos...")
	var photoMontureURL, photoBrancheURL *string
	if url, err := w.storageService.Upload(barcode+"/monture.jpg", montureBytes, "image/jpeg"); err != nil {
		log.Printf("⚠️  Erreur upload photo monture: %v", err)
	} else {
		photoMontureURL = &url
	}
	if len(brancheBytes) > 0 {
		if url, err := w.storageService.Upload(barcode+"/branche.jpg", brancheBytes, "image/jpeg"); err != nil {
			log.Printf("⚠️  Erreur upload photo branche: %v", err)
		} else {
			photoBrancheURL = &url
		}
	}

	// Étape: Trouver un emplacement libre
	log.Println("📍 Recherche emplacement libre...")
	var location *models.StorageLocation
	var allocErr error

	if req.Gender != nil || req.Shape != nil || req.Price != nil {
		candidates, err := w.glassRepo.FindByStationAndStatuses(req.StationID, []string{string(models.StatusEnStockGeneral)})
		if err == nil && len(candidates) > 0 {
			reference := &models.GlassListItem{
				Gender: req.Gender,
				Shape:  req.Shape,
				Price:  req.Price,
			}
			best, score, err := w.similarityService.FindBestMatch(reference, candidates)
			if err == nil && best != nil && score > 0 {
				log.Printf("🔎 Meilleure monture similaire trouvée: %s (score %.2f)", best.Barcode, score)
				if best.LocationCode != nil && *best.LocationCode != "" {
					location, allocErr = w.allocationService.FindFreeLocationNearCode(req.StationID, models.ZoneStock, *best.LocationCode)
					if allocErr == nil {
						log.Printf("✅ Emplacement proche trouvé autour de %s", *best.LocationCode)
					}
				}
			}
		}
	}

	if location == nil {
		var gamme string
		if req.Gamme != nil {
			gamme = *req.Gamme
		}
		location, allocErr = w.allocationService.FindFreeLocationForPrice(req.StationID, models.ZoneStock, req.Price, gamme)
		if allocErr != nil {
			return nil, fmt.Errorf("erreur allocation: %w", allocErr)
		}
	}
	log.Printf("✅ Emplacement trouvé: %s", location.Code)

	// Étape: Créer le glass
	log.Println("💾 Création glass...")
	glass := &models.Glass{
		Barcode:         barcode,
		StationID:       req.StationID,
		LocationID:      &location.ID,
		SupplierID:      req.SupplierID,
		Status:          models.StatusEnStockGeneral,
		Price:           req.Price,
		PhotoMontureURL: photoMontureURL,
		PhotoBrancheURL: photoBrancheURL,
		Notes:           req.Notes,
	}

	if err := w.glassRepo.Create(glass); err != nil {
		// Rollback: libérer l'emplacement
		w.allocationService.FreeLocation(location.ID)
		return nil, fmt.Errorf("erreur création glass: %w", err)
	}
	log.Printf("✅ Glass créé: ID=%d", glass.ID)

	// Étape: Journaliser une éventuelle correction manuelle de la forme détectée par l'IA
	if req.DetectedShape != nil && req.Shape != nil && *req.DetectedShape != "" && *req.Shape != "" && *req.DetectedShape != *req.Shape {
		correction := &models.ShapeCorrection{
			GlassID:        glass.ID,
			DetectedShape:  *req.DetectedShape,
			CorrectedShape: *req.Shape,
			UserID:         &userID,
		}
		if err := w.shapeCorrectionRepo.Create(correction); err != nil {
			log.Printf("⚠️  Erreur journalisation correction de forme: %v", err)
		} else {
			log.Printf("✏️  Correction de forme journalisée: %s → %s", *req.DetectedShape, *req.Shape)
		}
	}

	// Étape: Créer le mouvement
	log.Println("📝 Création mouvement...")
	movement := &models.Movement{
		GlassID:      glass.ID,
		ToStationID:  &req.StationID,
		ToLocationID: &location.ID,
		Action:       models.ActionReceptionFournisseur,
		UserID:       userID,
	}
	if err := w.movementService.CreateMovement(movement); err != nil {
		log.Printf("⚠️  Erreur création mouvement: %v", err)
	} else {
		log.Printf("✅ Mouvement créé: ID=%d", movement.ID)
	}

	// Étape: Sauvegarder l'analyse (IA + corrections manuelles)
	log.Println("🧠 Sauvegarde analyse...")
	analysis := &models.GlassAnalysis{
		GlassID:             glass.ID,
		ModelVersion:        "1.0.0",
		Shape:               &analysisResult.Shape,
		ShapeConfidence:     &analysisResult.ShapeConf,
		Color:               &analysisResult.Color,
		ColorConfidence:     &analysisResult.ColorConf,
		Material:            &analysisResult.Material,
		MaterialConfidence:  &analysisResult.MatConf,
		MountType:           &analysisResult.MountType,
		MountTypeConfidence: &analysisResult.MountConf,
		Gender:              &analysisResult.Gender,
		Size:                &analysisResult.Size,
		Brand:               &analysisResult.Brand,
		Reference:           &analysisResult.Reference,
		ProcessingTimeMs:    &analysisResult.ProcessingMs,
	}
	if err := w.analysisService.SaveAnalysis(analysis); err != nil {
		log.Printf("⚠️  Erreur sauvegarde analyse: %v", err)
	} else {
		// Lier l'analyse au glass
		w.glassRepo.UpdateAnalysis(glass.ID, analysis.ID)
		log.Printf("✅ Analyse sauvegardée: ID=%d", analysis.ID)
	}

	// Étape: Construire la réponse
	response := &dto.ReceptionResponse{
		GlassID:      glass.ID,
		Barcode:      barcode,
		Status:       string(glass.Status),
		Location:     location.Code,
		LocationCode: location.Code,
		Price:        glass.Price,
		Analysis:     analysisResult,
		Movement: &dto.MovementInfo{
			ID:        movement.ID,
			Action:    string(movement.Action),
			ToLoc:     location.Code,
			Timestamp: movement.CreatedAt.String(),
		},
	}

	log.Printf("🎉 Workflow réception terminé avec succès! Glass #%d", glass.ID)
	return response, nil
}

// buildAnalysisResult construit le résultat d'analyse final à partir des champs vérifiés/
// corrigés par l'utilisateur à l'écran de vérification (issus au départ de POST /inventory/analyze).
func buildAnalysisResult(req dto.ReceptionRequest) *dto.AnalysisResult {
	result := &dto.AnalysisResult{
		Shape:     "Non déterminé",
		Color:     "Non déterminé",
		Material:  "Non déterminé",
		MountType: "Non déterminé",
	}
	if req.Shape != nil && *req.Shape != "" {
		result.Shape = *req.Shape
	}
	if req.Color != nil && *req.Color != "" {
		result.Color = *req.Color
	}
	if req.Material != nil && *req.Material != "" {
		result.Material = *req.Material
	}
	if req.MountType != nil && *req.MountType != "" {
		result.MountType = *req.MountType
	}
	if req.Gender != nil {
		result.Gender = *req.Gender
	}
	if req.Size != nil {
		result.Size = *req.Size
	}
	if brand := req.EffectiveBrand(); brand != nil {
		result.Brand = *brand
	}
	if req.Reference != nil {
		result.Reference = *req.Reference
	}
	return result
}
