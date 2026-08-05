package services

import (
	"fmt"

	"github.com/boombuler/barcode"
	"github.com/boombuler/barcode/code128"
	"github.com/jmoiron/sqlx"
)

// BarcodeService gère la génération de code-barres
type BarcodeService struct {
	db *sqlx.DB
}

// NewBarcodeService crée une nouvelle instance
func NewBarcodeService(db *sqlx.DB) *BarcodeService {
	return &BarcodeService{db: db}
}

// GenerateBarcode génère un code-barres unique Code128, format LUN-CNG-0001
// (numéro séquentiel atomique via la séquence Postgres "barcode_seq"). Le zéro-padding
// sur 4 chiffres est une largeur minimale, pas une troncature : au-delà de 9999 le
// numéro s'affiche simplement en entier (ex. LUN-CNG-12345), sans collision possible.
func (s *BarcodeService) GenerateBarcode() (string, error) {
	var seq int64
	if err := s.db.Get(&seq, `SELECT nextval('barcode_seq')`); err != nil {
		return "", fmt.Errorf("erreur génération numéro de code-barres: %w", err)
	}
	code := fmt.Sprintf("LUN-CNG-%04d", seq)

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
