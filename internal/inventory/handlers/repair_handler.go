package handlers

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lunetterie/backend/internal/inventory/dto"
	"github.com/lunetterie/backend/internal/inventory/models"
	"github.com/lunetterie/backend/internal/inventory/repositories"
	"github.com/lunetterie/backend/internal/inventory/services"
	"github.com/lunetterie/backend/internal/shared"
)

type RepairHandler struct {
	glassRepo       *repositories.GlassRepository
	analysisService *services.AnalysisService
	aiService       *services.AIService
	httpClient      *http.Client
}

func NewRepairHandler(glassRepo *repositories.GlassRepository, analysisService *services.AnalysisService, aiService *services.AIService) *RepairHandler {
	return &RepairHandler{
		glassRepo:       glassRepo,
		analysisService: analysisService,
		aiService:       aiService,
		httpClient:      &http.Client{Timeout: 30 * time.Second},
	}
}

type repairResult struct {
	ID      int64  `json:"id"`
	Barcode string `json:"barcode"`
	Status  string `json:"status"`
	Error   string `json:"error,omitempty"`
}

// RepairLunCngAnalysis réanalyse par petits lots les anciennes montures LUN-CNG dont les
// photos existent mais dont les champs métier sont absents.
func (h *RepairHandler) RepairLunCngAnalysis(c *gin.Context) {
	limit := 25
	if raw := strings.TrimSpace(c.Query("limit")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 {
			shared.BadRequest(c, "limit invalide")
			return
		}
		if parsed > 100 {
			parsed = 100
		}
		limit = parsed
	}

	candidates, err := h.glassRepo.FindLunCngAnalysisRepairCandidates(limit)
	if err != nil {
		shared.InternalError(c, err.Error())
		return
	}

	results := make([]repairResult, 0, len(candidates))
	repaired := 0
	for _, candidate := range candidates {
		if err := h.repairCandidate(candidate); err != nil {
			status := "error"
			if h.markPermanentImageError(candidate, err) == nil {
				status = "skipped_permanent_image_error"
			}
			results = append(results, repairResult{
				ID:      candidate.ID,
				Barcode: candidate.Barcode,
				Status:  status,
				Error:   err.Error(),
			})
			continue
		}
		repaired++
		results = append(results, repairResult{ID: candidate.ID, Barcode: candidate.Barcode, Status: "repaired"})
	}

	shared.Success(c, http.StatusOK, gin.H{
		"scanned":  len(candidates),
		"repaired": repaired,
		"results":  results,
	})
}

func (h *RepairHandler) repairCandidate(candidate models.GlassAnalysisRepairCandidate) error {
	montureBytes, montureType, err := h.downloadImage(candidate.PhotoMontureURL)
	if err != nil {
		return fmt.Errorf("photo monture: %w", err)
	}
	brancheBytes, brancheType, err := h.downloadImage(candidate.PhotoBrancheURL)
	if err != nil {
		return fmt.Errorf("photo branche: %w", err)
	}

	face, err := h.aiService.Analyze(montureBytes, imageFilename(candidate.PhotoMontureURL, "monture.jpg"), montureType)
	if err != nil {
		return fmt.Errorf("analyse monture: %w", err)
	}
	branche, err := h.aiService.AnalyzeBranche(brancheBytes, imageFilename(candidate.PhotoBrancheURL, "branche.jpg"), brancheType)
	if err != nil {
		return fmt.Errorf("analyse branche: %w", err)
	}

	analysis := buildRepairedAnalysis(candidate.ID, face, branche)
	if err := h.analysisService.SaveAnalysis(analysis); err != nil {
		return err
	}
	if err := h.glassRepo.UpdateAnalysis(candidate.ID, analysis.ID); err != nil {
		return err
	}
	return nil
}

func (h *RepairHandler) markPermanentImageError(candidate models.GlassAnalysisRepairCandidate, err error) error {
	var statusErr imageDownloadStatusError
	if !errors.As(err, &statusErr) {
		return err
	}
	if statusErr.StatusCode != http.StatusBadRequest && statusErr.StatusCode != http.StatusForbidden && statusErr.StatusCode != http.StatusNotFound {
		return err
	}

	analysis := &models.GlassAnalysis{
		GlassID:      candidate.ID,
		ModelVersion: "repair-img-err-1.0",
	}
	if saveErr := h.analysisService.SaveAnalysis(analysis); saveErr != nil {
		return saveErr
	}
	return h.glassRepo.UpdateAnalysis(candidate.ID, analysis.ID)
}

type imageDownloadStatusError struct {
	StatusCode int
}

func (e imageDownloadStatusError) Error() string {
	return fmt.Sprintf("status %d", e.StatusCode)
}

func (h *RepairHandler) downloadImage(url string) ([]byte, string, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, "", err
	}
	resp, err := h.httpClient.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", imageDownloadStatusError{StatusCode: resp.StatusCode}
	}
	contentType := strings.Split(resp.Header.Get("Content-Type"), ";")[0]
	if contentType != "image/jpeg" && contentType != "image/png" && contentType != "image/webp" {
		contentType = contentTypeFromURL(url)
	}
	if contentType == "" {
		contentType = "image/jpeg"
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}
	if len(body) == 0 {
		return nil, "", fmt.Errorf("image vide")
	}
	return body, contentType, nil
}

func buildRepairedAnalysis(glassID int64, face *dto.AnalysisResult, branche *dto.AnalysisResult) *models.GlassAnalysis {
	result := dto.AnalysisResult{}
	if face != nil {
		result = *face
	}
	if branche != nil {
		if strings.TrimSpace(branche.Reference) != "" {
			result.Reference = branche.Reference
		}
		if strings.TrimSpace(branche.Brand) != "" {
			result.Brand = branche.Brand
		}
	}

	return &models.GlassAnalysis{
		GlassID:             glassID,
		ModelVersion:        "repair-lun-cng-1.0.0",
		Shape:               &result.Shape,
		ShapeConfidence:     &result.ShapeConf,
		Color:               &result.Color,
		ColorConfidence:     &result.ColorConf,
		Material:            &result.Material,
		MaterialConfidence:  &result.MatConf,
		MountType:           &result.MountType,
		MountTypeConfidence: &result.MountConf,
		Gender:              &result.Gender,
		Size:                &result.Size,
		Brand:               &result.Brand,
		Reference:           &result.Reference,
		ProcessingTimeMs:    &result.ProcessingMs,
	}
}

func imageFilename(rawURL string, fallback string) string {
	name := path.Base(rawURL)
	if name == "." || name == "/" || name == "" {
		return fallback
	}
	return name
}

func contentTypeFromURL(rawURL string) string {
	ext := strings.ToLower(path.Ext(rawURL))
	switch ext {
	case ".png":
		return "image/png"
	case ".webp":
		return "image/webp"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	default:
		return ""
	}
}
