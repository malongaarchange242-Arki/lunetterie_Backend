package dto

type RegisterFingerprintRequest struct {
	FirstName       string `json:"first_name" binding:"required"`
	LastName        string `json:"last_name" binding:"required"`
	Email           string `json:"email" binding:"required,email"`
	FingerprintHash string `json:"fingerprint_hash" binding:"required"`
	RoleID          int64  `json:"role_id" binding:"required"`
	StationID       *int64 `json:"station_id" binding:"omitempty"`
}
