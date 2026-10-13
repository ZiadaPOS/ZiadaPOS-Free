package localpos

import (
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const accessCookie = "pos_access_token"

// Handler exposes the desktop-only local POS API through the same origin as
// the embedded UI. It must be mounted only by the desktop shell, never by the
// public web deployment.
type Handler struct {
	store *Store
}

func NewHandler(store *Store) *Handler { return &Handler{store: store} }

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if r.URL.Path == "/api/local/status" {
		h.localStatus(w, r)
		return
	}
	if r.URL.Path == "/api/local/setup" {
		h.setup(w, r)
		return
	}
	if r.URL.Path == "/api/pos-session" {
		h.session(w, r)
		return
	}
	if !strings.HasPrefix(r.URL.Path, "/api/v1/") {
		h.writeError(w, http.StatusNotFound, "local route not found")
		return
	}
	if r.URL.Path == "/api/v1/health" && r.Method == http.MethodGet {
		h.writeOK(w, map[string]any{"status": "ok", "mode": "local", "storage": "sqlite"})
		return
	}
	user, err := h.authenticatedUser(r)
	if err != nil {
		h.writeError(w, http.StatusUnauthorized, "sign in to the local shop to continue")
		return
	}
	switch {
	case r.URL.Path == "/api/v1/accounts/me" && r.Method == http.MethodGet:
		h.me(w, user)
	case r.URL.Path == "/api/v1/inventory/categories" && r.Method == http.MethodGet:
		h.writeOK(w, map[string]any{"categories": []any{}})
	case r.URL.Path == "/api/v1/customers" && r.Method == http.MethodGet:
		h.writeOK(w, map[string]any{"customers": []any{}})
	case r.URL.Path == "/api/v1/inventory/products" && r.Method == http.MethodGet:
		h.listProducts(w, r, user)
	case r.URL.Path == "/api/v1/inventory/products" && r.Method == http.MethodPost:
		h.createProduct(w, r, user)
	case strings.HasPrefix(r.URL.Path, "/api/v1/inventory/products/barcode/") && r.Method == http.MethodGet:
		h.productByBarcode(w, r, user)
	case strings.HasPrefix(r.URL.Path, "/api/v1/inventory/products/"):
		h.productRoute(w, r, user)
	case r.URL.Path == "/api/v1/pos/sale" && r.Method == http.MethodPost:
		h.completeSale(w, r, user)
	case r.URL.Path == "/api/v1/pos/transactions" && r.Method == http.MethodGet:
		h.listTransactions(w, r, user)
	case strings.HasPrefix(r.URL.Path, "/api/v1/pos/transactions/"):
		h.transactionRoute(w, r, user)
	case r.URL.Path == "/api/v1/pos/summary" && r.Method == http.MethodGet:
		h.summary(w, r, user)
	default:
		h.writeError(w, http.StatusNotImplemented, "this feature is not yet available in local mode")
	}
}

func (h *Handler) localStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	configured, err := h.store.Configured(r.Context())
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "could not read local shop status")
		return
	}
	pending, err := h.store.OutboxCount(r.Context())
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "could not read local outbox status")
		return
	}
	h.writeOK(w, map[string]any{"local_mode": true, "configured": configured, "sync_configured": false, "pending_local_events": pending})
}

func (h *Handler) setup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var input SetupInput
	if err := decodeJSON(r, &input); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid setup details")
		return
	}
	user, token, err := h.store.Setup(r.Context(), input)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, ErrAlreadySetup) {
			status = http.StatusConflict
		}
		h.writeError(w, status, err.Error())
		return
	}
	h.setSessionCookies(w, token)
	w.Header().Set("Location", "/overview")
	h.writeOK(w, map[string]any{"user": user, "business_type": "retail", "offline_mode": true})
}

