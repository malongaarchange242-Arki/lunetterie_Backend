package dto

// StationTemplate est conservé pour compatibilité avec les anciens clients.
type StationTemplate struct {
	Name             string `json:"name,omitempty"`
	NumRayons        int    `json:"num_rayons" binding:"required"`
	EtageresParRayon int    `json:"etageres_par_rayon" binding:"required"`
	BacsParEtagere   int    `json:"bacs_par_etagere" binding:"required"`
	PositionsParBac  int    `json:"positions_par_bac" binding:"required"`
}

// GenerateLocationsRequest représente la requête de génération d'emplacements
type GenerateLocationsRequest struct {
	StationID int64           `json:"station_id" binding:"required"`
	Template  StationTemplate `json:"template" binding:"required"`
}

type CreateStorageLocationRequest struct {
	StationID        int64  `json:"station_id" binding:"required"`
	ParentLocationID *int64 `json:"parent_location_id"`
	Type             string `json:"type" binding:"required"`
	Capacity         *int   `json:"capacity"`
}

// FindFreeLocationRequest représente la requête de recherche d'un emplacement libre
type FindFreeLocationRequest struct {
	StationID int64    `json:"station_id" binding:"required"`
	Zone      string   `json:"zone,omitempty"`
	Price     *float64 `json:"price,omitempty"`
	Gamme     string   `json:"gamme,omitempty"`
}
