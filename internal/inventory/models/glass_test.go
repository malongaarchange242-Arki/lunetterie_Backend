package models

import "testing"

func TestGlassListItem_ValiseCartonDisplayPrefersPreRegistrationRefs(t *testing.T) {
	item := GlassListItem{
		ValiseCode:   StrPtr("VAL-999"),
		LocationCode: StrPtr("LOC-1"),
		CaseCode:     StrPtr("VAL-018"),
		BoxCode:      StrPtr("CTN-001"),
	}

	got := item.ValiseCartonDisplay()
	want := "VAL-018 / CTN-001"
	if got != want {
		t.Fatalf("ValiseCartonDisplay() = %q, want %q", got, want)
	}
}

func StrPtr(s string) *string { return &s }
