package models

import (
	"fmt"
	"math"
	"time"
)

// StockType : les deux stocks d'un magasin qui savent alerter. Ils se comptent sur la même
// station et se distinguent par le statut de la monture — une mise en présentoir change le
// station_id de la monture pour celui du magasin (display_service.go
// UpdateStationAndLocation), le présentoir n'est donc pas un compteur à part.
type StockType string

const (
	StockTypePresentoir StockType = "PRESENTOIR"
	StockTypeLocal      StockType = "LOCAL"
)

// GlassStatus correspondant à chaque type de stock.
func (t StockType) GlassStatus() GlassStatus {
	if t == StockTypePresentoir {
		return StatusEnPresentoir
	}
	return StatusEnStockSousStation
}

func (t StockType) Label() string {
	if t == StockTypePresentoir {
		return "présentoir"
	}
	return "local"
}

// TracksStock dit si un statut entre dans l'un des deux stocks surveillés. Sert à écarter
// d'emblée les mutations qui ne peuvent rien changer aux compteurs — une réception
// fournisseur ou un passage au laboratoire ne touche ni le présentoir ni le stock local, et
// n'a donc aucun seuil à faire franchir.
func TracksStock(status GlassStatus) bool {
	return status == StatusEnPresentoir || status == StatusEnStockSousStation
}

// StockAlertRatio : un stock est signalé quand il tombe à 10 % de sa quantité de référence.
//
// À ne pas confondre avec RestockAlertRatio (send_list.go), qui vaut la même chose mais
// répond à une autre question : là-bas la référence est le dernier carton reçu par une
// ville, ici c'est la quantité normale d'une station. Les deux seuils coexistent tant que
// le réapprovisionnement n'est pas migré sur ce mécanisme.
const StockAlertRatio = 0.10

// StockAlertThreshold : le seuil en nombre entier de montures.
//
// L'arrondi va vers le haut, et c'est un choix : une référence de 37 donne 3,7, et arrondir
// vers le bas laisserait un magasin à 3 montures sans alerte alors qu'il est sous les 10 %.
// Comparer directement à la valeur décimale reviendrait au même que l'arrondi bas, une
// quantité de montures étant toujours entière.
func StockAlertThreshold(reference int) int {
	if reference <= 0 {
		return 0
	}
	return int(math.Ceil(float64(reference) * StockAlertRatio))
}

// Les types de notification. Un seul usage aujourd'hui, mais la colonne `type` existe pour
// que la table serve aussi aux notifications suivantes sans nouvelle migration.
const (
	NotificationTypeStockPresentoirBas = "STOCK_PRESENTOIR_BAS"
	NotificationTypeStockLocalBas      = "STOCK_LOCAL_BAS"
)

func StockAlertNotificationType(stockType StockType) string {
	if stockType == StockTypePresentoir {
		return NotificationTypeStockPresentoirBas
	}
	return NotificationTypeStockLocalBas
}

// Notification destinée à un utilisateur précis. Une ligne par destinataire : c'est ce qui
// permet à chacun de marquer la sienne comme lue sans toucher celle des autres.
type Notification struct {
	ID             int64      `db:"id" json:"id"`
	UserID         int64      `db:"user_id" json:"user_id"`
	Type           string     `db:"type" json:"type"`
	Message        string     `db:"message" json:"message"`
	StationID      *int64     `db:"station_id" json:"station_id,omitempty"`
	StockType      *string    `db:"stock_type" json:"stock_type,omitempty"`
	CurrentStock   *int       `db:"current_stock" json:"current_stock,omitempty"`
	ReferenceStock *int       `db:"reference_stock" json:"reference_stock,omitempty"`
	Threshold      *int       `db:"threshold" json:"threshold,omitempty"`
	ReadAt         *time.Time `db:"read_at" json:"read_at,omitempty"`
	CreatedAt      time.Time  `db:"created_at" json:"created_at"`
}

// StockAlertMessage : la phrase lue par le destinataire. Elle porte les trois chiffres qui
// ont motivé l'alerte, pour qu'elle reste compréhensible relue plus tard, quand le stock
// aura bougé.
func StockAlertMessage(stockType StockType, stationName string, current, reference int) string {
	where := "Stock présentoir"
	if stockType == StockTypeLocal {
		where = "Stock local"
	}
	unit := "montures restantes"
	if current <= 1 {
		unit = "monture restante"
	}
	if stationName == "" {
		return fmt.Sprintf("%s faible : %d %s sur une référence de %d.", where, current, unit, reference)
	}
	return fmt.Sprintf("%s faible à %s : %d %s sur une référence de %d.", where, stationName, current, unit, reference)
}
