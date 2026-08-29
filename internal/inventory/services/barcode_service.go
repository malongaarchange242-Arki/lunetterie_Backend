package services

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"

	"github.com/boombuler/barcode"
	"github.com/boombuler/barcode/code128"
	"github.com/jmoiron/sqlx"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
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

	label := image.NewRGBA(image.Rect(0, 0, 300, 130))
	draw.Draw(label, label.Bounds(), image.NewUniform(color.White), image.Point{}, draw.Src)
	draw.Draw(label, image.Rect(0, 0, scaledBarcode.Bounds().Dx(), scaledBarcode.Bounds().Dy()), scaledBarcode, image.Point{}, draw.Over)
	textWidth := font.MeasureString(basicfont.Face7x13, code).Ceil()
	drawer := font.Drawer{
		Dst:  label,
		Src:  image.NewUniform(color.Black),
		Face: basicfont.Face7x13,
		Dot:  fixed.P((300 - textWidth) / 2, 118),
	}
	drawer.DrawString(code)

	var output bytes.Buffer
	if err := png.Encode(&output, label); err != nil {
		return nil, fmt.Errorf("erreur encodage PNG: %w", err)
	}
	return output.Bytes(), nil
}
