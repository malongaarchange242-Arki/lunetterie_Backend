package models

import "testing"

func TestPreRegistrationBoxPhotosRoundTrip(t *testing.T) {
	box := PreRegistrationBox{
		Code: "CTN-0001",
		Photos: []PreRegistrationPhoto{{
			Kind: "face",
			URL:  "https://bucket.example/pre-registration/VAL-001/CTN-0001/face.jpg",
		}},
	}

	if len(box.Photos) != 1 {
		t.Fatalf("expected one photo, got %d", len(box.Photos))
	}
	if box.Photos[0].Kind != "face" {
		t.Fatalf("expected photo kind face, got %q", box.Photos[0].Kind)
	}
	if box.Photos[0].URL == "" {
		t.Fatal("expected photo URL to be populated")
	}
}
