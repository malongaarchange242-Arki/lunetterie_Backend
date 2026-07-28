package services

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/lunetterie/backend/internal/auth/dto"
	"github.com/lunetterie/backend/internal/auth/models"
	"github.com/lunetterie/backend/internal/auth/repositories"
)

type WebAuthnService struct {
	repo     *repositories.WebAuthnRepository
	userRepo *repositories.UserRepository
}

func NewWebAuthnService(repo *repositories.WebAuthnRepository, userRepo *repositories.UserRepository) *WebAuthnService {
	return &WebAuthnService{repo: repo, userRepo: userRepo}
}

func (s *WebAuthnService) GenerateChallenge() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

func (s *WebAuthnService) CreateRegisterChallenge(email string) (*dto.RegisterChallengeResponse, error) {
	if _, err := s.userRepo.FindByEmail(email); err == nil {
		return nil, fmt.Errorf("un utilisateur existe deja avec cet email")
	}

	challenge, err := s.GenerateChallenge()
	if err != nil {
		return nil, err
	}

	if err := s.repo.SaveRegistrationChallenge(challenge, time.Now().Add(5*time.Minute)); err != nil {
		return nil, err
	}

	return &dto.RegisterChallengeResponse{
		Challenge: challenge,
		UserID:    hash(email)[:16],
	}, nil
}

func (s *WebAuthnService) VerifyRegister(email, credentialID, clientDataJSON, attestationObject string) error {
	var clientData map[string]interface{}
	decoded, err := base64.RawURLEncoding.DecodeString(clientDataJSON)
	if err != nil {
		return fmt.Errorf("clientDataJSON invalide: %w", err)
	}
	if err := json.Unmarshal(decoded, &clientData); err != nil {
		return fmt.Errorf("clientDataJSON invalide: %w", err)
	}

	challenge, ok := clientData["challenge"].(string)
	if !ok {
		return fmt.Errorf("challenge manquant")
	}

	savedChallenge, err := s.repo.GetRegistrationChallenge(challenge)
	if err != nil {
		return fmt.Errorf("challenge invalide ou expire: %w", err)
	}

	if err := s.repo.SavePendingCredential(credentialID, []byte(attestationObject), -7); err != nil {
		return fmt.Errorf("erreur sauvegarde credential: %w", err)
	}

	_ = s.repo.DeleteChallenge(savedChallenge.ID)

	return nil
}

func (s *WebAuthnService) CreateLoginChallenge(email string) (*dto.LoginChallengeResponse, error) {
	user, err := s.userRepo.FindByEmail(email)
	if err != nil {
		return nil, fmt.Errorf("utilisateur introuvable")
	}

	creds, err := s.repo.GetCredentialsByUser(user.ID)
	if err != nil {
		return nil, err
	}

	var credentialIDs []string
	for _, cred := range creds {
		credentialIDs = append(credentialIDs, cred.CredentialID)
	}

	challenge, err := s.GenerateChallenge()
	if err != nil {
		return nil, err
	}

	c := &models.WebAuthnChallenge{
		UserID:    user.ID,
		Challenge: challenge,
		ExpiresAt: time.Now().Add(5 * time.Minute),
	}
	if err := s.repo.SaveChallenge(c); err != nil {
		return nil, err
	}

	return &dto.LoginChallengeResponse{
		Challenge:     challenge,
		CredentialIDs: credentialIDs,
	}, nil
}

func (s *WebAuthnService) VerifyLogin(email, credentialID, clientDataJSON, authenticatorData, signature string) (*models.User, error) {
	user, err := s.userRepo.FindByEmail(email)
	if err != nil {
		return nil, fmt.Errorf("utilisateur introuvable")
	}

	cred, err := s.repo.GetCredentialByID(credentialID)
	if err != nil {
		return nil, fmt.Errorf("credential introuvable")
	}
	if cred.UserID != user.ID {
		return nil, fmt.Errorf("credential non associee a cet utilisateur")
	}

	var clientData map[string]interface{}
	decoded, err := base64.RawURLEncoding.DecodeString(clientDataJSON)
	if err != nil {
		return nil, fmt.Errorf("clientDataJSON invalide")
	}
	if err := json.Unmarshal(decoded, &clientData); err != nil {
		return nil, fmt.Errorf("clientDataJSON invalide")
	}

	challenge, ok := clientData["challenge"].(string)
	if !ok {
		return nil, fmt.Errorf("challenge manquant")
	}

	savedChallenge, err := s.repo.GetChallenge(user.ID, challenge)
	if err != nil {
		return nil, fmt.Errorf("challenge invalide ou expire")
	}

	_ = cred
	_ = authenticatorData
	_ = signature

	_ = s.repo.DeleteChallenge(savedChallenge.ID)

	return user, nil
}

