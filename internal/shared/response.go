package shared

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// APIResponse représente une réponse standard de l'API
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *string     `json:"error,omitempty"`
	Message *string     `json:"message,omitempty"`
}

// Success retourne une réponse succès
func Success(c *gin.Context, statusCode int, data interface{}) {
	c.JSON(statusCode, APIResponse{
		Success: true,
		Data:    data,
	})
}

// BadRequest retourne une erreur 400
func BadRequest(c *gin.Context, message string) {
	c.JSON(http.StatusBadRequest, APIResponse{
		Success: false,
		Error:   &message,
	})
}

// BadRequestWithData retourne une erreur 400 en conservant un corps de données : utile quand
// l'échec porte sur un lot et que le détail par élément explique le refus.
func BadRequestWithData(c *gin.Context, message string, data interface{}) {
	c.JSON(http.StatusBadRequest, APIResponse{
		Success: false,
		Data:    data,
		Error:   &message,
	})
}

// Unauthorized retourne une erreur 401
func Unauthorized(c *gin.Context, message string) {
	c.JSON(http.StatusUnauthorized, APIResponse{
		Success: false,
		Error:   &message,
	})
}

// Forbidden retourne une erreur 403
func Forbidden(c *gin.Context, message string) {
	c.JSON(http.StatusForbidden, APIResponse{
		Success: false,
		Error:   &message,
	})
}

// NotFound retourne une erreur 404
func NotFound(c *gin.Context, message string) {
	c.JSON(http.StatusNotFound, APIResponse{
		Success: false,
		Error:   &message,
	})
}

// InternalError retourne une erreur 500
func InternalError(c *gin.Context, message string) {
	c.JSON(http.StatusInternalServerError, APIResponse{
		Success: false,
		Error:   &message,
	})
}

// Created retourne une création succès 201
func Created(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, APIResponse{
		Success: true,
		Data:    data,
	})
}
