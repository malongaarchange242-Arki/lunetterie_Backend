package dto

// CreateDeliveryRequest représente la requête envoyée depuis le présentoir/laboratoire
type CreateDeliveryRequest struct {
	StationID int64    `json:"station_id" binding:"required"`
	Barcodes  []string `json:"barcodes" binding:"required"`
}
