package services

import (
	"strings"
	"sync"
	"testing"

	"github.com/lunetterie/backend/internal/inventory/models"
)

type alertKey struct {
	stationID int64
	stockType models.StockType
}

// fakeAlertStore rejoue le contrat de NotificationRepository en mémoire. Arm y reproduit la
// clause `WHERE active = false` du ON CONFLICT : c'est le point qui décide s'il faut
// notifier, le tester ailleurs ne testerait rien.
type fakeAlertStore struct {
	mu          sync.Mutex
	reference   map[alertKey]int
	current     map[alertKey]int
	active      map[alertKey]bool
	recipients  map[alertKey][]int64
	stationName string

	sent     []*models.Notification
	armCalls int
}

func newFakeAlertStore() *fakeAlertStore {
	return &fakeAlertStore{
		reference:   map[alertKey]int{},
		current:     map[alertKey]int{},
		active:      map[alertKey]bool{},
		recipients:  map[alertKey][]int64{},
		stationName: "Station Pointe-Noire",
	}
}

func (f *fakeAlertStore) StationReference(stationID int64, stockType models.StockType) (int, string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.reference[alertKey{stationID, stockType}], f.stationName, nil
}

func (f *fakeAlertStore) CurrentStock(stationID int64, stockType models.StockType) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.current[alertKey{stationID, stockType}], nil
}

func (f *fakeAlertStore) Arm(stationID int64, stockType models.StockType) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.armCalls++
	key := alertKey{stationID, stockType}
	if f.active[key] {
		return false, nil
	}
	f.active[key] = true
	return true, nil
}

func (f *fakeAlertStore) Disarm(stationID int64, stockType models.StockType) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.active[alertKey{stationID, stockType}] = false
	return nil
}

func (f *fakeAlertStore) Recipients(stationID int64, stockType models.StockType) ([]int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.recipients[alertKey{stationID, stockType}], nil
}

func (f *fakeAlertStore) CreateMany(notifications []*models.Notification) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sent = append(f.sent, notifications...)
	return nil
}

func (f *fakeAlertStore) sentCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.sent)
}

// setStock place le stock courant sans repasser par une mutation de monture.
func (f *fakeAlertStore) setStock(stationID int64, stockType models.StockType, qty int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.current[alertKey{stationID, stockType}] = qty
}

// presentoirStore : une station 1 dont le présentoir vaut 100 de référence (seuil 10) et
// dont le responsable est l'utilisateur 7. Le stock local reste sans référence pour que les
// tests du présentoir n'aient qu'une alerte possible.
func presentoirStore() *fakeAlertStore {
	store := newFakeAlertStore()
	store.reference[alertKey{1, models.StockTypePresentoir}] = 100
	store.recipients[alertKey{1, models.StockTypePresentoir}] = []int64{7}
	return store
}

// ── 1 à 3 : la condition de seuil ──────────────────────────────────────────────

func TestStockAlertAboveThresholdStaysSilent(t *testing.T) {
	store := presentoirStore()
	svc := NewStockAlertService(store)

	for _, qty := range []int{100, 50, 20, 11} {
		store.setStock(1, models.StockTypePresentoir, qty)
		if err := svc.CheckStation(1); err != nil {
			t.Fatalf("stock %d: %v", qty, err)
		}
		if store.sentCount() != 0 {
			t.Fatalf("stock %d au-dessus du seuil de 10 : %d notification(s) émise(s)", qty, store.sentCount())
		}
	}
}

func TestStockAlertExactlyAtThresholdNotifies(t *testing.T) {
	store := presentoirStore()
	svc := NewStockAlertService(store)

	// Le seuil est atteint, pas dépassé : la comparaison est `<=`, 10 doit alerter.
	store.setStock(1, models.StockTypePresentoir, 10)
	if err := svc.CheckStation(1); err != nil {
		t.Fatalf("CheckStation: %v", err)
	}
	if store.sentCount() != 1 {
		t.Fatalf("stock pile au seuil : attendu 1 notification, obtenu %d", store.sentCount())
	}
}

func TestStockAlertBelowThresholdNotifies(t *testing.T) {
	store := presentoirStore()
	svc := NewStockAlertService(store)

	store.setStock(1, models.StockTypePresentoir, 5)
	if err := svc.CheckStation(1); err != nil {
		t.Fatalf("CheckStation: %v", err)
	}
	if store.sentCount() != 1 {
		t.Fatalf("stock sous le seuil : attendu 1 notification, obtenu %d", store.sentCount())
	}
}

// ── 4 à 7 : la machine d'état ──────────────────────────────────────────────────

