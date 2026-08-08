package dto

type CreateUserRequest struct {
	FirstName string `json:"first_name" binding:"required"`
	LastName  string `json:"last_name" binding:"required"`
	Email     string `json:"email"`
	Password  string `json:"password"`
	Phone     string `json:"phone"`
	Gender    string `json:"gender"`
	City      string `json:"city"`
	RoleID    int64  `json:"role_id" binding:"required"`
	StationID *int64 `json:"station_id"`
}
