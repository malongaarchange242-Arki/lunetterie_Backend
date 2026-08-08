package dto

// SetInitialPasswordRequest : première connexion d'un compte créé sans code.
// Le format est vérifié par IsValidPIN dans le handler plutôt que par un `min=` : le
// message doit dire « 4 ou 6 chiffres », pas « trop court ».
type SetInitialPasswordRequest struct {
	Name     string `json:"name" binding:"required"`
	Password string `json:"password" binding:"required"`
}
