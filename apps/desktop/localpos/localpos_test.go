package localpos

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

func TestLocalDatabasePersistsAcrossRestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "pos.sqlite")
	store, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	user, _, err := store.Setup(context.Background(), SetupInput{BusinessName: "Mtaa Shop", FirstName: "Asha", LastName: "Mushi", Phone: "0712345678", Password: "local-password-42"})
	if err != nil {
		t.Fatal(err)
	}
	name := "Maize flour"
	price, stock := int64(1000), int64(7)
	product, err := store.CreateProduct(context.Background(), user, ProductInput{Name: &name, SellingPrice: &price, Stock: &stock})
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.CompleteSale(context.Background(), user, SaleInput{ExpectedStoreID: user.StoreID, IdempotencyKey: "persistent-request", PaymentMethod: "Cash", Lines: []SaleLineInput{{ProductID: product.ID, Qty: 2}}})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}

	reopened, err := Open(path)
	if err != nil {
		t.Fatalf("reopen local database: %v", err)
	}
	defer reopened.Close()
	loggedIn, _, err := reopened.Login(context.Background(), "0712345678", "local-password-42")
	if err != nil || loggedIn.ID != user.ID {
		t.Fatalf("login after restart user=%+v err=%v", loggedIn, err)
	}
	updated, err := reopened.Product(context.Background(), loggedIn, product.ID)
	if err != nil || updated.Stock != 5 {
		t.Fatalf("product after restart=%+v err=%v", updated, err)
	}
	transactions, err := reopened.ListTransactions(context.Background(), loggedIn, 10)
	if err != nil || len(transactions) != 1 {
		t.Fatalf("receipts after restart=%d err=%v", len(transactions), err)
	}
}

func testStore(t *testing.T) (*Store, User, string) {
	t.Helper()
	store, err := Open(":memory:")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	user, token, err := store.Setup(context.Background(), SetupInput{
		BusinessName: "Mtaa Shop",
		FirstName:    "Asha",
		LastName:     "Mushi",
		Phone:        "+255 712 345 678",
		Password:     "local-password-42",
	})
	if err != nil {
		t.Fatalf("Setup: %v", err)
	}
	return store, user, token
}

func TestLocalSetupLoginAndSession(t *testing.T) {
	store, user, token := testStore(t)
	if user.Role != "owner" || user.StoreID == "" || user.OrganisationID == "" {
		t.Fatalf("setup user missing local scope: %+v", user)
	}
	loggedIn, loginToken, err := store.Login(context.Background(), "0712345678", "local-password-42")
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	if loggedIn.ID != user.ID || loginToken == "" {
		t.Fatalf("login result = %+v, token length %d", loggedIn, len(loginToken))
	}
	if _, err := store.UserForToken(context.Background(), token); err != nil {
		t.Fatalf("UserForToken: %v", err)
	}
	if _, _, err := store.Login(context.Background(), "0712345678", "wrong-password"); err != ErrInvalidLogin {
		t.Fatalf("wrong-password error = %v, want ErrInvalidLogin", err)
	}
	if _, _, err := store.Setup(context.Background(), SetupInput{BusinessName: "Second shop", FirstName: "A", LastName: "B", Phone: "0711111111", Password: "password-123"}); err != ErrAlreadySetup {
		t.Fatalf("second setup error = %v, want ErrAlreadySetup", err)
	}
}

func TestOfflineCashSaleIsAtomicAndIdempotent(t *testing.T) {
	store, user, _ := testStore(t)
	name := "Maize flour"
	price, cost, stock := int64(100), int64(50), int64(5)
	product, err := store.CreateProduct(context.Background(), user, ProductInput{Name: &name, SellingPrice: &price, CostPrice: &cost, Stock: &stock})
	if err != nil {
		t.Fatalf("CreateProduct: %v", err)
	}
	input := SaleInput{
		ExpectedStoreID: user.StoreID,
		IdempotencyKey:  "browser-request-1",
		PaymentMethod:   "Cash",
		DiscountPct:     10,
		Lines:           []SaleLineInput{{ProductID: product.ID, Qty: 2}},
	}
	first, err := store.CompleteSale(context.Background(), user, input)
	if err != nil {
		t.Fatalf("CompleteSale: %v", err)
	}
	if first.Subtotal != 200 || first.DiscountAmount != 20 || first.TaxAmount != 32 || first.Total != 212 || first.Profit != 80 {
		t.Fatalf("sale totals incorrect: %+v", first)
	}
	if first.SyncStatus != "pending" || first.TxnNumber != "LOCAL-000001" || len(first.Lines) != 1 {
		t.Fatalf("sale should be local, pending, and itemized: %+v", first)
	}
	second, err := store.CompleteSale(context.Background(), user, input)
	if err != nil {
		t.Fatalf("idempotent retry: %v", err)
	}
	if second.ID != first.ID {
		t.Fatalf("retry created a second sale: first=%s second=%s", first.ID, second.ID)
	}
	changedRequest := input
	changedRequest.Lines = []SaleLineInput{{ProductID: product.ID, Qty: 1}}
	if _, err := store.CompleteSale(context.Background(), user, changedRequest); err != ErrIdempotencyConflict {
		t.Fatalf("reused key with changed sale returned %v, want ErrIdempotencyConflict", err)
	}
	updated, err := store.Product(context.Background(), user, product.ID)
	if err != nil {
		t.Fatalf("read product: %v", err)
	}
	if updated.Stock != 3 {
		t.Fatalf("stock after sale/retry = %d, want 3", updated.Stock)
	}
	transactions, err := store.ListTransactions(context.Background(), user, 50)
	if err != nil || len(transactions) != 1 {
		t.Fatalf("sales = %d, err = %v; want exactly one", len(transactions), err)
	}
	outbox, err := store.OutboxCount(context.Background())
	if err != nil || outbox != 1 {
		t.Fatalf("pending outbox events = %d, err = %v; want exactly one", outbox, err)
	}
}

