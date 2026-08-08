package handlers

import (
	"strings"
	"testing"
)

func TestNormalizeEmailUsesPlaceholderForBlankEmail(t *testing.T) {
	email := normalizeEmail("", "Jean", "Dupont")
	if email == "" {
		t.Fatal("expected generated email, got empty string")
	}
	if !strings.HasSuffix(email, "@local.invalid") {
		t.Fatalf("expected local invalid email, got %q", email)
	}
	if strings.Contains(email, "-0@local.invalid") {
		// no-op, just ensure the generated value is not empty
	}
}

func TestNormalizeEmailPreservesProvidedEmail(t *testing.T) {
	email := normalizeEmail(" jean.dupont@example.com ", "Jean", "Dupont")
	if email != "jean.dupont@example.com" {
		t.Fatalf("expected trimmed provided email, got %q", email)
	}
}