func (h *Handler) session(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		user, err := h.authenticatedUser(r)
		if err != nil {
			h.clearSessionCookies(w)
			h.writeError(w, http.StatusUnauthorized, "local session is not active")
			return
		}
		h.writeOK(w, map[string]any{"authenticated": true, "business_type": "retail", "offline_mode": true, "user": user})
	case http.MethodPost:
		var input struct {
			Identifier string `json:"identifier"`
			Password   string `json:"password"`
		}
		if err := decodeJSON(r, &input); err != nil {
			h.writeError(w, http.StatusBadRequest, "invalid sign-in details")
			return
		}
		user, token, err := h.store.Login(r.Context(), input.Identifier, input.Password)
		if err != nil {
			if errors.Is(err, ErrNotConfigured) {
				h.writeError(w, http.StatusConflict, "set up this computer's local shop first")
			} else {
				h.writeError(w, http.StatusUnauthorized, ErrInvalidLogin.Error())
			}
			return
		}
		h.setSessionCookies(w, token)
		h.writeOK(w, map[string]any{"user": user, "business_type": "retail", "offline_mode": true})
	case http.MethodDelete:
		if cookie, err := r.Cookie(accessCookie); err == nil {
			_ = h.store.Logout(r.Context(), cookie.Value)
		}
		h.clearSessionCookies(w)
		h.writeOK(w, map[string]any{"signed_out": true})
	default:
		h.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *Handler) authenticatedUser(r *http.Request) (User, error) {
	cookie, err := r.Cookie(accessCookie)
	if err != nil {
		return User{}, ErrInvalidLogin
	}
	return h.store.UserForToken(r.Context(), cookie.Value)
}

func (h *Handler) setSessionCookies(w http.ResponseWriter, token string) {
	for _, name := range []string{"pos_access_token", "pos_refresh_token"} {
		http.SetCookie(w, &http.Cookie{Name: name, Value: token, Path: "/", HttpOnly: true, SameSite: http.SameSiteLaxMode, MaxAge: int((30 * 24 * time.Hour).Seconds())})
	}
}
func (h *Handler) clearSessionCookies(w http.ResponseWriter) {
	for _, name := range []string{"pos_access_token", "pos_refresh_token"} {
		http.SetCookie(w, &http.Cookie{Name: name, Value: "", Path: "/", HttpOnly: true, SameSite: http.SameSiteLaxMode, MaxAge: -1})
	}
}

func (h *Handler) me(w http.ResponseWriter, user User) {
	data := map[string]any{
		"user":         user,
		"store":        map[string]any{"id": user.StoreID, "name": user.StoreName},
		"organisation": map[string]any{"id": user.OrganisationID, "name": user.BusinessName, "business_type": "retail", "outlet_type": ""},
		"offline_mode": true,
	}
	h.writeOK(w, data)
}

func (h *Handler) listProducts(w http.ResponseWriter, r *http.Request, user User) {
	products, err := h.store.ListProducts(r.Context(), user, r.URL.Query())
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "could not read local products")
		return
	}
	h.writeOK(w, map[string]any{"products": products})
}

func (h *Handler) createProduct(w http.ResponseWriter, r *http.Request, user User) {
	var input ProductInput
	if err := decodeJSON(r, &input); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid product details")
		return
	}
	product, err := h.store.CreateProduct(r.Context(), user, input)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	h.writeJSON(w, http.StatusCreated, map[string]any{"success": true, "message": "Product saved locally", "data": product})
}

func (h *Handler) productByBarcode(w http.ResponseWriter, r *http.Request, user User) {
	value := strings.TrimPrefix(r.URL.Path, "/api/v1/inventory/products/barcode/")
	barcode, err := url.PathUnescape(value)
	if err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid barcode")
		return
	}
	product, err := h.store.ProductByBarcode(r.Context(), user, barcode)
	if err != nil {
		h.writeStoreError(w, err, "barcode not found")
		return
	}
	h.writeOK(w, product)
}

