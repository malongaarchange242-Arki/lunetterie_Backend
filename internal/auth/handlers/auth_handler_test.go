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

func TestFrontendRoleIndexToRoleIDIncludesPreEnregistrement(t *testing.T) {
	if got := frontendRoleIndexToRoleID(0); got != 3 {
		t.Fatalf("expected stock general role id 3, got %d", got)
	}
	if got := frontendRoleIndexToRoleID(1); got != 11 {
		t.Fatalf("expected pre-enregistrement role id 11, got %d", got)
	}
	if got := frontendRoleIndexToRoleID(2); got != 6 {
		t.Fatalf("expected store manager role id 6, got %d", got)
	}
	if got := frontendRoleIndexToRoleID(3); got != 2 {
		t.Fatalf("expected admin role id 2, got %d", got)
	}
	if got := frontendRoleIndexToRoleID(99); got != 0 {
		t.Fatalf("expected invalid role to resolve to 0, got %d", got)
	}
}
