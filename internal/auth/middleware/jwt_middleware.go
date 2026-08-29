package middleware

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	authServices "github.com/lunetterie/backend/internal/auth/services"
	identityServices "github.com/lunetterie/backend/internal/identity/services"
	"github.com/lunetterie/backend/internal/shared"
)

// RequireAuth vérifie le JWT (Authorization: Bearer <token>) et place l'ID utilisateur
// dans le contexte gin sous la clé "user_id" (chaîne, attendue telle quelle par les handlers).
// Il accepte le service legacy et le service Identity durant la migration de structure.
func RequireAuth(authService interface{}) gin.HandlerFunc {
	parseToken := func(token string) (int64, int64, error) {
		switch s := authService.(type) {
		case *authServices.AuthService:
			claims, err := s.ParseToken(token)
			if err != nil {
				return 0, 0, err
			}
			return claims.UserID, claims.RoleID, nil
		case *identityServices.AuthService:
			claims, err := s.ParseToken(token)
			if err != nil {
				return 0, 0, err
			}
			return claims.UserID, claims.RoleID, nil
		default:
			return 0, 0, fmt.Errorf("service d'authentification non supporté")
		}
	}

	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" {
			shared.Unauthorized(c, "Authentification requise")
			c.Abort()
			return
		}

		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			shared.Unauthorized(c, "En-tête Authorization invalide")
			c.Abort()
			return
		}

		userID, roleID, err := parseToken(parts[1])
		if err != nil {
			shared.Unauthorized(c, fmt.Sprintf("Token invalide: %v", err))
			c.Abort()
			return
		}

		c.Set("user_id", strconv.FormatInt(userID, 10))
		c.Set("role_id", roleID)
		c.Next()
	}
}
