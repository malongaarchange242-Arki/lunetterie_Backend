package dto

type CheckEmailRequest struct {
	Email string `json:"email" binding:"required,email"`
}
