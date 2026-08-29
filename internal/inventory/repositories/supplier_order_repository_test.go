package repositories

import (
	"testing"
	"time"

	"github.com/lunetterie/backend/internal/inventory/models"
)

func TestSupplierOrderInsertSpec_LegacySchema(t *testing.T) {
	columns := []string{"id", "supplier", "quantity", "order_date", "note", "created_by", "created_at", "updated_at"}
	note := "test"
	order := &models.SupplierOrder{
		Supplier:  "Fournisseur A",
		Quantity:  42,
		OrderDate: time.Date(2026, 8, 29, 0, 0, 0, 0, time.UTC),
		Note:      &note,
	}

	cols, vals := supplierOrderInsertSpec(columns, order)
	if len(cols) != 5 {
		t.Fatalf("legacy insert should keep only legacy columns, got %d: %v", len(cols), cols)
	}
	if cols[0] != "supplier" || cols[1] != "quantity" || cols[2] != "order_date" || cols[3] != "note" || cols[4] != "created_by" {
		t.Fatalf("unexpected legacy columns: %v", cols)
	}
	if len(vals) != 5 {
		t.Fatalf("legacy values length mismatch: %d", len(vals))
	}
}

func TestSupplierOrderSelectColumns_EnhancedSchema(t *testing.T) {
	columns := []string{"id", "supplier", "reference", "provenance", "destination", "quantity", "gender", "gamme", "order_date", "transport", "barcode_num", "status", "note", "created_by", "created_at", "updated_at"}
	cols := supplierOrderSelectColumns(columns)
	if len(cols) < 12 {
		t.Fatalf("expected enhanced select set, got %d: %v", len(cols), cols)
	}
	if !containsColumn(cols, "reference") || !containsColumn(cols, "status") || !containsColumn(cols, "barcode_num") {
		t.Fatalf("enhanced schema should include modern supplier order fields: %v", cols)
	}
}

func containsColumn(cols []string, want string) bool {
	for _, col := range cols {
		if col == want {
			return true
		}
	}
	return false
}
