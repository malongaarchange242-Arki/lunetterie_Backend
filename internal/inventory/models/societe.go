package models

import "time"

// Societe : une entreprise conventionnée, dont les salariés se font équiper au compte de
// leur employeur. La liste est tenue par la Direction ; les postes magasin la consultent
// pour remplir le champ « Société » d'une proforma, sans pouvoir y écrire.
type Societe struct {
	ID      int64   `db:"id" json:"id"`
	Name    string  `db:"name" json:"name"`
	Contact *string `db:"contact" json:"contact,omitempty"`
	Phone   *string `db:"phone" json:"phone,omitempty"`
	// Une société désactivée sort de la liste proposée à la vendeuse mais reste liée aux
	// proformas qu'elle a déjà portées.
	Active    bool      `db:"active" json:"active"`
	CreatedBy *int64    `db:"created_by" json:"created_by,omitempty"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

type SocieteCreateRequest struct {
	Name    string `json:"name" binding:"required"`
	Contact string `json:"contact"`
	Phone   string `json:"phone"`
}

// SocieteUpdateRequest : tous les champs sont facultatifs, seuls les fournis sont appliqués.
// `Active` est un pointeur pour distinguer « passer à false » de « non transmis » — sans
// quoi une simple correction de numéro de téléphone désactiverait la société.
type SocieteUpdateRequest struct {
	Name    string `json:"name"`
	Contact string `json:"contact"`
	Phone   string `json:"phone"`
	Active  *bool  `json:"active"`
}