func (h *Handler) productRoute(w http.ResponseWriter, r *http.Request, user User) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/inventory/products/")
	parts := strings.Split(path, "/")
	if len(parts) == 2 && parts[1] == "adjust" && r.Method == http.MethodPost {
		var input struct {
			QtyChange int64 `json:"qty_change"`
		}
		if err := decodeJSON(r, &input); err != nil {
			h.writeError(w, http.StatusBadRequest, "invalid stock adjustment")
			return
		}
		product, err := h.store.AdjustStock(r.Context(), user, parts[0], input.QtyChange)
		if err != nil {
			h.writeStoreError(w, err, "could not adjust local stock")
			return
		}
		h.writeOK(w, product)
		return
	}
	if len(parts) != 1 {
		h.writeError(w, http.StatusNotImplemented, "this inventory operation is not yet available in local mode")
		return
	}
	switch r.Method {
	case http.MethodGet:
		product, err := h.store.Product(r.Context(), user, parts[0])
		if err != nil {
			h.writeStoreError(w, err, "product not found")
			return
		}
		h.writeOK(w, product)
	case http.MethodPatch:
		var input ProductInput
		if err := decodeJSON(r, &input); err != nil {
			h.writeError(w, http.StatusBadRequest, "invalid product details")
			return
		}
		product, err := h.store.UpdateProduct(r.Context(), user, parts[0], input)
		if err != nil {
			h.writeStoreError(w, err, "could not update local product")
			return
		}
		h.writeOK(w, product)
	case http.MethodDelete:
		if err := h.store.ArchiveProduct(r.Context(), user, parts[0]); err != nil {
			h.writeStoreError(w, err, "could not archive local product")
			return
		}
		h.writeJSON(w, http.StatusNoContent, nil)
	default:
		h.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (h *Handler) completeSale(w http.ResponseWriter, r *http.Request, user User) {
	var input SaleInput
	if err := decodeJSON(r, &input); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid sale details")
		return
	}
	transaction, err := h.store.CompleteSale(r.Context(), user, input)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, ErrInsufficientQty) || errors.Is(err, ErrIdempotencyConflict) {
			status = http.StatusConflict
		}
		h.writeError(w, status, err.Error())
		return
	}
	h.writeJSON(w, http.StatusCreated, map[string]any{"success": true, "message": "Cash sale saved on this computer; sync is not configured", "data": transaction})
}

func (h *Handler) listTransactions(w http.ResponseWriter, r *http.Request, user User) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
	transactions, err := h.store.ListTransactions(r.Context(), user, limit)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "could not read local sales")
		return
	}
	h.writeOK(w, map[string]any{"transactions": transactions})
}

func (h *Handler) transactionRoute(w http.ResponseWriter, r *http.Request, user User) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/pos/transactions/")
	if r.Method != http.MethodGet {
		h.writeError(w, http.StatusNotImplemented, "refunds, voids, and remote receipts are not yet available in local mode")
		return
	}
	transaction, err := h.store.Transaction(r.Context(), user, path)
	if err != nil {
		h.writeStoreError(w, err, "sale not found")
		return
	}
	h.writeOK(w, transaction)
}

func (h *Handler) summary(w http.ResponseWriter, r *http.Request, user User) {
	start := time.Now().In(time.Local).Format("2006-01-02") + "T00:00:00"
	var gross, count int64
	err := h.store.db.QueryRowContext(r.Context(), "SELECT COALESCE(SUM(total),0),COUNT(*) FROM sales WHERE store_id=? AND created_at>=?", user.StoreID, start).Scan(&gross, &count)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, "could not calculate local sales summary")
		return
	}
	h.writeOK(w, map[string]any{"gross_sales": gross, "net_sales": gross, "transactions": count, "profit": 0, "offline_mode": true})
}

func decodeJSON(r *http.Request, destination any) error {
	defer r.Body.Close()
	decoder := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	return decoder.Decode(destination)
}
func (h *Handler) writeStoreError(w http.ResponseWriter, err error, fallback string) {
	switch {
	case errors.Is(err, ErrNotFound), errors.Is(err, sql.ErrNoRows):
		h.writeError(w, http.StatusNotFound, fallback)
	case errors.Is(err, ErrInsufficientQty):
		h.writeError(w, http.StatusConflict, err.Error())
	default:
		h.writeError(w, http.StatusBadRequest, err.Error())
	}
}
func (h *Handler) writeOK(w http.ResponseWriter, data any) {
	h.writeJSON(w, http.StatusOK, map[string]any{"success": true, "message": "OK", "data": data})
}
func (h *Handler) writeError(w http.ResponseWriter, status int, message string) {
	h.writeJSON(w, status, map[string]any{"success": false, "message": message})
}
func (h *Handler) writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if status != http.StatusNoContent && value != nil {
		_ = json.NewEncoder(w).Encode(value)
	}
}
