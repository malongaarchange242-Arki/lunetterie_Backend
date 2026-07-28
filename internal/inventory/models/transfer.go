package models

import "time"

// TransferStatus représente le statut d'un transfert inter-station
type TransferStatus string

const (
	TransferStatusPreparation TransferStatus = "PREPARATION"
	TransferStatusInTransit   TransferStatus = "IN_TRANSIT"
	TransferStatusReceived    TransferStatus = "RECEIVED"
	TransferStatusCancelled   TransferStatus = "CANCELLED"
)

// TransferItemStatus représente le statut d'une monture au sein d'un transfert
type TransferItemStatus string

const (
	TransferItemStatusPending   TransferItemStatus = "PENDING"
	TransferItemStatusInTransit TransferItemStatus = "IN_TRANSIT"
	TransferItemStatusReceived  TransferItemStatus = "RECEIVED"
	TransferItemStatusMissing   TransferItemStatus = "MISSING"
)

// Transfer représente un transfert de montures entre deux stations
type Transfer struct {
	ID            int64          `db:"id" json:"id"`
	FromStationID int64          `db:"from_station_id" json:"from_station_id"`
	ToStationID   int64          `db:"to_station_id" json:"to_station_id"`
	CreatedBy     int64          `db:"created_by" json:"created_by"`
	ReceivedBy    *int64         `db:"received_by" json:"received_by,omitempty"`
	Status        TransferStatus `db:"status" json:"status"`
	Notes         *string        `db:"notes" json:"notes,omitempty"`
	CreatedAt     time.Time      `db:"created_at" json:"created_at"`
	ReceivedAt    *time.Time     `db:"received_at" json:"received_at,omitempty"`
}

// TransferItem représente une monture incluse dans un transfert
type TransferItem struct {
	ID         int64              `db:"id" json:"id"`
	TransferID int64              `db:"transfer_id" json:"transfer_id"`
	GlassID    int64              `db:"glass_id" json:"glass_id"`
	Status     TransferItemStatus `db:"status" json:"status"`
	ScannedAt  *time.Time         `db:"scanned_at" json:"scanned_at,omitempty"`
}
