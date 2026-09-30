package inventory

import (
	"context"
	cryptorand "crypto/rand"
	"errors"
	"fmt"
	"math"
	"strconv"
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
	ReasonDisposal   = "disposal"
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
	if err := s.ensureBusinessDefaults(ctx, orgID); err != nil {
		return nil, err
	}
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
		name = strings.TrimSpace(name)
		if cat.Name != name {
			if existing, err := s.repo.FindCategoryByName(orgID, name); err == nil && existing.ID != cat.ID {
				return nil, apierr.Conflict("A category with this name already exists")
			} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, err
			}
			cat.IsDefault = false
			cat.Name = name
		}
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
		Where("category_id = ?", id).Count(&count).Error; err != nil {
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

// CreateUnit adds a reusable organisation-level product unit.
func (s *Service) CreateUnit(ctx context.Context, orgID uuid.UUID, name, description string) (*Unit, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, apierr.BadRequest("Unit name is required")
	}
	if existing, err := s.repo.FindUnitByName(orgID, name); err == nil && existing != nil {
		return nil, apierr.Conflict("A unit with this name already exists")
	} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	unit := &Unit{OrganisationID: orgID, Name: name, Description: strings.TrimSpace(description), IsActive: true}
	if err := s.repo.CreateUnit(unit); err != nil {
		return nil, err
	}
	return unit, nil
}
func (s *Service) ListUnits(ctx context.Context, orgID uuid.UUID) ([]Unit, error) {
	if err := s.ensureBusinessDefaults(ctx, orgID); err != nil {
		return nil, err
	}
	return s.repo.ListUnits(orgID)
}
func (s *Service) UpdateUnit(ctx context.Context, orgID, id uuid.UUID, name, description string, isActive *bool) (*Unit, error) {
	unit, err := s.repo.GetUnit(orgID, id)
	if err != nil {
		return nil, apierr.NotFound("Unit not found")
	}
	oldName := unit.Name
	if name = strings.TrimSpace(name); name != "" && !strings.EqualFold(name, unit.Name) {
		if existing, err := s.repo.FindUnitByName(orgID, name); err == nil && existing != nil {
			return nil, apierr.Conflict("A unit with this name already exists")
		} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		unit.Name = name
		unit.IsDefault = false
	}
	unit.Description = strings.TrimSpace(description)
	if isActive != nil {
		unit.IsActive = *isActive
	}
	if err := s.repo.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(unit).Error; err != nil {
			return err
		}
		if oldName == unit.Name {
			return nil
		}
		storeIDs := tx.Table("stores").Select("id").Where("organisation_id = ?", orgID)
		if err := tx.Model(&Product{}).Where("store_id IN (?) AND lower(unit) = lower(?)", storeIDs, oldName).
			Update("unit", unit.Name).Error; err != nil {
			return err
		}
		return tx.Table("medicine_profiles").Where("store_id IN (?) AND lower(unit) = lower(?)", storeIDs, oldName).
			Update("unit", unit.Name).Error
	}); err != nil {
		return nil, err
	}
	return unit, nil
}
func (s *Service) DeleteUnit(ctx context.Context, orgID, id uuid.UUID) error {
	unit, err := s.repo.GetUnit(orgID, id)
	if err != nil {
		return apierr.NotFound("Unit not found")
	}
	var productUsage, medicineUsage int64
	if err := s.repo.db.WithContext(ctx).Table("products AS p").
		Joins("JOIN stores ON stores.id = p.store_id").
		Where("stores.organisation_id = ? AND lower(p.unit) = lower(?)", orgID, unit.Name).
		Count(&productUsage).Error; err != nil {
		return err
	}
	if err := s.repo.db.WithContext(ctx).Table("medicine_profiles AS mp").
		Joins("JOIN stores ON stores.id = mp.store_id").
		Where("stores.organisation_id = ? AND lower(mp.unit) = lower(?)", orgID, unit.Name).
		Count(&medicineUsage).Error; err != nil {
		return err
	}
	if productUsage > 0 || medicineUsage > 0 {
		return apierr.Conflict("Unit is used by products; change their unit before deleting it")
	}
	return s.repo.DeleteUnit(orgID, id)
}

// ── Product operations ──────────────────────────────────────────────────────

