package dto

type CreateStationRequest struct {
	City string `json:"city" binding:"required"`
}
