package models

import "time"

// SavFollowup est la couche que le poste SAV ajoute à une proforma : le client, sa
// commande et son paiement restent dans proformas, seule la relation client vit ici.
// Une proforma sans ligne de suivi est un client que le SAV n'a jamais touché.
type SavFollowup struct {
	ID           int64      `db:"id" json:"id"`
	ProformaID   int64      `db:"proforma_id" json:"proforma_id"`
	Called       bool       `db:"called" json:"called"`
	CalledAt     *time.Time `db:"called_at" json:"called_at,omitempty"`
	NoAnswer     bool       `db:"no_answer" json:"no_answer"`
	RelanceAt    *time.Time `db:"relance_at" json:"relance_at,omitempty"`
	Observations *string    `db:"observations" json:"observations,omitempty"`
	Message      *string    `db:"message" json:"message,omitempty"`
	UpdatedBy    *int64     `db:"updated_by" json:"updated_by,omitempty"`
	CreatedAt    time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time  `db:"updated_at" json:"updated_at"`
}

// SavFollowupSaveRequest : chaque champ est un pointeur pour distinguer « absent » de
// « vidé ». L'écran coche une case sans toucher aux observations, et efface une
// observation en envoyant une chaîne vide — les deux doivent rester possibles.
type SavFollowupSaveRequest struct {
	Called       *bool   `json:"called"`
	NoAnswer     *bool   `json:"no_answer"`
	RelanceAt    *string `json:"relance_at"`
	Observations *string `json:"observations"`
	Message      *string `json:"message"`
}
