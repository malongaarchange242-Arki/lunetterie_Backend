package models

import "time"

// Statuts de l'en-tête d'une proforma. Ils se déduisent des lignes : tant qu'une ligne
// attend l'arbitrage de la Caisse, la proforma reste EN_ATTENTE.
const (
	ProformaStatusEnAttente = "EN_ATTENTE"
	ProformaStatusReglee    = "REGLEE"
	ProformaStatusAnnulee   = "ANNULEE"
)

// Décisions possibles de la Caisse, ligne par ligne : un client peut garder une paire et
// renoncer à l'autre.
const (
	ProformaOutcomeVendue = "VENDUE"
	ProformaOutcomeRetour = "RETOUR_PRESENTOIR"
)

// Proforma : le document émis au Présentoir quand un client choisit des montures. Les
// montures qu'elle porte sont bloquées jusqu'à l'arbitrage de la Caisse.
type Proforma struct {
	ID          int64          `db:"id" json:"id"`
	Code        string         `db:"code" json:"code"`
	StationID   int64          `db:"station_id" json:"station_id"`
	ClientName  string         `db:"client_name" json:"client_name"`
	ClientPhone *string        `db:"client_phone" json:"client_phone,omitempty"`
	TotalAmount float64        `db:"total_amount" json:"total_amount"`
	Status      string         `db:"status" json:"status"`
	Note        *string        `db:"note" json:"note,omitempty"`
	CreatedBy   *int64         `db:"created_by" json:"created_by,omitempty"`
	SettledBy   *int64         `db:"settled_by" json:"settled_by,omitempty"`
	SettledAt   *time.Time     `db:"settled_at" json:"settled_at,omitempty"`
	CreatedAt   time.Time      `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time      `db:"updated_at" json:"updated_at"`
	Items       []ProformaItem `db:"-" json:"items,omitempty"`
}

// ProformaItem recopie les attributs de la monture au moment de l'émission : le document
// doit rester lisible même si la monture est vendue, déplacée ou supprimée ensuite.
type ProformaItem struct {
	ID         int64      `db:"id" json:"id"`
	ProformaID int64      `db:"proforma_id" json:"proforma_id"`
	GlassID    *int64     `db:"glass_id" json:"glass_id,omitempty"`
	Barcode    *string    `db:"barcode" json:"barcode,omitempty"`
	Reference  *string    `db:"reference" json:"reference,omitempty"`
	Brand      *string    `db:"brand" json:"brand,omitempty"`
	UnitPrice  float64    `db:"unit_price" json:"unit_price"`
	Outcome    *string    `db:"outcome" json:"outcome,omitempty"`
	SettledAt  *time.Time `db:"settled_at" json:"settled_at,omitempty"`
	IsPending  bool       `db:"is_pending" json:"is_pending"`
	CreatedAt  time.Time  `db:"created_at" json:"created_at"`
}

type ProformaCreateRequest struct {
	StationID   int64    `json:"station_id" binding:"required"`
	ClientName  string   `json:"client_name" binding:"required"`
	ClientPhone string   `json:"client_phone"`
	Note        string   `json:"note"`
	Barcodes    []string `json:"barcodes" binding:"required"`
}

type ProformaItemDecision struct {
	ItemID  int64  `json:"item_id" binding:"required"`
	Outcome string `json:"outcome" binding:"required"`
}

type ProformaSettleRequest struct {
	StationID int64                  `json:"station_id" binding:"required"`
	Decisions []ProformaItemDecision `json:"decisions" binding:"required"`
}
