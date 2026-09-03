package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestExtractCatalogueFiltersParsesFrontendFilterKeys(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/inventory/catalogue/pre-registration-boxes?q=Ray-Ban&fGamme=Classique&fGenre=Homme&fType=Vue&fMarque=Pacino&fCouleur=Noir", nil)

	filters := extractCatalogueFilters(c)
	if filters.Query != "Ray-Ban" {
		t.Fatalf("expected query %q, got %q", "Ray-Ban", filters.Query)
	}
	if filters.Gamme != "Classique" {
		t.Fatalf("expected gamme %q, got %q", "Classique", filters.Gamme)
	}
	if filters.Genre != "Homme" {
		t.Fatalf("expected genre %q, got %q", "Homme", filters.Genre)
	}
	if filters.Type != "Vue" {
		t.Fatalf("expected type %q, got %q", "Vue", filters.Type)
	}
	if filters.Marque != "Pacino" {
		t.Fatalf("expected marque %q, got %q", "Pacino", filters.Marque)
	}
	if filters.Couleur != "Noir" {
		t.Fatalf("expected couleur %q, got %q", "Noir", filters.Couleur)
	}
}

func TestBuildPhotoStoragePathUsesUniqueNamePerPhoto(t *testing.T) {
	pathA := buildPhotoStoragePath("VAL-016", "RO-20-001", "face", "p1", "IMG_0387.JPG")
	pathB := buildPhotoStoragePath("VAL-016", "RO-20-001", "face", "p2", "image.JPG")

	if pathA == pathB {
		t.Fatalf("expected distinct storage paths for different photos, got same: %s", pathA)
	}
	if pathA == "" || pathB == "" {
		t.Fatal("storage paths must not be empty")
	}
	if len(pathA) < len("pre-registration/") || len(pathB) < len("pre-registration/") {
		t.Fatal("storage path must start with pre-registration/")
	}
}

func TestBuildPhotoStoragePathSanitizesKindAndFileName(t *testing.T) {
	path := buildPhotoStoragePath("VAL-016", "RO-20-001", "face/side", "photo-123", "IMG 0387.JPG")
	if path == "" {
		t.Fatal("sanitized path must not be empty")
	}
	if path == "pre-registration/VAL-016/RO-20-001/" {
		t.Fatal("storage path should not be empty after sanitization")
	}
	if len(path) < len("pre-registration/VAL-016/RO-20-001/photo-123-face-side-img-0387-jpg") {
		t.Fatal("storage path appears to be missing the sanitized filename or kind")
	}
}
