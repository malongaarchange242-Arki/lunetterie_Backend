package models

import "time"

type PreRegistrationPhoto struct {
	ID   string `db:"id" json:"id,omitempty"`
	Kind string `db:"kind" json:"kind,omitempty"`
	Src  string `db:"src" json:"src,omitempty"`
	URL  string `db:"url" json:"url,omitempty"`
	Name string `db:"name" json:"name,omitempty"`
}

type PreRegistrationCase struct {
	ID                 int64                `db:"id" json:"id"`
	ReceptionCommandID int64                `db:"reception_command_id" json:"reception_command_id"`
	Code               string               `db:"code" json:"code"`
	Couleur            string               `db:"couleur" json:"couleur"`
	Hex                string               `db:"hex" json:"hex,omitempty"`
	Gamme              string               `db:"gamme" json:"gamme"`
	Genre              string               `db:"genre" json:"genre"`
	Montures           int                  `db:"montures" json:"montures"`
	Validated          bool                 `db:"validated" json:"validated"`
	ShipmentScanned    bool                 `db:"shipment_scanned" json:"shipment_scanned"`
	ShipmentScannedAt  *time.Time           `db:"shipment_scanned_at" json:"shipment_scanned_at,omitempty"`
	Cartons            []PreRegistrationBox `db:"-" json:"cartons"`
	CreatedAt          time.Time            `db:"created_at" json:"created_at"`
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
	ID        int64                  `db:"id" json:"id"`
	CaseID    int64                  `db:"case_id" json:"case_id"`
	Code      string                 `db:"code" json:"code"`
	Quantity  int                    `db:"quantity" json:"qty"`
	Formes    map[string]int         `db:"formes" json:"formes"`
	Marques   []string               `db:"marques" json:"marques"`
	Couleurs  []string               `db:"couleurs" json:"couleurs"`
	Matieres  []string               `db:"matieres" json:"matieres"`
	Photos    []PreRegistrationPhoto `db:"photos" json:"photos"`
	Gamme     string                 `db:"gamme" json:"gamme"`
	Type      string                 `db:"type_lunette" json:"type"`
	Prix      float64                `db:"prix" json:"prix"`
	CreatedAt time.Time              `db:"created_at" json:"created_at"`
	UpdatedAt time.Time              `db:"updated_at" json:"updated_at"`
}

type PreRegistrationCatalogueBox struct {
	ID           int64                           `json:"id"`
	CaseID       int64                           `json:"case_id"`
	Code         string                          `json:"code"`
	Quantity     int                             `json:"quantity"`
	Formes       map[string]int                  `json:"formes"`
	Marques      []string                        `json:"marques"`
	Couleurs     []string                        `json:"couleurs"`
	Matieres     []string                        `json:"matieres"`
	Photos       []PreRegistrationPhoto          `json:"photos"`
	Gamme        string                          `json:"gamme"`
	Type         string                          `json:"type"`
	Prix         float64                         `json:"prix"`
	CreatedAt    time.Time                       `json:"created_at"`
	UpdatedAt    time.Time                       `json:"updated_at"`
	CaseCode     string                          `json:"valise_code"`
	CaseCouleur  string                          `json:"valise_couleur"`
	CaseGenre    string                          `json:"valise_genre"`
	CaseMontures int                             `json:"valise_montures"`
	Montures     []PreRegistrationCatalogueGlass `json:"montures"`
}

type PreRegistrationCatalogueGlass struct {
	ID        int64  `json:"id"`
	Reference string `json:"reference"`
	Barcode   string `json:"barcode"`
	Marque    string `json:"marque"`
	Couleur   string `json:"couleur"`
	Forme     string `json:"forme"`
	Matiere   string `json:"matiere"`
}

type PreRegistrationCatalogueFilters struct {
	Query   string
	Gamme   string
	Genre   string
	Type    string
	Marque  string
	Couleur string
}

type PreRegistrationBoxRequest struct {
	Code        string                 `json:"code"`
	Quantity    int                    `json:"qty"`
	QuantityAlt int                    `json:"quantity"`
	Formes      map[string]int         `json:"formes" binding:"required"`
	Marques     []string               `json:"marques"`
	Couleurs    []string               `json:"couleurs"`
	Matieres    []string               `json:"matieres"`
	Photos      []PreRegistrationPhoto `json:"photos"`
	Gamme       string                 `json:"gamme" binding:"required"`
	Type        string                 `json:"type" binding:"required"`
	Prix        float64                `json:"prix"`
}
