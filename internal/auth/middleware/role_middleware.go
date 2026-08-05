package middleware

import (
	"github.com/gin-gonic/gin"

	"github.com/lunetterie/backend/internal/shared"
)

// RequireRoles n'autorise que les role_id listés. À chaîner après RequireAuth, qui place
// "role_id" dans le contexte. IDs vérifiés directement dans la table `roles` de la base de
// production (migrations/001_init.up.sql y est désynchronisé, ne pas s'y fier).
func RequireRoles(allowed ...int64) gin.HandlerFunc {
	allowedSet := make(map[int64]bool, len(allowed))
	for _, id := range allowed {
		allowedSet[id] = true
	}
	return func(c *gin.Context) {
		roleID, ok := c.Get("role_id")
		id, isInt64 := roleID.(int64)
		if !ok || !isInt64 || !allowedSet[id] {
			shared.Forbidden(c, "Accès réservé à certains rôles")
			c.Abort()
			return
		}
		c.Next()
	}
}
