package inventory

import (
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/troubleman96/ziadapos/apps/accounts"
)

// Repo provides inventory data access.
type Repo struct {
	db *gorm.DB
}

// NewRepo builds a Repo.
func NewRepo(db *gorm.DB) *Repo { return &Repo{db: db} }

func (r *Repo) DB() *gorm.DB { return r.db }

// StoreOrgID returns the organisation owning a store.
func (r *Repo) StoreOrgID(storeID uuid.UUID) (uuid.UUID, error) {
	var orgIDs []uuid.UUID
	err := r.db.Model(&accounts.Store{}).
		Where("id = ?", storeID).
		Pluck("organisation_id", &orgIDs).Error
	if err != nil {
		return uuid.Nil, err
	}
	if len(orgIDs) == 0 {
		return uuid.Nil, gorm.ErrRecordNotFound
	}
	return orgIDs[0], nil
}

// ── Categories ──────────────────────────────────────────────────────────────

func (r *Repo) CreateCategory(c *Category) error { return r.db.Create(c).Error }

func (r *Repo) GetCategory(orgID uuid.UUID, id uuid.UUID) (*Category, error) {
	var c Category
	if err := r.db.Where("id = ? AND organisation_id = ?", id, orgID).First(&c).Error; err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *Repo) FindCategoryByName(orgID uuid.UUID, name string) (*Category, error) {
	var c Category
	err := r.db.Where("lower(name) = lower(?) AND organisation_id = ?", name, orgID).First(&c).Error
	return &c, err
}

func (r *Repo) ListCategories(orgID uuid.UUID) ([]Category, error) {
	var cats []Category
	err := r.db.Where("organisation_id = ?", orgID).
		Order("sort_order asc, name asc").
		Find(&cats).Error
	if err != nil {
		return nil, err
	}
	// Attach live product counts.
	for i := range cats {
		var n int64
		if err := r.db.Model(&Product{}).
			Where("category_id = ?", cats[i].ID).
			Count(&n).Error; err == nil {
			cats[i].ProductCount = n
		}
	}
	return cats, nil
}

func (r *Repo) UpdateCategory(c *Category) error { return r.db.Save(c).Error }

func (r *Repo) DeleteCategory(orgID uuid.UUID, id uuid.UUID) error {
	return r.db.Where("id = ? AND organisation_id = ?", id, orgID).Delete(&Category{}).Error
}

func (r *Repo) CreateUnit(unit *Unit) error { return r.db.Create(unit).Error }
func (r *Repo) GetUnit(orgID, id uuid.UUID) (*Unit, error) {
	var unit Unit
	if err := r.db.Where("id = ? AND organisation_id = ?", id, orgID).First(&unit).Error; err != nil {
		return nil, err
	}
	return &unit, nil
}
func (r *Repo) FindUnitByName(orgID uuid.UUID, name string) (*Unit, error) {
	var unit Unit
	err := r.db.Where("lower(name) = lower(?) AND organisation_id = ?", name, orgID).First(&unit).Error
	return &unit, err
}
func (r *Repo) ListUnits(orgID uuid.UUID) ([]Unit, error) {
	var units []Unit
	err := r.db.Where("organisation_id = ?", orgID).Order("name asc").Find(&units).Error
	if err != nil {
		return nil, err
	}
	for i := range units {
		const usageQuery = `SELECT COUNT(DISTINCT product_id) FROM (
			SELECT p.id AS product_id FROM products p JOIN stores s ON s.id = p.store_id
			WHERE s.organisation_id = ? AND lower(p.unit) = lower(?)
			UNION
			SELECT mp.product_id FROM medicine_profiles mp JOIN stores s ON s.id = mp.store_id
			WHERE s.organisation_id = ? AND lower(mp.unit) = lower(?)
		) AS unit_usage`
		if err := r.db.Raw(usageQuery, orgID, units[i].Name, orgID, units[i].Name).Scan(&units[i].UsageCount).Error; err != nil {
			return nil, err
		}
	}
	return units, err
}
func (r *Repo) UpdateUnit(unit *Unit) error { return r.db.Save(unit).Error }
func (r *Repo) DeleteUnit(orgID, id uuid.UUID) error {
	return r.db.Where("id = ? AND organisation_id = ?", id, orgID).Delete(&Unit{}).Error
}

