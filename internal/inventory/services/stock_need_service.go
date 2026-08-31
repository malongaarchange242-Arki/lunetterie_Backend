package services

import (
	"sort"

	"github.com/lunetterie/backend/internal/inventory/models"
)

// DefaultStockThresholdConfig renvoie la configuration locale de seuils utilisée par
// le poste Stock Général pour calculer les manques à commander.
func DefaultStockThresholdConfig() map[string]interface{} {
	return map[string]interface{}{
		"ventesMoisBoutique": 400,
		"boutiques":          5,
		"coefGeneral":        6.5,
		"genre": map[string]int{
			"Femme": 240,
			"Homme": 210,
			"Mixte": 90,
			"Enfant": 60,
		},
		"type": map[string]int{
			"Vue":      330,
			"Solaire":  180,
			"Lecture":  60,
			"Sécurité": 30,
		},
		"gamme": map[string]int{
			"Moyenne gamme": 270,
			"Classique":     210,
			"Luxe":          72,
			"Enfant":        48,
		},
		"formesMini": 10,
	}
}

// ComputeStockNeeds compare le stock actuel à un seuil théorique, puis renvoie les
// écarts positifs à commander par catégorie et sous-catégorie.
func ComputeStockNeeds(config map[string]interface{}, counts map[string]map[string]int) []models.StockNeedItem {
	axes := []struct {
		category string
		key      string
	}{
		{category: "Genre", key: "genre"},
		{category: "Type", key: "type"},
		{category: "Gamme", key: "gamme"},
	}

	items := make([]models.StockNeedItem, 0)
	for _, axis := range axes {
		thresholds, ok := config[axis.key].(map[string]int)
		if !ok {
			continue
		}
		axisCounts := counts[axis.key]
		if axisCounts == nil {
			axisCounts = map[string]int{}
		}
		for name, threshold := range thresholds {
			if threshold <= 0 {
				continue
			}
			qty := threshold - axisCounts[name]
			if qty > 0 {
				items = append(items, models.StockNeedItem{
					Category: axis.category,
					Name:     name,
					Quantity: qty,
				})
			}
		}
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].Quantity == items[j].Quantity {
			if items[i].Category == items[j].Category {
				return items[i].Name < items[j].Name
			}
			return items[i].Category < items[j].Category
		}
		return items[i].Quantity > items[j].Quantity
	})
	return items
}
