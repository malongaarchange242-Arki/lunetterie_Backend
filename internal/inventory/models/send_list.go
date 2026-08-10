package models

import "time"

// SendList est la liste des montures d'une session de réception, envoyée par la direction
// vers un magasin. Elle sert d'ordre de préparation au poste de scan du stock général.
type SendList struct {
	ID                     int64     `db:"id" json:"id"`
	SessionCode            string    `db:"session_code" json:"session_code"`
	City                   string    `db:"city" json:"city"`
	ItemCount              int       `db:"item_count" json:"item_count"`
	Status                 string    `db:"status" json:"status"`
	SentCount              int       `db:"sent_count" json:"sent_count"`
	DestinationStationName *string   `db:"destination_station_name" json:"destination_station_name,omitempty"`
	CreatedBy              *int64    `db:"created_by" json:"created_by,omitempty"`
	CreatedAt              time.Time `db:"created_at" json:"created_at"`
	UpdatedAt              time.Time `db:"updated_at" json:"updated_at"`
}

// SendListItem recopie les attributs de la monture au moment de l'envoi : la liste doit
// rester lisible même si la monture bouge ou est supprimée ensuite.
type SendListItem struct {
	ID           int64     `db:"id" json:"id"`
	ListID       int64     `db:"list_id" json:"list_id"`
	GlassID      *int64    `db:"glass_id" json:"glass_id,omitempty"`
	Barcode      *string   `db:"barcode" json:"barcode,omitempty"`
	Reference    *string   `db:"reference" json:"reference,omitempty"`
	Brand        *string   `db:"brand" json:"brand,omitempty"`
	LocationCode *string   `db:"location_code" json:"location_code,omitempty"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
}

type SendListItemRequest struct {
	GlassID      *int64 `json:"glass_id"`
	Barcode      string `json:"barcode"`
	Reference    string `json:"reference"`
	Brand        string `json:"brand"`
	LocationCode string `json:"location_code"`
}

type SendListCreateRequest struct {
	SessionCode string                `json:"session_code" binding:"required"`
	City        string                `json:"city" binding:"required"`
	Items       []SendListItemRequest `json:"items"`
}

// Statuts d'un carton, dans l'ordre de son voyage :
//
//	CREATED — parti du stock général, montures EN_TRANSIT, pas encore vu au magasin.
//	OPENED  — son étiquette a été scannée à l'arrivée : le pointage est en cours, et il peut
//	          être repris autant de fois qu'il le faut (page fermée, poste changé).
//	CLOSED  — le magasinier a clos le pointage. Les montures jamais scannées restent
//	          EN_TRANSIT : elles n'entrent pas au stock, et `missing_count` fige l'écart.
const (
	SendBoxStatusCreated = "CREATED"
	SendBoxStatusOpened  = "OPENED"
	SendBoxStatusClosed  = "CLOSED"
)

// SendBox represents a physical carton shipped for one full send-list dispatch.
type SendBox struct {
	ID          int64  `db:"id" json:"id"`
	ListID      int64  `db:"list_id" json:"list_id"`
	Code        string `db:"code" json:"code"`
	Reference   string `db:"reference" json:"reference"`
	City        string `db:"city" json:"city"`
	SessionCode string `db:"session_code" json:"session_code"`
	ItemCount   int    `db:"item_count" json:"item_count"`
	Status      string `db:"status" json:"status"`
	// Le transfert que ce carton transporte. C'est lui qui porte l'état de réception
	// monture par monture (transfer_items) : le carton n'en tient pas de copie.
	TransferID      *int64     `db:"transfer_id" json:"transfer_id,omitempty"`
	CreatedBy       *int64     `db:"created_by" json:"created_by,omitempty"`
	OpenedAt        *time.Time `db:"opened_at" json:"opened_at,omitempty"`
	OpenedBy        *int64     `db:"opened_by" json:"opened_by,omitempty"`
	OpenedStationID *int64     `db:"opened_station_id" json:"opened_station_id,omitempty"`
	ClosedAt        *time.Time `db:"closed_at" json:"closed_at,omitempty"`
	ClosedBy        *int64     `db:"closed_by" json:"closed_by,omitempty"`
	MissingCount    int        `db:"missing_count" json:"missing_count"`
	CreatedAt       time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt       time.Time  `db:"updated_at" json:"updated_at"`
}

// RestockAlertRatio : un magasin est signalé quand il ne lui reste plus que 10 % de sa
// dernière livraison. La référence est le dernier carton et non le cumul de tout ce qui a
// été livré — un cumul ne fait que grossir, le seuil deviendrait vite hors d'atteinte et la
// quantité à renvoyer correspondrait à tout ce qui a été vendu depuis l'ouverture.
const RestockAlertRatio = 0.10

// RestockSuggestion : ce qu'il faut renvoyer à un magasin pour le remettre au niveau de sa
// dernière livraison.
type RestockSuggestion struct {
	City         string    `db:"city" json:"city"`
	LastBoxQty   int       `db:"last_box_qty" json:"last_box_qty"`
	LastBoxAt    time.Time `db:"last_box_at" json:"last_box_at"`
	CurrentStock int       `db:"current_stock" json:"current_stock"`
	ToSend       int       `db:"-" json:"to_send"`
	Alert        bool      `db:"-" json:"alert"`
}

// SendBoxItem stores each send-list snapshot line that is attached to a box.
type SendBoxItem struct {
	ID         int64   `db:"id" json:"id"`
	BoxID      int64   `db:"box_id" json:"box_id"`
	ListItemID int64   `db:"list_item_id" json:"list_item_id"`
	GlassID    *int64  `db:"glass_id" json:"glass_id,omitempty"`
	Barcode    *string `db:"barcode" json:"barcode,omitempty"`
	Reference  *string `db:"reference" json:"reference,omitempty"`
	// D'où la monture est partie, figé à l'expédition. Sans intérêt pour le magasin qui la
	// reçoit — c'est une case du stock général.
	LocationCode *string `db:"location_code" json:"location_code,omitempty"`
	// Où la ranger ici : l'emplacement que la station d'arrivée lui a attribué au pointage.
	// Nul tant que la monture est en transit — l'expédition libère sa case de départ et ne
	// lui en donne pas d'autre avant d'être reçue. C'est la seule des deux que le
	// magasinier a besoin de lire, une monture en main.
	StockLocationCode *string   `db:"stock_location_code" json:"stock_location_code,omitempty"`
	CreatedAt         time.Time `db:"created_at" json:"created_at"`
}