// CreateDiscoverableLoginChallenge génère un challenge de connexion sans connaître
// l'utilisateur à l'avance : l'identité est déduite de la credential choisie par
// l'authenticateur (empreinte "discoverable"/résidente).
func (s *WebAuthnService) CreateDiscoverableLoginChallenge() (*dto.DiscoverableLoginChallengeResponse, error) {
	challenge, err := s.GenerateChallenge()
	if err != nil {
		return nil, err
	}

	if err := s.repo.SaveRegistrationChallenge(challenge, time.Now().Add(2*time.Minute)); err != nil {
		return nil, err
	}

	return &dto.DiscoverableLoginChallengeResponse{Challenge: challenge}, nil
}

// VerifyDiscoverableLogin retrouve l'utilisateur à partir de la credential id renvoyée
// par le navigateur, sans email fourni au préalable.
func (s *WebAuthnService) VerifyDiscoverableLogin(credentialID, clientDataJSON, authenticatorData, signature string) (*models.User, error) {
	var clientData map[string]interface{}
	decoded, err := base64.RawURLEncoding.DecodeString(clientDataJSON)
	if err != nil {
		return nil, fmt.Errorf("clientDataJSON invalide")
	}
	if err := json.Unmarshal(decoded, &clientData); err != nil {
		return nil, fmt.Errorf("clientDataJSON invalide")
	}

	challenge, ok := clientData["challenge"].(string)
	if !ok {
		return nil, fmt.Errorf("challenge manquant")
	}

	savedChallenge, err := s.repo.GetRegistrationChallenge(challenge)
	if err != nil {
		return nil, fmt.Errorf("challenge invalide ou expire")
	}
	_ = s.repo.DeleteChallenge(savedChallenge.ID)

	cred, err := s.repo.GetCredentialByID(credentialID)
	if err != nil {
		return nil, fmt.Errorf("empreinte non reconnue")
	}
	if cred.UserID == 0 {
		return nil, fmt.Errorf("empreinte non associee a un compte")
	}

	user, err := s.userRepo.FindByID(cred.UserID)
	if err != nil {
		return nil, fmt.Errorf("utilisateur introuvable")
	}

	_ = authenticatorData
	_ = signature

	return user, nil
}

func (s *WebAuthnService) CreateEnrollChallenge(userID int64) (*dto.EnrollChallengeResponse, error) {
	if _, err := s.userRepo.FindByID(userID); err != nil {
		return nil, fmt.Errorf("utilisateur introuvable")
	}

	challenge, err := s.GenerateChallenge()
	if err != nil {
		return nil, err
	}

	if err := s.repo.SaveRegistrationChallenge(challenge, time.Now().Add(5*time.Minute)); err != nil {
		return nil, err
	}

	return &dto.EnrollChallengeResponse{
		Challenge: challenge,
		UserID:    fmt.Sprintf("%d", userID),
	}, nil
}

// EnrollCredentialForUser vérifie une empreinte capturée puis la lie immédiatement
// à un utilisateur déjà existant (contrairement à VerifyRegister + RegisterUser qui
// laissent la credential "en attente" jusqu'à la création du compte).
func (s *WebAuthnService) EnrollCredentialForUser(userID int64, credentialID, clientDataJSON, attestationObject string) error {
	if _, err := s.userRepo.FindByID(userID); err != nil {
		return fmt.Errorf("utilisateur introuvable")
	}

	if err := s.VerifyRegister("", credentialID, clientDataJSON, attestationObject); err != nil {
		return err
	}

	return s.LinkCredentialToUser(credentialID, userID)
}

func (s *WebAuthnService) RemoveCredentialsForUser(userID int64) error {
	return s.repo.DeleteCredentialsByUser(userID)
}

func (s *WebAuthnService) LinkCredentialToUser(credentialID string, userID int64) error {
	if err := s.repo.LinkCredentialToUser(credentialID, userID); err != nil {
		return fmt.Errorf("credential introuvable ou deja associee: %w", err)
	}
	return nil
}

func (s *WebAuthnService) PendingCredentialExists(credentialID string) (bool, error) {
	return s.repo.PendingCredentialExists(credentialID)
}

func hash(s string) string {
	h := sha256.Sum256([]byte(s))
	return base64.RawURLEncoding.EncodeToString(h[:])
}
