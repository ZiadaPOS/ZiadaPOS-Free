package localpos

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
	_ "modernc.org/sqlite"
)

var (
	ErrNotConfigured       = errors.New("local shop has not been set up")
	ErrAlreadySetup        = errors.New("local shop is already set up")
	ErrNotFound            = errors.New("record not found")
	ErrInvalidLogin        = errors.New("phone number or password is incorrect")
	ErrInsufficientQty     = errors.New("insufficient stock for one or more products")
	ErrIdempotencyConflict = errors.New("idempotency key was already used for different sale details")
)

const schemaVersion = 2

// Store owns the local, single-computer POS database. It never connects to a
// hosted service; pending outbox events remain on this computer until a sync
// protocol is implemented and explicitly configured.
type Store struct {
	db *sql.DB
}

func Open(path string) (*Store, error) {
	if path != ":memory:" {
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			return nil, fmt.Errorf("create local data directory: %w", err)
		}
		file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
		if err != nil {
			return nil, fmt.Errorf("create local database: %w", err)
		}
		if err := file.Chmod(0o600); err != nil {
			_ = file.Close()
			return nil, fmt.Errorf("protect local database permissions: %w", err)
		}
		if err := file.Close(); err != nil {
			return nil, err
		}
	}

	dsn := path
	if path != ":memory:" {
		dsn = "file:" + filepath.ToSlash(path) + "?_pragma=busy_timeout(5000)"
	}
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open local SQLite database: %w", err)
	}
	// A single connection keeps SQLite pragmas consistent and is appropriate
	// for one local POS process; SQLite still provides atomic write transactions.
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	store := &Store{db: db}
	if err := store.migrate(context.Background()); err != nil {
		_ = db.Close()
		return nil, err
	}
	return store, nil
}

func (s *Store) Close() error { return s.db.Close() }

