package inventory

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/troubleman96/ziadapos/internal/apierr"
	"github.com/troubleman96/ziadapos/internal/cache"
	"github.com/troubleman96/ziadapos/internal/events"
)

// Stock adjustment reasons for manual changes.
const (
	ReasonRestock    = "restock"
	ReasonDamage     = "damage"
	ReasonCorrection = "correction"
	ReasonTheft      = "theft"
	ReasonSale       = "sale"
	ReasonRefund     = "refund"
)

// Service implements inventory business rules with Redis caching.
type Service struct {
	repo  *Repo
	cache *cache.Cache
	bus   *events.Bus
}

// NewService wires an inventory Service.
func NewService(repo *Repo, cache *cache.Cache, bus *events.Bus) *Service {
	return &Service{repo: repo, cache: cache, bus: bus}
}

func (s *Service) categoryKey(orgID uuid.UUID) string { return "inv:categories:" + orgID.String() }
func (s *Service) productKey(id uuid.UUID) string     { return "inv:product:" + id.String() }
func (s *Service) lowStockKey(storeID uuid.UUID) string {
	return "inv:lowstock:" + storeID.String()
}
func (s *Service) barcodeKey(storeID uuid.UUID, code string) string {
	return "inv:barcode:" + storeID.String() + ":" + code
}

func (s *Service) invalidate(storeID, orgID *uuid.UUID, productIDs ...uuid.UUID) {
	if s.cache == nil {
		return
	}
	ctx := context.Background()
	if storeID != nil {
		_ = s.cache.DeletePattern(ctx, "inv:products:"+storeID.String()+":*")
		_ = s.cache.DeletePattern(ctx, "inv:barcode:"+storeID.String()+":*")
		_ = s.cache.Del(ctx, s.lowStockKey(*storeID))
	}
	if orgID != nil {
		_ = s.cache.Del(ctx, s.categoryKey(*orgID))
	}
	for _, id := range productIDs {
		_ = s.cache.Del(ctx, s.productKey(id))
	}
}

// InvalidateProductCache is exported for other apps (e.g. transactions) to
// evict product/low-stock/category entries after they mutate stock.
func (s *Service) InvalidateProductCache(storeID, orgID *uuid.UUID, productIDs ...uuid.UUID) {
	s.invalidate(storeID, orgID, productIDs...)
}

// ── Category operations ─────────────────────────────────────────────────────

// CreateCategory adds a category, refusing duplicate names (case-insensitive).
func (s *Service) CreateCategory(ctx context.Context, orgID uuid.UUID, name, description string) (*Category, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, apierr.BadRequest("Category name is required")
	}
	existing, err := s.repo.FindCategoryByName(orgID, name)
	if err == nil && existing != nil {
		return nil, apierr.Conflict("A category with this name already exists")
	} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	cat := &Category{OrganisationID: orgID, Name: name, Description: description, IsActive: true}
	if err := s.repo.CreateCategory(cat); err != nil {
		return nil, err
	}
	s.invalidate(nil, &orgID)
	s.bus.Publish("category.created", map[string]interface{}{"category_id": cat.ID, "org_id": orgID})
	return cat, nil
}

// ListCategories returns an organisation's categories with product counts,
// cached in Redis for 5 minutes.
func (s *Service) ListCategories(ctx context.Context, orgID uuid.UUID) ([]Category, error) {
	var cats []Category
	key := s.categoryKey(orgID)
	if s.cache != nil {
		if err := s.cache.GetJSON(ctx, key, &cats); err == nil && cats != nil {
			return cats, nil
		}
	}
	cats, err := s.repo.ListCategories(orgID)
	if err != nil {
		return nil, err
	}
	if s.cache != nil {
		_ = s.cache.SetJSONTTL(ctx, key, cats, 5*time.Minute)
	}
	return cats, nil
}

// UpdateCategory edits name/description/sort order.
func (s *Service) UpdateCategory(ctx context.Context, orgID, id uuid.UUID, name, description string, sortOrder int, isActive *bool) (*Category, error) {
	cat, err := s.repo.GetCategory(orgID, id)
	if err != nil {
		return nil, apierr.NotFound("Category not found")
	}
	if strings.TrimSpace(name) != "" {
		cat.Name = strings.TrimSpace(name)
	}
	cat.Description = description
	cat.SortOrder = sortOrder
	if isActive != nil {
		cat.IsActive = *isActive
	}
	if err := s.repo.UpdateCategory(cat); err != nil {
		return nil, err
	}
	s.invalidate(nil, &orgID)
	return cat, nil
}

