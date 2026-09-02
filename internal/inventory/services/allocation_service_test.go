package services

import (
	"testing"

	"github.com/lunetterie/backend/internal/inventory/models"
	"github.com/lunetterie/backend/internal/shared"
)

func TestCreateStockCartonAutomatically(t *testing.T) {
	db := shared.GetTestDB()
	if db == nil {
		t.Skip("Pas de DB de test disponible")
	}
	defer db.Close()

	// Setup: créer une station de test
	var stationID int64
	err := db.Get(&stationID, `
		INSERT INTO stations (name, type, is_active)
		VALUES ('Stock Général Test', 'STOCK', true)
		RETURNING id
	`)
	if err != nil {
		t.Fatalf("Impossible de créer station: %v", err)
	}

	t.Logf("✅ Station créée: %d", stationID)

	// Setup: vider tous les cartons pour forcer la création
	_, err = db.Exec(`
		DELETE FROM storage_locations
		WHERE station_id = $1 AND zone = 'STOCK'
	`, stationID)
	if err != nil {
		t.Fatalf("Impossible de vider les cartons: %v", err)
	}

	// Test: FindOrCreateStockLocation doit créer auto une VALISE et un CARTON
	service := NewAllocationService(db)

	location, err := service.FindOrCreateStockLocation(stationID)
	if err != nil {
		t.Fatalf("FindOrCreateStockLocation échoué: %v", err)
	}

	if location == nil {
		t.Fatal("Emplacement retourné est nil")
	}

	t.Logf("✅ Emplacement créé: %s (id=%d, type=%s, parent_id=%v)", location.Code, location.ID, location.Type, location.ParentLocationID)

	// Vérifications
	if location.Type != "CARTON" {
		t.Errorf("Type attendu: CARTON, obtenu: %s", location.Type)
	}

	if location.Capacity == nil || *location.Capacity != 50 {
		t.Errorf("Capacité attendue: 50, obtenue: %v", location.Capacity)
	}

	if location.ParentLocationID == nil {
		t.Fatal("ParentLocationID doit être renseigné (VALISE)")
	}

	// Vérifier que la VALISE existe
	var valise models.StorageLocation
	err = db.Get(&valise, `
		SELECT id, station_id, zone, code, name, type, capacity, status, parent_location_id, created_at
		FROM storage_locations
		WHERE id = $1
	`, location.ParentLocationID)

	if err != nil {
		t.Fatalf("Impossible de récupérer la VALISE: %v", err)
	}

	t.Logf("✅ VALISE trouvée: %s (id=%d, type=%s)", valise.Code, valise.ID, valise.Type)

	if valise.Type != "VALISE" {
		t.Errorf("Type VALISE attendu, obtenu: %s", valise.Type)
	}

	if valise.ParentLocationID != nil {
		t.Errorf("ParentLocationID de VALISE doit être nil, obtenu: %v", valise.ParentLocationID)
	}

	// Test: appel suivant doit réutiliser le carton créé (si pas plein)
	location2, err := service.FindOrCreateStockLocation(stationID)
	if err != nil {
		t.Fatalf("FindOrCreateStockLocation (2ème appel) échoué: %v", err)
	}

	if location2.ID != location.ID {
		t.Errorf("Attendu réutilisation du carton %d, obtenu nouveau carton %d", location.ID, location2.ID)
	}

	t.Log("✅ Test réussi: création automatique VALISE → CARTON")
}

func TestFillCartonThenCreateNew(t *testing.T) {
	db := shared.GetTestDB()
	if db == nil {
		t.Skip("Pas de DB de test disponible")
	}
	defer db.Close()

	// Setup: créer une station
	var stationID int64
	err := db.Get(&stationID, `
		INSERT INTO stations (name, type, is_active)
		VALUES ('Stock Général Capacité Test', 'STOCK', true)
		RETURNING id
	`)
	if err != nil {
		t.Fatalf("Impossible de créer station: %v", err)
	}

	t.Logf("✅ Station créée: %d", stationID)

	// Vider la station
	_, err = db.Exec(`DELETE FROM storage_locations WHERE station_id = $1 AND zone = 'STOCK'`, stationID)
	if err != nil {
		t.Fatalf("Impossible de vider: %v", err)
	}

	service := NewAllocationService(db)

	// Créer le premier carton
	loc1, err := service.FindOrCreateStockLocation(stationID)
	if err != nil {
		t.Fatalf("Création 1er carton échouée: %v", err)
	}

	t.Logf("✅ 1er Carton: %s (id=%d)", loc1.Code, loc1.ID)

	// Simuler : remplir le carton avec 50 montures
	for i := 1; i <= 50; i++ {
		barcode := "LB-TEST-" + string(rune('A'+int64(i%26)))
		_, err := db.Exec(`
			INSERT INTO glasses (barcode, station_id, location_id, status)
			VALUES ($1, $2, $3, 'EN_STOCK_GENERAL')
		`, barcode, stationID, loc1.ID)
		if err != nil {
			t.Fatalf("Impossible d'insérer verre %d: %v", i, err)
		}
	}

	t.Logf("✅ Carton rempli avec 50 montures")

	// Créer une 2ème location → doit créer un nouveau carton
	loc2, err := service.FindOrCreateStockLocation(stationID)
	if err != nil {
		t.Fatalf("Création 2e location échouée: %v", err)
	}

	if loc2.ID == loc1.ID {
		t.Errorf("Attendu un nouveau carton, obtenu le même (%d)", loc1.ID)
	}

	t.Logf("✅ 2e Carton créé automatiquement: %s (id=%d)", loc2.Code, loc2.ID)

	// Les deux cartons doivent avoir la même VALISE comme parent
	if loc2.ParentLocationID == nil || loc1.ParentLocationID == nil {
		t.Fatal("Les deux cartons doivent avoir une VALISE parente")
	}

	if *loc1.ParentLocationID != *loc2.ParentLocationID {
		t.Errorf("Les deux cartons doivent partager la même VALISE, obtenu %d et %d", *loc1.ParentLocationID, *loc2.ParentLocationID)
	}

	t.Logf("✅ Les deux cartons sont dans la même VALISE: %d", *loc1.ParentLocationID)
}
