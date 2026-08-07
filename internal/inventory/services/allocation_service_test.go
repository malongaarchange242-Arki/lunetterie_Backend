package services

import (
	"testing"

	"github.com/lunetterie/backend/internal/inventory/models"
)

func TestSelectLocationByTierUsesDedicatedPool(t *testing.T) {
	locations := []models.StorageLocation{
		{Code: "RAYON-A-ETA-01-BAC-A-POS-06"},
		{Code: "RAYON-A-ETA-01-BAC-A-POS-07"},
	}

	selected := selectLocationByTier(locations, "moyenne")
	if selected == nil {
		t.Fatal("expected a location, got nil")
	}
	if selected.Code != "RAYON-A-ETA-01-BAC-A-POS-06" {
		t.Fatalf("expected the moyenne pool to pick the first dedicated position, got %s", selected.Code)
	}
}

func TestSelectLocationByTierFallsBackToFirstFreePosition(t *testing.T) {
	locations := []models.StorageLocation{
		{Code: "RAYON-A-ETA-01-BAC-A-POS-01"},
		{Code: "RAYON-A-ETA-01-BAC-A-POS-02"},
	}

	selected := selectLocationByTier(locations, "luxe")
	if selected == nil {
		t.Fatal("expected a location, got nil")
	}
	if selected.Code != "RAYON-A-ETA-01-BAC-A-POS-01" {
		t.Fatalf("expected the first available position when the luxe pool is empty, got %s", selected.Code)
	}
}

func TestClassifyPriceTierUsesExplicitGamme(t *testing.T) {
	tier := classifyPriceTier(nil, "luxe")
	if tier != "luxe" {
		t.Fatalf("expected luxe tier from explicit gamme, got %s", tier)
	}
}

func TestClassifyPriceTierUsesPriceWhenNoGammeIsProvided(t *testing.T) {
	price := 85000.0
	tier := classifyPriceTier(&price, "")
	if tier != "moyenne" {
		t.Fatalf("expected moyenne tier from price, got %s", tier)
	}
}
