package services

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestStorageServiceUploadUsesUpsert(t *testing.T) {
	var gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path + "?" + r.URL.RawQuery
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	svc := NewStorageService(server.URL, "service-key", "photos")

	_, err := svc.Upload("LUN-001/monture.jpg", []byte("abc"), "image/jpeg")
	if err != nil {
		t.Fatalf("Upload returned error: %v", err)
	}

	if gotPath == "" {
		t.Fatal("expected request path to be called")
	}
	if gotPath != "/storage/v1/object/photos/LUN-001/monture.jpg?upsert=true" {
		t.Fatalf("expected upsert=true in upload URL, got %q", gotPath)
	}
}
