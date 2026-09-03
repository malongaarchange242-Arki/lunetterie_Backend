package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lunetterie/backend/internal/inventory/models"
)

func TestToValidationPayloadKeepsRealItemAttributes(t *testing.T) {
	list := models.SendList{
		ID:          12,
		SessionCode: "STK-2026-0001",
		City:        "Pointe-Noire",
		ItemCount:   1,
		CreatedAt:   time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC),
	}
	brand := "Pacino"
	shape := "Rectangle"
	color := "Noir"
	location := "Valise A-02 · Carton 4"
	item := models.SendListItem{
		ID:           5,
		ListID:       12,
		Reference:    strPtr("PA-5197-BL"),
		Brand:        strPtr(brand),
		Shape:        strPtr(shape),
		Color:        strPtr(color),
		LocationCode: strPtr(location),
		Barcode:      strPtr("BAR-001"),
	}

	payload := toValidationPayload(list, []models.SendListItem{item})
	items, ok := payload["items"].([]gin.H)
	if !ok {
		t.Fatalf("expected items to be a []gin.H, got %#v", payload["items"])
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	first := items[0]
	if got := first["brand"]; got != brand {
		t.Fatalf("expected brand %q, got %#v", brand, got)
	}
	if got := first["shape"]; got != shape {
		t.Fatalf("expected shape %q, got %#v", shape, got)
	}
	if got := first["color"]; got != color {
		t.Fatalf("expected color %q, got %#v", color, got)
	}
	if got := first["location_code"]; got != location {
		t.Fatalf("expected location %q, got %#v", location, got)
	}
	if got := first["emplacement"]; got != location {
		t.Fatalf("expected emplacement %q, got %#v", location, got)
	}
}

func TestToValidationPayloadTagsAdminCreatedListsAsAdmin(t *testing.T) {
	userID := int64(42)
	list := models.SendList{
		ID:          12,
		SessionCode: "STK-2026-0002",
		City:        "Pointe-Noire",
		CreatedBy:   &userID,
		CreatedAt:   time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC),
	}

	payload := toValidationPayload(list, nil)
	if got := payload["origine"]; got != "admin" {
		t.Fatalf("expected origine %q for admin-created list, got %#v", "admin", got)
	}
}

func TestToValidationPayloadMarksAdminCreatedListsAsValidated(t *testing.T) {
	userID := int64(42)
	list := models.SendList{
		ID:          12,
		SessionCode: "STK-2026-0003",
		City:        "Pointe-Noire",
		Status:      models.SendListStatusNouvelle,
		CreatedBy:   &userID,
		CreatedAt:   time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC),
	}

	payload := toValidationPayload(list, nil)
	if got := payload["statut"]; got != "valide" {
		t.Fatalf("expected statut %q for admin-created list, got %#v", "valide", got)
	}
}

func TestCreateAcceptsGlassIDWithoutBarcode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &stubSendListRepo{nextCode: "STK-2026-0001"}
	handler := NewSendListHandler(repo, nil)

	body := `{"city":"Pointe-Noire","items":[{"glass_id":42,"reference":"REF-42","brand":"Pacino","shape":"Vue","color":"Noir","location_code":"A-01"}]}`
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/inventory/send-lists", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Create(c)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusCreated, w.Code, w.Body.String())
	}
	if len(repo.createdItems) != 1 {
		t.Fatalf("expected one item created, got %d", len(repo.createdItems))
	}
	if repo.createdItems[0].GlassID == nil || *repo.createdItems[0].GlassID != 42 {
		t.Fatalf("expected glass_id 42 to be kept, got %#v", repo.createdItems[0].GlassID)
	}
}

func TestCreateAcceptsReferenceAndLocationWithoutBarcodeOrGlassID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &stubSendListRepo{nextCode: "STK-2026-0002"}
	handler := NewSendListHandler(repo, nil)

	body := `{"city":"Pointe-Noire","items":[{"reference":"REF-PRE-1","brand":"Pacino","shape":"Vue","color":"Noir","location_code":"VAL-001 / CTN-07"}]}`
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/inventory/send-lists", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Create(c)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d body=%s", http.StatusCreated, w.Code, w.Body.String())
	}
	if len(repo.createdItems) != 1 {
		t.Fatalf("expected one item created, got %d", len(repo.createdItems))
	}
	if repo.createdItems[0].Reference != "REF-PRE-1" {
		t.Fatalf("expected reference to be kept, got %#v", repo.createdItems[0].Reference)
	}
	if repo.createdItems[0].LocationCode != "VAL-001 / CTN-07" {
		t.Fatalf("expected location_code to be kept, got %#v", repo.createdItems[0].LocationCode)
	}
}

type stubSendListRepo struct {
	nextCode     string
	createdItems []models.SendListItemRequest
}

func (s *stubSendListRepo) Create(list *models.SendList, items []models.SendListItemRequest) error {
	s.createdItems = append([]models.SendListItemRequest{}, items...)
	list.ID = 101
	list.Status = models.SendListStatusNouvelle
	list.CreatedAt = time.Now()
	list.UpdatedAt = time.Now()
	return nil
}

func (s *stubSendListRepo) NextStockListCode() (string, error) { return s.nextCode, nil }

func (s *stubSendListRepo) SplitAvailableBarcodes(barcodes []string) ([]string, map[string]string, error) {
	if len(barcodes) == 0 {
		return []string{}, map[string]string{"": "no barcode"}, nil
	}
	return barcodes, map[string]string{}, nil
}

func (s *stubSendListRepo) List(status string) ([]models.SendList, error) { return nil, nil }

func (s *stubSendListRepo) ListItems(listID int64, query string) ([]models.SendListItem, error) {
	return nil, nil
}

func (s *stubSendListRepo) Cancel(id int64) (*models.SendList, error) { return nil, nil }

func (s *stubSendListRepo) MarkSeen(ids []int64) (int64, error) { return 0, nil }

func (s *stubSendListRepo) MarkProcessed(ids []int64) (int64, error) { return 0, nil }

func strPtr(s string) *string { return &s }
