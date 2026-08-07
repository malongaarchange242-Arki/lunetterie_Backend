package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/lunetterie/backend/internal/inventory/models"
)

type stubCountryRepository struct {
	countries []models.Country
	err       error
}

func (s *stubCountryRepository) ListCountries() ([]models.Country, error) {
	return s.countries, s.err
}

func TestCountryHandlerListReturnsCountries(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := NewCountryHandler(&stubCountryRepository{countries: []models.Country{{ID: 1, Name: "Congo", Code: "CG"}}})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodGet, "/countries", nil)
	c.Request = req

	h.List(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var payload struct {
		Success bool `json:"success"`
		Data    struct {
			Countries []models.Country `json:"countries"`
		} `json:"data"`
	}

	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}

	if !payload.Success {
		t.Fatalf("expected success response")
	}
	if len(payload.Data.Countries) != 1 {
		t.Fatalf("expected one country, got %d", len(payload.Data.Countries))
	}
	if payload.Data.Countries[0].Name != "Congo" {
		t.Fatalf("expected country name Congo, got %s", payload.Data.Countries[0].Name)
	}
}
