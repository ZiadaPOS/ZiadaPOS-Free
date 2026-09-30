package pharmacy

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/troubleman96/ziadapos/apps/inventory"
)

var ErrInsufficientUnexpiredStock = errors.New("insufficient unexpired batch stock")

type BatchAllocation struct {
	BatchID  uuid.UUID
	Quantity int
	UnitCost int64
}

// AllocateFEFO takes the requested quantity from earliest-expiring eligible
// lots. Input order is expiry_date ASC, then receipt order ASC, as queried under
// row locks by CompleteSale. Expired lots and nonpositive balances are skipped.
func AllocateFEFO(batches []inventory.MedicineBatch, requested int, today time.Time) ([]BatchAllocation, error) {
	if requested <= 0 {
		return nil, errors.New("requested quantity must be positive")
	}
	today = dateOnly(today)
	remaining := requested
	allocations := make([]BatchAllocation, 0, len(batches))
	for _, batch := range batches {
		if batch.Quantity <= 0 || dateOnly(batch.ExpiryDate).Before(today) {
			continue
		}
		quantity := batch.Quantity
		if quantity > remaining {
			quantity = remaining
		}
		allocations = append(allocations, BatchAllocation{BatchID: batch.ID, Quantity: quantity, UnitCost: batch.CostPrice})
		remaining -= quantity
		if remaining == 0 {
			return allocations, nil
		}
	}
	return nil, ErrInsufficientUnexpiredStock
}

func dateOnly(value time.Time) time.Time {
	year, month, day := value.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, value.Location())
}
