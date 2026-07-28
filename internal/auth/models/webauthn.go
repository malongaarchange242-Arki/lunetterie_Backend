package models

import "time"

type WebAuthnCredential struct {
	ID           int64     `json:"id" db:"id"`
	UserID       int64     `json:"user_id" db:"user_id"`
	CredentialID string    `json:"credential_id" db:"credential_id"`
	PublicKey    []byte    `json:"public_key" db:"public_key"`
	Algorithm    int       `json:"algorithm" db:"algorithm"`
	AAGUID       string    `json:"aaguid" db:"aaguid"`
	SignCount    int       `json:"sign_count" db:"sign_count"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

type WebAuthnChallenge struct {
	ID        int64     `json:"id" db:"id"`
	UserID    int64     `json:"user_id" db:"user_id"`
	Challenge string    `json:"challenge" db:"challenge"`
	ExpiresAt time.Time `json:"expires_at" db:"expires_at"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}
