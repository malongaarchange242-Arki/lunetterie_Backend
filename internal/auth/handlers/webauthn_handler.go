package handlers

import (
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/lunetterie/backend/internal/auth/dto"
	"github.com/lunetterie/backend/internal/auth/services"
	"github.com/lunetterie/backend/internal/shared"
)

type WebAuthnHandler struct {
	service     *services.WebAuthnService
	authService *services.AuthService
}

func NewWebAuthnHandler(service *services.WebAuthnService, authService *services.AuthService) *WebAuthnHandler {
	return &WebAuthnHandler{service: service, authService: authService}
}

// RegisterChallenge - POST /api/v1/auth/webauthn/register-challenge
func (h *WebAuthnHandler) RegisterChallenge(c *gin.Context) {
	var req dto.RegisterChallengeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("❌ Erreur parsing JSON: %v", err)
		shared.BadRequest(c, "Données invalides: "+err.Error())
		return
	}

	log.Printf("📝 Demande challenge WebAuthn pour: %s", req.Email)
	result, err := h.service.CreateRegisterChallenge(req.Email)
	if err != nil {
		log.Printf("❌ Erreur service: %v", err)
		shared.InternalError(c, "Erreur création challenge: "+err.Error())
		return
	}

	log.Printf("✅ Challenge créé pour: %s", req.Email)
	shared.Success(c, http.StatusOK, result)
}

// RegisterVerify - POST /api/v1/auth/webauthn/register-verify
func (h *WebAuthnHandler) RegisterVerify(c *gin.Context) {
	var req dto.RegisterVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.BadRequest(c, "Données invalides: "+err.Error())
		return
	}

	err := h.service.VerifyRegister(req.Email, req.ID, req.Response.ClientDataJSON, req.Response.AttestationObject)
	if err != nil {
		shared.BadRequest(c, "Vérification échouée: "+err.Error())
		return
	}

	shared.Success(c, http.StatusOK, gin.H{"message": "Empreinte enregistrée avec succès"})
}

// LoginChallenge - POST /api/v1/auth/webauthn/login-challenge
func (h *WebAuthnHandler) LoginChallenge(c *gin.Context) {
	var req dto.LoginChallengeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.BadRequest(c, "Données invalides: "+err.Error())
		return
	}

	result, err := h.service.CreateLoginChallenge(req.Email)
	if err != nil {
		shared.Unauthorized(c, "Utilisateur introuvable")
		return
	}

	shared.Success(c, http.StatusOK, result)
}

// LoginVerify - POST /api/v1/auth/webauthn/login-verify
func (h *WebAuthnHandler) LoginVerify(c *gin.Context) {
	var req dto.LoginVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.BadRequest(c, "Données invalides: "+err.Error())
		return
	}

	user, err := h.service.VerifyLogin(req.Email, req.ID, req.Response.ClientDataJSON, req.Response.AuthenticatorData, req.Response.Signature)
	if err != nil {
		shared.Unauthorized(c, "Authentification échouée: "+err.Error())
		return
	}

	// Générer le JWT
	token, err := h.authService.GenerateToken(user)
	if err != nil {
		shared.InternalError(c, "Erreur génération token")
		return
	}

	shared.Success(c, http.StatusOK, gin.H{
		"token": token,
		"user":  user,
	})
}

// DiscoverableLoginChallenge - POST /api/v1/auth/webauthn/discoverable-login-challenge
// Connexion 100% biométrique : aucun email requis, l'identité est déduite de l'empreinte scannée.
func (h *WebAuthnHandler) DiscoverableLoginChallenge(c *gin.Context) {
	result, err := h.service.CreateDiscoverableLoginChallenge()
	if err != nil {
		shared.InternalError(c, "Erreur creation challenge: "+err.Error())
		return
	}

	shared.Success(c, http.StatusOK, result)
}

// DiscoverableLoginVerify - POST /api/v1/auth/webauthn/discoverable-login-verify
func (h *WebAuthnHandler) DiscoverableLoginVerify(c *gin.Context) {
	var req dto.DiscoverableLoginVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.BadRequest(c, "Donnees invalides: "+err.Error())
		return
	}

	user, err := h.service.VerifyDiscoverableLogin(req.ID, req.Response.ClientDataJSON, req.Response.AuthenticatorData, req.Response.Signature)
	if err != nil {
		shared.Unauthorized(c, "Authentification echouee: "+err.Error())
		return
	}

	token, err := h.authService.GenerateToken(user)
	if err != nil {
		shared.InternalError(c, "Erreur generation token")
		return
	}

	shared.Success(c, http.StatusOK, gin.H{
		"token": token,
		"user":  user,
	})
}

// EnrollChallenge - POST /api/v1/auth/webauthn/enroll-challenge
// Permet de générer un challenge pour ajouter une empreinte à un employé déjà créé.
func (h *WebAuthnHandler) EnrollChallenge(c *gin.Context) {
	var req dto.EnrollChallengeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.BadRequest(c, "Données invalides: "+err.Error())
		return
	}

	result, err := h.service.CreateEnrollChallenge(req.UserID)
	if err != nil {
		shared.BadRequest(c, err.Error())
		return
	}

	shared.Success(c, http.StatusOK, result)
}

// EnrollVerify - POST /api/v1/auth/webauthn/enroll-verify
// Vérifie l'empreinte capturée et la lie immédiatement à l'employé existant.
func (h *WebAuthnHandler) EnrollVerify(c *gin.Context) {
	var req dto.EnrollVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.BadRequest(c, "Données invalides: "+err.Error())
		return
	}

	if err := h.service.EnrollCredentialForUser(req.UserID, req.ID, req.Response.ClientDataJSON, req.Response.AttestationObject); err != nil {
		shared.BadRequest(c, "Vérification échouée: "+err.Error())
		return
	}

	shared.Success(c, http.StatusOK, gin.H{"message": "Empreinte enregistrée avec succès"})
}

// RemoveCredentials - DELETE /api/v1/auth/webauthn/credentials/:userId
func (h *WebAuthnHandler) RemoveCredentials(c *gin.Context) {
	userID, err := strconv.ParseInt(c.Param("userId"), 10, 64)
	if err != nil {
		shared.BadRequest(c, "Identifiant utilisateur invalide")
		return
	}

	if err := h.service.RemoveCredentialsForUser(userID); err != nil {
		shared.InternalError(c, "Impossible de supprimer l'empreinte")
		return
	}

	shared.Success(c, http.StatusOK, gin.H{"message": "Empreinte supprimée"})
}
