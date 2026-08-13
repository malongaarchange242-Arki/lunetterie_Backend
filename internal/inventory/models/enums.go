package models

// GlassStatus représente le statut d'une monture
type GlassStatus string

const (
	StatusRecuFournisseur    GlassStatus = "RECU_FOURNISSEUR"
	StatusEnStockGeneral     GlassStatus = "EN_STOCK_GENERAL"
	StatusEnTransit          GlassStatus = "EN_TRANSIT"
	StatusEnStockSousStation GlassStatus = "EN_STOCK_SOUS_STATION"
	StatusEnPresentoir       GlassStatus = "EN_PRESENTOIR"
	StatusEnCaisse           GlassStatus = "EN_CAISSE"
	StatusReservee           GlassStatus = "RESERVEE"
	StatusEnLaboratoire      GlassStatus = "EN_LABORATOIRE"
	StatusPreteALivrer       GlassStatus = "PRETE_A_LIVRER"
	// Remise au client : le dernier état du cycle. La monture a quitté le magasin, elle ne
	// doit plus figurer dans aucun compte de stock.
	StatusLivree    GlassStatus = "LIVREE"
	StatusVendue    GlassStatus = "VENDUE"
	StatusPerdue    GlassStatus = "PERDUE"
	StatusCassee    GlassStatus = "CASSEE"
	StatusRetournee GlassStatus = "RETOURNEE"
)

// MovementAction représente le type d'action de mouvement
type MovementAction string

const (
	ActionReceptionFournisseur MovementAction = "RECEPTION_FOURNISSEUR"
	ActionRangement            MovementAction = "RANGEMENT"
	ActionExpedition           MovementAction = "EXPEDITION"
	ActionReceptionStation     MovementAction = "RECEPTION_STATION"
	ActionMiseEnPresentoir     MovementAction = "PRESENTOIR"
	ActionMiseEnCaisse         MovementAction = "MISE_EN_CAISSE"
	ActionRetraitPresentoir    MovementAction = "RETRAIT_PRESENTOIR"
	ActionReservation          MovementAction = "RESERVATION"
	ActionEnvoiLaboratoire     MovementAction = "LABORATOIRE"
	ActionControleQualite      MovementAction = "CONTROLE_QUALITE"
	// LIVRAISON est écrite par le laboratoire quand il termine un montage, pas quand le
	// client repart : c'est REMISE_CLIENT qui marque la sortie du magasin.
	ActionLivraison    MovementAction = "LIVRAISON"
	ActionRemiseClient MovementAction = "REMISE_CLIENT"
	ActionRetour       MovementAction = "RETOUR"
	ActionInventaire   MovementAction = "INVENTAIRE"
	ActionPerte        MovementAction = "PERTE"
	ActionCasse        MovementAction = "CASSE"
)

// ZoneType représente le type de zone de stockage
type ZoneType string

const (
	ZoneStock       ZoneType = "STOCK"
	ZonePresentoir  ZoneType = "PRESENTOIR"
	ZoneLaboratoire ZoneType = "LABORATOIRE"
)
