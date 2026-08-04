package handlers

import (
	"io"

	"github.com/gin-gonic/gin"
	"github.com/lunetterie/backend/internal/inventory/services"
	"github.com/lunetterie/backend/internal/shared"
)

// Rôles autorisés à utiliser le chatbot (voir migrations/001_init.up.sql, insert roles).
// DIRECTION (7) est le rôle prévu pour ce module, mais en pratique le compte utilisé au
// quotidien pour la gestion est en ADMIN (2) : on autorise donc aussi ADMIN/SUPER_ADMIN
// plutôt que de forcer un changement de rôle en base.
var chatAllowedRoles = map[int64]bool{
	1: true, // SUPER_ADMIN
	2: true, // ADMIN
	7: true, // DIRECTION
}

// ChatHandler expose le chatbot de direction (résumés/questions sur l'activité du magasin)
type ChatHandler struct {
	aiService *services.AIService
}

// NewChatHandler crée une nouvelle instance
func NewChatHandler(aiService *services.AIService) *ChatHandler {
	return &ChatHandler{aiService: aiService}
}

// HandleChat relaie {message, history, context} (déjà construit par direction.js) au
// chatbot de direction du service IA. Réservé au rôle DIRECTION.
// POST /api/v1/ai/chat
func (h *ChatHandler) HandleChat(c *gin.Context) {
	roleID, _ := c.Get("role_id")
	if id, ok := roleID.(int64); !ok || !chatAllowedRoles[id] {
		shared.Forbidden(c, "Réservé à la direction/administration")
		return
	}

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		shared.BadRequest(c, "Corps de requête invalide")
		return
	}

	reply, err := h.aiService.Chat(body)
	if err != nil {
		shared.InternalError(c, "Erreur lors de la réponse du chatbot: "+err.Error())
		return
	}

	shared.Success(c, 200, gin.H{"reply": reply})
}
