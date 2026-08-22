package settings

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/lunetterie/backend/internal/shared"
)

type Handler struct {
	repo *Repository
}

func NewHandler(repo *Repository) *Handler {
	return &Handler{repo: repo}
}

func (h *Handler) Get(c *gin.Context) {
	config, err := h.repo.Get()
	if err != nil {
		shared.InternalError(c, err.Error())
		return
	}
	shared.Success(c, http.StatusOK, gin.H{"settings": config})
}

func (h *Handler) Save(c *gin.Context) {
	var config Config
	if err := c.ShouldBindJSON(&config); err != nil {
		shared.BadRequest(c, "Données de réglages invalides")
		return
	}
	if err := h.repo.Save(config); err != nil {
		shared.InternalError(c, err.Error())
		return
	}
	shared.Success(c, http.StatusOK, gin.H{"settings": config})
}
