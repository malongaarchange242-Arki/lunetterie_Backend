package handlers

import (
	"io"

	"github.com/gin-gonic/gin"
	"github.com/lunetterie/backend/internal/inventory/services"
	"github.com/lunetterie/backend/internal/shared"
)

// Rôles autorisés à utiliser le chatbot. IDs vérifiés directement dans la table `roles`
// de la base de production (SELECT id, name FROM roles) : le fichier
// migrations/001_init.up.sql est désynchronisé de la prod (DIRECTION y est insérée en
// 7e position, mais vaut 8 en base ; SUPER_DIRECTEUR n'y figure même pas), ne pas s'y fier
// pour ces IDs.
var chatAllowedRoles = map[int64]bool{
	1:  true, // SUPER_ADMIN
	2:  true, // ADMIN
	8:  true, // DIRECTION
	12: true, // SUPER_DIRECTEUR
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