func generateOpeningBatchNumber() (string, error) {
	const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	var randomBytes [8]byte
	if _, err := cryptorand.Read(randomBytes[:]); err != nil {
		return "", err
	}

	var suffix [8]byte
	for i, value := range randomBytes {
		suffix[i] = alphabet[value&31]
	}
	return "OPN-" + string(suffix[:]), nil
}

// CreateProduct adds a product to a store.
func (s *Service) CreateProduct(ctx context.Context, storeID uuid.UUID, in *ProductInput) (*Product, error) {
	orgID, _ := s.repo.StoreOrgID(storeID)
	var businessType string
	if err := s.repo.db.WithContext(ctx).Table("organisations").Select("business_type").Where("id = ?", orgID).Scan(&businessType).Error; err != nil {
		return nil, err
	}
	var initialExpiry time.Time
	initialBatchNumber := strings.TrimSpace(in.InitialBatchNumber)
	if businessType == "pharmacy" && !in.IsService && in.Stock > 0 {
		if initialBatchNumber == "" {
			var err error
			initialBatchNumber, err = generateOpeningBatchNumber()
			if err != nil {
				return nil, fmt.Errorf("generate opening medicine batch number: %w", err)
			}
		}
		var err error
		initialExpiry, err = ParseMedicineExpiryDate(in.InitialExpiryDate, time.Now())
		if err != nil {
			return nil, err
		}
	}
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, apierr.BadRequest("Product name is required")
	}
	openingStock := in.Stock
	p := &Product{
		StoreID:          storeID,
		Name:             name,
		Unit:             strings.TrimSpace(in.Unit),
		SKU:              optStr(in.SKU),
		Barcode:          optStr(in.Barcode),
		CostPrice:        in.CostPrice,
		SellingPrice:     in.SellingPrice,
		WholesalePrice:   in.WholesalePrice,
		WholesaleMinimum: in.WholesaleMinimum,
		Stock:            in.Stock,
		ReorderPoint:     in.ReorderPoint,
		Image:            in.Image,
		TaxRate:          in.TaxRate,
		Description:      in.Description,
		IsActive:         true,
		IsService:        in.IsService,
	}
	if p.SellingPrice <= 0 {
		return nil, apierr.BadRequest("Selling price must be greater than zero").WithField("selling_price", "required")
	}
	if (businessType == "wholesale" || businessType == "retail_wholesale") && p.WholesalePrice == nil {
		defaultWholesale := p.SellingPrice
		p.WholesalePrice = &defaultWholesale
	}
	if p.WholesalePrice != nil && *p.WholesalePrice <= 0 {
		return nil, apierr.BadRequest("Wholesale price must be greater than zero").WithField("wholesale_price", "invalid")
	}
	if p.WholesaleMinimum <= 0 {
		p.WholesaleMinimum = 1
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
	if businessType == "pharmacy" {
		p.Stock = 0
	}

	if err := s.repo.WithTx(func(tx *gorm.DB) error {
		if err := tx.Create(p).Error; err != nil {
			return err
		}
		if !p.IsService && openingStock > 0 {
			if businessType == "pharmacy" {
				batch := &MedicineBatch{StoreID: storeID, ProductID: p.ID, BatchNumber: initialBatchNumber, Quantity: 0, ExpiryDate: initialExpiry, CostPrice: p.CostPrice}
				if err := tx.Create(batch).Error; err != nil {
					return err
				}
				if err := tx.Create(&MedicineStockMovement{StoreID: storeID, ProductID: p.ID, BatchID: batch.ID, Quantity: openingStock, MovementType: "opening_balance", Reason: "Initial product stock", PerformedBy: in.CreatedBy}).Error; err != nil {
					return err
				}
				if err := ReconcilePharmacyStock(tx, storeID, p.ID, 0); err != nil {
					return err
				}
				p.Stock = openingStock
			}
			adj := &StockAdjustment{
				ProductID: p.ID, QtyChange: openingStock, Reason: ReasonRestock,
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
	orgID, orgErr := s.repo.StoreOrgID(storeID)
	if orgErr != nil {
		return nil, orgErr
	}
	var businessType string
	if err := s.repo.db.WithContext(ctx).Table("organisations").Select("business_type").Where("id = ?", orgID).Scan(&businessType).Error; err != nil {
		return nil, err
	}
	if businessType == "pharmacy" && len(products) > 0 {
		location, err := time.LoadLocation("Africa/Dar_es_Salaam")
		if err != nil {
			location = time.FixedZone("EAT", 3*60*60)
		}
		today := time.Now().In(location).Format("2006-01-02")
		var summaries []struct {
			ProductID     uuid.UUID `gorm:"column:product_id"`
			SellableStock int       `gorm:"column:sellable_stock"`
		}
		if err := s.repo.db.WithContext(ctx).Table("medicine_batches").Select("product_id, COALESCE(SUM(quantity), 0) AS sellable_stock").Where("store_id = ? AND product_id IN ? AND stock_status = ? AND quantity > 0 AND expiry_date >= ?", storeID, productIDs(products), "available", today).Group("product_id").Scan(&summaries).Error; err != nil {
			return nil, err
		}
		sellableByProduct := make(map[uuid.UUID]int, len(summaries))
		for _, summary := range summaries {
			sellableByProduct[summary.ProductID] = summary.SellableStock
		}
		for i, product := range products {
			items[i]["sellable_stock"] = sellableByProduct[product.ID]
			items[i]["low_stock"] = !product.IsService && sellableByProduct[product.ID] <= product.ReorderPoint
		}
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
func (s *Service) UpdateProduct(ctx context.Context, storeID, id uuid.UUID, in *UpdateProductInput) (*Product, error) {
	orgID, _ := s.repo.StoreOrgID(storeID)
	p, err := s.repo.GetProduct(storeID, id)
	if err != nil {
		return nil, apierr.NotFound("Product not found")
	}
	if in.Name != nil {
		name := strings.TrimSpace(*in.Name)
		if name == "" {
			return nil, apierr.BadRequest("Product name is required")
		}
		p.Name = name
	}
	if in.Unit != nil {
		p.Unit = strings.TrimSpace(*in.Unit)
	}
	if in.SKU != nil {
		p.SKU = optStr(*in.SKU)
	}
	if in.Barcode != nil {
		p.Barcode = optStr(*in.Barcode)
	}
	if in.CostPrice != nil {
		p.CostPrice = *in.CostPrice
	}
	if in.SellingPrice != nil {
		p.SellingPrice = *in.SellingPrice
	}
	if in.WholesalePrice != nil {
		p.WholesalePrice = in.WholesalePrice
		if *in.WholesalePrice <= 0 {
			return nil, apierr.BadRequest("Wholesale price must be greater than zero")
		}
	}
	if in.WholesaleMinimum != nil {
		if *in.WholesaleMinimum <= 0 {
			return nil, apierr.BadRequest("Wholesale minimum quantity must be positive")
		}
		p.WholesaleMinimum = *in.WholesaleMinimum
	}
	if in.ReorderPoint != nil {
		p.ReorderPoint = *in.ReorderPoint
	}
	if in.Image != nil {
		p.Image = *in.Image
	}
	if in.TaxRate != nil {
		p.TaxRate = *in.TaxRate
	}
	if in.Description != nil {
		p.Description = *in.Description
	}
	if in.IsService != nil {
		p.IsService = *in.IsService
	}
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
	var businessType string
	if err := s.repo.db.WithContext(ctx).Table("stores").Select("organisations.business_type").Joins("JOIN organisations ON organisations.id = stores.organisation_id").Where("stores.id = ?", storeID).Scan(&businessType).Error; err != nil {
		return nil, err
	}
	if businessType == "pharmacy" {
		pharmacyItems, err := s.pharmacyLowStock(ctx, storeID)
		if err != nil {
			return nil, err
		}
		if s.cache != nil {
			_ = s.cache.SetJSONTTL(ctx, key, pharmacyItems, time.Minute)
		}
		return pharmacyItems, nil
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

type pharmacyStockSummary struct {
	ProductID      uuid.UUID  `gorm:"column:product_id"`
	TrackedStock   int64      `gorm:"column:tracked_stock"`
	SellableStock  int64      `gorm:"column:sellable_stock"`
	ExpiredStock   int64      `gorm:"column:expired_stock"`
	NextExpiryDate *time.Time `gorm:"column:next_expiry_date"`
}

func (s *Service) pharmacyLowStock(ctx context.Context, storeID uuid.UUID) ([]map[string]interface{}, error) {
	location, err := time.LoadLocation("Africa/Dar_es_Salaam")
	if err != nil {
		location = time.FixedZone("EAT", 3*60*60)
	}
	today := time.Now().In(location)
	today = time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, location)
	var products []Product
	if err := s.repo.db.WithContext(ctx).Where("store_id = ? AND is_service = false AND is_active = true", storeID).Preload("Category").Find(&products).Error; err != nil {
		return nil, err
	}
	if len(products) == 0 {
		return []map[string]interface{}{}, nil
	}
	var summaries []pharmacyStockSummary
	if err := s.repo.db.WithContext(ctx).Table("medicine_batches").Select(`
		product_id,
		COALESCE(SUM(quantity), 0) AS tracked_stock,
		COALESCE(SUM(CASE WHEN expiry_date >= ? AND stock_status = 'available' THEN quantity ELSE 0 END), 0) AS sellable_stock,
		COALESCE(SUM(CASE WHEN expiry_date < ? THEN quantity ELSE 0 END), 0) AS expired_stock,
		MIN(CASE WHEN expiry_date >= ? AND quantity > 0 AND stock_status = 'available' THEN expiry_date END) AS next_expiry_date`, today.Format("2006-01-02"), today.Format("2006-01-02"), today.Format("2006-01-02")).
		Where("store_id = ? AND product_id IN ?", storeID, productIDs(products)).Group("product_id").Scan(&summaries).Error; err != nil {
		return nil, err
	}
	byProduct := make(map[uuid.UUID]pharmacyStockSummary, len(summaries))
	for _, summary := range summaries {
		byProduct[summary.ProductID] = summary
	}
	items := make([]map[string]interface{}, 0)
	for i := range products {
		product := &products[i]
		summary := byProduct[product.ID]
		if summary.SellableStock > int64(product.ReorderPoint) {
			continue
		}
		item := productResponse(product)
		item["sellable_stock"] = summary.SellableStock
		item["expired_stock"] = summary.ExpiredStock
		item["untracked_stock"] = max(int64(0), int64(product.Stock)-summary.TrackedStock)
		item["low_stock"] = true
		if summary.NextExpiryDate != nil {
			item["next_expiry_date"] = summary.NextExpiryDate
		}
		items = append(items, item)
	}
	return items, nil
}

func productIDs(products []Product) []uuid.UUID {
	ids := make([]uuid.UUID, len(products))
	for i := range products {
		ids[i] = products[i].ID
	}
	return ids
}

// AdjustStock applies a manual stock change and records the reason trail.
func (s *Service) AdjustStock(ctx context.Context, storeID, productID, userID uuid.UUID, qtyChange int, reason, note string) (*Product, error) {
	if qtyChange == 0 {
		return nil, apierr.BadRequest("qty_change cannot be zero")
	}
	switch reason {
	case ReasonRestock, ReasonDamage, ReasonCorrection, ReasonTheft, ReasonDisposal:
	default:
		return nil, apierr.BadRequest("Invalid reason").WithField("reason", "must be restock, damage, correction, theft or disposal")
	}

	// sale/refund adjustments are recorded by the transactions app.
	p, err := s.repo.GetProduct(storeID, productID)
	if err != nil {
		return nil, apierr.NotFound("Product not found")
	}
	if p.IsService {
		return nil, apierr.BadRequest("Service products do not hold stock")
	}
	orgID, err := s.repo.StoreOrgID(storeID)
	if err != nil {
		return nil, err
	}
	var businessType string
	if err := s.repo.db.WithContext(ctx).Table("organisations").Select("business_type").Where("id = ?", orgID).Scan(&businessType).Error; err != nil {
		return nil, err
	}
	if businessType == "pharmacy" && qtyChange > 0 {
		return nil, apierr.Conflict("Receive positive pharmacy stock with its batch number and expiry date")
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
		if businessType == "pharmacy" {
			remaining := -qtyChange
			var batches []MedicineBatch
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("store_id = ? AND product_id = ? AND quantity > 0", storeID, productID).Order("expiry_date asc, created_at asc").Find(&batches).Error; err != nil {
				return err
			}
			for _, batch := range batches {
				if remaining == 0 {
					break
				}
				used := batch.Quantity
				if used > remaining {
					used = remaining
				}
				if err := tx.Create(&MedicineStockMovement{StoreID: storeID, ProductID: productID, BatchID: batch.ID, Quantity: -used, MovementType: reason, Reason: note, PerformedBy: userID}).Error; err != nil {
					return err
				}
				batchID := batch.ID
				if err := tx.Create(&StockAdjustment{ProductID: productID, MedicineBatchID: &batchID, QtyChange: -used, Reason: reason, Note: note, CreatedBy: userID}).Error; err != nil {
					return err
				}
				remaining -= used
			}
			if remaining > 0 {
				return apierr.Conflict("Not enough batch-tracked stock; assign the remaining stock to a batch first")
			}
			if err := ReconcilePharmacyStock(tx, storeID, productID, 0); err != nil {
				return err
			}
		} else if err := tx.Model(&Product{}).Where("id = ?", productID).Update("stock", newStock).Error; err != nil {
			return err
		}
		if businessType == "pharmacy" {
			locked.Stock = newStock
		} else {
			adj := &StockAdjustment{ProductID: productID, QtyChange: qtyChange, Reason: reason, Note: note, CreatedBy: userID}
			if err := tx.Create(adj).Error; err != nil {
				return err
			}
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

// ── Bulk import ─────────────────────────────────────────────────────────────

// FindOrCreateCategory returns an organisation's category by name, creating it
// (with default properties) when it does not exist yet.
func (s *Service) FindOrCreateCategory(ctx context.Context, orgID uuid.UUID, name string) (*Category, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, apierr.BadRequest("Category name is required")
	}
	cat, err := s.repo.FindCategoryByName(orgID, name)
	if err == nil && cat != nil {
		return cat, nil
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	cat = &Category{OrganisationID: orgID, Name: name, IsActive: true}
	if err := s.repo.CreateCategory(cat); err != nil {
		return nil, err
	}
	s.invalidate(nil, &orgID)
	s.bus.Publish("category.created", map[string]interface{}{"category_id": cat.ID, "org_id": orgID})
	return cat, nil
}

// ImportProducts creates products from parsed CSV rows (header row + data rows).
// Returns a summary the web-app renders: created/failed counts and per-row errors.
func (s *Service) ImportProducts(ctx context.Context, orgID, storeID, userID uuid.UUID, rows [][]string) (map[string]interface{}, error) {
	if len(rows) < 2 {
		return nil, apierr.BadRequest("CSV must have a header row and at least one product")
	}
	idx := map[string]int{}
	for i, h := range rows[0] {
		idx[strings.ToLower(strings.TrimSpace(h))] = i
	}
	col := func(row []string, name string) string {
		i, ok := idx[name]
		if !ok || i >= len(row) {
			return ""
		}
		return strings.TrimSpace(row[i])
	}
	for _, key := range []string{"name", "price"} {
		if _, ok := idx[key]; !ok {
			return nil, apierr.BadRequest("CSV is missing the '" + key + "' column")
		}
	}

	created, failed := 0, 0
	errList := make([]map[string]interface{}, 0)
	recordErr := func(rowNum int, msg string) {
		failed++
		errList = append(errList, map[string]interface{}{"row": rowNum, "message": msg})
	}

	for n := 1; n < len(rows); n++ {
		row := rows[n]
		if len(row) == 1 && strings.TrimSpace(row[0]) == "" {
			continue
		}
		rowNum := n + 1 // human-friendly 1-based row number

		name := col(row, "name")
		if name == "" {
			recordErr(rowNum, "name is required")
			continue
		}
		price, err := parseMoney(col(row, "price"))
		if err != nil || price <= 0 {
			recordErr(rowNum, "price must be a positive number")
			continue
		}
		cost, _ := parseMoney(col(row, "cost"))
		stock, _ := strconv.Atoi(col(row, "stock"))
		if stock < 0 {
			stock = 0
		}
		minStock, _ := strconv.Atoi(col(row, "min_stock"))
		if minStock < 0 {
			minStock = 0
		}

		var categoryID *uuid.UUID
		if cn := col(row, "category"); cn != "" {
			cat, err := s.FindOrCreateCategory(ctx, orgID, cn)
			if err != nil {
				recordErr(rowNum, "category: "+apierr.Translate(err).Message)
				continue
			}
			categoryID = &cat.ID
		}

		desc := ""
		if unit := col(row, "unit"); unit != "" {
			desc = "Unit: " + unit
		}
		if _, err := s.CreateProduct(ctx, storeID, &ProductInput{
			Name:         name,
			CategoryID:   categoryID,
			SKU:          col(row, "sku"),
			Barcode:      col(row, "barcode"),
			CostPrice:    cost,
			SellingPrice: price,
			Stock:        stock,
			ReorderPoint: minStock,
			Description:  desc,
			CreatedBy:    userID,
		}); err != nil {
			recordErr(rowNum, apierr.Translate(err).Message)
			continue
		}
		created++
	}

	return map[string]interface{}{
		"created": created,
		"failed":  failed,
		"errors":  errList,
	}, nil
}

// parseMoney parses an amount string (TZS) allowing "28500", "28,500" or "28500.00".
func parseMoney(s string) (int64, error) {
	s = strings.ReplaceAll(strings.TrimSpace(s), ",", "")
	if s == "" {
		return 0, nil
	}
	if i, err := strconv.ParseInt(s, 10, 64); err == nil {
		return i, nil
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, err
	}
	return int64(math.Round(f)), nil
}

// ── DTOs ────────────────────────────────────────────────────────────────────

// ProductInput is the create/update payload.
type ProductInput struct {
	Name               string     `json:"name" binding:"required"`
	Unit               string     `json:"unit"`
	CategoryID         *uuid.UUID `json:"category_id"`
	SKU                string     `json:"sku"`
	Barcode            string     `json:"barcode"`
	CostPrice          int64      `json:"cost_price"`
	SellingPrice       int64      `json:"selling_price" binding:"required"`
	WholesalePrice     *int64     `json:"wholesale_price"`
	WholesaleMinimum   int        `json:"wholesale_minimum"`
	Stock              int        `json:"stock"`
	ReorderPoint       int        `json:"reorder_point"`
	Image              string     `json:"image"`
	TaxRate            float64    `json:"tax_rate"`
	Description        string     `json:"description"`
	IsService          bool       `json:"is_service"`
	IsActive           *bool      `json:"is_active"`
	CreatedBy          uuid.UUID  `json:"-"`
	InitialBatchNumber string     `json:"initial_batch_number"`
	InitialExpiryDate  string     `json:"initial_expiry_date"`
}

// UpdateProductInput is used for PATCH /inventory/products/{id}. Every field is
// optional (pointer) so callers can update a single field (e.g. image) without
// resending the whole product.
type UpdateProductInput struct {
	Name             *string    `json:"name"`
	Unit             *string    `json:"unit"`
	CategoryID       *uuid.UUID `json:"category_id"`
	SKU              *string    `json:"sku"`
	Barcode          *string    `json:"barcode"`
	CostPrice        *int64     `json:"cost_price"`
	SellingPrice     *int64     `json:"selling_price"`
	WholesalePrice   *int64     `json:"wholesale_price"`
	WholesaleMinimum *int       `json:"wholesale_minimum"`
	Stock            *int       `json:"stock"`
	ReorderPoint     *int       `json:"reorder_point"`
	Image            *string    `json:"image"`
	TaxRate          *float64   `json:"tax_rate"`
	Description      *string    `json:"description"`
	IsService        *bool      `json:"is_service"`
	IsActive         *bool      `json:"is_active"`
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
		"id":                p.ID,
		"store_id":          p.StoreID,
		"category_id":       p.CategoryID,
		"category":          cat,
		"name":              p.Name,
		"unit":              p.Unit,
		"sku":               sku,
		"barcode":           barcode,
		"cost_price":        p.CostPrice,
		"selling_price":     p.SellingPrice,
		"wholesale_price":   p.WholesalePrice,
		"wholesale_minimum": p.WholesaleMinimum,
		"stock":             p.Stock,
		"reorder_point":     p.ReorderPoint,
		"low_stock":         p.IsLowStock(),
		"image":             p.Image,
		"tax_rate":          p.TaxRate,
		"description":       p.Description,
		"is_active":         p.IsActive,
		"is_service":        p.IsService,
		"created_at":        p.CreatedAt,
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