func TestStockAlertCrossingDownNotifiesOnce(t *testing.T) {
	store := presentoirStore()
	svc := NewStockAlertService(store)

	// 11 → 10 : le franchissement.
	store.setStock(1, models.StockTypePresentoir, 11)
	mustCheck(t, svc)
	if store.sentCount() != 0 {
		t.Fatalf("11 est au-dessus du seuil, aucune notification attendue")
	}

	store.setStock(1, models.StockTypePresentoir, 10)
	mustCheck(t, svc)
	if store.sentCount() != 1 {
		t.Fatalf("11 → 10 : attendu 1 notification, obtenu %d", store.sentCount())
	}
}

func TestStockAlertStayingLowDoesNotRepeat(t *testing.T) {
	store := presentoirStore()
	svc := NewStockAlertService(store)

	store.setStock(1, models.StockTypePresentoir, 10)
	mustCheck(t, svc)

	// C'est le cas qui motive toute la machine d'état : sans elle, chacune de ces quatre
	// mutations renotifierait.
	for _, qty := range []int{9, 8, 7, 5} {
		store.setStock(1, models.StockTypePresentoir, qty)
		mustCheck(t, svc)
	}

	if store.sentCount() != 1 {
		t.Fatalf("stock resté bas : attendu 1 seule notification, obtenu %d", store.sentCount())
	}
}

func TestStockAlertRearmsWhenStockRecovers(t *testing.T) {
	store := presentoirStore()
	svc := NewStockAlertService(store)

	store.setStock(1, models.StockTypePresentoir, 5)
	mustCheck(t, svc)

	// Remontée au-dessus du seuil : l'alerte se réarme sans rien émettre.
	store.setStock(1, models.StockTypePresentoir, 20)
	mustCheck(t, svc)
	if store.sentCount() != 1 {
		t.Fatalf("une remontée ne notifie pas : obtenu %d notifications", store.sentCount())
	}
	if store.active[alertKey{1, models.StockTypePresentoir}] {
		t.Fatal("l'alerte devait être réarmée au-dessus du seuil")
	}
}

func TestStockAlertNotifiesAgainAfterRearm(t *testing.T) {
	store := presentoirStore()
	svc := NewStockAlertService(store)

	store.setStock(1, models.StockTypePresentoir, 5)
	mustCheck(t, svc)
	store.setStock(1, models.StockTypePresentoir, 20)
	mustCheck(t, svc)
	store.setStock(1, models.StockTypePresentoir, 15)
	mustCheck(t, svc)

	// 20 → 15 reste au-dessus, seule la redescente à 10 doit renotifier.
	if store.sentCount() != 1 {
		t.Fatalf("15 est au-dessus du seuil : obtenu %d notifications", store.sentCount())
	}

	store.setStock(1, models.StockTypePresentoir, 10)
	mustCheck(t, svc)
	if store.sentCount() != 2 {
		t.Fatalf("second franchissement : attendu 2 notifications au total, obtenu %d", store.sentCount())
	}
}

// ── 8 à 10 : les destinataires ─────────────────────────────────────────────────

func TestStockAlertPresentoirGoesToStationManager(t *testing.T) {
	store := presentoirStore()
	svc := NewStockAlertService(store)

	store.setStock(1, models.StockTypePresentoir, 8)
	mustCheck(t, svc)

	if store.sentCount() != 1 {
		t.Fatalf("attendu 1 notification, obtenu %d", store.sentCount())
	}
	got := store.sent[0]
	if got.UserID != 7 {
		t.Fatalf("l'alerte présentoir doit aller au responsable 7, obtenu %d", got.UserID)
	}
	if got.Type != models.NotificationTypeStockPresentoirBas {
		t.Fatalf("type attendu %q, obtenu %q", models.NotificationTypeStockPresentoirBas, got.Type)
	}
	// Le message doit rester lisible relu plus tard, quand le stock aura bougé.
	if !strings.Contains(got.Message, "8") || !strings.Contains(got.Message, "100") {
		t.Fatalf("le message doit porter le stock et la référence, obtenu %q", got.Message)
	}
	if got.CurrentStock == nil || *got.CurrentStock != 8 {
		t.Fatal("le stock au déclenchement doit être figé dans la notification")
	}
	if got.Threshold == nil || *got.Threshold != 10 {
		t.Fatal("le seuil doit être figé dans la notification")
	}
}

func TestStockAlertLocalGoesToAdmins(t *testing.T) {
	store := newFakeAlertStore()
	store.reference[alertKey{1, models.StockTypeLocal}] = 200
	// Tous les ADMIN et SUPER_ADMIN actifs : les comptes ADMIN semés ont station_id à NULL,
	// un filtre par station ne rendrait personne.
	store.recipients[alertKey{1, models.StockTypeLocal}] = []int64{1, 2}
	svc := NewStockAlertService(store)

	// Référence 200 → seuil 20.
	store.setStock(1, models.StockTypeLocal, 15)
	mustCheck(t, svc)

	if store.sentCount() != 2 {
		t.Fatalf("attendu une notification par administrateur (2), obtenu %d", store.sentCount())
	}
	for _, n := range store.sent {
		if n.Type != models.NotificationTypeStockLocalBas {
			t.Fatalf("type attendu %q, obtenu %q", models.NotificationTypeStockLocalBas, n.Type)
		}
		if n.UserID != 1 && n.UserID != 2 {
			t.Fatalf("destinataire inattendu %d", n.UserID)
		}
	}
}

