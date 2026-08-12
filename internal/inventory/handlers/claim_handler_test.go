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
	// Retiennent ce que le handler a transmis : c'est la seule façon de vérifier qu'un
	// `station_id` de l'URL descend bien jusqu'au dépôt.
	lastStationID int64
	lastStatus    string
}

func (s *stubClaimRepository) Create(claim *models.Claim) error {
	if s.err != nil {
		return s.err
	}
	claim.ID = int64(len(s.claims) + 1)
	s.claims = append(s.claims, *claim)
	return nil
}

func (s *stubClaimRepository) List(stationID int64, status string) ([]models.Claim, error) {
	if s.err != nil {
		return nil, s.err
	}
	s.lastStationID = stationID
	s.lastStatus = status

	claims := []models.Claim{}
	for _, claim := range s.claims {
		if stationID > 0 && claim.StationID != stationID {
			continue
		}
		if status != "" && claim.Status != status {
			continue
		}
		claims = append(claims, claim)
	}
	return claims, nil
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

func TestClaimHandlerListReturnsClaims(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &stubClaimRepository{claims: []models.Claim{
		{ID: 1, StationID: 6, ClientName: "Arki malonga", Motif: "CASSE", Status: models.ClaimStatusOuverte},
	}}
	h := NewClaimHandler(repo)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/claims", nil)

	h.List(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var payload struct {
		Success bool `json:"success"`
		Data    struct {
			Claims []models.Claim `json:"claims"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}

	// La clé `claims` est celle que le front lit en premier (`data.claims`) : la renommer
	// lui ferait afficher une liste vide sans la moindre erreur.
	if len(payload.Data.Claims) != 1 {
		t.Fatalf("expected 1 claim, got %d", len(payload.Data.Claims))
	}
	if payload.Data.Claims[0].ClientName != "Arki malonga" {
		t.Fatalf("expected client name Arki malonga, got %s", payload.Data.Claims[0].ClientName)
	}
}

func TestClaimHandlerListPassesFilters(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &stubClaimRepository{}
	h := NewClaimHandler(repo)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/claims?station_id=6&status=OUVERTE", nil)

	h.List(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if repo.lastStationID != 6 {
		t.Fatalf("expected station_id 6 to reach the repository, got %d", repo.lastStationID)
	}
	if repo.lastStatus != models.ClaimStatusOuverte {
		t.Fatalf("expected status OUVERTE to reach the repository, got %s", repo.lastStatus)
	}
}

func TestClaimHandlerListRejectsInvalidStation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := NewClaimHandler(&stubClaimRepository{})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/claims?station_id=abc", nil)

	h.List(c)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}
