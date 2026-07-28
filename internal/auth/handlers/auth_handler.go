package handlers

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/lunetterie/backend/internal/auth/dto"
	"github.com/lunetterie/backend/internal/auth/models"
	"github.com/lunetterie/backend/internal/auth/repositories"
	"github.com/lunetterie/backend/internal/auth/services"
	"github.com/lunetterie/backend/internal/shared"
)

type AuthHandler struct {
	userRepo        *repositories.UserRepository
	authService     *services.AuthService
	webauthnService *services.WebAuthnService
}

func stringPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func NewAuthHandler(userRepo *repositories.UserRepository, authService *services.AuthService, webauthnService *services.WebAuthnService) *AuthHandler {
	return &AuthHandler{userRepo: userRepo, authService: authService, webauthnService: webauthnService}
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

	shared.Success(c, http.StatusCreated, gin.H{
		"user": user,
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

	if err := h.userRepo.Create(user); err != nil {
		shared.InternalError(c, "Impossible de créer l'utilisateur")
		return
	}

	shared.Success(c, http.StatusCreated, gin.H{
		"user": user,
	})
}
