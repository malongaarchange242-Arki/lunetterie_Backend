package handlers

import (
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lunetterie/backend/internal/inventory/models"
)

func TestToValidationPayloadKeepsRealItemAttributes(t *testing.T) {
	list := models.SendList{
		ID:          12,
		SessionCode: "STK-2026-0001",
		City:        "Pointe-Noire",
		ItemCount:   1,
		CreatedAt:   time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC),
	}
	brand := "Pacino"
	shape := "Rectangle"
	color := "Noir"
	location := "Valise A-02 · Carton 4"
	item := models.SendListItem{
		ID:           5,
		ListID:       12,
		Reference:    strPtr("PA-5197-BL"),
		Brand:        strPtr(brand),
		Shape:        strPtr(shape),
		Color:        strPtr(color),
		LocationCode: strPtr(location),
		Barcode:      strPtr("BAR-001"),
	}

	payload := toValidationPayload(list, []models.SendListItem{item})
	items, ok := payload["items"].([]gin.H)
	if !ok {
		t.Fatalf("expected items to be a []gin.H, got %#v", payload["items"])
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	first := items[0]
	if got := first["brand"]; got != brand {
		t.Fatalf("expected brand %q, got %#v", brand, got)
	}
	if got := first["shape"]; got != shape {
		t.Fatalf("expected shape %q, got %#v", shape, got)
	}
	if got := first["color"]; got != color {
		t.Fatalf("expected color %q, got %#v", color, got)
	}
	if got := first["location_code"]; got != location {
		t.Fatalf("expected location %q, got %#v", location, got)
	}
	if got := first["emplacement"]; got != location {
		t.Fatalf("expected emplacement %q, got %#v", location, got)
	}
}

func strPtr(s string) *string { return &s }
