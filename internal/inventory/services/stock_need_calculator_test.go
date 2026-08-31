package services

import "testing"

func TestDefaultStockThresholdConfigIncludesExpectedAxes(t *testing.T) {
	config := DefaultStockThresholdConfig()

	if got := config["ventesMoisBoutique"]; got == nil {
		t.Fatal("ventesMoisBoutique absente")
	}
	if got := config["coefGeneral"]; got == nil {
		t.Fatal("coefGeneral absente")
	}
	if got := config["genre"]; got == nil {
		t.Fatal("genre absente")
	}
	if got := config["type"]; got == nil {
		t.Fatal("type absente")
	}
	if got := config["gamme"]; got == nil {
		t.Fatal("gamme absente")
	}
}

func TestComputeStockNeedsProducesDeficitsForLowCounts(t *testing.T) {
	config := DefaultStockThresholdConfig()
	counts := map[string]map[string]int{
		"genre": {"Femme": 60},
		"type":  {"Vue": 200},
		"gamme": {"Moyenne gamme": 120},
	}

	needs := ComputeStockNeeds(config, counts)
	if len(needs) == 0 {
		t.Fatal("aucun besoin calculé")
	}

	found := false
	for _, item := range needs {
		if item.Category == "Genre" && item.Name == "Femme" {
			found = true
			if item.Quantity <= 0 {
				t.Fatalf("quantité de besoin invalide pour Femme: %d", item.Quantity)
			}
		}
	}
	if !found {
		t.Fatal("besoin Genre/Femme non trouvé")
	}
}