// ── Products ────────────────────────────────────────────────────────────────

// ProductFilter constrains product list queries.
type ProductFilter struct {
	StoreID    uuid.UUID
	CategoryID uuid.UUID
	Search     string
	Status     string // "", "active", "inactive"
	LowStock   bool
	Minimal    bool
	NoCache    bool
	Page       int
	PageSize   int
}

func (r *Repo) ListProducts(f ProductFilter) ([]Product, int64, error) {
	q := r.db.Model(&Product{}).Where("store_id = ?", f.StoreID)

	if f.CategoryID != uuid.Nil {
		q = q.Where("category_id = ?", f.CategoryID)
	}
	if f.Search != "" {
		like := "%" + strings.ToLower(f.Search) + "%"
		q = q.Where("(lower(name) LIKE ? OR lower(sku) LIKE ? OR barcode LIKE ?)", like, like, like)
	}
	if f.Status == "active" {
		q = q.Where("is_active = true")
	} else if f.Status == "inactive" {
		q = q.Where("is_active = false")
	}
	if f.LowStock {
		q = q.Where("is_service = false AND stock <= reorder_point")
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if f.Minimal {
		q = q.Select("id, store_id, category_id, name, sku, barcode, selling_price, stock, reorder_point, image, is_active, is_service, created_at")
	}
	q = q.Preload("Category").Order("name asc")
	if f.Page > 0 && f.PageSize > 0 {
		q = q.Offset((f.Page - 1) * f.PageSize).Limit(f.PageSize)
	}

	var products []Product
	if err := q.Find(&products).Error; err != nil {
		return nil, 0, err
	}
	return products, total, nil
}

func (r *Repo) GetProduct(storeID, id uuid.UUID) (*Product, error) {
	var p Product
	if err := r.db.Where("id = ? AND store_id = ?", id, storeID).Preload("Category").First(&p).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *Repo) CreateProduct(p *Product) error { return r.db.Create(p).Error }

func (r *Repo) UpdateProduct(p *Product) error { return r.db.Save(p).Error }

func (r *Repo) SoftDeleteProduct(storeID, id uuid.UUID) error {
	return r.db.Model(&Product{}).Where("id = ? AND store_id = ?", id, storeID).
		Update("is_active", false).Error
}

func (r *Repo) GetProductByBarcode(storeID uuid.UUID, code string) (*Product, error) {
	var p Product
	err := r.db.Where("store_id = ? AND barcode = ?", storeID, code).
		Preload("Category").First(&p).Error
	return &p, err
}

func (r *Repo) LowStockProducts(storeID uuid.UUID) ([]Product, error) {
	var products []Product
	err := r.db.Where("store_id = ? AND is_service = false AND is_active = true AND stock <= reorder_point", storeID).
		Preload("Category").Order("stock asc").Find(&products).Error
	return products, err
}

// ── Stock adjustments ───────────────────────────────────────────────────────

func (r *Repo) CreateAdjustment(a *StockAdjustment) error { return r.db.Create(a).Error }

func (r *Repo) ListAdjustments(productID uuid.UUID, limit int) ([]StockAdjustment, error) {
	var adj []StockAdjustment
	err := r.db.Where("product_id = ?", productID).Order("created_at desc").Limit(limit).Find(&adj).Error
	return adj, err
}

// WithTx runs a closure in a transaction.
func (r *Repo) WithTx(fn func(tx *gorm.DB) error) error { return r.db.Transaction(fn) }
