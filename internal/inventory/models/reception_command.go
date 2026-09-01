package models

import "time"

type ReceptionCommand struct {
	ID                    int64      `db:"id" json:"id"`
	Code                  string     `db:"code" json:"code"`
	TargetCount           int        `db:"target_count" json:"target_count"`
	RegisteredCount       int        `db:"registered_count" json:"registered_count"`
	Status                string     `db:"status" json:"status"`
	PreRegistrationStatus string     `db:"pre_registration_status" json:"pre_registration_status"`
	ShipmentStatus        string     `db:"shipment_status" json:"shipment_status"`
	DispatchedAt          *time.Time `db:"dispatched_at" json:"dispatched_at,omitempty"`
	ArrivedAt             *time.Time `db:"arrived_at" json:"arrived_at,omitempty"`
	SupplierOrderID       *int64     `db:"supplier_order_id" json:"supplier_order_id,omitempty"`
	OrderGender           string     `db:"order_gender" json:"gender,omitempty"`
	OrderProvenance       string     `db:"order_provenance" json:"provenance,omitempty"`
	OrderGamme            string     `db:"order_gamme" json:"gamme,omitempty"`
	CreatedBy             *int64     `db:"created_by" json:"created_by,omitempty"`
	ActivatedAt           *time.Time `db:"activated_at" json:"activated_at,omitempty"`
	CreatedAt             time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt             time.Time  `db:"updated_at" json:"updated_at"`
}

type ReceptionCommandCreateRequest struct {
	TargetCount     int    `json:"target_count" binding:"required"`
	SupplierOrderID *int64 `json:"supplier_order_id"`
}

type ReceptionCommandResponse struct {
	Command ReceptionCommand `json:"command"`
}

type ShipmentData struct {
	Valises    []PreRegistrationCase `json:"valises"`
	Total      int                   `json:"total"`
	Date       *time.Time            `json:"date,omitempty"`
	ArriveDate *time.Time            `json:"arriveDate,omitempty"`
}

type ReceptionCommandWithShip struct {
	ID                    int64         `json:"id"`
	Code                  string        `json:"code"`
	TargetCount           int           `json:"target_count"`
	RegisteredCount       int           `json:"registered_count"`
	Status                string        `json:"status"`
	PreRegistrationStatus string        `json:"pre_registration_status"`
	ShipmentStatus        string        `json:"shipment_status"`
	DispatchedAt          *time.Time    `json:"dispatched_at,omitempty"`
	ArrivedAt             *time.Time    `json:"arrived_at,omitempty"`
	SupplierOrderID       *int64        `json:"supplier_order_id,omitempty"`
	OrderGender           string        `json:"gender,omitempty"`
	OrderProvenance       string        `json:"provenance,omitempty"`
	OrderGamme            string        `json:"gamme,omitempty"`
	CreatedBy             *int64        `json:"created_by,omitempty"`
	ActivatedAt           *time.Time    `json:"activated_at,omitempty"`
	CreatedAt             time.Time     `json:"created_at"`
	UpdatedAt             time.Time     `json:"updated_at"`
	Ship                  *ShipmentData `json:"ship,omitempty"`
}
