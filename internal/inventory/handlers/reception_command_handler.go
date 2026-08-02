package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/lunetterie/backend/internal/inventory/models"
	"github.com/lunetterie/backend/internal/inventory/repositories"
	"github.com/lunetterie/backend/internal/shared"
)

type ReceptionCommandHandler struct {
	repo *repositories.ReceptionCommandRepository
}

func NewReceptionCommandHandler(repo *repositories.ReceptionCommandRepository) *ReceptionCommandHandler {
	return &ReceptionCommandHandler{repo: repo}
}

func (h *ReceptionCommandHandler) Create(c *gin.Context) {
	var req models.ReceptionCommandCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.BadRequest(c, "Données invalides")
		return
	}
	if req.TargetCount < 1 {
		shared.BadRequest(c, "target_count doit être supérieur à 0")
		return
	}

	code := fmt.Sprintf("SESSION-%s-%s", strconv.FormatInt(int64(len(strings.ToUpper(fmt.Sprintf("%d", req.TargetCount)))+1000), 36), strconv.FormatInt(int64(1000+req.TargetCount), 36))
	command := &models.ReceptionCommand{
		Code:            strings.ToUpper(code),
		TargetCount:     req.TargetCount,
		RegisteredCount: 0,
		Status:          "active",
	}
	if userID, exists := c.Get("user_id"); exists {
		if id, err := strconv.ParseInt(fmt.Sprintf("%v", userID), 10, 64); err == nil {
			command.CreatedBy = &id
		}
	}

	if err := h.repo.Create(command); err != nil {
		shared.InternalError(c, "Erreur lors de la création de la commande")
		return
	}

	shared.Created(c, gin.H{"command": command})
}

func (h *ReceptionCommandHandler) GetByCode(c *gin.Context) {
	code := strings.TrimSpace(c.Param("code"))
	command, err := h.repo.GetByCode(code)
	if err != nil {
		shared.InternalError(c, err.Error())
		return
	}
	if command == nil {
		shared.NotFound(c, "commande introuvable")
		return
	}
	shared.Success(c, http.StatusOK, gin.H{"command": command})
}

func (h *ReceptionCommandHandler) Increment(c *gin.Context) {
	code := strings.TrimSpace(c.Param("code"))
	command, err := h.repo.Increment(code)
	if err != nil {
		shared.InternalError(c, err.Error())
		return
	}
	if command == nil {
		shared.NotFound(c, "commande introuvable")
		return
	}
	shared.Success(c, http.StatusOK, gin.H{"command": command})
}
