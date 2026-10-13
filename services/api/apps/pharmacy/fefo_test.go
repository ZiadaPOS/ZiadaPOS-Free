package pharmacy

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/troubleman96/ziadapos/apps/inventory"
	"github.com/troubleman96/ziadapos/internal/models"
)

func TestAllocateFEFOUsesEarliestLotsAndSkipsExpired(t *testing.T) {
	today := time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC)
	expired := inventory.MedicineBatch{BaseModel: models.BaseModel{ID: uuid.New()}, Quantity: 50, ExpiryDate: today.AddDate(0, 0, -1)}
	first := inventory.MedicineBatch{BaseModel: models.BaseModel{ID: uuid.New()}, Quantity: 3, ExpiryDate: today, CostPrice: 100}
	second := inventory.MedicineBatch{BaseModel: models.BaseModel{ID: uuid.New()}, Quantity: 4, ExpiryDate: today.AddDate(0, 0, 12), CostPrice: 200}
	allocations, err := AllocateFEFO([]inventory.MedicineBatch{expired, first, second}, 5, today)
	if err != nil {
		t.Fatal(err)
	}
	if len(allocations) != 2 || allocations[0].BatchID != first.ID || allocations[0].Quantity != 3 || allocations[1].BatchID != second.ID || allocations[1].Quantity != 2 {
		t.Fatalf("unexpected allocation: %+v", allocations)
	}
}

func TestAllocateFEFORejectsInsufficientUnexpiredStock(t *testing.T) {
	today := time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC)
	batches := []inventory.MedicineBatch{{BaseModel: models.BaseModel{ID: uuid.New()}, Quantity: 8, ExpiryDate: today.AddDate(0, 0, -1)}}
	if _, err := AllocateFEFO(batches, 1, today); !errors.Is(err, ErrInsufficientUnexpiredStock) {
		t.Fatalf("expected insufficient stock error, got %v", err)
	}
}
