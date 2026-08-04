package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
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

// rawAnalysisResponse reflète le schéma réellement renvoyé par POST /glasses/analyze
// (app/ai/predictor.py, GlassesPredictor.predict_image) : une confiance globale (pas une par
// attribut), et un sous-objet product_fiche pour marque/référence (généralement vides pour
// l'instant, ce pipeline ne les déduit pas encore de l'image).
type rawAnalysisResponse struct {
	Detected     bool    `json:"detected"`
	Confidence   float64 `json:"confidence"`
	FrameShape   string  `json:"frame_shape"`
	Color        string  `json:"color"`
	Material     string  `json:"material"`
	HasBranches  bool    `json:"has_branches"`
	MountType    string  `json:"mount_type"`
	Gender       *string `json:"gender"`
	ProductFiche struct {
		Brand     *string `json:"brand"`
		Reference *string `json:"reference"`
	} `json:"product_fiche"`
}

// rawBrancheOCRResponse reflète le schéma renvoyé par POST /glasses/analyze-branche
// (app/ai/claude_vision.py, ocr_branche) : lecture du texte gravé sur la branche.
type rawBrancheOCRResponse struct {
	Reference  *string `json:"reference"`
	Brand      *string `json:"brand"`
	Confidence float64 `json:"confidence"`
}

var shapeTranslation = map[string]string{
	"carrée":        "Carré",
	"carree":        "Carré",
	"square":        "Carré",
	"ovale":         "Ovale",
	"oval":          "Ovale",
	"papillon":      "Papillon",
	"pilote":        "Aviateur",
	"aviateur":      "Aviateur",
	"aviator":       "Aviateur",
	"rectangulaire": "Rectangulaire",
	"rectangle":     "Rectangulaire",
	"ronde":         "Rond",
	"rond":          "Rond",
	"round":         "Rond",
	"unknown":       "",
}

var colorTranslation = map[string]string{
	"black":        "Noir",
	"gray":         "Gris",
	"grey":         "Gris",
	"white":        "Blanc",
	"brown":        "Marron",
	"green":        "Vert",
	"blue":         "Bleu",
	"red":          "Rouge",
	"gold":         "Doré",
	"silver":       "Argenté",
	"yellow":       "Jaune",
	"orange":       "Orange",
	"pink":         "Rose",
	"rose":         "Rose",
	"beige":        "Beige",
	"cream":        "Beige",
	"transparent":  "Transparent",
	"tortoise":     "Écaille",
	"écaille":      "Écaille",
	"multicolor":   "Multicolore",
	"multicolore":  "Multicolore",
	"multicolored": "Multicolore",
	"bronze":       "Bronze",
	"cuivre":       "Cuivré",
	"cuivré":       "Cuivré",
	"or":           "Doré",
	"argent":       "Argenté",
	"unknown":      "",
	"inconnu":      "",
}

var materialTranslation = map[string]string{
	"plastic":  "Plastique",
	"metal":    "Métal",
	"acetate":  "Acétate",
	"titanium": "Titane",
	"unknown":  "",
	"inconnu":  "",
}

