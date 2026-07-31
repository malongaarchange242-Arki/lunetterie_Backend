package dto

type CreateSaleRequest struct {
	StationID int64    `json:"station_id" binding:"required"`
	Barcodes  []string `json:"barcodes" binding:"required"`
}

type CreateReserveRequest struct {
	StationID int64    `json:"station_id" binding:"required"`
	Barcodes  []string `json:"barcodes" binding:"required"`
}
