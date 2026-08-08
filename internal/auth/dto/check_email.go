package dto

// CheckUserRequest interroge l'existence d'un compte à partir du nom de l'employé, avant
// d'afficher le champ du code.
type CheckUserRequest struct {
	Name string `json:"name" binding:"required"`
}
