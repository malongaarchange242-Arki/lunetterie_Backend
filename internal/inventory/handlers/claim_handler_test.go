package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/lunetterie/backend/internal/inventory/models"
)

type stubClaimRepository struct {
	claims []models.Claim
	err    error
}

func (s *stubClaimRepository) Create(claim *models.Claim) error {
	if s.err != nil {
		return s.err
	}
	claim.ID = int64(len(s.claims) + 1)
	s.claims = append(s.claims, *claim)
	return nil
}

func TestClaimHandlerCreatePersistsClaim(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &stubClaimRepository{}
	h := NewClaimHandler(repo)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodPost, "/claims", strings.NewReader(`{"station_id":7,"client_name":"Alice","barcode":"LUN-001","motif":"CASSE","detail":"Monture cassée"}`))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req
	c.Set("user_id", "42")

	h.Create(c)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, w.Code)
	}

	var payload struct {
		Success bool `json:"success"`
		Data    struct {
			Claim models.Claim `json:"claim"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}

	if !payload.Success {
		t.Fatalf("expected success response")
	}
	if payload.Data.Claim.ClientName != "Alice" {
		t.Fatalf("expected client name Alice, got %s", payload.Data.Claim.ClientName)
	}
	if payload.Data.Claim.Motif != "CASSE" {
		t.Fatalf("expected motif CASSE, got %s", payload.Data.Claim.Motif)
	}
	if repo.claims[0].CreatedBy == nil || *repo.claims[0].CreatedBy != 42 {
		t.Fatalf("expected created_by to be set from auth context")
	}
}
