package dto

import "strings"

// ReceptionRequest représente la requête de réception d'une monture
type ReceptionRequest struct {
	SupplierID           *int64   `json:"supplier_id" form:"supplier_id"`
	DeliveryRef          *string  `json:"delivery_ref" form:"delivery_ref"`
	StationID            int64    `json:"station_id" form:"station_id" binding:"required"`
	Notes                *string  `json:"notes" form:"notes"`
	Price                *float64 `json:"price" form:"price"`
	Gamme                *string  `json:"gamme" form:"gamme"`
	ReceptionCommandID   *int64   `json:"reception_command_id" form:"reception_command_id"`
	ReceptionCommandCode *string  `json:"reception_command_code" form:"reception_command_code"`
	PreRegistrationBoxID *int64   `json:"pre_registration_box_id" form:"pre_registration_box_id"`
	CartonCode           *string  `json:"carton_code" form:"carton_code"`
	// Code-barres réservé d'avance par GET /inventory/barcodes/next, pour que l'aperçu de
	// l'étiquette montre le vrai numéro au lieu d'un numéro de convenance. Facultatif : sans
	// lui le workflow en génère un, comme il l'a toujours fait.
	Barcode *string `json:"barcode" form:"barcode"`
	// Champs saisis/corrigés manuellement (étape de vérification) : prioritaires sur l'analyse IA.
	Reference *string `json:"reference" form:"reference"`
	Brand     *string `json:"brand" form:"brand"`
	Marque    *string `json:"marque" form:"marque"`
	Gender    *string `json:"gender" form:"gender"`
	Shape     *string `json:"shape" form:"shape"`
	// Forme telle que détectée par l'IA (avant correction éventuelle par l'utilisateur) :
	// sert uniquement à journaliser les corrections, jamais persistée telle quelle.
	DetectedShape *string `json:"detected_shape" form:"detected_shape"`
	Color         *string `json:"color" form:"color"`
	Size          *string `json:"size" form:"size"`
	Material      *string `json:"material" form:"material"`
	MountType     *string `json:"mount_type" form:"mount_type"`
}

// EffectiveBrand retourne la marque effective à enregistrer, en priorisant le champ brand
// puis le fallback marque pour rester compatible avec les anciens clients.
func (r *ReceptionRequest) EffectiveBrand() *string {
	if r.Brand != nil && strings.TrimSpace(*r.Brand) != "" {
		return r.Brand
	}
	if r.Marque != nil && strings.TrimSpace(*r.Marque) != "" {
		return r.Marque
	}
	return nil
}

// ReceptionResponse représente la réponse après réception
type ReceptionResponse struct {
	GlassID            int64           `json:"glass_id"`
	Barcode            string          `json:"barcode"`
	ReceptionCommandID *int64          `json:"reception_command_id,omitempty"`
	Status             string          `json:"status"`
	Location           string          `json:"location"`
	LocationCode       string          `json:"location_code"`
	Price              *float64        `json:"price,omitempty"`
	Analysis           *AnalysisResult `json:"analysis"`
	Movement           *MovementInfo   `json:"movement"`
}

// AnalysisResult représente les résultats d'analyse IA
type AnalysisResult struct {
	Shape        string  `json:"shape"`
	ShapeConf    float64 `json:"shape_confidence"`
	Color        string  `json:"color"`
	ColorConf    float64 `json:"color_confidence"`
	Material     string  `json:"material"`
	MatConf      float64 `json:"material_confidence"`
	MountType    string  `json:"mount_type"`
	MountConf    float64 `json:"mount_type_confidence"`
	Gender       string  `json:"gender,omitempty"`
	Size         string  `json:"size,omitempty"`
	Brand        string  `json:"brand,omitempty"`
	Reference    string  `json:"reference,omitempty"`
	ProcessingMs float64 `json:"processing_time_ms"`
}

// BranchAnalysisResult représente les résultats d'extraction de référence depuis la branche
type BranchAnalysisResult struct {
	Reference string `json:"reference,omitempty"`
}

// MovementInfo représente les infos du mouvement créé
type MovementInfo struct {
	ID        int64  `json:"id"`
	Action    string `json:"action"`
	FromLoc   string `json:"from_location,omitempty"`
	ToLoc     string `json:"to_location,omitempty"`
	Timestamp string `json:"timestamp"`
}
