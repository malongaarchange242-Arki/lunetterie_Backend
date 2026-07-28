package middleware

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/lunetterie/backend/internal/auth/services"
	"github.com/lunetterie/backend/internal/shared"
)

// RequireAuth vérifie le JWT (Authorization: Bearer <token>) et place l'ID utilisateur
// dans le contexte gin sous la clé "user_id" (chaîne, attendue telle quelle par les handlers).
func RequireAuth(authService *services.AuthService) gin.HandlerFunc {
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

		claims, err := authService.ParseToken(parts[1])
		if err != nil {
			shared.Unauthorized(c, fmt.Sprintf("Token invalide: %v", err))
			c.Abort()
			return
		}

		c.Set("user_id", strconv.FormatInt(claims.UserID, 10))
		c.Set("role_id", claims.RoleID)
		c.Next()
	}
}
