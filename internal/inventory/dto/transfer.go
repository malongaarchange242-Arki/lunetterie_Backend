package dto

// CreateTransferRequest représente la création d'un transfert entre deux stations
type CreateTransferRequest struct {
	FromStationID int64   `json:"from_station_id" binding:"required"`
	ToStationID   int64   `json:"to_station_id" binding:"required"`
	Notes         *string `json:"notes"`
}

// TransferItemRequest représente le scan d'une monture (ajout au transfert ou réception)
type TransferItemRequest struct {
	Barcode string `json:"barcode" binding:"required"`
}

// TransferItemResponse représente une monture au sein d'un transfert
type TransferItemResponse struct {
	ID        int64   `json:"id"`
	GlassID   int64   `json:"glass_id"`
	Barcode   string  `json:"barcode"`
	Status    string  `json:"status"`
	ScannedAt *string `json:"scanned_at,omitempty"`
}

// TransferResponse représente un transfert
type TransferResponse struct {
	ID            int64                  `json:"id"`
	FromStationID int64                  `json:"from_station_id"`
	ToStationID   int64                  `json:"to_station_id"`
	Status        string                 `json:"status"`
	CreatedBy     int64                  `json:"created_by"`
	ReceivedBy    *int64                 `json:"received_by,omitempty"`
	Notes         *string                `json:"notes,omitempty"`
	CreatedAt     string                 `json:"created_at"`
	ReceivedAt    *string                `json:"received_at,omitempty"`
	Items         []TransferItemResponse `json:"items,omitempty"`
}

// ReceiveTransferItemResponse représente le résultat de la réception d'une monture
type ReceiveTransferItemResponse struct {
	TransferItem   TransferItemResponse `json:"transfer_item"`
	NewLocation    string               `json:"new_location"`
	GlassStatus    string               `json:"glass_status"`
	TransferStatus string               `json:"transfer_status"`
}
