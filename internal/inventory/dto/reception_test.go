package dto

import "testing"

func TestEffectiveBrandPrefersBrandAndFallsBackToMarque(t *testing.T) {
	brand := "Ray-Ban"
	marque := "Gucci"

	req := ReceptionRequest{
		Brand:  &brand,
		Marque: &marque,
	}

	got := req.EffectiveBrand()
	if got == nil || *got != brand {
		t.Fatalf("expected brand %q, got %#v", brand, got)
	}
}

func TestEffectiveBrandUsesMarqueWhenBrandIsEmpty(t *testing.T) {
	marque := "Prada"
	req := ReceptionRequest{Marque: &marque}

	got := req.EffectiveBrand()
	if got == nil || *got != marque {
		t.Fatalf("expected marque %q, got %#v", marque, got)
	}
}
