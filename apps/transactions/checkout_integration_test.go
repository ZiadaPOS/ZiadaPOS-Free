package transactions

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/troubleman96/ziadapos/apps/accounts"
	"github.com/troubleman96/ziadapos/apps/inventory"
	"github.com/troubleman96/ziadapos/internal/config"
	"github.com/troubleman96/ziadapos/internal/events"
	"github.com/troubleman96/ziadapos/internal/security"
	"github.com/troubleman96/ziadapos/internal/server"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestCatalogProductCanBeSoldUnderSameStoreSession(t *testing.T) {
	dsn := os.Getenv("ZIADA_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set ZIADA_TEST_DATABASE_URL to run the PostgreSQL checkout integration test")
	}
	db, cleanup := isolatedCheckoutSchema(t, dsn)
	defer cleanup()
	for _, migrate := range []func(*gorm.DB) error{accounts.Migrate, inventory.Migrate, Migrate} {
		if err := migrate(db); err != nil {
			t.Fatalf("migration failed: %v", err)
		}
	}

	org := accounts.Organisation{Name: "Checkout Integration", Slug: "checkout-integration", BusinessType: "retail", IsActive: true}
	if err := db.Create(&org).Error; err != nil {
		t.Fatal(err)
	}
	store := accounts.Store{OrganisationID: org.ID, Name: "Checkout Integration Store", IsMainStore: true, Status: "open", IsActive: true}
	if err := db.Create(&store).Error; err != nil {
		t.Fatal(err)
	}
	otherStore := accounts.Store{OrganisationID: org.ID, Name: "Other Checkout Store", Status: "open", IsActive: true}
	if err := db.Create(&otherStore).Error; err != nil {
		t.Fatal(err)
	}
	product := inventory.Product{StoreID: store.ID, Name: "Panadol", SellingPrice: 65, CostPrice: 30, Stock: 10, IsActive: true}
	if err := db.Create(&product).Error; err != nil {
		t.Fatal(err)
	}

	gin.SetMode(gin.TestMode)
	router := gin.New()
	base := router.Group("/api/v1")
	bus := events.New(1, 8)
	defer bus.Close()
	deps := &server.Deps{
		DB:  db,
		Bus: bus,
		Tokens: security.NewManager(&config.Config{
			JWTAccessSecret:  "checkout-integration-access-secret",
			JWTRefreshSecret: "checkout-integration-refresh-secret",
			JWTAccessTTL:     15 * time.Minute,
			JWTRefreshTTL:    24 * time.Hour,
		}),
	}
	inventory.RegisterRoutes(base, deps)
	RegisterRoutes(base, deps)

	const cashierID = "34c2cdb8-437a-47c3-8a1b-23f168d1bc80"
	pair, err := deps.Tokens.Issue(uuid.MustParse(cashierID), accounts.RoleOwner, org.ID, store.ID)
	if err != nil {
		t.Fatal(err)
	}

	listRequest := httptest.NewRequest(http.MethodGet, "/api/v1/inventory/products?status=active&page=1&page_size=100", nil)
	listRequest.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	listResponse := httptest.NewRecorder()
	router.ServeHTTP(listResponse, listRequest)
	if listResponse.Code != http.StatusOK {
		t.Fatalf("catalog returned %d: %s", listResponse.Code, listResponse.Body.String())
	}
	var catalog struct {
		Success bool `json:"success"`
		Data    struct {
			Items []struct {
				ID      uuid.UUID `json:"id"`
				StoreID uuid.UUID `json:"store_id"`
			} `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal(listResponse.Body.Bytes(), &catalog); err != nil {
		t.Fatalf("decode catalog response: %v", err)
	}
	if !catalog.Success || len(catalog.Data.Items) != 1 {
		t.Fatalf("unexpected catalog response: %s", listResponse.Body.String())
	}
	listedProduct := catalog.Data.Items[0]
	if listedProduct.ID != product.ID || listedProduct.StoreID != store.ID {
		t.Fatalf("catalog product context mismatch: product=%s/%s store=%s", listedProduct.ID, product.ID, listedProduct.StoreID)
	}

	// A checkout tab can retain its old catalog after another tab switches the
	// shared session cookie to a different store. Reject that stale sale with a
	// specific conflict before product validation obscures the cause.
	otherStorePair, err := deps.Tokens.Issue(uuid.MustParse(cashierID), accounts.RoleOwner, org.ID, otherStore.ID)
	if err != nil {
		t.Fatal(err)
	}
	expectedStoreID := store.ID
	stalePayload, err := json.Marshal(SaleInput{
		ExpectedStoreID: &expectedStoreID,
		Lines:           []SaleLineInput{{ProductID: listedProduct.ID, Qty: 2}},
		PaymentMethod:   MethodCash,
		PriceMode:       "retail",
	})
	if err != nil {
		t.Fatal(err)
	}
	staleRequest := httptest.NewRequest(http.MethodPost, "/api/v1/pos/sale", bytes.NewReader(stalePayload))
	staleRequest.Header.Set("Authorization", "Bearer "+otherStorePair.AccessToken)
	staleRequest.Header.Set("Content-Type", "application/json")
	staleResponse := httptest.NewRecorder()
	router.ServeHTTP(staleResponse, staleRequest)
	if staleResponse.Code != http.StatusConflict || !strings.Contains(staleResponse.Body.String(), "Active store changed") {
		t.Fatalf("stale checkout returned %d: %s", staleResponse.Code, staleResponse.Body.String())
	}

	payload, err := json.Marshal(SaleInput{
		ExpectedStoreID: &expectedStoreID,
		Lines:           []SaleLineInput{{ProductID: listedProduct.ID, Qty: 2}},
		PaymentMethod:   MethodCash,
		PriceMode:       "retail",
	})
	if err != nil {
		t.Fatal(err)
	}
	saleRequest := httptest.NewRequest(http.MethodPost, "/api/v1/pos/sale", bytes.NewReader(payload))
	saleRequest.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	saleRequest.Header.Set("Content-Type", "application/json")
	saleResponse := httptest.NewRecorder()
	router.ServeHTTP(saleResponse, saleRequest)
	if saleResponse.Code != http.StatusCreated {
		t.Fatalf("sale returned %d: %s", saleResponse.Code, saleResponse.Body.String())
	}
	var sale struct {
		Success bool `json:"success"`
		Data    struct {
			ID       uuid.UUID `json:"id"`
			Subtotal int64     `json:"subtotal"`
			Total    int64     `json:"total"`
		} `json:"data"`
	}
	if err := json.Unmarshal(saleResponse.Body.Bytes(), &sale); err != nil {
		t.Fatalf("decode sale response: %v", err)
	}
	if !sale.Success || sale.Data.ID == uuid.Nil || sale.Data.Subtotal != 130 || sale.Data.Total != 153 {
		t.Fatalf("unexpected sale response: %s", saleResponse.Body.String())
	}
	var persistedProduct inventory.Product
	if err := db.First(&persistedProduct, "id = ? AND store_id = ?", product.ID, store.ID).Error; err != nil {
		t.Fatal(err)
	}
	if persistedProduct.Stock != 8 {
		t.Fatalf("product stock=%d after sale; want 8", persistedProduct.Stock)
	}
}

func isolatedCheckoutSchema(t *testing.T, dsn string) (*gorm.DB, func()) {
	t.Helper()
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("open database pool: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	schema := "ziada_checkout_it_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if err := db.Exec("CREATE SCHEMA \"" + schema + "\"").Error; err != nil {
		t.Fatalf("create isolated schema: %v", err)
	}
	if err := db.Exec(fmt.Sprintf("SET search_path TO \"%s\"", schema)).Error; err != nil {
		t.Fatalf("set isolated schema: %v", err)
	}
	return db, func() {
		if err := db.Exec("DROP SCHEMA IF EXISTS \"" + schema + "\" CASCADE").Error; err != nil {
			t.Errorf("drop isolated schema: %v", err)
		}
		_ = sqlDB.Close()
	}
}
