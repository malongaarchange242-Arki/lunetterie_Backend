package models

import "testing"

func TestPreRegistrationBoxPhotosRoundTrip(t *testing.T) {
	box := PreRegistrationBox{
		Code: "CTN-0001",
		Photos: []PreRegistrationPhoto{{
			ID:  "photo-1",
			Src: "data:image/jpeg;base64,abc123",
		}},
	}

	if len(box.Photos) != 1 {
		t.Fatalf("expected one photo, got %d", len(box.Photos))
	}
	if box.Photos[0].ID != "photo-1" {
		t.Fatalf("expected photo id photo-1, got %q", box.Photos[0].ID)
	}
}
