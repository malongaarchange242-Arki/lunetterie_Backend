package models

import "time"

type PreRegistrationCase struct {
	ID                 int64                `db:"id" json:"id"`
	ReceptionCommandID int64                `db:"reception_command_id" json:"reception_command_id"`
	Code               string               `db:"code" json:"code"`
	Couleur            string               `db:"couleur" json:"couleur"`
	Hex                string               `db:"hex" json:"hex,omitempty"`
	Gamme              string               `db:"gamme" json:"gamme"`
	Genre              string               `db:"genre" json:"genre"`
	Montures              int                  `db:"montures" json:"montures"`
	Validated             bool                 `db:"validated" json:"validated"`
	ShipmentScanned       bool                 `db:"shipment_scanned" json:"shipment_scanned"`
	ShipmentScannedAt     *time.Time           `db:"shipment_scanned_at" json:"shipment_scanned_at,omitempty"`
	Cartons               []PreRegistrationBox `db:"-" json:"cartons"`
	CreatedAt             time.Time            `db:"created_at" json:"created_at"`
	UpdatedAt          time.Time            `db:"updated_at" json:"updated_at"`
}

type PreRegistrationCaseRequest struct {
	Couleur  string `json:"couleur" binding:"required"`
	Hex      string `json:"hex"`
	Gamme    string `json:"gamme" binding:"required"`
	Genre    string `json:"genre" binding:"required"`
	Montures int    `json:"montures" binding:"required"`
}

type PreRegistrationBox struct {
	ID        int64          `db:"id" json:"id"`
	CaseID    int64          `db:"case_id" json:"case_id"`
	Code      string         `db:"code" json:"code"`
	Quantity  int            `db:"quantity" json:"qty"`
	Formes    map[string]int `db:"formes" json:"formes"`
	Marques   []string       `db:"marques" json:"marques"`
	Couleurs  []string       `db:"couleurs" json:"couleurs"`
	Matieres  []string       `db:"matieres" json:"matieres"`
	Gamme     string         `db:"gamme" json:"gamme"`
	Type      string         `db:"type_lunette" json:"type"`
	Prix      float64        `db:"prix" json:"prix"`
	CreatedAt time.Time      `db:"created_at" json:"created_at"`
	UpdatedAt time.Time      `db:"updated_at" json:"updated_at"`
}

type PreRegistrationBoxRequest struct {
	Code     string         `json:"code"`
	Quantity int            `json:"qty" binding:"required"`
	Formes   map[string]int `json:"formes" binding:"required"`
	Marques  []string       `json:"marques"`
	Couleurs []string       `json:"couleurs"`
	Matieres []string       `json:"matieres"`
	Gamme    string         `json:"gamme" binding:"required"`
	Type     string         `json:"type" binding:"required"`
	Prix     float64        `json:"prix"`
}
