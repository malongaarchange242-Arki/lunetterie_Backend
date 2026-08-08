package dto

import "regexp"

// pinPattern : le code de connexion fait exactement 4 ou 6 chiffres.
var pinPattern = regexp.MustCompile(`^(\d{4}|\d{6})$`)

// IsValidPIN valide le format du code. Centralisé ici pour que la connexion et la
// définition initiale appliquent exactement la même règle.
func IsValidPIN(pin string) bool {
	return pinPattern.MatchString(pin)
}

// PasswordLoginRequest : connexion par nom d'employé et code chiffré.
type PasswordLoginRequest struct {
	Name     string `json:"name" binding:"required"`
	Password string `json:"password" binding:"required"`
}
