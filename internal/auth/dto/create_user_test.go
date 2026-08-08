package dto

import (
	"encoding/json"
	"testing"
)

func TestCreateUserRequestBindsCity(t *testing.T) {
	payload := []byte(`{"first_name":"Jean","last_name":"Dupont","city":"Pointe-Noire","role_id":1}`)

	var req CreateUserRequest
	if err := json.Unmarshal(payload, &req); err != nil {
		t.Fatalf("unexpected unmarshal error: %v", err)
	}

	if req.City != "Pointe-Noire" {
		t.Fatalf("expected city to be bound, got %q", req.City)
	}
}
