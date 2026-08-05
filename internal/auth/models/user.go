package models

import "time"

type User struct {
	ID                     int64      `db:"id" json:"id"`
	FirstName              string     `db:"first_name" json:"first_name"`
	LastName               string     `db:"last_name" json:"last_name"`
	Email                  string     `db:"email" json:"email"`
	PasswordHashDeprecated *string    `db:"password_hash_deprecated" json:"-"`
	PasswordHash           *string    `db:"password_hash" json:"-"`
	HasPassword            bool       `db:"has_password" json:"has_password"`
	FingerprintHash        *string    `db:"fingerprint_hash" json:"fingerprint_hash,omitempty"`
	WebAuthnRegistered     bool       `db:"webauthn_registered" json:"webauthn_registered"`
	Gender                 *string    `db:"gender" json:"gender,omitempty"`
	Phone                  *string    `db:"phone" json:"phone,omitempty"`
	RoleID                 int64      `db:"role_id" json:"role_id"`
	RoleName               string     `db:"role_name" json:"role_name,omitempty"`
	StationID              *int64     `db:"station_id" json:"station_id,omitempty"`
	StationName            *string    `db:"station_name" json:"station_name,omitempty"`
	IsActive               bool       `db:"is_active" json:"is_active"`
	LastLogin              *time.Time `db:"last_login" json:"last_login,omitempty"`
	CreatedAt              time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt              time.Time  `db:"updated_at" json:"updated_at"`
}