// DeleteCategory removes a category if no active products reference it.
func (s *Service) DeleteCategory(ctx context.Context, orgID, id uuid.UUID) error {
	var count int64
	if err := s.repo.db.Model(&Product{}).
		Where("category_id = ? AND is_active = true", id).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return apierr.Conflict("Category still has active products; move or deactivate them first")
	}
	if err := s.repo.DeleteCategory(orgID, id); err != nil {
		return apierr.NotFound("Category not found")
	}
	s.invalidate(nil, &orgID)
	return nil
}

// ── Product operations ──────────────────────────────────────────────────────

// CreateProduct adds a product to a store.
func (s *Service) CreateProduct(ctx context.Context, storeID uuid.UUID, in *ProductInput) (*Product, error) {
	orgID, _ := s.repo.StoreOrgID(storeID)
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, apierr.BadRequest("Product name is required")
	}
	p := &Product{
		StoreID:      storeID,
		Name:         name,
		SKU:          optStr(in.SKU),
		Barcode:      optStr(in.Barcode),
		CostPrice:    in.CostPrice,
		SellingPrice: in.SellingPrice,
		Stock:        in.Stock,
		ReorderPoint: in.ReorderPoint,
		Image:        in.Image,
		TaxRate:      in.TaxRate,
		Description:  in.Description,
		IsActive:     true,
		IsService:    in.IsService,
	}
	if p.SellingPrice <= 0 {
		return nil, apierr.BadRequest("Selling price must be greater than zero").WithField("selling_price", "required")
	}
	if in.CategoryID != nil {
		if orgID != uuid.Nil {
			if cat, err := s.repo.GetCategory(orgID, *in.CategoryID); err == nil {
				p.CategoryID = in.CategoryID
				p.Category = cat
			}
		}
	}
	if p.ReorderPoint < 0 {
		p.ReorderPoint = 0
	}

	if err := s.repo.WithTx(func(tx *gorm.DB) error {
		if err := tx.Create(p).Error; err != nil {
			return err
		}
		if !p.IsService && p.Stock > 0 {
			adj := &StockAdjustment{
				ProductID: p.ID, QtyChange: p.Stock, Reason: ReasonRestock,
				Note: "Initial stock", CreatedBy: in.CreatedBy,
			}
			return tx.Create(adj).Error
		}
		return nil
	}); err != nil {
		if isUniqueViolation(err) {
			return nil, apierr.Conflict("A product with this SKU or barcode already exists")
		}
		return nil, err
	}
	s.invalidate(&storeID, &orgID, p.ID)
	s.bus.Publish("product.created", map[string]interface{}{"product_id": p.ID, "store_id": storeID})
	return p, nil
}

// ListProducts returns products (with category) for a store, cached.
func (s *Service) ListProducts(ctx context.Context, storeID uuid.UUID, f ProductFilter) (map[string]interface{}, error) {
	if f.Page <= 0 {
		f.Page = 1
	}
	if f.PageSize <= 0 {
		f.PageSize = 20
	}
	if f.PageSize > 200 {
		f.PageSize = 200
	}
	queryKey := fmt.Sprintf("inv:products:%s:p%d:s%d:cat%s:q%s:st%s:lo%t",
		storeID, f.Page, f.PageSize, f.CategoryID, f.Search, f.Status, f.LowStock)

	var out map[string]interface{}
	if s.cache != nil && !f.Minimal {
		if err := s.cache.GetJSON(ctx, queryKey, &out); err == nil && out != nil {
			return out, nil
		}
	}

	products, total, err := s.repo.ListProducts(f)
	if err != nil {
		return nil, err
	}
	items := make([]map[string]interface{}, 0, len(products))
	for i := range products {
		items = append(items, productResponse(&products[i]))
	}
	out = map[string]interface{}{
		"items":     items,
		"total":     total,
		"page":      f.Page,
		"page_size": f.PageSize,
	}
	if f.PageSize > 0 {
		pages := int(total) / f.PageSize
		if int(total)%f.PageSize != 0 {
			pages++
		}
		out["pages"] = pages
	}
	if s.cache != nil && !f.Minimal {
		_ = s.cache.SetJSONTTL(ctx, queryKey, out, 2*time.Minute)
	}
	return out, nil
}

