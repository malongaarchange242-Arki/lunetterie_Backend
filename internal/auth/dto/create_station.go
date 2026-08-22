package dto

type CreateStationRequest struct {
	Country string `json:"country" binding:"required"`
	City    string `json:"city" binding:"required"`
}
