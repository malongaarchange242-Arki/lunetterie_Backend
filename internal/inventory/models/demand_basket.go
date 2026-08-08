package models

import "time"

// DemandBasketItem est une ligne de demande déposée dans le panier d'un magasin : ce qu'un
// client a cherché via le chatbot pour cette ville. Le panier lui-même n'est pas une entité,
// c'est le regroupement des lignes d'une même ville.
//
// Les quatre critères sont facultatifs : le chatbot ne les obtient pas tous à chaque
// recherche (un client peut demander « une ovale pour femme » sans préciser la taille).
type DemandBasketItem struct {
	ID        int64     `db:"id" json:"id"`
	City      string    `db:"city" json:"city"`
	Genre     *string   `db:"genre" json:"genre,omitempty"`
	Forme     *string   `db:"forme" json:"forme,omitempty"`
	Gamme     *string   `db:"gamme" json:"gamme,omitempty"`
	Taille    *string   `db:"taille" json:"taille,omitempty"`
	Source    string    `db:"source" json:"source"`
	Status    string    `db:"status" json:"status"`
	CreatedBy *int64    `db:"created_by" json:"created_by,omitempty"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

// DemandBasketCount est le compteur affiché sur le panier d'une ville.
type DemandBasketCount struct {
	City  string `db:"city" json:"city"`
	Count int    `db:"count" json:"count"`
}

type DemandBasketCreateRequest struct {
	City   string `json:"city" binding:"required"`
	Genre  string `json:"genre"`
	Forme  string `json:"forme"`
	Gamme  string `json:"gamme"`
	Taille string `json:"taille"`
	Source string `json:"source"`
}

type DemandBasketMarkSentRequest struct {
	IDs []int64 `json:"ids" binding:"required"`
}
