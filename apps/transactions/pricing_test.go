package transactions

import (
	"testing"

	"github.com/troubleman96/ziadapos/apps/inventory"
)

func TestSaleUnitPriceUsesConfiguredWholesaleTier(t *testing.T) {
	wholesale := int64(850)
	product := inventory.Product{SellingPrice: 1000, WholesalePrice: &wholesale, WholesaleMinimum: 6}
	price, err := saleUnitPrice(product, 6, "wholesale")
	if err != nil || price != wholesale {
		t.Fatalf("got price=%d err=%v", price, err)
	}
}

func TestSaleUnitPriceRejectsWholesaleBelowMinimum(t *testing.T) {
	wholesale := int64(850)
	product := inventory.Product{SellingPrice: 1000, WholesalePrice: &wholesale, WholesaleMinimum: 6}
	if _, err := saleUnitPrice(product, 5, "wholesale"); err == nil {
		t.Fatal("expected minimum quantity error")
	}
}

func TestSaleUnitPriceUsesRetailForRetailMode(t *testing.T) {
	wholesale := int64(850)
	product := inventory.Product{SellingPrice: 1000, WholesalePrice: &wholesale, WholesaleMinimum: 6}
	price, err := saleUnitPrice(product, 1, "retail")
	if err != nil || price != product.SellingPrice {
		t.Fatalf("got price=%d err=%v", price, err)
	}
}