func TestStockAlertSpareOtherStations(t *testing.T) {
	store := presentoirStore()
	// La station 2 a son propre responsable. Le filtrage réel est SQL
	// (`u.station_id = $1`) ; ce test vérifie que le service interroge bien la station sur
	// laquelle la mutation a porté, et n'élargit pas au-delà.
	store.reference[alertKey{2, models.StockTypePresentoir}] = 100
	store.recipients[alertKey{2, models.StockTypePresentoir}] = []int64{9}
	svc := NewStockAlertService(store)

	store.setStock(1, models.StockTypePresentoir, 4)
	mustCheck(t, svc)

	if store.sentCount() != 1 {
		t.Fatalf("attendu 1 notification, obtenu %d", store.sentCount())
	}
	if store.sent[0].UserID == 9 {
		t.Fatal("le responsable de la station 2 ne doit pas recevoir l'alerte de la station 1")
	}
	if store.sent[0].StationID == nil || *store.sent[0].StationID != 1 {
		t.Fatal("la notification doit porter la station concernée")
	}
}

// ── 11 : la concurrence ────────────────────────────────────────────────────────

func TestStockAlertConcurrentMutationsNotifyOnce(t *testing.T) {
	store := presentoirStore()
	svc := NewStockAlertService(store)

	// Vingt mutations simultanées sur une station déjà sous le seuil. En base, la
	// sérialisation vient de UNIQUE(station_id, stock_type) et du `WHERE active = false` du
	// ON CONFLICT ; le faux dépôt rejoue ce contrat sous verrou.
	store.setStock(1, models.StockTypePresentoir, 6)

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = svc.CheckStation(1)
		}()
	}
	wg.Wait()

	if store.sentCount() != 1 {
		t.Fatalf("20 mutations concurrentes : attendu 1 notification, obtenu %d", store.sentCount())
	}
	if store.armCalls < 20 {
		t.Fatalf("les 20 passages devaient tous tenter l'armement, obtenu %d", store.armCalls)
	}
}

// ── 12 : l'absence de référence ────────────────────────────────────────────────

func TestStockAlertWithoutReferenceStaysSilent(t *testing.T) {
	store := newFakeAlertStore()
	// Aucune référence pour la station 1 : colonne NULL en base, lue à zéro.
	store.recipients[alertKey{1, models.StockTypePresentoir}] = []int64{7}
	svc := NewStockAlertService(store)

	store.setStock(1, models.StockTypePresentoir, 0)
	mustCheck(t, svc)

	// Un stock à zéro sans référence connue n'est pas une anomalie : c'est un magasin qu'on
	// n'a jamais mesuré. Alerter reviendrait à le signaler en permanence.
	if store.sentCount() != 0 {
		t.Fatalf("sans référence, aucune alerte ne doit partir, obtenu %d", store.sentCount())
	}
}

// ── L'arrondi du seuil ─────────────────────────────────────────────────────────

func TestStockAlertThresholdRoundsUp(t *testing.T) {
	cases := []struct{ reference, want int }{
		{100, 10},
		{200, 20},
		// 3,7 arrondi vers le haut : à 3 montures le magasin est bien sous les 10 %,
		// arrondir vers le bas le laisserait sans alerte.
		{37, 4},
		{1, 1},
		{5, 1},
		{0, 0},
		{-3, 0},
	}
	for _, tc := range cases {
		if got := models.StockAlertThreshold(tc.reference); got != tc.want {
			t.Fatalf("seuil pour une référence de %d : attendu %d, obtenu %d", tc.reference, tc.want, got)
		}
	}
}

func TestStockAlertIgnoresUnknownStation(t *testing.T) {
	store := presentoirStore()
	svc := NewStockAlertService(store)

	// Une monture sans station (station_id NULL) ne doit rien déclencher ni paniquer.
	if err := svc.CheckStation(0); err != nil {
		t.Fatalf("station inconnue: %v", err)
	}
	if store.sentCount() != 0 {
		t.Fatal("aucune notification attendue pour une station inconnue")
	}
}

func mustCheck(t *testing.T, svc *StockAlertService) {
	t.Helper()
	if err := svc.CheckStation(1); err != nil {
		t.Fatalf("CheckStation: %v", err)
	}
}
