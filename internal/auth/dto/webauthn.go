package dto

// RegisterChallengeRequest - Requête pour obtenir un challenge d'enregistrement
type RegisterChallengeRequest struct {
	Email     string `json:"email" binding:"required,email"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
}

// RegisterChallengeResponse - Réponse du challenge d'enregistrement
type RegisterChallengeResponse struct {
	Challenge string `json:"challenge"`
	UserID    string `json:"userId"`
}

// RegisterVerifyRequest - Requête de vérification d'enregistrement
type RegisterVerifyRequest struct {
	Email    string                     `json:"email" binding:"required"`
	ID       string                     `json:"id" binding:"required"`
	RawID    string                     `json:"rawId" binding:"required"`
	Type     string                     `json:"type" binding:"required"`
	Response RegisterVerifyResponseData `json:"response" binding:"required"`
}

type RegisterVerifyResponseData struct {
	ClientDataJSON    string `json:"clientDataJSON" binding:"required"`
	AttestationObject string `json:"attestationObject" binding:"required"`
}

// LoginChallengeRequest - Requête de challenge de connexion
type LoginChallengeRequest struct {
	Email string `json:"email" binding:"required,email"`
}

// LoginChallengeResponse - Réponse du challenge de connexion
type LoginChallengeResponse struct {
	Challenge     string   `json:"challenge"`
	CredentialIDs []string `json:"credentialIds"`
}

// LoginVerifyRequest - Requête de vérification de connexion
type LoginVerifyRequest struct {
	Email    string                  `json:"email" binding:"required"`
	ID       string                  `json:"id" binding:"required"`
	RawID    string                  `json:"rawId" binding:"required"`
	Type     string                  `json:"type" binding:"required"`
	Response LoginVerifyResponseData `json:"response" binding:"required"`
}

type LoginVerifyResponseData struct {
	ClientDataJSON    string `json:"clientDataJSON" binding:"required"`
	AuthenticatorData string `json:"authenticatorData" binding:"required"`
	Signature         string `json:"signature" binding:"required"`
	UserHandle        string `json:"userHandle"`
}

// RegisterUserRequest - Requête d'inscription finale
type RegisterUserRequest struct {
	FirstName    string `json:"first_name" binding:"required"`
	LastName     string `json:"last_name" binding:"required"`
	Email        string `json:"email" binding:"required,email"`
	Phone        string `json:"phone"`
	Gender       string `json:"gender"`
	RoleID       int64  `json:"role_id" binding:"required"`
	StationID    *int64 `json:"station_id"`
	CredentialID string `json:"credential_id" binding:"required"`
}

// DiscoverableLoginChallengeResponse - Réponse du challenge de connexion sans email
// (empreinte seule, credential "discoverable"/résidente)
type DiscoverableLoginChallengeResponse struct {
	Challenge string `json:"challenge"`
}

// DiscoverableLoginVerifyRequest - Requête de vérification de connexion sans email
type DiscoverableLoginVerifyRequest struct {
	ID       string                  `json:"id" binding:"required"`
	RawID    string                  `json:"rawId" binding:"required"`
	Type     string                  `json:"type" binding:"required"`
	Response LoginVerifyResponseData `json:"response" binding:"required"`
}

// EnrollChallengeRequest - Requête de challenge pour ajouter une empreinte à un utilisateur déjà existant
type EnrollChallengeRequest struct {
	UserID int64 `json:"user_id" binding:"required"`
}

// EnrollChallengeResponse - Réponse du challenge d'ajout d'empreinte
type EnrollChallengeResponse struct {
	Challenge string `json:"challenge"`
	UserID    string `json:"userId"`
}

// EnrollVerifyRequest - Requête de vérification pour lier une empreinte à un utilisateur existant
type EnrollVerifyRequest struct {
	UserID   int64                      `json:"user_id" binding:"required"`
	ID       string                     `json:"id" binding:"required"`
	RawID    string                     `json:"rawId" binding:"required"`
	Type     string                     `json:"type" binding:"required"`
	Response RegisterVerifyResponseData `json:"response" binding:"required"`
}
