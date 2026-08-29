package services

import (
	"errors"
	"testing"

	"github.com/lunetterie/backend/internal/inventory/models"
	"github.com/lunetterie/backend/internal/inventory/ports"
)

type testTx struct{ commits, rollbacks int }

func (t *testTx) Commit() error   { t.commits++; return nil }
func (t *testTx) Rollback() error { t.rollbacks++; return nil }
func (t *testTx) Unwrap() any     { return nil }

type testTransactions struct{ tx *testTx }

func (m *testTransactions) Begin() (ports.Transaction, error) { return m.tx, nil }

type testBarcode struct{}

func (testBarcode) GenerateBarcode() (string, error) { return "LUN-CNG-1", nil }

type testGlasses struct {
	glass                                             *models.Glass
	failCreate, failLocation, failStatus, failReserve bool
}

func (r *testGlasses) GetByID(_ int64) (*models.Glass, error)       { return r.glass, nil }
func (r *testGlasses) Create(*models.Glass) error                   { return nil }
func (r *testGlasses) UpdateLocation(int64, int64) error            { return nil }
func (r *testGlasses) UpdateStatus(int64, models.GlassStatus) error { return nil }
func (r *testGlasses) UpdateReservedState(int64, bool) error        { return nil }
func (r *testGlasses) GetByIDTx(_ ports.Transaction, _ int64) (*models.Glass, error) {
	return r.glass, nil
}
func (r *testGlasses) CreateTx(_ ports.Transaction, _ *models.Glass) error {
	if r.failCreate {
		return errors.New("create")
	}
	return nil
}
func (r *testGlasses) UpdateLocationTx(_ ports.Transaction, _, _ int64) error {
	if r.failLocation {
		return errors.New("location")
	}
	return nil
}
func (r *testGlasses) UpdateStatusTx(_ ports.Transaction, _ int64, _ models.GlassStatus) error {
	if r.failStatus {
		return errors.New("status")
	}
	return nil
}
func (r *testGlasses) UpdateReservedStateTx(_ ports.Transaction, _ int64, _ bool) error {
	if r.failReserve {
		return errors.New("reserve")
	}
	return nil
}

type testStorage struct{ carton *models.StorageLocation }

func (r *testStorage) GetByID(int64) (*models.StorageLocation, error) { return r.carton, nil }
func (r *testStorage) CountGlassesAtLocation(int64) (int, error)      { return 0, nil }
func (r *testStorage) UpdateStatus(int64, string) error               { return nil }
func (r *testStorage) GetByIDTx(_ ports.Transaction, _ int64) (*models.StorageLocation, error) {
	return r.carton, nil
}
func (r *testStorage) CountGlassesAtLocationTx(_ ports.Transaction, _ int64) (int, error) {
	return 0, nil
}
func (r *testStorage) UpdateStatusTx(_ ports.Transaction, _ int64, _ string) error { return nil }

type testMovements struct{ fail bool }

func (r *testMovements) Create(*models.Movement) error { return nil }
func (r *testMovements) CreateTx(_ ports.Transaction, _ *models.Movement) error {
	if r.fail {
		return errors.New("movement")
	}
	return nil
}

func newTransactionalTestService(tx *testTx, movements *testMovements, glasses *testGlasses) *TransactionalGlassService {
	return NewTransactionalGlassService(&testTransactions{tx}, glasses, &testStorage{carton: &models.StorageLocation{ID: 2, Type: "CARTON", Status: "LIBRE"}}, movements, testBarcode{})
}

func TestTransactionalGlassServiceCommitsAllMutations(t *testing.T) {
	locationID := int64(1)
	for name, mutation := range map[string]func(*TransactionalGlassService) error{
		"create":  func(s *TransactionalGlassService) error { return s.CreateGlass(&models.Glass{StationID: 1}) },
		"assign":  func(s *TransactionalGlassService) error { return s.AssignGlass(1, 2, 9) },
		"move":    func(s *TransactionalGlassService) error { return s.MoveGlass(1, 2, 9) },
		"reserve": func(s *TransactionalGlassService) error { return s.ReserveGlass(1, 99, 9) },
	} {
		t.Run(name, func(t *testing.T) {
			tx := &testTx{}
			glass := &models.Glass{ID: 1, StationID: 1, LocationID: &locationID, Status: models.StatusEnPresentoir}
			if err := mutation(newTransactionalTestService(tx, &testMovements{}, &testGlasses{glass: glass})); err != nil {
				t.Fatal(err)
			}
			if tx.commits != 1 || tx.rollbacks != 0 {
				t.Fatalf("commit=%d rollback=%d", tx.commits, tx.rollbacks)
			}
		})
	}
}

func TestTransactionalGlassServiceRollsBackWhenMovementFails(t *testing.T) {
	locationID := int64(1)
	tx := &testTx{}
	glass := &models.Glass{ID: 1, StationID: 1, LocationID: &locationID, Status: models.StatusEnStockGeneral}
	err := newTransactionalTestService(tx, &testMovements{fail: true}, &testGlasses{glass: glass}).MoveGlass(1, 2, 9)
	if err == nil {
		t.Fatal("expected mutation error")
	}
	if tx.commits != 0 || tx.rollbacks != 1 {
		t.Fatalf("commit=%d rollback=%d", tx.commits, tx.rollbacks)
	}
}
