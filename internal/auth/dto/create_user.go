package dto

type CreateUserRequest struct {
	FirstName string `json:"first_name" binding:"required"`
	LastName  string `json:"last_name" binding:"required"`
    Email     string `json:"email" binding:"omitempty,email"`
	RoleID    int64  `json:"role_id" binding:"required"`
	StationID *int64 `json:"station_id"`
}
