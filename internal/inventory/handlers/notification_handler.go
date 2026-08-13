package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/lunetterie/backend/internal/inventory/models"
	"github.com/lunetterie/backend/internal/shared"
)

type notificationRepository interface {
	ListForUser(userID int64, unreadOnly bool) ([]models.Notification, error)
	MarkRead(userID, notificationID int64) error
	MarkAllRead(userID int64) error
}

type NotificationHandler struct {
	repo notificationRepository
}

func NewNotificationHandler(repo notificationRepository) *NotificationHandler {
	return &NotificationHandler{repo: repo}
}

// List rend les notifications de l'appelant, jamais celles d'un autre : l'identifiant vient
// du jeton et non d'un paramètre, sinon n'importe quel poste lirait la boîte de la Direction.
// GET /api/v1/inventory/notifications?unread=1
func (h *NotificationHandler) List(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	unreadOnly := false
	switch strings.ToLower(strings.TrimSpace(c.Query("unread"))) {
	case "1", "true", "oui":
		unreadOnly = true
	}

	notifications, err := h.repo.ListForUser(userID, unreadOnly)
	if err != nil {
		shared.InternalError(c, err.Error())
		return
	}

	unread := 0
	for i := range notifications {
		if notifications[i].ReadAt == nil {
			unread++
		}
	}

	shared.Success(c, http.StatusOK, gin.H{
		"notifications": notifications,
		// Le compteur accompagne la liste : la pastille du bandeau n'a pas à la parcourir,
		// et il reste juste quand `unread=1` a déjà filtré.
		"unread_count": unread,
	})
}

// MarkRead marque une notification comme lue.
// POST /api/v1/inventory/notifications/:id/read
func (h *NotificationHandler) MarkRead(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	notificationID, err := strconv.ParseInt(strings.TrimSpace(c.Param("id")), 10, 64)
	if err != nil || notificationID <= 0 {
		shared.BadRequest(c, "Identifiant de notification invalide")
		return
	}

	if err := h.repo.MarkRead(userID, notificationID); err != nil {
		shared.InternalError(c, err.Error())
		return
	}

	shared.Success(c, http.StatusOK, gin.H{"marked": true})
}

// MarkAllRead vide la pastille d'un coup.
// POST /api/v1/inventory/notifications/read-all
func (h *NotificationHandler) MarkAllRead(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}

	if err := h.repo.MarkAllRead(userID); err != nil {
		shared.InternalError(c, err.Error())
		return
	}

	shared.Success(c, http.StatusOK, gin.H{"marked": true})
}
