package handlers

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/lunetterie/backend/internal/shared"
)

func currentUserID(c *gin.Context) (int64, bool) {
	userIDVal, ok := c.Get("user_id")
	if !ok {
		shared.Unauthorized(c, "Utilisateur non authentifié")
		return 0, false
	}
	userIDStr, _ := userIDVal.(string)
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		shared.BadRequest(c, "ID utilisateur invalide")
		return 0, false
	}
	return userID, true
}
