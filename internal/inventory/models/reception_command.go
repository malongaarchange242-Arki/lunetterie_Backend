package models

import "time"

type ReceptionCommand struct {
	ID              int64     `db:"id" json:"id"`
	Code            string    `db:"code" json:"code"`
	TargetCount     int       `db:"target_count" json:"target_count"`
	RegisteredCount int       `db:"registered_count" json:"registered_count"`
	Status          string    `db:"status" json:"status"`
	SupplierOrderID *int64    `db:"supplier_order_id" json:"supplier_order_id,omitempty"`
	CreatedBy       *int64    `db:"created_by" json:"created_by,omitempty"`
	CreatedAt       time.Time `db:"created_at" json:"created_at"`
	UpdatedAt       time.Time `db:"updated_at" json:"updated_at"`
}

type ReceptionCommandCreateRequest struct {
	TargetCount     int    `json:"target_count" binding:"required"`
	SupplierOrderID *int64 `json:"supplier_order_id"`
}

type ReceptionCommandResponse struct {
	Command ReceptionCommand `json:"command"`
}
