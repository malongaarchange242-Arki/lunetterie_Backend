package workflows

import (
	"testing"

	"github.com/lunetterie/backend/internal/inventory/dto"
	"github.com/lunetterie/backend/internal/inventory/models"
)

func TestResolveReceptionCommandIDPrefersExplicitID(t *testing.T) {
	id := int64(42)
	code := "S260808-0001"
	req := dto.ReceptionRequest{
		ReceptionCommandID:   &id,
		ReceptionCommandCode: &code,
	}

	resolved, err := resolveReceptionCommandID(req, func(code string) (*models.ReceptionCommand, error) {
		t.Fatalf("lookup should not run when explicit id is provided")
		return nil, nil
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resolved == nil || *resolved != id {
		t.Fatalf("expected explicit id %d, got %#v", id, resolved)
	}
}

func TestResolveReceptionCommandIDFallsBackToCodeLookup(t *testing.T) {
	code := "S260808-0001"
	lookupID := int64(7)
	req := dto.ReceptionRequest{ReceptionCommandCode: &code}

	resolved, err := resolveReceptionCommandID(req, func(requestedCode string) (*models.ReceptionCommand, error) {
		if requestedCode != code {
			t.Fatalf("expected code %q, got %q", code, requestedCode)
		}
		return &models.ReceptionCommand{ID: lookupID}, nil
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resolved == nil || *resolved != lookupID {
		t.Fatalf("expected resolved id %d, got %#v", lookupID, resolved)
	}
}

func TestGlassSupportsRearPhotoURL(t *testing.T) {
	url := "https://example.com/rear.jpg"
	glass := models.Glass{PhotoArriereURL: &url}
	if glass.PhotoArriereURL == nil || *glass.PhotoArriereURL != url {
		t.Fatalf("expected rear photo URL %q, got %#v", url, glass.PhotoArriereURL)
	}
}