// GetProduct returns one product by id, cached for 5 minutes.
func (s *Service) GetProduct(ctx context.Context, storeID, id uuid.UUID) (*Product, error) {
	var cached Product
	key := s.productKey(id)
	if s.cache != nil {
		if err := s.cache.GetJSON(ctx, key, &cached); err == nil && cached.ID != uuid.Nil {
			return &cached, nil
		}
	}
	p, err := s.repo.GetProduct(storeID, id)
	if err != nil {
		return nil, apierr.NotFound("Product not found")
	}
	if s.cache != nil {
		_ = s.cache.SetJSONTTL(ctx, key, p, 5*time.Minute)
	}
	return p, nil
}

// UpdateProduct edits a product and resets caches.
func (s *Service) UpdateProduct(ctx context.Context, storeID, id uuid.UUID, in *ProductInput) (*Product, error) {
	orgID, _ := s.repo.StoreOrgID(storeID)
	p, err := s.repo.GetProduct(storeID, id)
	if err != nil {
		return nil, apierr.NotFound("Product not found")
	}
	if strings.TrimSpace(in.Name) != "" {
		p.Name = strings.TrimSpace(in.Name)
	}
	p.SKU = optStr(in.SKU)
	p.Barcode = optStr(in.Barcode)
	p.CostPrice = in.CostPrice
	p.SellingPrice = in.SellingPrice
	p.ReorderPoint = in.ReorderPoint
	p.Image = in.Image
	p.TaxRate = in.TaxRate
	p.Description = in.Description
	p.IsService = in.IsService
	if in.CategoryID != nil {
		p.CategoryID = in.CategoryID
	}
	if in.IsActive != nil {
		p.IsActive = *in.IsActive
	}
	if err := s.repo.UpdateProduct(p); err != nil {
		if isUniqueViolation(err) {
			return nil, apierr.Conflict("A product with this SKU or barcode already exists")
		}
		return nil, err
	}
	s.invalidate(&storeID, &orgID, p.ID)
	fresh, err := s.repo.GetProduct(storeID, id)
	if err == nil {
		p = fresh
	}
	return p, nil
}
// DeleteProduct archives a product (soft delete).
func (s *Service) DeleteProduct(ctx context.Context, storeID, id uuid.UUID) error {
	orgID, _ := s.repo.StoreOrgID(storeID)
	if err := s.repo.SoftDeleteProduct(storeID, id); err != nil {
		return apierr.NotFound("Product not found")
	}
	s.invalidate(&storeID, &orgID, id)
	s.bus.Publish("product.deleted", map[string]interface{}{"product_id": id, "store_id": storeID})
	return nil
}

// ByBarcode looks up a product by barcode, cached for 5 minutes.
func (s *Service) ByBarcode(ctx context.Context, storeID uuid.UUID, code string) (*Product, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return nil, apierr.BadRequest("barcode is required")
	}
	key := s.barcodeKey(storeID, code)
	if s.cache != nil {
		var cached Product
		if err := s.cache.GetJSON(ctx, key, &cached); err == nil && cached.ID != uuid.Nil {
			return &cached, nil
		}
	}
	p, err := s.repo.GetProductByBarcode(storeID, code)
	if err != nil {
		return nil, apierr.NotFound("No product found for this barcode")
	}
	if s.cache != nil {
		_ = s.cache.SetJSONTTL(ctx, key, p, 5*time.Minute)
	}
	return p, nil
}

// LowStock lists products at or below their reorder point, cached 1 minute.
func (s *Service) LowStock(ctx context.Context, storeID uuid.UUID) ([]map[string]interface{}, error) {
	key := s.lowStockKey(storeID)
	var items []map[string]interface{}
	if s.cache != nil {
		if err := s.cache.GetJSON(ctx, key, &items); err == nil && items != nil {
			return items, nil
		}
	}
	products, err := s.repo.LowStockProducts(storeID)
	if err != nil {
		return nil, err
	}
	items = make([]map[string]interface{}, 0, len(products))
	for i := range products {
		items = append(items, productResponse(&products[i]))
	}
	if s.cache != nil {
		_ = s.cache.SetJSONTTL(ctx, key, items, time.Minute)
	}
	return items, nil
}

