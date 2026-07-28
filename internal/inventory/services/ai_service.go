package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"

	"github.com/lunetterie/backend/internal/inventory/dto"
)

// AIService appelle le service d'analyse IA (détection + classification) en Python
type AIService struct {
	baseURL    string
	httpClient *http.Client
}

// NewAIService crée une nouvelle instance
func NewAIService(baseURL string) *AIService {
	return &AIService{baseURL: baseURL, httpClient: &http.Client{}}
}

// rawAnalysisResponse reflète le schéma AnalysisResponse du service Python (app/schemas/analysis.py)
type rawAnalysisResponse struct {
	Shape                string   `json:"shape"`
	ShapeConfidence      float64  `json:"shape_confidence"`
	Color                string   `json:"color"`
	ColorConfidence      float64  `json:"color_confidence"`
	Material             string   `json:"material"`
	MaterialConfidence   float64  `json:"material_confidence"`
	MountType            string   `json:"mount_type"`
	MountTypeConfidence  float64  `json:"mount_type_confidence"`
	Gender               *string  `json:"gender"`
	Brand                *string  `json:"brand"`
	Reference             *string `json:"reference"`
	Size                 *string  `json:"size"`
	ModelVersion         string   `json:"model_version"`
	ProcessingTimeMs     float64  `json:"processing_time_ms"`
}

var shapeTranslation = map[string]string{
	"carrée":        "Carré",
	"carree":        "Carré",
	"ovale":         "Ovale",
	"papillon":      "Papillon",
	"pilote":        "Aviateur",
	"aviateur":      "Aviateur",
	"rectangulaire": "Rectangulaire",
	"ronde":         "Rond",
	"rond":          "Rond",
}

var colorTranslation = map[string]string{
	"black":  "Noir",
	"gray":   "Gris",
	"grey":   "Gris",
	"white":  "Blanc",
	"brown":  "Marron",
	"green":  "Vert",
	"blue":   "Bleu",
	"red":    "Rouge",
	"gold":   "Doré",
	"silver": "Argenté",
}

var materialTranslation = map[string]string{
	"plastic":  "Plastique",
	"metal":    "Métal",
	"acetate":  "Acétate",
	"titanium": "Titane",
}

var mountTypeTranslation = map[string]string{
	"pleine":        "Pleine monture",
	"semi-cerclée":  "Semi-cerclée",
	"semi-cerclee":  "Semi-cerclée",
	"percée":        "Percée",
	"percee":        "Percée",
}

func translate(table map[string]string, raw string) string {
	if raw == "" {
		return ""
	}
	if translated, ok := table[strings.ToLower(raw)]; ok {
		return translated
	}
	return raw
}

// Analyze envoie une image au service IA et traduit le résultat pour l'UI (labels français)
func (s *AIService) Analyze(imageBytes []byte, filename string) (*dto.AnalysisResult, error) {
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return nil, fmt.Errorf("erreur création form: %w", err)
	}
	if _, err := io.Copy(part, bytes.NewReader(imageBytes)); err != nil {
		return nil, fmt.Errorf("erreur copie image: %w", err)
	}
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("erreur fermeture writer: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, s.baseURL+"/analyze", body)
	if err != nil {
		return nil, fmt.Errorf("erreur création requête IA: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("service IA injoignable: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("erreur lecture réponse IA: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("erreur service IA (status %d): %s", resp.StatusCode, string(respBody))
	}

	var raw rawAnalysisResponse
	if err := json.Unmarshal(respBody, &raw); err != nil {
		return nil, fmt.Errorf("réponse IA invalide: %w", err)
	}

	result := &dto.AnalysisResult{
		Shape:        translate(shapeTranslation, raw.Shape),
		ShapeConf:    raw.ShapeConfidence * 100,
		Color:        translate(colorTranslation, raw.Color),
		ColorConf:    raw.ColorConfidence * 100,
		Material:     translate(materialTranslation, raw.Material),
		MatConf:      raw.MaterialConfidence * 100,
		MountType:    translate(mountTypeTranslation, raw.MountType),
		MountConf:    raw.MountTypeConfidence * 100,
		ProcessingMs: raw.ProcessingTimeMs,
	}
	if raw.Gender != nil {
		result.Gender = *raw.Gender
	}
	if raw.Brand != nil {
		result.Brand = *raw.Brand
	}
	if raw.Reference != nil {
		result.Reference = *raw.Reference
	}
	if raw.Size != nil {
		result.Size = *raw.Size
	}

	return result, nil
}
