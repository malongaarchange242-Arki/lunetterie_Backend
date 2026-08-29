package handlers

import "github.com/gin-gonic/gin"

// ReceptionHandler expose le point d'entrée HTTP de Reception.
type ReceptionHandler struct{}

func NewReceptionHandler() *ReceptionHandler {
	return &ReceptionHandler{}
}

func (h *ReceptionHandler) Handle(c *gin.Context) {
	c.JSON(501, gin.H{"error": "Reception handler is scaffolded and waiting for the production workflow wiring"})
}
