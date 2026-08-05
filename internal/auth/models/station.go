package models

import "time"

type Station struct {
	ID        int64     `db:"id" json:"id"`
	Name      string    `db:"name" json:"name"`
	Type      string    `db:"type" json:"type"`
	City      *string   `db:"city" json:"city,omitempty"`
	Address   *string   `db:"address" json:"address,omitempty"`
	Phone     *string   `db:"phone" json:"phone,omitempty"`
	IsActive  bool      `db:"is_active" json:"is_active"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}
