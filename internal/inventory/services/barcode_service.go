package services

import (
	"fmt"
	"time"

	"github.com/boombuler/barcode"
	"github.com/boombuler/barcode/code128"
	"github.com/google/uuid"
)

// BarcodeService gère la génération de code-barres
type BarcodeService struct{}

// NewBarcodeService crée une nouvelle instance
func NewBarcodeService() *BarcodeService {
	return &BarcodeService{}
}

// GenerateBarcode génère un code-barres unique Code128
func (s *BarcodeService) GenerateBarcode(prefix string) (string, error) {
	// Format: LB-YYYYMMDD-UUID8 (Lunetterie + date + uuid court)
	date := time.Now().Format("20060102")
	shortUUID := uuid.New().String()[:8]
	code := fmt.Sprintf("%s-%s-%s", prefix, date, shortUUID)

	// Vérifier que le code peut être encodé en Code128
	_, err := code128.Encode(code)
	if err != nil {
		return "", fmt.Errorf("erreur génération code-barres: %w", err)
	}

	return code, nil
}

// GenerateBarcodeImage génère une image PNG du code-barres (optionnel pour plus tard)
func (s *BarcodeService) GenerateBarcodeImage(code string) ([]byte, error) {
	barcodeImg, err := code128.Encode(code)
	if err != nil {
		return nil, fmt.Errorf("erreur encodage code128: %w", err)
	}

	// Redimensionner à une taille standard
	scaledBarcode, err := barcode.Scale(barcodeImg, 300, 100)
	if err != nil {
		return nil, fmt.Errorf("erreur redimensionnement: %w", err)
	}

	// TODO: Convertir en PNG
	_ = scaledBarcode
	return nil, fmt.Errorf("encodage PNG à implémenter")
}
