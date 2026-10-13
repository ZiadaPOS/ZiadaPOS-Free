package transactions

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/troubleman96/ziadapos/internal/apierr"
)

func TestCompleteSaleRejectsStaleStoreBeforeDatabaseWork(t *testing.T) {
	expectedStoreID := uuid.New()
	activeStoreID := uuid.New()
	service := &Service{}

	_, err := service.CompleteSale(context.Background(), uuid.New(), activeStoreID, uuid.New(), &SaleInput{
		ExpectedStoreID: &expectedStoreID,
	})
	if err == nil {
		t.Fatal("expected a stale store conflict")
	}
	domainErr := apierr.Translate(err)
	if domainErr.Status != 409 || domainErr.Message != "Active store changed; refresh checkout before completing the sale" {
		t.Fatalf("unexpected error: status=%d message=%q", domainErr.Status, domainErr.Message)
	}
}
