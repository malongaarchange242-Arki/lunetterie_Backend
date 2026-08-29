package services

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	authModels "github.com/lunetterie/backend/internal/auth/models"
	authServices "github.com/lunetterie/backend/internal/auth/services"
	identityModels "github.com/lunetterie/backend/internal/identity/models"
)

// AuthService adapte le service historique vers le module Identity.
type AuthService struct {
	delegate *authServices.AuthService
	jwtSecret string
}

func NewAuthService(delegate *authServices.AuthService) *AuthService {
	if delegate == nil {
		return &AuthService{jwtSecret: ""}
	}
	return &AuthService{delegate: delegate, jwtSecret: ""}
}

// AuthClaims est la version Identity du payload JWT.
type AuthClaims struct {
	UserID int64  `json:"user_id"`
	Email  string `json:"email"`
	RoleID int64  `json:"role_id"`
	jwt.RegisteredClaims
}

func (s *AuthService) ParseToken(token string) (*AuthClaims, error) {
	if token == "" {
		return nil, fmt.Errorf("token vide")
	}
	if s.delegate != nil {
		legacyClaims, err := s.delegate.ParseToken(token)
		if err != nil {
			return nil, err
		}
		return &AuthClaims{
			UserID: legacyClaims.UserID,
			Email:  legacyClaims.Email,
			RoleID: legacyClaims.RoleID,
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: legacyClaims.ExpiresAt,
				IssuedAt:  legacyClaims.IssuedAt,
				Subject:   legacyClaims.Subject,
			},
		}, nil
	}
	claims := &AuthClaims{}
	parsed, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("méthode de signature inattendue: %v", t.Header["alg"])
		}
		return []byte(s.jwtSecret), nil
	})
	if err != nil || !parsed.Valid {
		return nil, fmt.Errorf("token invalide: %w", err)
	}
	return claims, nil
}

func (s *AuthService) ValidateToken(token string) error {
	_, err := s.ParseToken(token)
	return err
}

func (s *AuthService) GenerateToken(user *identityModels.User) (string, error) {
	if user == nil {
		return "", fmt.Errorf("utilisateur requis")
	}
	if s.delegate != nil {
		legacyUser := &authModels.User{ID: user.ID, Email: user.Email, RoleID: user.RoleID}
		return s.delegate.GenerateToken(legacyUser)
	}
	claims := AuthClaims{
		UserID: user.ID,
		Email:  user.Email,
		RoleID: user.RoleID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   fmt.Sprintf("%d", user.ID),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.jwtSecret))
}
