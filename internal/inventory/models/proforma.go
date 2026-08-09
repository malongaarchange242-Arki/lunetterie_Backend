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
	// Chargée seulement par Get : la Caisse liste des en-têtes, elle n'a que faire de
	// l'axe de l'œil gauche.
	Prescription *ProformaPrescription `db:"-" json:"prescription,omitempty"`
}

// ProformaPrescription : l'ordonnance et le détail de facturation, en colonnes.
//
// Le même contenu reste sérialisé dans Proforma.Note, qui n'a pas disparu : la Caisse
// réaffiche la note telle quelle, et elle sert de filet quand une valeur saisie à la main
// ne se convertit pas en nombre. Les pointeurs distinguent « non renseigné » de zéro —
// une sphère à 0 est une correction valide.
type ProformaPrescription struct {
	ProformaID int64   `db:"proforma_id" json:"proforma_id"`
	Societe    *string `db:"societe" json:"societe,omitempty"`
	Foyer      *string `db:"foyer" json:"foyer,omitempty"`
	Teinte     *string `db:"teinte" json:"teinte,omitempty"`

	ODSphere   *float64 `db:"od_sphere" json:"od_sphere,omitempty"`
	ODCylindre *float64 `db:"od_cylindre" json:"od_cylindre,omitempty"`
	ODAxe      *int     `db:"od_axe" json:"od_axe,omitempty"`
	ODAddition *float64 `db:"od_addition" json:"od_addition,omitempty"`
	ODPrix     float64  `db:"od_prix" json:"od_prix"`

	OGSphere   *float64 `db:"og_sphere" json:"og_sphere,omitempty"`
	OGCylindre *float64 `db:"og_cylindre" json:"og_cylindre,omitempty"`
	OGAxe      *int     `db:"og_axe" json:"og_axe,omitempty"`
	OGAddition *float64 `db:"og_addition" json:"og_addition,omitempty"`
	OGPrix     float64  `db:"og_prix" json:"og_prix"`

	AccessoiresLabel *string `db:"accessoires_label" json:"accessoires_label,omitempty"`
	AccessoiresPrix  float64 `db:"accessoires_prix" json:"accessoires_prix"`
	MontagePrix      float64 `db:"montage_prix" json:"montage_prix"`
	RemisePct        float64 `db:"remise_pct" json:"remise_pct"`
	NoteLibre        *string `db:"note_libre" json:"note_libre,omitempty"`

	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
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
	Shape      *string    `db:"shape" json:"shape,omitempty"`
	Color      *string    `db:"color" json:"color,omitempty"`
	UnitPrice  float64    `db:"unit_price" json:"unit_price"`
	// Monture offerte : dit explicitement pourquoi UnitPrice vaut 0, qu'on ne peut pas
	// distinguer autrement d'un prix simplement non renseigné sur la fiche.
	Offerte bool    `db:"offerte" json:"offerte"`
	Outcome *string `db:"outcome" json:"outcome,omitempty"`
	SettledAt  *time.Time `db:"settled_at" json:"settled_at,omitempty"`
	IsPending  bool       `db:"is_pending" json:"is_pending"`
	CreatedAt  time.Time  `db:"created_at" json:"created_at"`
}

// ProformaLineRequest : une monture de la proforma. Remplace l'ancien envoi d'un simple
// code-barres, qui ne laissait aucune place au geste « offerte » — le serveur relisait le
// prix de la fiche et facturait plein tarif.
type ProformaLineRequest struct {
	Barcode string `json:"barcode" binding:"required"`
	Offerte bool   `json:"offerte"`
}

// Les champs de l'ordonnance arrivent en chaînes : le formulaire laisse saisir « +1.00 »,
// « 60° » ou rien du tout. La conversion en nombre se fait à l'entrée du handler, qui pose
// NULL sur ce qu'il ne sait pas lire plutôt que de refuser toute la proforma.
type ProformaPrescriptionRequest struct {
	Societe string `json:"societe"`
	Foyer   string `json:"foyer"`
	Teinte  string `json:"teinte"`

	ODSphere   string  `json:"od_sphere"`
	ODCylindre string  `json:"od_cylindre"`
	ODAxe      string  `json:"od_axe"`
	ODAddition string  `json:"od_addition"`
	ODPrix     float64 `json:"od_prix"`

	OGSphere   string  `json:"og_sphere"`
	OGCylindre string  `json:"og_cylindre"`
	OGAxe      string  `json:"og_axe"`
	OGAddition string  `json:"og_addition"`
	OGPrix     float64 `json:"og_prix"`

	AccessoiresLabel string  `json:"accessoires_label"`
	AccessoiresPrix  float64 `json:"accessoires_prix"`
	MontagePrix      float64 `json:"montage_prix"`
	RemisePct        float64 `json:"remise_pct"`
	NoteLibre        string  `json:"note_libre"`
}

type ProformaCreateRequest struct {
	StationID   int64  `json:"station_id" binding:"required"`
	ClientName  string `json:"client_name" binding:"required"`
	ClientPhone string `json:"client_phone"`
	// L'ordonnance lisible, réaffichée telle quelle par la Caisse. Toujours envoyée et
	// toujours stockée, même depuis que Prescription porte les mêmes valeurs en colonnes.
	Note string `json:"note"`
	// Barcodes reste accepté pour les clients qui n'ont pas encore migré vers Lines :
	// la Caisse et le poste magasin appellent le même endpoint.
	Barcodes     []string                     `json:"barcodes"`
	Lines        []ProformaLineRequest        `json:"lines"`
	Prescription *ProformaPrescriptionRequest `json:"prescription"`
}

type ProformaItemDecision struct {
	ItemID  int64  `json:"item_id" binding:"required"`
	Outcome string `json:"outcome" binding:"required"`
}

type ProformaSettleRequest struct {
	StationID int64                  `json:"station_id" binding:"required"`
	Decisions []ProformaItemDecision `json:"decisions" binding:"required"`
}
