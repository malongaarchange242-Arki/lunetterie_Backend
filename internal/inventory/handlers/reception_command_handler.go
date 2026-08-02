package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lunetterie/backend/internal/inventory/models"
	"github.com/lunetterie/backend/internal/inventory/repositories"
	"github.com/lunetterie/backend/internal/shared"
)

// newSessionCode génère un code lisible et unique au format SYYMMDD-####.
func newSessionCode(repo *repositories.ReceptionCommandRepository) (string, error) {
	now := time.Now()
	prefix := fmt.Sprintf("S%s-", now.Format("060102"))

	latestCode, err := repo.GetLatestCodeWithPrefix(prefix)
	if err != nil {
		return "", err
	}

	sequence := 0
	if latestCode != "" {
		if n, parseErr := strconv.Atoi(strings.TrimPrefix(latestCode, prefix)); parseErr == nil {
			sequence = n
		}
	}

	return fmt.Sprintf("%s%04d", prefix, sequence+1), nil
}

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

	code, err := newSessionCode(h.repo)
	if err != nil {
		shared.InternalError(c, "Erreur génération du code de session")
		return
	}

	command := &models.ReceptionCommand{
		Code:            code,
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

// List retourne les sessions de réception, tous créateurs confondus — le
// bandeau "session active" ne doit pas se limiter à ce que le poste courant a
// créé localement, mais refléter l'état réel partagé côté serveur.
func (h *ReceptionCommandHandler) List(c *gin.Context) {
	status := strings.TrimSpace(c.Query("status"))
	commands, err := h.repo.List(status)
	if err != nil {
		shared.InternalError(c, err.Error())
		return
	}
	shared.Success(c, http.StatusOK, gin.H{"commands": commands})
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
