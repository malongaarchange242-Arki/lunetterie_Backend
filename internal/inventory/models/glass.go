package models

import "time"

// Glass représente une monture physique exemplaire
type Glass struct {
	ID                 int64       `db:"id" json:"id"`
	Barcode            string      `db:"barcode" json:"barcode"`
	SerialNumber       *string     `db:"serial_number" json:"serial_number,omitempty"`
	FrameModelID       *int64      `db:"frame_model_id" json:"frame_model_id,omitempty"`
	StationID          int64       `db:"station_id" json:"station_id"`
	LocationID         *int64      `db:"location_id" json:"location_id,omitempty"`
	SupplierID         *int64      `db:"supplier_id" json:"supplier_id,omitempty"`
	DeliveryID         *int64      `db:"delivery_id" json:"delivery_id,omitempty"`
	AnalysisID         *int64      `db:"analysis_id" json:"analysis_id,omitempty"`
	Status             GlassStatus `db:"status" json:"status"`
	IsReserved         bool        `db:"is_reserved" json:"is_reserved"`
	ReservedForOrder   *int64      `db:"reserved_for_order" json:"reserved_for_order,omitempty"`
	Price              *float64    `db:"price" json:"price,omitempty"`
	PhotoMontureURL    *string     `db:"photo_monture_url" json:"photo_monture_url,omitempty"`
	PhotoBrancheURL    *string     `db:"photo_branche_url" json:"photo_branche_url,omitempty"`
	PhotoArriereURL    *string     `db:"photo_arriere_url" json:"photo_arriere_url,omitempty"`
	ReceptionCommandID *int64      `db:"reception_command_id" json:"reception_command_id,omitempty"`
	Notes              *string     `db:"notes" json:"notes,omitempty"`
	CreatedAt          time.Time   `db:"created_at" json:"created_at"`
	UpdatedAt          time.Time   `db:"updated_at" json:"updated_at"`
}

// GlassListItem représente une monture pour affichage en liste (jointure glasses + glass_analysis + storage_locations)
type GlassListItem struct {
	ID                 int64     `db:"id" json:"id"`
	Barcode            string    `db:"barcode" json:"barcode"`
	StationID          int64     `db:"station_id" json:"station_id"`
	StationName        *string   `db:"station_name" json:"station_name,omitempty"`
	Status             string    `db:"status" json:"status"`
	Price              *float64  `db:"price" json:"price,omitempty"`
	CreatedAt          time.Time `db:"created_at" json:"created_at"`
	Reference          *string   `db:"reference" json:"reference,omitempty"`
	Brand              *string   `db:"brand" json:"brand,omitempty"`
	Gender             *string   `db:"gender" json:"gender,omitempty"`
	Shape              *string   `db:"shape" json:"shape,omitempty"`
	Color              *string   `db:"color" json:"color,omitempty"`
	Size               *string   `db:"size" json:"size,omitempty"`
	Material           *string   `db:"material" json:"material,omitempty"`
	MountType          *string   `db:"mount_type" json:"mount_type,omitempty"`
	Gamme              *string   `db:"gamme" json:"gamme,omitempty"`
	LocationCode       *string   `db:"location_code" json:"location_code,omitempty"`
	ValiseCode         *string   `db:"valise_code" json:"valise_code,omitempty"`
	PhotoMontureURL    *string   `db:"photo_monture_url" json:"photo_monture_url,omitempty"`
	ReceptionCommandID *int64    `db:"reception_command_id" json:"reception_command_id,omitempty"`
	// Ville de la liste d'envoi qui réserve cette monture (statut RESERVEE_ENVOI), tant que
	// cette liste n'est pas TRAITEE. Nul pour tout autre statut.
	ReservedForCity *string `db:"reserved_for_city" json:"reserved_for_city,omitempty"`
	// La table glasses ne porte pas d'auteur : celui qui a enregistré la monture est
	// l'utilisateur de son mouvement RECEPTION_FOURNISSEUR.
	RegisteredBy *string `db:"registered_by" json:"registered_by,omitempty"`
}

