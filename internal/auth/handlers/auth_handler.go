package handlers

import (
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/lunetterie/backend/internal/auth/dto"
	"github.com/lunetterie/backend/internal/auth/models"
	"github.com/lunetterie/backend/internal/auth/repositories"
	"github.com/lunetterie/backend/internal/auth/services"
	"github.com/lunetterie/backend/internal/shared"
)

type AuthHandler struct {
	userRepo        *repositories.UserRepository
	stationRepo     *repositories.StationRepository
	authService     *services.AuthService
	webauthnService *services.WebAuthnService
}

func stringPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func NewAuthHandler(userRepo *repositories.UserRepository, stationRepo *repositories.StationRepository, authService *services.AuthService, webauthnService *services.WebAuthnService) *AuthHandler {
	return &AuthHandler{userRepo: userRepo, stationRepo: stationRepo, authService: authService, webauthnService: webauthnService}
}

func (h *AuthHandler) RegisterUser(c *gin.Context) {
	var req dto.RegisterUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.BadRequest(c, "Donnees invalides: "+err.Error())
		return
	}

	exists, err := h.webauthnService.PendingCredentialExists(req.CredentialID)
	if err != nil {
		shared.InternalError(c, "Impossible de verifier l'empreinte")
		return
	}
	if !exists {
		shared.BadRequest(c, "Empreinte non verifiee ou deja utilisee")
		return
	}

	user := &models.User{
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Email:     req.Email,
		Phone:     stringPtr(req.Phone),
		Gender:    stringPtr(req.Gender),
		RoleID:    req.RoleID,
		StationID: req.StationID,
		IsActive:  true,
	}

	if err := h.userRepo.Create(user); err != nil {
		log.Printf("register user create failed: %v", err)
		shared.InternalError(c, "Impossible de creer l'utilisateur")
		return
	}

	if err := h.webauthnService.LinkCredentialToUser(req.CredentialID, user.ID); err != nil {
		shared.BadRequest(c, "Impossible de lier l'empreinte: "+err.Error())
		return
	}

	shared.Success(c, http.StatusCreated, gin.H{
		"user": user,
	})
}

func (h *AuthHandler) RegisterFingerprintUser(c *gin.Context) {
	var req dto.RegisterFingerprintRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.BadRequest(c, "Données invalides")
		return
	}

	user := &models.User{
		FirstName:       req.FirstName,
		LastName:        req.LastName,
		Email:           req.Email,
		FingerprintHash: stringPtr(req.FingerprintHash),
		RoleID:          req.RoleID,
		StationID:       req.StationID,
		IsActive:        true,
	}

	if err := h.userRepo.CreateFingerprintUser(user); err != nil {
		shared.InternalError(c, "Impossible de créer l'utilisateur")
		return
	}

	createdUser, err := h.userRepo.FindByID(user.ID)
	if err != nil {
		shared.InternalError(c, "Impossible de récupérer l'utilisateur créé")
		return
	}

	shared.Success(c, http.StatusCreated, gin.H{
		"user": createdUser,
	})
}

func (h *AuthHandler) LoginWithFingerprint(c *gin.Context) {
	var req dto.FingerprintLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.BadRequest(c, "Données invalides")
		return
	}

	user, err := h.userRepo.FindByFingerprint(req.FingerprintHash)
	if err != nil {
		shared.Unauthorized(c, "Empreinte non reconnue")
		return
	}

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

func (h *AuthHandler) ListUsers(c *gin.Context) {
	users, err := h.userRepo.FindAll()
	if err != nil {
		shared.InternalError(c, "Impossible de récupérer les utilisateurs")
		return
	}

	shared.Success(c, http.StatusOK, gin.H{
		"users": users,
	})
}

func (h *AuthHandler) ListStations(c *gin.Context) {
	stations, err := h.stationRepo.FindAll()
	if err != nil {
		shared.InternalError(c, "Impossible de récupérer les stations")
		return
	}

	shared.Success(c, http.StatusOK, gin.H{
		"stations": stations,
	})
}

// CheckEmail répond juste "ce compte existe-t-il, a-t-il déjà un mot de passe ?", pour
// l'étape email de la page de connexion (avant authentification). Ne renvoie PAS le
// reste du dossier utilisateur : contrairement à GET /auth/users (désormais réservé aux
// rôles admin), cette route reste publique mais expose le minimum nécessaire.
func (h *AuthHandler) CheckEmail(c *gin.Context) {
	var req dto.CheckEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.BadRequest(c, "Email invalide")
		return
	}

	user, err := h.userRepo.FindByEmail(req.Email)
	if err != nil {
		shared.Success(c, http.StatusOK, gin.H{"exists": false})
		return
	}

	shared.Success(c, http.StatusOK, gin.H{"exists": true, "has_password": user.HasPassword})
}

