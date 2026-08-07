package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lunetterie/backend/internal/inventory/models"
)

type stubExpeditionRepository struct {
	orders []models.SupplierOrder
	err    error
}

func (s *stubExpeditionRepository) Create(order *models.SupplierOrder) error {
	return nil
}

func (s *stubExpeditionRepository) List() ([]models.SupplierOrder, error) {
	return s.orders, s.err
}

func TestExpeditionHandlerListReturnsOrders(t *testing.T) {
	gin.SetMode(gin.TestMode)

	note := "Transit prioritaire"
	h := NewExpeditionHandler(&stubExpeditionRepository{orders: []models.SupplierOrder{{
		ID:        7,
		Supplier:  "Dubai",
		Quantity:  120,
		OrderDate: time.Date(2026, 8, 7, 0, 0, 0, 0, time.UTC),
		Note:      &note,
	}}})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodGet, "/expeditions", nil)
	c.Request = req

	h.List(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var payload struct {
		Success bool `json:"success"`
		Data    struct {
			Orders []models.SupplierOrder `json:"orders"`
		} `json:"data"`
	}

	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}

	if !payload.Success {
		t.Fatalf("expected success response")
	}
	if len(payload.Data.Orders) != 1 {
		t.Fatalf("expected one order, got %d", len(payload.Data.Orders))
	}
	if payload.Data.Orders[0].Supplier != "Dubai" {
		t.Fatalf("expected supplier Dubai, got %s", payload.Data.Orders[0].Supplier)
	}
}