type StockNeedItem struct {
	Category string `json:"category"`
	Name     string `json:"name"`
	Quantity int    `json:"quantity"`
}

type GlassAnalysisRepairCandidate struct {
	ID              int64   `db:"id" json:"id"`
	Barcode         string  `db:"barcode" json:"barcode"`
	PhotoMontureURL string  `db:"photo_monture_url" json:"photo_monture_url"`
	AnalysisID      *int64  `db:"analysis_id" json:"analysis_id,omitempty"`
	Reference       *string `db:"reference" json:"reference,omitempty"`
	Brand           *string `db:"brand" json:"brand,omitempty"`
	Shape           *string `db:"shape" json:"shape,omitempty"`
	Gender          *string `db:"gender" json:"gender,omitempty"`
}

// SimilarGlass représente une monture candidate au classement de similarité, avec le score
// composite (genre/forme/prix) et son détail par critère pour transparence côté UI.
type SimilarGlass struct {
	GlassListItem
	Score      float64 `json:"score"`
	ScoreGenre float64 `json:"score_genre"`
	ScoreForme float64 `json:"score_forme"`
	ScorePrix  float64 `json:"score_prix"`
}

// StockSummaryItem représente le stock actif (hors vendu/perdu/cassé/retourné) d'une référence,
// réparti entre Stock Général, Stock Local (station régionale) et Présentoir.
type StockSummaryItem struct {
	Reference     *string `db:"reference" json:"reference,omitempty"`
	Brand         *string `db:"brand" json:"brand,omitempty"`
	QtyGeneral    int     `db:"qty_general" json:"qty_general"`
	QtyLocal      int     `db:"qty_local" json:"qty_local"`
	QtyPresentoir int     `db:"qty_presentoir" json:"qty_presentoir"`
	QtyTotal      int     `db:"qty_total" json:"qty_total"`
	IsCritical    bool    `db:"is_critical" json:"is_critical"`
}

// Movement représente un mouvement de monture
type Movement struct {
	ID             int64          `db:"id" json:"id"`
	GlassID        int64          `db:"glass_id" json:"glass_id"`
	FromStationID  *int64         `db:"from_station_id" json:"from_station_id,omitempty"`
	ToStationID    *int64         `db:"to_station_id" json:"to_station_id,omitempty"`
	FromLocationID *int64         `db:"from_location_id" json:"from_location_id,omitempty"`
	ToLocationID   *int64         `db:"to_location_id" json:"to_location_id,omitempty"`
	Action         MovementAction `db:"action" json:"action"`
	UserID         int64          `db:"user_id" json:"user_id"`
	Notes          *string        `db:"notes" json:"notes,omitempty"`
	CreatedAt      time.Time      `db:"created_at" json:"created_at"`
}

// StorageLocation représente un emplacement de stockage
type StorageLocation struct {
	ID               int64     `db:"id" json:"id"`
	StationID        int64     `db:"station_id" json:"station_id"`
	ParentLocationID *int64    `db:"parent_location_id" json:"parent_location_id,omitempty"`
	Zone             ZoneType  `db:"zone" json:"zone"`
	Code             string    `db:"code" json:"code"`
	Name             string    `db:"name" json:"name"`
	Barcode          *string   `db:"barcode" json:"barcode,omitempty"`
	Type             string    `db:"type" json:"type"`
	Capacity         *int      `db:"capacity" json:"capacity,omitempty"`
	Status           string    `db:"status" json:"status"`
	CreatedAt        time.Time `db:"created_at" json:"created_at"`
}

