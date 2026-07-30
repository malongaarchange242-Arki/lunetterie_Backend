package models

import "time"

// Glass représente une monture physique exemplaire
type Glass struct {
	ID               int64       `db:"id" json:"id"`
	Barcode          string      `db:"barcode" json:"barcode"`
	SerialNumber     *string     `db:"serial_number" json:"serial_number,omitempty"`
	FrameModelID     *int64      `db:"frame_model_id" json:"frame_model_id,omitempty"`
	StationID        int64       `db:"station_id" json:"station_id"`
	LocationID       *int64      `db:"location_id" json:"location_id,omitempty"`
	SupplierID       *int64      `db:"supplier_id" json:"supplier_id,omitempty"`
	DeliveryID       *int64      `db:"delivery_id" json:"delivery_id,omitempty"`
	AnalysisID       *int64      `db:"analysis_id" json:"analysis_id,omitempty"`
	Status           GlassStatus `db:"status" json:"status"`
	IsReserved       bool        `db:"is_reserved" json:"is_reserved"`
	ReservedForOrder *int64      `db:"reserved_for_order" json:"reserved_for_order,omitempty"`
	Price            *float64    `db:"price" json:"price,omitempty"`
	PhotoMontureURL  *string     `db:"photo_monture_url" json:"photo_monture_url,omitempty"`
	PhotoBrancheURL  *string     `db:"photo_branche_url" json:"photo_branche_url,omitempty"`
	Notes            *string     `db:"notes" json:"notes,omitempty"`
	CreatedAt        time.Time   `db:"created_at" json:"created_at"`
	UpdatedAt        time.Time   `db:"updated_at" json:"updated_at"`
}

// GlassListItem représente une monture pour affichage en liste (jointure glasses + glass_analysis + storage_locations)
type GlassListItem struct {
	ID           int64    `db:"id" json:"id"`
	Barcode      string   `db:"barcode" json:"barcode"`
	StationID    int64    `db:"station_id" json:"station_id"`
	StationName  *string  `db:"station_name" json:"station_name,omitempty"`
	Status       string   `db:"status" json:"status"`
	Price        *float64 `db:"price" json:"price,omitempty"`
	Reference    *string  `db:"reference" json:"reference,omitempty"`
	Brand        *string  `db:"brand" json:"brand,omitempty"`
	Gender       *string  `db:"gender" json:"gender,omitempty"`
	Shape        *string  `db:"shape" json:"shape,omitempty"`
	Color        *string  `db:"color" json:"color,omitempty"`
	Size         *string  `db:"size" json:"size,omitempty"`
	Material     *string  `db:"material" json:"material,omitempty"`
	LocationCode *string  `db:"location_code" json:"location_code,omitempty"`
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
	Type             string    `db:"type" json:"type"`
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

// Delivery représente un arrivage
type Delivery struct {
	ID         int64     `db:"id" json:"id"`
	SupplierID int64     `db:"supplier_id" json:"supplier_id"`
	Reference  *string   `db:"reference" json:"reference,omitempty"`
	ReceivedBy *int64    `db:"received_by" json:"received_by,omitempty"`
	StationID  int64     `db:"station_id" json:"station_id"`
	Notes      *string   `db:"notes" json:"notes,omitempty"`
	ReceivedAt time.Time `db:"received_at" json:"received_at"`
	CreatedAt  time.Time `db:"created_at" json:"created_at"`
}