func (s *Store) migrate(ctx context.Context) error {
	for _, pragma := range []string{"PRAGMA foreign_keys=ON", "PRAGMA journal_mode=WAL", "PRAGMA synchronous=FULL", "PRAGMA busy_timeout=5000"} {
		if _, err := s.db.ExecContext(ctx, pragma); err != nil {
			return fmt.Errorf("configure local SQLite: %w", err)
		}
	}
	var version int
	if err := s.db.QueryRowContext(ctx, "PRAGMA user_version").Scan(&version); err != nil {
		return fmt.Errorf("read local schema version: %w", err)
	}
	if version > schemaVersion {
		return fmt.Errorf("local database schema version %d is newer than this app supports (%d)", version, schemaVersion)
	}
	if version == schemaVersion {
		return nil
	}
	if _, err := s.db.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS businesses (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  business_type TEXT NOT NULL DEFAULT 'retail',
  created_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS stores (
  id TEXT PRIMARY KEY,
  business_id TEXT NOT NULL REFERENCES businesses(id),
  name TEXT NOT NULL,
  next_txn_number INTEGER NOT NULL DEFAULT 0
);
CREATE TABLE IF NOT EXISTS users (
  id TEXT PRIMARY KEY,
  business_id TEXT NOT NULL REFERENCES businesses(id),
  store_id TEXT NOT NULL REFERENCES stores(id),
  first_name TEXT NOT NULL,
  last_name TEXT NOT NULL,
  phone TEXT NOT NULL UNIQUE,
  password_hash TEXT NOT NULL,
  role TEXT NOT NULL DEFAULT 'owner'
);
CREATE TABLE IF NOT EXISTS sessions (
  token_hash TEXT PRIMARY KEY,
  user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  expires_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS products (
  id TEXT PRIMARY KEY,
  store_id TEXT NOT NULL REFERENCES stores(id),
  name TEXT NOT NULL,
  unit TEXT NOT NULL DEFAULT 'unit',
  sku TEXT NOT NULL DEFAULT '',
  barcode TEXT NOT NULL DEFAULT '',
  selling_price INTEGER NOT NULL CHECK (selling_price >= 0),
  cost_price INTEGER NOT NULL DEFAULT 0 CHECK (cost_price >= 0),
  stock INTEGER NOT NULL DEFAULT 0 CHECK (stock >= 0),
  reorder_point INTEGER NOT NULL DEFAULT 0,
  category_id TEXT,
  image TEXT NOT NULL DEFAULT '',
  description TEXT NOT NULL DEFAULT '',
  is_service INTEGER NOT NULL DEFAULT 0,
  is_active INTEGER NOT NULL DEFAULT 1,
  tax_rate INTEGER NOT NULL DEFAULT 18,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_products_store_active ON products(store_id, is_active, name);
CREATE INDEX IF NOT EXISTS idx_products_store_barcode ON products(store_id, barcode);
CREATE TABLE IF NOT EXISTS sales (
  id TEXT PRIMARY KEY,
  store_id TEXT NOT NULL REFERENCES stores(id),
  user_id TEXT NOT NULL REFERENCES users(id),
  txn_number TEXT NOT NULL,
  idempotency_key TEXT NOT NULL UNIQUE,
  request_hash TEXT NOT NULL DEFAULT '',
  payment_method TEXT NOT NULL,
  status TEXT NOT NULL,
  subtotal INTEGER NOT NULL,
  discount_amount INTEGER NOT NULL,
  tax_amount INTEGER NOT NULL,
  total INTEGER NOT NULL,
  profit INTEGER NOT NULL,
  customer_name TEXT NOT NULL DEFAULT 'Walk-in customer',
  customer_phone TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  UNIQUE(store_id, txn_number)
);
CREATE INDEX IF NOT EXISTS idx_sales_store_created ON sales(store_id, created_at DESC);
CREATE TABLE IF NOT EXISTS sale_lines (
  id TEXT PRIMARY KEY,
  sale_id TEXT NOT NULL REFERENCES sales(id) ON DELETE CASCADE,
  product_id TEXT NOT NULL,
  product_name TEXT NOT NULL,
  qty INTEGER NOT NULL CHECK (qty > 0),
  unit_price INTEGER NOT NULL,
  total INTEGER NOT NULL
);
CREATE TABLE IF NOT EXISTS outbox (
  id TEXT PRIMARY KEY,
  event_type TEXT NOT NULL,
  payload TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'pending',
  created_at TEXT NOT NULL
);
PRAGMA user_version=2;`); err != nil {
		return fmt.Errorf("migrate local SQLite database: %w", err)
	}
	if version == 1 {
		if _, err := s.db.ExecContext(ctx, `ALTER TABLE sales ADD COLUMN request_hash TEXT NOT NULL DEFAULT ''; PRAGMA user_version=2;`); err != nil {
			return fmt.Errorf("migrate local sale request fingerprints: %w", err)
		}
	}
	return nil
}

func (s *Store) Configured(ctx context.Context) (bool, error) {
	var count int
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM businesses").Scan(&count); err != nil {
		return false, err
	}
	return count > 0, nil
}

type User struct {
	ID             string `json:"id"`
	FullName       string `json:"full_name"`
	FirstName      string `json:"first_name"`
	LastName       string `json:"last_name"`
	Phone          string `json:"phone"`
	Role           string `json:"role"`
	OrganisationID string `json:"organisation_id"`
	StoreID        string `json:"store_id"`
	StoreName      string `json:"-"`
	BusinessName   string `json:"-"`
}

type SetupInput struct {
	BusinessName string `json:"business_name"`
	FirstName    string `json:"first_name"`
	LastName     string `json:"last_name"`
	Phone        string `json:"phone"`
	Password     string `json:"password"`
}

func (s *Store) Setup(ctx context.Context, input SetupInput) (User, string, error) {
	input.BusinessName = strings.TrimSpace(input.BusinessName)
	input.FirstName = strings.TrimSpace(input.FirstName)
	input.LastName = strings.TrimSpace(input.LastName)
	phone, err := normalizePhone(input.Phone)
	if err != nil {
		return User{}, "", err
	}
	if input.BusinessName == "" || input.FirstName == "" || input.LastName == "" {
		return User{}, "", errors.New("business name, first name, and last name are required")
	}
	if len(input.Password) < 8 {
		return User{}, "", errors.New("password must be at least 8 characters")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return User{}, "", fmt.Errorf("secure local password: %w", err)
	}
	token, err := randomToken()
	if err != nil {
		return User{}, "", err
	}
	businessID, storeID, userID := newID(), newID(), newID()
	now := time.Now().UTC().Format(time.RFC3339Nano)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return User{}, "", err
	}
	defer tx.Rollback()
	var existing int
	if err := tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM businesses").Scan(&existing); err != nil {
		return User{}, "", err
	}
	if existing > 0 {
		return User{}, "", ErrAlreadySetup
	}
	if _, err := tx.ExecContext(ctx, "INSERT INTO businesses(id,name,business_type,created_at) VALUES(?,?,?,?)", businessID, input.BusinessName, "retail", now); err != nil {
		return User{}, "", err
	}
	if _, err := tx.ExecContext(ctx, "INSERT INTO stores(id,business_id,name) VALUES(?,?,?)", storeID, businessID, input.BusinessName); err != nil {
		return User{}, "", err
	}
	if _, err := tx.ExecContext(ctx, "INSERT INTO users(id,business_id,store_id,first_name,last_name,phone,password_hash,role) VALUES(?,?,?,?,?,?,?,?)", userID, businessID, storeID, input.FirstName, input.LastName, phone, string(hash), "owner"); err != nil {
		return User{}, "", err
	}
	if err := insertSession(ctx, tx, userID, token); err != nil {
		return User{}, "", err
	}
	if err := tx.Commit(); err != nil {
		return User{}, "", err
	}
	return User{ID: userID, FirstName: input.FirstName, LastName: input.LastName, FullName: strings.TrimSpace(input.FirstName + " " + input.LastName), Phone: phone, Role: "owner", OrganisationID: businessID, StoreID: storeID, StoreName: input.BusinessName, BusinessName: input.BusinessName}, token, nil
}

func (s *Store) Login(ctx context.Context, identifier, password string) (User, string, error) {
	phone, err := normalizePhone(identifier)
	if err != nil {
		return User{}, "", ErrInvalidLogin
	}
	var user User
	var passwordHash string
	err = s.db.QueryRowContext(ctx, `SELECT u.id,u.business_id,u.store_id,u.first_name,u.last_name,u.phone,u.role,u.password_hash,st.name,b.name
FROM users u JOIN stores st ON st.id=u.store_id JOIN businesses b ON b.id=u.business_id WHERE u.phone=?`, phone).Scan(
		&user.ID, &user.OrganisationID, &user.StoreID, &user.FirstName, &user.LastName, &user.Phone, &user.Role, &passwordHash, &user.StoreName, &user.BusinessName,
	)
	if err != nil || bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password)) != nil {
		return User{}, "", ErrInvalidLogin
	}
	user.FullName = strings.TrimSpace(user.FirstName + " " + user.LastName)
	token, err := randomToken()
	if err != nil {
		return User{}, "", err
	}
	if err := insertSession(ctx, s.db, user.ID, token); err != nil {
		return User{}, "", err
	}
	return user, token, nil
}

func insertSession(ctx context.Context, execer interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}, userID, token string) error {
	_, err := execer.ExecContext(ctx, "INSERT INTO sessions(token_hash,user_id,expires_at) VALUES(?,?,?)", hashToken(token), userID, time.Now().UTC().Add(30*24*time.Hour).Format(time.RFC3339Nano))
	return err
}

func (s *Store) UserForToken(ctx context.Context, token string) (User, error) {
	if token == "" {
		return User{}, ErrInvalidLogin
	}
	var user User
	var expiry string
	err := s.db.QueryRowContext(ctx, `SELECT u.id,u.business_id,u.store_id,u.first_name,u.last_name,u.phone,u.role,st.name,b.name,se.expires_at
FROM sessions se JOIN users u ON u.id=se.user_id JOIN stores st ON st.id=u.store_id JOIN businesses b ON b.id=u.business_id
WHERE se.token_hash=?`, hashToken(token)).Scan(&user.ID, &user.OrganisationID, &user.StoreID, &user.FirstName, &user.LastName, &user.Phone, &user.Role, &user.StoreName, &user.BusinessName, &expiry)
	if err != nil {
		return User{}, ErrInvalidLogin
	}
	expiresAt, err := time.Parse(time.RFC3339Nano, expiry)
	if err != nil || !expiresAt.After(time.Now().UTC()) {
		_, _ = s.db.ExecContext(ctx, "DELETE FROM sessions WHERE token_hash=?", hashToken(token))
		return User{}, ErrInvalidLogin
	}
	user.FullName = strings.TrimSpace(user.FirstName + " " + user.LastName)
	return user, nil
}

func (s *Store) Logout(ctx context.Context, token string) error {
	if token == "" {
		return nil
	}
	_, err := s.db.ExecContext(ctx, "DELETE FROM sessions WHERE token_hash=?", hashToken(token))
	return err
}

type Product struct {
	ID           string `json:"id"`
	StoreID      string `json:"store_id"`
	Name         string `json:"name"`
	Unit         string `json:"unit"`
	SKU          string `json:"sku"`
	Barcode      string `json:"barcode"`
	SellingPrice int64  `json:"selling_price"`
	CostPrice    int64  `json:"cost_price"`
	Stock        int64  `json:"stock"`
	ReorderPoint int64  `json:"reorder_point"`
	CategoryID   string `json:"category_id,omitempty"`
	Image        string `json:"image,omitempty"`
	Description  string `json:"description,omitempty"`
	IsService    bool   `json:"is_service"`
	IsActive     bool   `json:"is_active"`
	TaxRate      int64  `json:"tax_rate"`
	LowStock     bool   `json:"low_stock"`
}

type ProductInput struct {
	Name         *string `json:"name"`
	Unit         *string `json:"unit"`
	SKU          *string `json:"sku"`
	Barcode      *string `json:"barcode"`
	SellingPrice *int64  `json:"selling_price"`
	CostPrice    *int64  `json:"cost_price"`
	Stock        *int64  `json:"stock"`
	ReorderPoint *int64  `json:"reorder_point"`
	CategoryID   *string `json:"category_id"`
	Image        *string `json:"image"`
	Description  *string `json:"description"`
	IsService    *bool   `json:"is_service"`
	IsActive     *bool   `json:"is_active"`
	TaxRate      *int64  `json:"tax_rate"`
}

const productColumns = `id,store_id,name,unit,sku,barcode,selling_price,cost_price,stock,reorder_point,COALESCE(category_id,''),image,description,is_service,is_active,tax_rate`

func scanProduct(scanner interface{ Scan(...any) error }) (Product, error) {
	var p Product
	var service, active int
	err := scanner.Scan(&p.ID, &p.StoreID, &p.Name, &p.Unit, &p.SKU, &p.Barcode, &p.SellingPrice, &p.CostPrice, &p.Stock, &p.ReorderPoint, &p.CategoryID, &p.Image, &p.Description, &service, &active, &p.TaxRate)
	p.IsService, p.IsActive = service != 0, active != 0
	p.LowStock = !p.IsService && p.Stock <= p.ReorderPoint
	return p, err
}

func (s *Store) ListProducts(ctx context.Context, user User, values url.Values) ([]Product, error) {
	query := "SELECT " + productColumns + " FROM products WHERE store_id=?"
	args := []any{user.StoreID}
	if values.Get("status") == "active" {
		query += " AND is_active=1"
	}
	if search := strings.TrimSpace(values.Get("search")); search != "" {
		query += " AND (name LIKE ? OR sku LIKE ? OR barcode LIKE ?)"
		like := "%" + search + "%"
		args = append(args, like, like, like)
	}
	if ids := strings.TrimSpace(values.Get("ids")); ids != "" {
		want := strings.Split(ids, ",")
		marks := make([]string, len(want))
		for i, id := range want {
			marks[i] = "?"
			args = append(args, strings.TrimSpace(id))
		}
		query += " AND id IN (" + strings.Join(marks, ",") + ")"
	}
	query += " ORDER BY name COLLATE NOCASE LIMIT 500"
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	products := make([]Product, 0)
	for rows.Next() {
		p, err := scanProduct(rows)
		if err != nil {
			return nil, err
		}
		products = append(products, p)
	}
	return products, rows.Err()
}

func (s *Store) Product(ctx context.Context, user User, id string) (Product, error) {
	p, err := scanProduct(s.db.QueryRowContext(ctx, "SELECT "+productColumns+" FROM products WHERE id=? AND store_id=?", id, user.StoreID))
	if errors.Is(err, sql.ErrNoRows) {
		return Product{}, ErrNotFound
	}
	return p, err
}

func (s *Store) ProductByBarcode(ctx context.Context, user User, barcode string) (Product, error) {
	p, err := scanProduct(s.db.QueryRowContext(ctx, "SELECT "+productColumns+" FROM products WHERE barcode=? AND store_id=? AND is_active=1", barcode, user.StoreID))
	if errors.Is(err, sql.ErrNoRows) {
		return Product{}, ErrNotFound
	}
	return p, err
}

func (s *Store) CreateProduct(ctx context.Context, user User, input ProductInput) (Product, error) {
	if input.Name == nil || strings.TrimSpace(*input.Name) == "" || input.SellingPrice == nil || *input.SellingPrice < 0 {
		return Product{}, errors.New("name and a non-negative selling_price are required")
	}
	p := Product{ID: newID(), StoreID: user.StoreID, Name: strings.TrimSpace(*input.Name), Unit: valueOr(input.Unit, "unit"), SKU: valueOr(input.SKU, ""), Barcode: valueOr(input.Barcode, ""), SellingPrice: *input.SellingPrice, IsActive: true, TaxRate: 18}
	if input.CostPrice != nil {
		p.CostPrice = *input.CostPrice
	}
	if input.Stock != nil {
		p.Stock = *input.Stock
	}
	if input.ReorderPoint != nil {
		p.ReorderPoint = *input.ReorderPoint
	}
	if input.CategoryID != nil {
		p.CategoryID = *input.CategoryID
	}
	if input.Image != nil {
		p.Image = *input.Image
	}
	if input.Description != nil {
		p.Description = *input.Description
	}
	if input.IsService != nil {
		p.IsService = *input.IsService
	}
	if input.IsActive != nil {
		p.IsActive = *input.IsActive
	}
	if input.TaxRate != nil {
		p.TaxRate = *input.TaxRate
	}
	if p.Stock < 0 || p.CostPrice < 0 || p.ReorderPoint < 0 || p.TaxRate < 0 || p.TaxRate > 100 {
		return Product{}, errors.New("stock, cost, reorder point, or tax rate is outside the allowed range")
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	_, err := s.db.ExecContext(ctx, `INSERT INTO products(id,store_id,name,unit,sku,barcode,selling_price,cost_price,stock,reorder_point,category_id,image,description,is_service,is_active,tax_rate,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		p.ID, p.StoreID, p.Name, p.Unit, p.SKU, p.Barcode, p.SellingPrice, p.CostPrice, p.Stock, p.ReorderPoint, nullIfBlank(p.CategoryID), p.Image, p.Description, boolInt(p.IsService), boolInt(p.IsActive), p.TaxRate, now, now)
	if err != nil {
		return Product{}, err
	}
	return p, nil
}

func (s *Store) UpdateProduct(ctx context.Context, user User, id string, input ProductInput) (Product, error) {
	p, err := s.Product(ctx, user, id)
	if err != nil {
		return Product{}, err
	}
	if input.Name != nil {
		p.Name = strings.TrimSpace(*input.Name)
	}
	if input.Unit != nil {
		p.Unit = *input.Unit
	}
	if input.SKU != nil {
		p.SKU = *input.SKU
	}
	if input.Barcode != nil {
		p.Barcode = *input.Barcode
	}
	if input.SellingPrice != nil {
		p.SellingPrice = *input.SellingPrice
	}
	if input.CostPrice != nil {
		p.CostPrice = *input.CostPrice
	}
	if input.ReorderPoint != nil {
		p.ReorderPoint = *input.ReorderPoint
	}
	if input.CategoryID != nil {
		p.CategoryID = *input.CategoryID
	}
	if input.Image != nil {
		p.Image = *input.Image
	}
	if input.Description != nil {
		p.Description = *input.Description
	}
	if input.IsService != nil {
		p.IsService = *input.IsService
	}
	if input.IsActive != nil {
		p.IsActive = *input.IsActive
	}
	if input.TaxRate != nil {
		p.TaxRate = *input.TaxRate
	}
	if p.Name == "" || p.SellingPrice < 0 || p.CostPrice < 0 || p.ReorderPoint < 0 || p.TaxRate < 0 || p.TaxRate > 100 {
		return Product{}, errors.New("product fields are outside the allowed range")
	}
	_, err = s.db.ExecContext(ctx, `UPDATE products SET name=?,unit=?,sku=?,barcode=?,selling_price=?,cost_price=?,reorder_point=?,category_id=?,image=?,description=?,is_service=?,is_active=?,tax_rate=?,updated_at=? WHERE id=? AND store_id=?`,
		p.Name, p.Unit, p.SKU, p.Barcode, p.SellingPrice, p.CostPrice, p.ReorderPoint, nullIfBlank(p.CategoryID), p.Image, p.Description, boolInt(p.IsService), boolInt(p.IsActive), p.TaxRate, time.Now().UTC().Format(time.RFC3339Nano), p.ID, user.StoreID)
	if err != nil {
		return Product{}, err
	}
	return s.Product(ctx, user, id)
}

func (s *Store) ArchiveProduct(ctx context.Context, user User, id string) error {
	result, err := s.db.ExecContext(ctx, "UPDATE products SET is_active=0,updated_at=? WHERE id=? AND store_id=?", time.Now().UTC().Format(time.RFC3339Nano), id, user.StoreID)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err == nil && count == 0 {
		return ErrNotFound
	}
	return err
}

func (s *Store) AdjustStock(ctx context.Context, user User, id string, change int64) (Product, error) {
	if change == 0 {
		return Product{}, errors.New("qty_change must not be zero")
	}
	result, err := s.db.ExecContext(ctx, "UPDATE products SET stock=stock+?,updated_at=? WHERE id=? AND store_id=? AND is_service=0 AND stock+?>=0", change, time.Now().UTC().Format(time.RFC3339Nano), id, user.StoreID, change)
	if err != nil {
		return Product{}, err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return Product{}, err
	}
	if count == 0 {
		if _, err := s.Product(ctx, user, id); errors.Is(err, ErrNotFound) {
			return Product{}, ErrNotFound
		}
		return Product{}, ErrInsufficientQty
	}
	return s.Product(ctx, user, id)
}

type SaleLineInput struct {
	ProductID string `json:"product_id"`
	Qty       int64  `json:"qty"`
}
type SaleInput struct {
	ExpectedStoreID string          `json:"expected_store_id"`
	IdempotencyKey  string          `json:"idempotency_key"`
	PaymentMethod   string          `json:"payment_method"`
	CustomerName    string          `json:"customer_name"`
	CustomerPhone   string          `json:"customer_phone"`
	DiscountPct     float64         `json:"discount_pct"`
	Lines           []SaleLineInput `json:"lines"`
}

type Transaction struct {
	ID             string            `json:"id"`
	TxnNumber      string            `json:"txn_number"`
	PaymentMethod  string            `json:"payment_method"`
	Status         string            `json:"status"`
	Subtotal       int64             `json:"subtotal"`
	DiscountAmount int64             `json:"discount_amount"`
	TaxAmount      int64             `json:"tax_amount"`
	Total          int64             `json:"total"`
	Profit         int64             `json:"profit"`
	CustomerName   string            `json:"customer_name"`
	CustomerPhone  string            `json:"customer_phone,omitempty"`
	CreatedAt      string            `json:"created_at"`
	SyncStatus     string            `json:"sync_status"`
	Lines          []TransactionLine `json:"lines,omitempty"`
}

type TransactionLine struct {
	ID          string `json:"id"`
	ProductID   string `json:"product_id"`
	ProductName string `json:"product_name"`
	Qty         int64  `json:"qty"`
	UnitPrice   int64  `json:"unit_price"`
	Total       int64  `json:"total"`
}

func (s *Store) CompleteSale(ctx context.Context, user User, input SaleInput) (Transaction, error) {
	if input.ExpectedStoreID != "" && input.ExpectedStoreID != user.StoreID {
		return Transaction{}, errors.New("active store changed; refresh checkout before completing the sale")
	}
	if !strings.EqualFold(strings.TrimSpace(input.PaymentMethod), "cash") {
		return Transaction{}, errors.New("offline checkout currently supports cash only")
	}
	if len(input.Lines) == 0 {
		return Transaction{}, errors.New("add at least one product to the sale")
	}
	if math.IsNaN(input.DiscountPct) || math.IsInf(input.DiscountPct, 0) || input.DiscountPct < 0 || input.DiscountPct > 100 {
		return Transaction{}, errors.New("discount_pct must be between 0 and 100")
	}
	if strings.TrimSpace(input.IdempotencyKey) == "" {
		return Transaction{}, errors.New("idempotency_key is required for local checkout")
	}
	// Combine repeated product IDs before checking stock and writing the event.
	quantities := map[string]int64{}
	for _, line := range input.Lines {
		if strings.TrimSpace(line.ProductID) == "" || line.Qty <= 0 {
			return Transaction{}, errors.New("every sale line needs a product_id and positive qty")
		}
		quantities[line.ProductID] += line.Qty
		if quantities[line.ProductID] <= 0 {
			return Transaction{}, errors.New("sale quantity is too large")
		}
	}
	productIDs := make([]string, 0, len(quantities))
	for id := range quantities {
		productIDs = append(productIDs, id)
	}
	sort.Strings(productIDs)
	fingerprintLines := make([]map[string]any, 0, len(productIDs))
	for _, id := range productIDs {
		fingerprintLines = append(fingerprintLines, map[string]any{"product_id": id, "qty": quantities[id]})
	}
	fingerprintBytes, err := json.Marshal(map[string]any{
		"store_id":       user.StoreID,
		"payment_method": "Cash",
		"customer_name":  strings.TrimSpace(input.CustomerName),
		"customer_phone": strings.TrimSpace(input.CustomerPhone),
		"discount_pct":   input.DiscountPct,
		"lines":          fingerprintLines,
	})
	if err != nil {
		return Transaction{}, err
	}
	requestHash := hashToken(string(fingerprintBytes))
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Transaction{}, err
	}
	defer tx.Rollback()
	if existingID, err := findSaleID(ctx, tx, input.IdempotencyKey, user.StoreID, requestHash); err == nil {
		if err := tx.Commit(); err != nil {
			return Transaction{}, err
		}
		return s.Transaction(ctx, user, existingID)
	} else if !errors.Is(err, sql.ErrNoRows) {
		return Transaction{}, err
	}

	products := make([]Product, 0, len(productIDs))
	var subtotal, costTotal int64
	for _, productID := range productIDs {
		p, err := scanProduct(tx.QueryRowContext(ctx, "SELECT "+productColumns+" FROM products WHERE id=? AND store_id=? AND is_active=1", productID, user.StoreID))
		if errors.Is(err, sql.ErrNoRows) {
			return Transaction{}, errors.New("one or more products were not found in this store")
		}
		if err != nil {
			return Transaction{}, err
		}
		qty := quantities[productID]
		if !p.IsService && p.Stock < qty {
			return Transaction{}, fmt.Errorf("%w: %s", ErrInsufficientQty, p.Name)
		}
		lineTotal := p.SellingPrice * qty
		if qty != 0 && lineTotal/qty != p.SellingPrice {
			return Transaction{}, errors.New("sale total exceeds supported amount")
		}
		subtotal += lineTotal
		costTotal += p.CostPrice * qty
		products = append(products, p)
	}
	discount := int64(math.Floor(float64(subtotal)*input.DiscountPct/100 + 1e-9))
	tax := int64(math.Floor(float64(subtotal-discount)*18/100 + 1e-9))
	total := subtotal - discount + tax
	profit := subtotal - costTotal - discount
	now := time.Now().UTC().Format(time.RFC3339Nano)
	if _, err := tx.ExecContext(ctx, "UPDATE stores SET next_txn_number=next_txn_number+1 WHERE id=?", user.StoreID); err != nil {
		return Transaction{}, err
	}
	var sequence int64
	if err := tx.QueryRowContext(ctx, "SELECT next_txn_number FROM stores WHERE id=?", user.StoreID).Scan(&sequence); err != nil {
		return Transaction{}, err
	}
	saleID, txnNumber := newID(), fmt.Sprintf("LOCAL-%06d", sequence)
	customerName := strings.TrimSpace(input.CustomerName)
	if customerName == "" {
		customerName = "Walk-in customer"
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO sales(id,store_id,user_id,txn_number,idempotency_key,request_hash,payment_method,status,subtotal,discount_amount,tax_amount,total,profit,customer_name,customer_phone,created_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, saleID, user.StoreID, user.ID, txnNumber, input.IdempotencyKey, requestHash, "Cash", "paid", subtotal, discount, tax, total, profit, customerName, strings.TrimSpace(input.CustomerPhone), now); err != nil {
		return Transaction{}, err
	}
	lines := make([]TransactionLine, 0, len(products))
	for _, p := range products {
		qty := quantities[p.ID]
		lineTotal := p.SellingPrice * qty
		line := TransactionLine{ID: newID(), ProductID: p.ID, ProductName: p.Name, Qty: qty, UnitPrice: p.SellingPrice, Total: lineTotal}
		if _, err := tx.ExecContext(ctx, "INSERT INTO sale_lines(id,sale_id,product_id,product_name,qty,unit_price,total) VALUES(?,?,?,?,?,?,?)", line.ID, saleID, line.ProductID, line.ProductName, line.Qty, line.UnitPrice, line.Total); err != nil {
			return Transaction{}, err
		}
		if !p.IsService {
			result, err := tx.ExecContext(ctx, "UPDATE products SET stock=stock-?,updated_at=? WHERE id=? AND store_id=? AND is_active=1 AND stock>=?", qty, now, p.ID, user.StoreID, qty)
			if err != nil {
				return Transaction{}, err
			}
			affected, err := result.RowsAffected()
			if err != nil {
				return Transaction{}, err
			}
			if affected != 1 {
				return Transaction{}, fmt.Errorf("%w: %s", ErrInsufficientQty, p.Name)
			}
		}
		lines = append(lines, line)
	}
	eventID := newID()
	eventPayload, err := json.Marshal(map[string]any{
		"event_id":        eventID,
		"event_type":      "sale.created",
		"schema_version":  1,
		"created_at":      now,
		"device_store_id": user.StoreID,
		"idempotency_key": input.IdempotencyKey,
		"sale":            map[string]any{"id": saleID, "txn_number": txnNumber, "payment_method": "Cash", "subtotal": subtotal, "discount_amount": discount, "tax_amount": tax, "total": total, "lines": lines},
	})
	if err != nil {
		return Transaction{}, err
	}
	if _, err := tx.ExecContext(ctx, "INSERT INTO outbox(id,event_type,payload,status,created_at) VALUES(?,?,?,?,?)", eventID, "sale.created", string(eventPayload), "pending", now); err != nil {
		return Transaction{}, err
	}
	if err := tx.Commit(); err != nil {
		return Transaction{}, err
	}
	return Transaction{ID: saleID, TxnNumber: txnNumber, PaymentMethod: "Cash", Status: "paid", Subtotal: subtotal, DiscountAmount: discount, TaxAmount: tax, Total: total, Profit: profit, CustomerName: customerName, CustomerPhone: strings.TrimSpace(input.CustomerPhone), CreatedAt: now, SyncStatus: "pending", Lines: lines}, nil
}

type queryRower interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func findSaleID(ctx context.Context, q queryRower, idempotencyKey, storeID, requestHash string) (string, error) {
	var id string
	var storedHash string
	err := q.QueryRowContext(ctx, "SELECT id,request_hash FROM sales WHERE idempotency_key=? AND store_id=?", idempotencyKey, storeID).Scan(&id, &storedHash)
	if err == nil && storedHash != requestHash {
		return "", ErrIdempotencyConflict
	}
	return id, err
}

type queryer interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func (s *Store) Transaction(ctx context.Context, user User, id string) (Transaction, error) {
	return loadTransaction(ctx, s.db, user.StoreID, id)
}
func loadTransaction(ctx context.Context, q queryer, storeID, id string) (Transaction, error) {
	var t Transaction
	err := q.QueryRowContext(ctx, `SELECT id,txn_number,payment_method,status,subtotal,discount_amount,tax_amount,total,profit,customer_name,customer_phone,created_at FROM sales WHERE id=? AND store_id=?`, id, storeID).Scan(&t.ID, &t.TxnNumber, &t.PaymentMethod, &t.Status, &t.Subtotal, &t.DiscountAmount, &t.TaxAmount, &t.Total, &t.Profit, &t.CustomerName, &t.CustomerPhone, &t.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Transaction{}, ErrNotFound
	}
	if err != nil {
		return Transaction{}, err
	}
	t.SyncStatus = "pending"
	rows, err := q.QueryContext(ctx, "SELECT id,product_id,product_name,qty,unit_price,total FROM sale_lines WHERE sale_id=? ORDER BY rowid", id)
	if err != nil {
		return Transaction{}, err
	}
	defer rows.Close()
	t.Lines = make([]TransactionLine, 0)
	for rows.Next() {
		var line TransactionLine
		if err := rows.Scan(&line.ID, &line.ProductID, &line.ProductName, &line.Qty, &line.UnitPrice, &line.Total); err != nil {
			return Transaction{}, err
		}
		t.Lines = append(t.Lines, line)
	}
	return t, rows.Err()
}

func (s *Store) ListTransactions(ctx context.Context, user User, limit int) ([]Transaction, error) {
	if limit < 1 || limit > 500 {
		limit = 100
	}
	rows, err := s.db.QueryContext(ctx, "SELECT id FROM sales WHERE store_id=? ORDER BY created_at DESC LIMIT ?", user.StoreID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := make([]string, 0)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	transactions := make([]Transaction, 0, len(ids))
	for _, id := range ids {
		t, err := loadTransaction(ctx, s.db, user.StoreID, id)
		if err != nil {
			return nil, err
		}
		transactions = append(transactions, t)
	}
	return transactions, nil
}

func (s *Store) OutboxCount(ctx context.Context) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM outbox WHERE status='pending'").Scan(&count)
	return count, err
}

func normalizePhone(raw string) (string, error) {
	digits := regexp.MustCompile(`\D`).ReplaceAllString(raw, "")
	if strings.HasPrefix(digits, "255") && len(digits) == 12 {
		digits = "0" + digits[3:]
	}
	if len(digits) == 9 {
		digits = "0" + digits
	}
	if len(digits) != 10 || digits[0] != '0' {
		return "", errors.New("enter a valid Tanzanian mobile phone number")
	}
	return digits, nil
}
func newID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic("cryptographic random source unavailable: " + err.Error())
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	encoded := hex.EncodeToString(b[:])
	return encoded[:8] + "-" + encoded[8:12] + "-" + encoded[12:16] + "-" + encoded[16:20] + "-" + encoded[20:]
}
func randomToken() (string, error) {
	var bytes [32]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes[:]), nil
}
func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
func valueOr(value *string, fallback string) string {
	if value == nil {
		return fallback
	}
	return strings.TrimSpace(*value)
}
func nullIfBlank(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}
func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