// GlassAnalysis représente l'analyse IA d'une monture
type GlassAnalysis struct {
	ID                  int64     `db:"id" json:"id"`
	GlassID             int64     `db:"glass_id" json:"glass_id"`
	ModelVersion        string    `db:"model_version" json:"model_version"`
	Shape               *string   `db:"shape" json:"shape,omitempty"`
	ShapeConfidence     *float64  `db:"shape_confidence" json:"shape_confidence,omitempty"`
	Color               *string   `db:"color" json:"color,omitempty"`
	ColorConfidence     *float64  `db:"color_confidence" json:"color_confidence,omitempty"`
	Material            *string   `db:"material" json:"material,omitempty"`
	MaterialConfidence  *float64  `db:"material_confidence" json:"material_confidence,omitempty"`
	MountType           *string   `db:"mount_type" json:"mount_type,omitempty"`
	MountTypeConfidence *float64  `db:"mount_type_confidence" json:"mount_type_confidence,omitempty"`
	Gender              *string   `db:"gender" json:"gender,omitempty"`
	GenderConfidence    *float64  `db:"gender_confidence" json:"gender_confidence,omitempty"`
	Brand               *string   `db:"brand" json:"brand,omitempty"`
	BrandConfidence     *float64  `db:"brand_confidence" json:"brand_confidence,omitempty"`
	Reference           *string   `db:"reference" json:"reference,omitempty"`
	ReferenceConfidence *float64  `db:"reference_confidence" json:"reference_confidence,omitempty"`
	Size                *string   `db:"size" json:"size,omitempty"`
	CropImagePath       *string   `db:"crop_image_path" json:"crop_image_path,omitempty"`
	ProcessingTimeMs    *float64  `db:"processing_time_ms" json:"processing_time_ms,omitempty"`
	CreatedAt           time.Time `db:"created_at" json:"created_at"`
}

// Station représente une station de travail
type Station struct {
	ID        int64     `db:"id" json:"id"`
	Name      string    `db:"name" json:"name"`
	Type      string    `db:"type" json:"type"`
	City      *string   `db:"city" json:"city,omitempty"`
	Address   *string   `db:"address" json:"address,omitempty"`
	Phone     *string   `db:"phone" json:"phone,omitempty"`
	IsActive  bool      `db:"is_active" json:"is_active"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

// FrameModel représente un modèle de monture (référence catalogue)
type FrameModel struct {
	ID          int64     `db:"id" json:"id"`
	BrandID     *int64    `db:"brand_id" json:"brand_id,omitempty"`
	Reference   *string   `db:"reference" json:"reference,omitempty"`
	ShapeID     *int64    `db:"shape_id" json:"shape_id,omitempty"`
	ColorID     *int64    `db:"color_id" json:"color_id,omitempty"`
	MaterialID  *int64    `db:"material_id" json:"material_id,omitempty"`
	MountTypeID *int64    `db:"mount_type_id" json:"mount_type_id,omitempty"`
	Gender      *string   `db:"gender" json:"gender,omitempty"`
	Size        *string   `db:"size" json:"size,omitempty"`
	ImageURL    *string   `db:"image_url" json:"image_url,omitempty"`
	IsActive    bool      `db:"is_active" json:"is_active"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time `db:"updated_at" json:"updated_at"`
}

// Supplier représente un fournisseur
type Supplier struct {
	ID          int64     `db:"id" json:"id"`
	Name        string    `db:"name" json:"name"`
	Phone       *string   `db:"phone" json:"phone,omitempty"`
	Email       *string   `db:"email" json:"email,omitempty"`
	Address     *string   `db:"address" json:"address,omitempty"`
	ContactName *string   `db:"contact_name" json:"contact_name,omitempty"`
	IsActive    bool      `db:"is_active" json:"is_active"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time `db:"updated_at" json:"updated_at"`
}

// NOTE: Delivery model moved to models/delivery.go

// EmptySlot représente un emplacement présentoir devenu libre aujourd'hui, avec la monture qui
// l'occupait (vendue ou réservée) — pour savoir quoi remettre en place et où.
type EmptySlot struct {
	Code      string  `db:"code" json:"code"`
	Barcode   string  `db:"barcode" json:"barcode"`
	Reference *string `db:"reference" json:"reference,omitempty"`
	Brand     *string `db:"brand" json:"brand,omitempty"`
}
