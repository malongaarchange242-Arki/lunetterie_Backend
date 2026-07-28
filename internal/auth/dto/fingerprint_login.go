package dto

type FingerprintLoginRequest struct {
	FingerprintHash string `json:"fingerprint_hash" binding:"required"`
	DeviceID        string `json:"device_id" binding:"required"`
}
