package services

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
)

// StorageService gère l'upload des photos vers le bucket Supabase Storage
type StorageService struct {
	supabaseURL string
	serviceKey  string
	bucket      string
	httpClient  *http.Client
}

// NewStorageService crée une nouvelle instance
func NewStorageService(supabaseURL, serviceKey, bucket string) *StorageService {
	return &StorageService{
		supabaseURL: supabaseURL,
		serviceKey:  serviceKey,
		bucket:      bucket,
		httpClient:  &http.Client{},
	}
}

// Upload envoie une image dans le bucket et renvoie son URL publique
func (s *StorageService) Upload(path string, data []byte, contentType string) (string, error) {
	url := fmt.Sprintf("%s/storage/v1/object/%s/%s", s.supabaseURL, s.bucket, path)

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("erreur création requête upload: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+s.serviceKey)
	req.Header.Set("apikey", s.serviceKey)
	req.Header.Set("Content-Type", contentType)
	// On ne fait pas de upsert car deux photos du même type doivent rester distinctes
	// et ne doivent pas écraser la précédente avec un nom identique.

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("erreur upload vers Supabase Storage: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("upload échoué (status %d): %s", resp.StatusCode, string(body))
	}

	publicURL := fmt.Sprintf("%s/storage/v1/object/public/%s/%s", s.supabaseURL, s.bucket, path)
	return publicURL, nil
}
