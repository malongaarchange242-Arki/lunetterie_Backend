package handlers

import "github.com/gin-gonic/gin"

// AuthHandler exposera les endpoints d'authentification du module Identity.
type AuthHandler struct{}

func NewAuthHandler() *AuthHandler {
	return &AuthHandler{}
}

func (h *AuthHandler) Login(c *gin.Context) {
	c.JSON(501, gin.H{"error": "Identity auth handler is scaffolded but not yet connected to production logic"})
}
