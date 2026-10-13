package main

import (
	"bytes"
	"context"
	"encoding/json"
	"image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ZiadaPOS/ZiadaPOS/apps/desktop/localpos"
)

func TestNewLoopbackProxyRejectsRemoteFrontend(t *testing.T) {
	for _, target := range []string{"http://example.com", "https://127.0.0.1:3000", "file:///tmp/app"} {
		t.Run(target, func(t *testing.T) {
			if _, err := newLoopbackProxy(target); err == nil {
				t.Fatalf("newLoopbackProxy(%q) succeeded; want error", target)
			}
		})
	}
}

func TestNewLoopbackProxyForwardsMethodPathAndBody(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read body: %v", err)
		}
		w.Header().Set("X-Forwarded-Test", "ok")
		_, _ = w.Write([]byte(r.Method + " " + r.URL.RequestURI() + " " + string(body)))
	}))
	defer backend.Close()

	proxy, err := newLoopbackProxy(backend.URL)
	if err != nil {
		t.Fatalf("newLoopbackProxy: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "http://wails.localhost/api/pos?store=1", strings.NewReader("payload"))
	proxy.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if got, want := recorder.Body.String(), "POST /api/pos?store=1 payload"; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
	if got := recorder.Header().Get("X-Forwarded-Test"); got != "ok" {
		t.Fatalf("forwarded response header = %q, want %q", got, "ok")
	}
}

func TestMergeEnvironmentUsesOverrides(t *testing.T) {
	got := mergeEnvironment([]string{"A=old", "B=kept"}, map[string]string{"A": "new", "C": "added"})
	values := make(map[string]string)
	for _, item := range got {
		key, value, ok := strings.Cut(item, "=")
		if ok {
			values[key] = value
		}
	}
	for key, want := range map[string]string{"A": "new", "B": "kept", "C": "added"} {
		if values[key] != want {
			t.Errorf("%s = %q, want %q", key, values[key], want)
		}
	}
}

func TestAppIconIsEmbeddedPNG(t *testing.T) {
	config, err := png.DecodeConfig(bytes.NewReader(appIcon))
	if err != nil {
		t.Fatalf("decode embedded app icon: %v", err)
	}
	if config.Width != 1024 || config.Height != 1024 {
		t.Fatalf("embedded app icon size = %dx%d, want 1024x1024", config.Width, config.Height)
	}
}

func TestDesktopLocalRoutesStayInsideLocalService(t *testing.T) {
	store, err := localpos.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	app := newDesktopApp(nil)
	app.localHandler = localpos.NewHandler(store)

	setup := httptest.NewRecorder()
	setupBody := strings.NewReader(`{"business_name":"Mtaa Shop","first_name":"Asha","last_name":"Mushi","phone":"0712345678","password":"local-password-42"}`)
	app.ServeHTTP(setup, httptest.NewRequest(http.MethodPost, "http://wails.localhost/api/local/setup", setupBody))
	if setup.Code != http.StatusOK {
		t.Fatalf("local setup status=%d body=%s", setup.Code, setup.Body.String())
	}
	var sessionCookie *http.Cookie
	for _, cookie := range setup.Result().Cookies() {
		if cookie.Name == "pos_access_token" {
			sessionCookie = cookie
		}
	}
	if sessionCookie == nil {
		t.Fatal("local setup did not set the access cookie")
	}

	profileRequest := httptest.NewRequest(http.MethodGet, "http://wails.localhost/api/pos/accounts/me", nil)
	profileRequest.AddCookie(sessionCookie)
	profile := httptest.NewRecorder()
	app.ServeHTTP(profile, profileRequest)
	if profile.Code != http.StatusOK {
		t.Fatalf("local profile status=%d body=%s", profile.Code, profile.Body.String())
	}
	var envelope struct {
		Success bool `json:"success"`
		Data    struct {
			Offline bool `json:"offline_mode"`
			Store   struct {
				ID string `json:"id"`
			} `json:"store"`
		} `json:"data"`
	}
	if err := json.Unmarshal(profile.Body.Bytes(), &envelope); err != nil || !envelope.Success || !envelope.Data.Offline {
		t.Fatalf("local profile envelope=%+v err=%v", envelope, err)
	}
	if _, err := store.UserForToken(context.Background(), sessionCookie.Value); err != nil {
		t.Fatalf("desktop profile did not authenticate against local SQLite: %v", err)
	}

	productRequest := httptest.NewRequest(http.MethodPost, "http://wails.localhost/api/pos/inventory/products", strings.NewReader(`{"name":"Tea","selling_price":500,"stock":3}`))
	productRequest.AddCookie(sessionCookie)
	product := httptest.NewRecorder()
	app.ServeHTTP(product, productRequest)
	var productEnvelope struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if product.Code != http.StatusCreated || json.Unmarshal(product.Body.Bytes(), &productEnvelope) != nil || productEnvelope.Data.ID == "" {
		t.Fatalf("local product create status=%d body=%s", product.Code, product.Body.String())
	}

	saleBody := `{"expected_store_id":"` + envelope.Data.Store.ID + `","idempotency_key":"desktop-integration-sale","payment_method":"Cash","lines":[{"product_id":"` + productEnvelope.Data.ID + `","qty":1}]}`
	saleRequest := httptest.NewRequest(http.MethodPost, "http://wails.localhost/api/pos/pos/sale", strings.NewReader(saleBody))
	saleRequest.AddCookie(sessionCookie)
	sale := httptest.NewRecorder()
	app.ServeHTTP(sale, saleRequest)
	if sale.Code != http.StatusCreated {
		t.Fatalf("local sale status=%d body=%s", sale.Code, sale.Body.String())
	}

	historyRequest := httptest.NewRequest(http.MethodGet, "http://wails.localhost/api/pos/pos/transactions?page=1&page_size=10", nil)
	historyRequest.AddCookie(sessionCookie)
	history := httptest.NewRecorder()
	app.ServeHTTP(history, historyRequest)
	var historyEnvelope struct {
		Data struct {
			Transactions []localpos.Transaction `json:"transactions"`
		} `json:"data"`
	}
	if history.Code != http.StatusOK || json.Unmarshal(history.Body.Bytes(), &historyEnvelope) != nil || len(historyEnvelope.Data.Transactions) != 1 {
		t.Fatalf("local history status=%d body=%s", history.Code, history.Body.String())
	}
}