// AdjustStock applies a manual stock change and records the reason trail.
func (s *Service) AdjustStock(ctx context.Context, storeID, productID, userID uuid.UUID, qtyChange int, reason, note string) (*Product, error) {
	if qtyChange == 0 {
		return nil, apierr.BadRequest("qty_change cannot be zero")
	}
	switch reason {
	case ReasonRestock, ReasonDamage, ReasonCorrection, ReasonTheft:
	default:
		return nil, apierr.BadRequest("Invalid reason").WithField("reason", "must be restock, damage, correction or theft")
	}

	// sale/refund adjustments are recorded by the transactions app.
	p, err := s.repo.GetProduct(storeID, productID)
	if err != nil {
		return nil, apierr.NotFound("Product not found")
	}
	if p.IsService {
		return nil, apierr.BadRequest("Service products do not hold stock")
	}

	var result *Product
	err = s.repo.WithTx(func(tx *gorm.DB) error {
		var locked Product
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND store_id = ?", productID, storeID).First(&locked).Error; err != nil {
			return err
		}
		newStock := locked.Stock + qtyChange
		if newStock < 0 {
			return apierr.Conflict("Insufficient stock").WithField("stock", fmt.Sprintf("only %d units available", locked.Stock))
		}
		if err := tx.Model(&Product{}).Where("id = ?", productID).Update("stock", newStock).Error; err != nil {
			return err
		}
		adj := &StockAdjustment{
			ProductID: productID, QtyChange: qtyChange, Reason: reason,
			Note: note, CreatedBy: userID,
		}
		if err := tx.Create(adj).Error; err != nil {
			return err
		}
		locked.Stock = newStock
		result = &locked
		return nil
	})
	if err != nil {
		return nil, err
	}
	s.invalidate(&storeID, nil, productID)
	s.bus.Publish("stock.adjusted", map[string]interface{}{
		"product_id": productID, "store_id": storeID, "qty_change": qtyChange, "reason": reason,
	})
	return result, nil
}

// ── DTOs ────────────────────────────────────────────────────────────────────

// ProductInput is the create/update payload.
type ProductInput struct {
	Name         string    `json:"name" binding:"required"`
	CategoryID   *uuid.UUID `json:"category_id"`
	SKU          string    `json:"sku"`
	Barcode      string    `json:"barcode"`
	CostPrice    int64     `json:"cost_price"`
	SellingPrice int64     `json:"selling_price" binding:"required"`
	Stock        int       `json:"stock"`
	ReorderPoint int       `json:"reorder_point"`
	Image        string    `json:"image"`
	TaxRate      float64   `json:"tax_rate"`
	Description  string    `json:"description"`
	IsService    bool      `json:"is_service"`
	IsActive     *bool     `json:"is_active"`
	CreatedBy    uuid.UUID `json:"-"`
}

func productResponse(p *Product) map[string]interface{} {
	var cat map[string]interface{}
	if p.Category != nil {
		cat = map[string]interface{}{
			"id": p.Category.ID, "name": p.Category.Name,
		}
	}
	sku, barcode := "", ""
	if p.SKU != nil {
		sku = *p.SKU
	}
	if p.Barcode != nil {
		barcode = *p.Barcode
	}
	return map[string]interface{}{
		"id":            p.ID,
		"store_id":      p.StoreID,
		"category_id":   p.CategoryID,
		"category":      cat,
		"name":          p.Name,
		"sku":           sku,
		"barcode":       barcode,
		"cost_price":    p.CostPrice,
		"selling_price": p.SellingPrice,
		"stock":         p.Stock,
		"reorder_point": p.ReorderPoint,
		"low_stock":     p.IsLowStock(),
		"image":         p.Image,
		"tax_rate":      p.TaxRate,
		"description":   p.Description,
		"is_active":     p.IsActive,
		"is_service":    p.IsService,
		"created_at":    p.CreatedAt,
	}
}

func isUniqueViolation(err error) bool {
	return err != nil && (strings.Contains(strings.ToLower(err.Error()), "duplicate") ||
		strings.Contains(strings.ToLower(err.Error()), "unique"))
}

// optStr normalises an optional string: empty input becomes NULL so Postgres
// unique indexes treat missing values as distinct.
func optStr(s string) *string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return &s
}