func (h *AuthHandler) CreateUser(c *gin.Context) {
	var req dto.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.BadRequest(c, "Données invalides: "+err.Error())
		return
	}

	firstName := req.FirstName
	lastName := req.LastName
	user := &models.User{
		FirstName: firstName,
		LastName:  lastName,
		Email:     req.Email,
		Phone:     stringPtr(req.Phone),
		Gender:    stringPtr(req.Gender),
		RoleID:    req.RoleID,
		StationID: req.StationID,
		IsActive:  true,
	}

	if req.Password != "" {
		hash, err := services.HashPassword(req.Password)
		if err != nil {
			shared.InternalError(c, "Impossible de sécuriser le mot de passe")
			return
		}
		user.PasswordHash = &hash
	}

	if err := h.userRepo.Create(user); err != nil {
		shared.InternalError(c, "Impossible de créer l'utilisateur")
		return
	}

	createdUser, err := h.userRepo.FindByID(user.ID)
	if err != nil {
		shared.InternalError(c, "Impossible de récupérer l'utilisateur créé")
		return
	}

	shared.Success(c, http.StatusCreated, gin.H{"user": createdUser})
}

func (h *AuthHandler) LoginWithPassword(c *gin.Context) {
	var req dto.PasswordLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.BadRequest(c, "Données invalides: "+err.Error())
		return
	}

	user, err := h.userRepo.FindByEmail(req.Email)
	if err != nil {
		shared.Unauthorized(c, "Identifiants incorrects")
		return
	}

	if user.PasswordHash == nil || !services.CheckPassword(*user.PasswordHash, req.Password) {
		shared.Unauthorized(c, "Identifiants incorrects")
		return
	}

	if !user.IsActive {
		shared.Unauthorized(c, "Compte désactivé")
		return
	}

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

func (h *AuthHandler) Logout(c *gin.Context) {
	rawUserID, ok := c.Get("user_id")
	if !ok {
		shared.Unauthorized(c, "Non authentifié")
		return
	}

	var userID int64
	switch id := rawUserID.(type) {
	case int64:
		userID = id
	case int:
		userID = int64(id)
	case string:
		parsed, err := strconv.ParseInt(id, 10, 64)
		if err != nil {
			shared.Unauthorized(c, "Identifiant utilisateur invalide")
			return
		}
		userID = parsed
	default:
		shared.Unauthorized(c, "Identifiant utilisateur invalide")
		return
	}

	if err := h.userRepo.SetActive(userID, false); err != nil {
		shared.InternalError(c, "Impossible de désactiver l'utilisateur")
		return
	}

	shared.Success(c, http.StatusOK, gin.H{
		"message": "Déconnexion réussie",
	})
}

// SetInitialPassword : première connexion d'un compte créé sans mot de passe
// (voir CreateUser). Email + mot de passe suffisent, sans jeton supplémentaire.
func (h *AuthHandler) SetInitialPassword(c *gin.Context) {
	var req dto.SetInitialPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.BadRequest(c, "Données invalides: "+err.Error())
		return
	}

	user, err := h.userRepo.FindByEmail(req.Email)
	if err != nil {
		shared.Unauthorized(c, "Compte introuvable")
		return
	}

	if user.PasswordHash != nil {
		shared.BadRequest(c, "Un mot de passe est déjà défini pour ce compte")
		return
	}

	hash, err := services.HashPassword(req.Password)
	if err != nil {
		shared.InternalError(c, "Impossible de sécuriser le mot de passe")
		return
	}

	if err := h.userRepo.SetPassword(user.ID, hash); err != nil {
		shared.InternalError(c, "Impossible d'enregistrer le mot de passe")
		return
	}
	user.HasPassword = true

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

func (h *AuthHandler) GetMe(c *gin.Context) {
	rawUserID, ok := c.Get("user_id")
	if !ok {
		shared.Unauthorized(c, "Non authentifié")
		return
	}

	var userID int64
	switch id := rawUserID.(type) {
	case int64:
		userID = id
	case int:
		userID = int64(id)
	case string:
		parsed, err := strconv.ParseInt(id, 10, 64)
		if err != nil {
			shared.Unauthorized(c, "Identifiant utilisateur invalide")
			return
		}
		userID = parsed
	default:
		shared.Unauthorized(c, "Identifiant utilisateur invalide")
		return
	}

	user, err := h.userRepo.FindByID(userID)
	if err != nil {
		shared.InternalError(c, "Utilisateur introuvable")
		return
	}

	shared.Success(c, http.StatusOK, gin.H{
		"user": user,
	})
}
