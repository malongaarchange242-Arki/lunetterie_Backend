package services

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
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

// Upload envoie une image dans le bucket et renvoie son URL publique.
// En cas de collision de clé, on retente une fois avec un suffixe unique pour éviter
// de casser la réception sur un upload redondant du même code-barres.
func (s *StorageService) Upload(path string, data []byte, contentType string) (string, error) {
	if strings.TrimSpace(s.supabaseURL) == "" || strings.TrimSpace(s.serviceKey) == "" || strings.TrimSpace(s.bucket) == "" {
		return "", fmt.Errorf("configuration Supabase Storage incomplète: SUPABASE_URL, SUPABASE_SERVICE_ROLE_KEY et bucket requis")
	}
	if !strings.HasPrefix(s.supabaseURL, "http://") && !strings.HasPrefix(s.supabaseURL, "https://") {
		return "", fmt.Errorf("SUPABASE_URL invalide: %q", s.supabaseURL)
	}

	attempts := []string{path}
	if idx := strings.LastIndex(path, "/"); idx >= 0 && idx < len(path)-1 {
		dir := path[:idx+1]
		base := path[idx+1:]
		suffix := strconv.FormatInt(time.Now().UnixNano(), 10)
		attempts = append(attempts, dir+"retry-"+suffix+"-"+base)
	}

	var lastErr error
	for _, candidate := range attempts {
		url := fmt.Sprintf("%s/storage/v1/object/%s/%s?upsert=true", s.supabaseURL, s.bucket, candidate)
		req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(data))
		if err != nil {
			return "", fmt.Errorf("erreur création requête upload: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+s.serviceKey)
		req.Header.Set("apikey", s.serviceKey)
		req.Header.Set("Content-Type", contentType)

		resp, err := s.httpClient.Do(req)
		if err != nil {
			return "", fmt.Errorf("erreur upload vers Supabase Storage: %w", err)
		}

		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusNoContent {
			publicURL := fmt.Sprintf("%s/storage/v1/object/public/%s/%s", s.supabaseURL, s.bucket, candidate)
			return publicURL, nil
		}

		lastErr = fmt.Errorf("upload échoué (status %d): %s", resp.StatusCode, string(body))
		if resp.StatusCode == http.StatusConflict || strings.Contains(strings.ToLower(string(body)), "keyalreadyexists") || strings.Contains(strings.ToLower(string(body)), "duplicate") {
			continue
		}
		break
	}

	if lastErr != nil {
		return "", lastErr
	}
	return "", fmt.Errorf("upload Storage sans réponse exploitable")
}
