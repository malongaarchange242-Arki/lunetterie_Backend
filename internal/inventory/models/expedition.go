package models

type ExpeditionCreateRequest struct {
	Supplier  string `json:"supplier" binding:"required"`
	Quantity  int    `json:"quantity" binding:"required"`
	OrderDate string `json:"order_date" binding:"required"`
	Note      string `json:"note"`
	Country   string `json:"country"`
	City      string `json:"city"`
}
