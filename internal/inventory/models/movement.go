package models

import "time"

// MovementListItem représente un mouvement enrichi (monture, postes, emplacements, utilisateur)
// pour l'affichage dans l'historique/traffic des montures.
type MovementListItem struct {
	ID        int64   `db:"id" json:"id"`
	GlassID   int64   `db:"glass_id" json:"glass_id"`
	Barcode   string  `db:"barcode" json:"barcode"`
	Reference *string `db:"reference" json:"reference,omitempty"`
	Brand     *string `db:"brand" json:"brand,omitempty"`
	// Attributs et photos de la monture : l'historique du poste de scan affiche des
	// vignettes et filtre par forme/genre, il lui faut autre chose que la référence.
	Gender           *string   `db:"gender" json:"gender,omitempty"`
	Shape            *string   `db:"shape" json:"shape,omitempty"`
	Color            *string   `db:"color" json:"color,omitempty"`
	Size             *string   `db:"size" json:"size,omitempty"`
	Price            *float64  `db:"price" json:"price,omitempty"`
	PhotoMontureURL  *string   `db:"photo_monture_url" json:"photo_monture_url,omitempty"`
	PhotoBrancheURL  *string   `db:"photo_branche_url" json:"photo_branche_url,omitempty"`
	FromStationID    *int64    `db:"from_station_id" json:"from_station_id,omitempty"`
	FromStationName  *string   `db:"from_station_name" json:"from_station_name,omitempty"`
	ToStationID      *int64    `db:"to_station_id" json:"to_station_id,omitempty"`
	ToStationName    *string   `db:"to_station_name" json:"to_station_name,omitempty"`
	FromLocationCode *string   `db:"from_location_code" json:"from_location_code,omitempty"`
	ToLocationCode   *string   `db:"to_location_code" json:"to_location_code,omitempty"`
	Action           string    `db:"action" json:"action"`
	UserFirstName    *string   `db:"user_first_name" json:"user_first_name,omitempty"`
	UserLastName     *string   `db:"user_last_name" json:"user_last_name,omitempty"`
	Notes            *string   `db:"notes" json:"notes,omitempty"`
	CreatedAt        time.Time `db:"created_at" json:"created_at"`
}

// MovementFilters décrit les filtres optionnels de consultation de l'historique des mouvements.
type MovementFilters struct {
	FromStationID *int64
	ToStationID   *int64
	Action        *string
	Barcode       *string
	UserID        *int64
	DateFrom      *string
	DateTo        *string
	Limit         int
	Offset        int
}
