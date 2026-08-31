package handlers

import (
	"testing"

	"github.com/lunetterie/backend/internal/inventory/models"
)

func TestParseCreateGlassRequest_AllowsStringNumbersAndNulls(t *testing.T) {
	payload := []byte(`{
		"barcode": "CTN-001-001-ABC",
		"reference": "REF-42",
		"price": "90000",
		"photo_monture_url": "",
		"photo_branche_url": "",
		"photo_arriere_url": "https://example.com/back.jpg",
		"reception_command_id": null,
		"notes": "test"
	}`)

	glass, err := parseCreateGlassRequest(payload)
	if err != nil {
		t.Fatalf("parseCreateGlassRequest returned error for valid payload: %v", err)
	}
	if glass.Barcode != "CTN-001-001-ABC" {
		t.Fatalf("barcode mismatch: %q", glass.Barcode)
	}
	if glass.Price == nil || *glass.Price != 90000 {
		t.Fatalf("price should be 90000, got %#v", glass.Price)
	}
	if glass.ReceptionCommandID != nil {
		t.Fatalf("reception_command_id should be nil, got %#v", glass.ReceptionCommandID)
	}
	if glass.Status != models.StatusEnStockGeneral {
		t.Fatalf("default status should be EN_STOCK_GENERAL, got %q", glass.Status)
	}
}

func TestParseCreateGlassRequest_AllowsEmptyBarcodeForBackendGeneration(t *testing.T) {
	payload := []byte(`{
		"barcode": "",
		"reference": "REF-43",
		"price": "120000",
		"notes": "test"
	}`)

	glass, err := parseCreateGlassRequest(payload)
	if err != nil {
		t.Fatalf("parseCreateGlassRequest returned error for empty barcode: %v", err)
	}
	if glass.Barcode != "" {
		t.Fatalf("empty barcode should remain empty for backend generation, got %q", glass.Barcode)
	}
	if glass.Price == nil || *glass.Price != 120000 {
		t.Fatalf("price should be 120000, got %#v", glass.Price)
	}
}