var mountTypeTranslation = map[string]string{
	"pleine":       "Pleine monture",
	"semi-cerclée": "Semi-cerclée",
	"semi-cerclee": "Semi-cerclée",
	"percée":       "Percée",
	"percee":       "Percée",
	"unknown":      "",
	"inconnu":      "",
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
func (s *AIService) Analyze(imageBytes []byte, filename string, contentType string) (*dto.AnalysisResult, error) {
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)

	// CreateFormFile fixe toujours Content-Type: application/octet-stream, ce que le service
	// Python rejette (il exige un type MIME "image/...") : on construit la part manuellement.
	header := make(textproto.MIMEHeader)
	header.Set("Content-Disposition", fmt.Sprintf(`form-data; name="file"; filename="%s"`, filename))
	header.Set("Content-Type", contentType)
	part, err := writer.CreatePart(header)
	if err != nil {
		return nil, fmt.Errorf("erreur création form: %w", err)
	}
	if _, err := io.Copy(part, bytes.NewReader(imageBytes)); err != nil {
		return nil, fmt.Errorf("erreur copie image: %w", err)
	}
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("erreur fermeture writer: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, s.baseURL+"/glasses/analyze", body)
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

	// Le pipeline ne renvoie qu'une confiance globale (pas une par attribut) : on la réutilise
	// pour les 4 champs. Size n'est pas déduit par ce pipeline : laissé vide, l'utilisateur le
	// renseigne manuellement à l'étape de vérification.
	confidence := raw.Confidence * 100
	result := &dto.AnalysisResult{
		Shape:     translate(shapeTranslation, raw.FrameShape),
		ShapeConf: confidence,
		Color:     translate(colorTranslation, raw.Color),
		ColorConf: confidence,
		Material:  translate(materialTranslation, raw.Material),
		MatConf:   confidence,
		MountType: translate(mountTypeTranslation, raw.MountType),
		MountConf: confidence,
	}
	// Gender vient de Claude vision (app/ai/claude_vision.py) déjà en français ("Homme",
	// "Femme", "Enfant", "Unisexe") : pas de table de traduction nécessaire.
	if raw.Gender != nil {
		result.Gender = *raw.Gender
	}
	if raw.ProductFiche.Brand != nil {
		result.Brand = *raw.ProductFiche.Brand
	}
	if raw.ProductFiche.Reference != nil {
		result.Reference = *raw.ProductFiche.Reference
	}

	return result, nil
}

// rawChatResponse reflète le schéma renvoyé par POST /assistant/chat (app/api/chat.py).
type rawChatResponse struct {
	Reply string `json:"reply"`
}

// Chat relaie tel quel (message + historique + contexte déjà construits par direction.js)
// au chatbot de direction du service IA, qui construit le prompt et appelle Claude
// (app/ai/chat.py). Contrairement à Analyze/AnalyzeBranche, pas de repli silencieux
// possible ici : une erreur remonte telle quelle à l'appelant.
func (s *AIService) Chat(payload []byte) (string, error) {
	req, err := http.NewRequest(http.MethodPost, s.baseURL+"/assistant/chat", bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("erreur création requête chat: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("service IA injoignable: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("erreur lecture réponse chat: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("erreur service IA (status %d): %s", resp.StatusCode, string(respBody))
	}

	var raw rawChatResponse
	if err := json.Unmarshal(respBody, &raw); err != nil {
		return "", fmt.Errorf("réponse chat invalide: %w", err)
	}
	return raw.Reply, nil
}

// AnalyzeBranche envoie la photo de la branche au service IA pour OCR de la référence et de
// la marque (Claude vision) — utilisé à l'étape 2 de scan.html, séparément de la photo de face.
func (s *AIService) AnalyzeBranche(imageBytes []byte, filename string, contentType string) (*dto.AnalysisResult, error) {
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)

	header := make(textproto.MIMEHeader)
	header.Set("Content-Disposition", fmt.Sprintf(`form-data; name="file"; filename="%s"`, filename))
	header.Set("Content-Type", contentType)
	part, err := writer.CreatePart(header)
	if err != nil {
		return nil, fmt.Errorf("erreur création form: %w", err)
	}
	if _, err := io.Copy(part, bytes.NewReader(imageBytes)); err != nil {
		return nil, fmt.Errorf("erreur copie image: %w", err)
	}
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("erreur fermeture writer: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, s.baseURL+"/glasses/analyze-branche", body)
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

	var raw rawBrancheOCRResponse
	if err := json.Unmarshal(respBody, &raw); err != nil {
		return nil, fmt.Errorf("réponse IA invalide: %w", err)
	}

	result := &dto.AnalysisResult{}
	if raw.Reference != nil {
		result.Reference = *raw.Reference
	}
	if raw.Brand != nil {
		result.Brand = *raw.Brand
	}
	return result, nil
}
