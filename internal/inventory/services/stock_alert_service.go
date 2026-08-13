package services

import (
	"github.com/lunetterie/backend/internal/inventory/models"
)

// stockAlertStore : ce que le service attend de la base. Une interface étroite plutôt que le
// dépôt concret, pour que la machine d'état se teste sans PostgreSQL — c'est elle qui porte
// la règle métier, pas les requêtes.
type stockAlertStore interface {
	StationReference(stationID int64, stockType models.StockType) (int, string, error)
	CurrentStock(stationID int64, stockType models.StockType) (int, error)
	Arm(stationID int64, stockType models.StockType) (bool, error)
	Disarm(stationID int64, stockType models.StockType) error
	Recipients(stationID int64, stockType models.StockType) ([]int64, error)
	CreateMany(notifications []*models.Notification) error
}

// StockAlertService prévient quand le stock d'un magasin tombe à 10 % de sa référence.
//
// La règle n'est pas « notifier tant que le stock est bas » mais « notifier quand il passe
// sous le seuil ». Sans cette distinction, chaque vente sous le seuil renotifierait :
// 10, 9, 8, 7 donneraient quatre alertes là où une seule a du sens. L'état vit en base
// (stock_alert_states), pas en mémoire : plusieurs instances de l'API doivent aboutir à la
// même décision, et un redémarrage ne doit pas réémettre une alerte déjà partie.
type StockAlertService struct {
	store stockAlertStore
}

func NewStockAlertService(store stockAlertStore) *StockAlertService {
	return &StockAlertService{store: store}
}

// CheckStation examine les deux stocks d'une station après une mutation.
//
// Les deux sont toujours examinés : une monture qui quitte le présentoir pour le stock local
// fait bouger les deux compteurs d'un coup, et n'en regarder qu'un manquerait un
// franchissement. Le premier échec n'interrompt pas le second contrôle — une alerte
// présentoir perdue ne doit pas emporter l'alerte stock local avec elle.
func (s *StockAlertService) CheckStation(stationID int64) error {
	if s == nil || s.store == nil || stationID <= 0 {
		return nil
	}

	var firstErr error
	for _, stockType := range []models.StockType{models.StockTypePresentoir, models.StockTypeLocal} {
		if err := s.checkOne(stationID, stockType); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// checkOne applique la machine d'état à un seul stock.
//
//	stock > seuil ──────────────► réarmement (aucune notification)
//	stock <= seuil ─┬─ alerte inactive ──► on vient de franchir : NOTIFIER
//	                └─ alerte active ────► déjà signalé : silence
func (s *StockAlertService) checkOne(stationID int64, stockType models.StockType) error {
	reference, stationName, err := s.store.StationReference(stationID, stockType)
	if err != nil {
		return err
	}
	// Pas de référence : la station n'a pas de normale connue, il n'y a pas de 10 % à
	// calculer. Alerter ici reviendrait à signaler en permanence un magasin qu'on n'a
	// jamais mesuré — c'est le même raisonnement que RestockSuggestions, qui laisse de côté
	// les villes jamais livrées.
	if reference <= 0 {
		return nil
	}

	threshold := models.StockAlertThreshold(reference)

	current, err := s.store.CurrentStock(stationID, stockType)
	if err != nil {
		return err
	}

	if current > threshold {
		return s.store.Disarm(stationID, stockType)
	}

	// Sous le seuil. Arm ne rend true qu'au franchissement : c'est lui, et non la condition
	// `current <= threshold`, qui décide s'il faut notifier.
	crossed, err := s.store.Arm(stationID, stockType)
	if err != nil {
		return err
	}
	if !crossed {
		return nil
	}

	recipients, err := s.store.Recipients(stationID, stockType)
	if err != nil {
		return err
	}
	if len(recipients) == 0 {
		return nil
	}

	message := models.StockAlertMessage(stockType, stationName, current, reference)
	notificationType := models.StockAlertNotificationType(stockType)
	rawStockType := string(stockType)

	notifications := make([]*models.Notification, 0, len(recipients))
	for _, userID := range recipients {
		// Les pointeurs sont pris sur des copies locales : les partager ferait écrire la
		// même adresse dans toutes les lignes.
		station, currentQty, referenceQty, thresholdQty := stationID, current, reference, threshold
		notifications = append(notifications, &models.Notification{
			UserID:         userID,
			Type:           notificationType,
			Message:        message,
			StationID:      &station,
			StockType:      &rawStockType,
			CurrentStock:   &currentQty,
			ReferenceStock: &referenceQty,
			Threshold:      &thresholdQty,
		})
	}

	return s.store.CreateMany(notifications)
}
