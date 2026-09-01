package workflows

import (
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"

	"github.com/lunetterie/backend/internal/inventory/dto"
	"github.com/lunetterie/backend/internal/inventory/models"
	"github.com/lunetterie/backend/internal/inventory/repositories"
	"github.com/lunetterie/backend/internal/inventory/services"
)

// derefString rend la chaîne pointée, ou vide si le pointeur est nul — les champs
// facultatifs du formulaire de réception arrivent tous en pointeurs.
func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func readReceptionPhoto(file multipart.File, rawURL *string, label string) ([]byte, error) {
	if file != nil {
		return io.ReadAll(file)
	}
	if rawURL == nil || strings.TrimSpace(*rawURL) == "" {
		return nil, fmt.Errorf("image %s requise", label)
	}
	parsed, err := url.Parse(strings.TrimSpace(*rawURL))
	if err != nil || parsed.Scheme != "https" || !strings.HasSuffix(parsed.Hostname(), ".supabase.co") {
		return nil, fmt.Errorf("URL image %s invalide", label)
	}
	response, err := http.Get(parsed.String())
	if err != nil {
		return nil, fmt.Errorf("image %s inaccessible: %w", label, err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("image %s inaccessible (HTTP %d)", label, response.StatusCode)
	}
	return io.ReadAll(response.Body)
}

// ReceptionWorkflow orchestrate le workflow complet de réception
type ReceptionWorkflow struct {
	allocationService    *services.AllocationService
	movementService      *services.MovementService
	barcodeService       *services.BarcodeService
	analysisService      *services.AnalysisService
	storageService       *services.StorageService
	similarityService    *services.SimilarityService
	glassRepo            *repositories.GlassRepository
	locationRepo         *repositories.LocationRepository
	analysisRepo         *repositories.AnalysisRepository
	shapeCorrectionRepo  *repositories.ShapeCorrectionRepository
	receptionCommandRepo *repositories.ReceptionCommandRepository
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
	receptionCommandRepo *repositories.ReceptionCommandRepository,
) *ReceptionWorkflow {
	return &ReceptionWorkflow{
		allocationService:    allocationSvc,
		movementService:      movementSvc,
		barcodeService:       barcodeSvc,
		analysisService:      analysisSvc,
		storageService:       storageSvc,
		similarityService:    similaritySvc,
		glassRepo:            glassRepo,
		locationRepo:         locationRepo,
		analysisRepo:         analysisRepo,
		shapeCorrectionRepo:  shapeCorrectionRepo,
		receptionCommandRepo: receptionCommandRepo,
	}
}

// Execute exécute le workflow complet de réception. Les caractéristiques (forme, couleur,
// matière, ...) proviennent de l'écran de vérification du scan : l'analyse IA a déjà tourné
// via POST /inventory/analyze et l'utilisateur les a validées/corrigées avant d'arriver ici.
func (w *ReceptionWorkflow) Execute(req dto.ReceptionRequest, montureImage multipart.File, brancheImage multipart.File, arriereImage multipart.File, userID int64) (*dto.ReceptionResponse, error) {
	log.Printf("🚀 Démarrage workflow réception - User: %d, Station: %d", userID, req.StationID)

	montureBytes, err := readReceptionPhoto(montureImage, req.PhotoMontureURL, "monture")
	if err != nil {
		return nil, fmt.Errorf("erreur lecture image monture: %w", err)
	}
	var brancheBytes []byte
	if brancheImage != nil || (req.PhotoBrancheURL != nil && strings.TrimSpace(*req.PhotoBrancheURL) != "") {
		brancheBytes, err = readReceptionPhoto(brancheImage, req.PhotoBrancheURL, "branche")
		if err != nil {
			return nil, fmt.Errorf("erreur lecture image branche: %w", err)
		}
	}
	var arriereBytes []byte
	if arriereImage != nil || (req.PhotoArriereURL != nil && strings.TrimSpace(*req.PhotoArriereURL) != "") {
		arriereBytes, err = readReceptionPhoto(arriereImage, req.PhotoArriereURL, "arrière")
		if err != nil {
			return nil, fmt.Errorf("erreur lecture image arrière: %w", err)
		}
	}
	analysisResult := buildAnalysisResult(req)

	// Étape: Générer un code-barres unique — sauf si le poste en a déjà réservé un.
	//
	// Le scan affiche l'étiquette avant d'enregistrer : sans code réservé, il y montrait un
	// numéro tiré au hasard, sans rapport avec celui qui allait être attribué. Le poste
	// demande donc son numéro d'avance et le renvoie ici.
	//
	// Un code déjà porté par une monture est ignoré au profit d'un neuf : deux étiquettes
	// identiques rendraient deux montures indiscernables au scan, et mieux vaut un aperçu
	// démenti qu'un doublon en rayon. Le poste réaffiche de toute façon le code retenu.
	barcode := strings.ToUpper(strings.TrimSpace(derefString(req.Barcode)))
	if barcode != "" {
		if existing, err := w.glassRepo.GetByBarcode(barcode); err == nil && existing != nil {
			log.Printf("⚠️  Code-barres réservé %s déjà attribué, un nouveau est généré", barcode)
			barcode = ""
		}
	}
	if barcode == "" {
		log.Println("🏷️  Génération code-barres...")
		generated, err := w.barcodeService.GenerateBarcode()
		if err != nil {
			return nil, fmt.Errorf("erreur code-barres: %w", err)
		}
		barcode = generated
	}
	log.Printf("✅ Code-barres retenu: %s", barcode)

	// Étape: Envoyer les photos dans le bucket Supabase Storage
	log.Println("📤 Upload des photos...")
	var photoMontureURL *string
	if url, err := w.storageService.Upload(barcode+"/monture.jpg", montureBytes, "image/jpeg"); err != nil {
		log.Printf("⚠️  Erreur upload photo monture: %v", err)
	} else {
		photoMontureURL = &url
	}
	var photoBrancheURL *string
	if url, err := w.storageService.Upload(barcode+"/branche.jpg", brancheBytes, "image/jpeg"); err != nil {
		log.Printf("⚠️  Erreur upload photo branche: %v", err)
	} else {
		photoBrancheURL = &url
	}
	var photoArriereURL *string
	if len(arriereBytes) > 0 {
		if url, err := w.storageService.Upload(barcode+"/arriere.jpg", arriereBytes, "image/jpeg"); err != nil {
			log.Printf("⚠️  Erreur upload photo arrière: %v", err)
		} else {
			photoArriereURL = &url
		}
	}

	// Étape: Trouver un emplacement libre
	log.Println("📍 Recherche emplacement libre...")
	var location *models.StorageLocation
	var allocErr error

	if req.PreRegistrationBoxID != nil || (req.CartonCode != nil && strings.TrimSpace(*req.CartonCode) != "") {
		location, allocErr = w.allocationService.FindOrCreatePreRegistrationCartonLocation(
			req.StationID,
			req.ReceptionCommandCode,
			req.PreRegistrationBoxID,
			req.CartonCode,
		)
		if allocErr != nil {
			return nil, fmt.Errorf("erreur allocation: %w", allocErr)
		}
	}

	if location == nil && (req.Gender != nil || req.Shape != nil || req.Price != nil) {
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
		// Premier emplacement libre dans l'ordre du code (POS-01, POS-02, ... POS-20 par
		// bac) : plus de découpage par tranche de prix — voir allocation_service.go pour
		// l'historique de ce choix.
		location, allocErr = w.allocationService.FindOrCreateStockLocation(req.StationID)
		if allocErr != nil {
			return nil, fmt.Errorf("erreur allocation: %w", allocErr)
		}
	}
	log.Printf("✅ Emplacement trouvé: %s", location.Code)

	receptionCommandID, err := resolveReceptionCommandID(req, func(code string) (*models.ReceptionCommand, error) {
		if w.receptionCommandRepo == nil {
			return nil, nil
		}
		return w.receptionCommandRepo.GetByCode(code)
	})
	if err != nil {
		log.Printf("⚠️  Erreur résolution session de réception: %v", err)
	}

	// Étape: Créer le glass
	log.Println("💾 Création glass...")
	glass := &models.Glass{
		Barcode:            barcode,
		StationID:          req.StationID,
		LocationID:         &location.ID,
		SupplierID:         req.SupplierID,
		Status:             models.StatusEnStockGeneral,
		Price:              req.Price,
		Reference:          req.Reference,
		PhotoMontureURL:    photoMontureURL,
		PhotoBrancheURL:    photoBrancheURL,
		PhotoArriereURL:    photoArriereURL,
		ReceptionCommandID: receptionCommandID,
		Notes:              req.Notes,
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
		GlassID:            glass.ID,
		Barcode:            barcode,
		ReceptionCommandID: receptionCommandID,
		Status:             string(glass.Status),
		Location:           location.Code,
		LocationCode:       location.Code,
		Price:              glass.Price,
		Analysis:           analysisResult,
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
func resolveReceptionCommandID(req dto.ReceptionRequest, lookup func(string) (*models.ReceptionCommand, error)) (*int64, error) {
	if req.ReceptionCommandID != nil {
		return req.ReceptionCommandID, nil
	}
	if req.ReceptionCommandCode == nil || strings.TrimSpace(*req.ReceptionCommandCode) == "" {
		return nil, nil
	}
	command, err := lookup(strings.TrimSpace(*req.ReceptionCommandCode))
	if err != nil {
		return nil, err
	}
	if command == nil || command.ID == 0 {
		return nil, fmt.Errorf("commande de réception introuvable pour le code %q", strings.TrimSpace(*req.ReceptionCommandCode))
	}
	return &command.ID, nil
}

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