func TestOfflineSaleRejectsNonCashAndInsufficientStockWithoutWrites(t *testing.T) {
	store, user, _ := testStore(t)
	name := "Tea"
	price, stock := int64(500), int64(1)
	product, err := store.CreateProduct(context.Background(), user, ProductInput{Name: &name, SellingPrice: &price, Stock: &stock})
	if err != nil {
		t.Fatalf("CreateProduct: %v", err)
	}
	base := SaleInput{ExpectedStoreID: user.StoreID, IdempotencyKey: "key-2", PaymentMethod: "Cash", Lines: []SaleLineInput{{ProductID: product.ID, Qty: 2}}}
	if _, err := store.CompleteSale(context.Background(), user, base); err == nil {
		t.Fatal("overselling sale succeeded")
	}
	base.PaymentMethod = "M-Pesa"
	base.IdempotencyKey = "key-3"
	if _, err := store.CompleteSale(context.Background(), user, base); err == nil {
		t.Fatal("offline non-cash sale succeeded")
	}
	transactions, err := store.ListTransactions(context.Background(), user, 50)
	if err != nil || len(transactions) != 0 {
		t.Fatalf("failed sale left transaction rows: %d, err=%v", len(transactions), err)
	}
	pending, err := store.OutboxCount(context.Background())
	if err != nil || pending != 0 {
		t.Fatalf("failed sale left outbox rows: %d, err=%v", pending, err)
	}
	unchanged, err := store.Product(context.Background(), user, product.ID)
	if err != nil || unchanged.Stock != 1 {
		t.Fatalf("failed sale changed stock: %+v, err=%v", unchanged, err)
	}
}

func TestHTTPSetupCookieAndLocalAPIAuth(t *testing.T) {
	store, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	handler := NewHandler(store)

	unauthenticated := httptest.NewRecorder()
	handler.ServeHTTP(unauthenticated, httptest.NewRequest(http.MethodGet, "/api/v1/accounts/me", nil))
	if unauthenticated.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated profile status=%d, want 401", unauthenticated.Code)
	}

	setupBody := []byte(`{"business_name":"Mtaa Shop","first_name":"Asha","last_name":"Mushi","phone":"0712345678","password":"local-password-42"}`)
	setup := httptest.NewRecorder()
	handler.ServeHTTP(setup, httptest.NewRequest(http.MethodPost, "/api/local/setup", bytes.NewReader(setupBody)))
	if setup.Code != http.StatusOK || len(setup.Result().Cookies()) != 2 {
		t.Fatalf("setup response status=%d cookies=%d body=%s", setup.Code, len(setup.Result().Cookies()), setup.Body.String())
	}
	var envelope struct {
		Success bool `json:"success"`
		Data    struct {
			Offline bool `json:"offline_mode"`
		} `json:"data"`
	}
	if err := json.Unmarshal(setup.Body.Bytes(), &envelope); err != nil || !envelope.Success || !envelope.Data.Offline {
		t.Fatalf("setup envelope invalid: %+v, err=%v", envelope, err)
	}

	profileRequest := httptest.NewRequest(http.MethodGet, "/api/v1/accounts/me", nil)
	profileRequest.AddCookie(setup.Result().Cookies()[0])
	profile := httptest.NewRecorder()
	handler.ServeHTTP(profile, profileRequest)
	if profile.Code != http.StatusOK {
		t.Fatalf("authenticated profile status=%d body=%s", profile.Code, profile.Body.String())
	}
}
