package dto

// SendToCaisseRequest représente la requête envoyée depuis le poste Présentoir quand le
// vendeur pousse les montures sélectionnées vers le comptoir d'encaissement.
type SendToCaisseRequest struct {
	StationID int64    `json:"station_id" binding:"required"`
	Barcodes  []string `json:"barcodes" binding:"required"`
}
