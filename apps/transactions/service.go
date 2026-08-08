package transactions

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/troubleman96/ziadapos/apps/accounts"
	"github.com/troubleman96/ziadapos/apps/inventory"
	"github.com/troubleman96/ziadapos/internal/apierr"
	"github.com/troubleman96/ziadapos/internal/cache"
	"github.com/troubleman96/ziadapos/internal/events"
)

const taxRatePct = 18

var lockForUpdate = clause.Locking{Strength: "UPDATE"}

// Service implements POS sale/refund flows with stock accounting.
type Service struct {
	repo       *Repo
	cache      *cache.Cache
	bus        *events.Bus
	invSvc     *inventory.Service
	accountsDB *gorm.DB
}

// NewService wires a transactions Service. invSvc is used only for cache
// invalidation after stock changes.
func NewService(repo *Repo, cache *cache.Cache, bus *events.Bus, invSvc *inventory.Service, db *gorm.DB) *Service {
	return &Service{repo: repo, cache: cache, bus: bus, invSvc: invSvc, accountsDB: db}
}

func (s *Service) summaryKey(storeID uuid.UUID, day string) string {
	return "pos:summary:" + storeID.String() + ":" + day
}

func (s *Service) invalidateStoreCaches(storeID, orgID uuid.UUID, productIDs ...uuid.UUID) {
	if s.cache != nil {
		ctx := context.Background()
		_ = s.cache.DeletePattern(ctx, "pos:list:"+storeID.String()+":*")
		_ = s.cache.DeletePattern(ctx, "pos:summary:"+storeID.String()+":*")
	}
	if s.invSvc != nil {
		s.invSvc.InvalidateProductCache(&storeID, &orgID, productIDs...)
	}
}

// ── Sale DTOs ────────────────────────────────────────────────────────────────

// SaleLineInput is one item in a sale.
type SaleLineInput struct {
	ProductID uuid.UUID `json:"product_id" binding:"required"`
	Qty       int       `json:"qty" binding:"required,gt=0"`
}

// SaleInput is a complete sale payload.
type SaleInput struct {
	Lines            []SaleLineInput `json:"lines" binding:"required,min=1,dive"`
	PaymentMethod    string          `json:"payment_method" binding:"required"`
	PaymentReference string          `json:"payment_reference"`
	DiscountPct      float64         `json:"discount_pct"`
	CustomerName     string          `json:"customer_name"`
	CustomerPhone    string          `json:"customer_phone"`
	CustomerID       *uuid.UUID      `json:"customer_id"`
	TillNumber       string          `json:"till_number"`
	Notes            string          `json:"notes"`
}

// CompleteSale validates, decrements stock and persists a sale.
func (s *Service) CompleteSale(ctx context.Context, orgID, storeID, cashierID uuid.UUID, in *SaleInput) (*Transaction, error) {
	method := strings.TrimSpace(in.PaymentMethod)
	if !contains(ValidMethods, method) {
		return nil, apierr.BadRequest("Invalid payment method").WithField("payment_method", "must be Cash, M-Pesa, Tigo Pesa, Bank or Credit")
	}
	if in.DiscountPct < 0 || in.DiscountPct > 100 {
		return nil, apierr.BadRequest("Discount must be between 0 and 100").WithField("discount_pct", "out of range")
	}

	// Load product snapshots with row locks.
	ids := make([]uuid.UUID, 0, len(in.Lines))
	for _, l := range in.Lines {
		ids = append(ids, l.ProductID)
	}

	var txn *Transaction
	err := s.repo.WithTx(func(tx *gorm.DB) error {
		// Fetch all products once with FOR UPDATE.
		var products []inventory.Product
		if err := tx.Clauses(lockForUpdate).Where("store_id = ? AND id IN ?", storeID, ids).Find(&products).Error; err != nil {
			return err
		}
		byID := make(map[uuid.UUID]inventory.Product, len(products))
		for _, p := range products {
			byID[p.ID] = p
		}
		if len(byID) != len(ids) {
			return apierr.BadRequest("One or more products were not found in this store").WithField("lines", "check product ids")
		}

		number, err := s.repo.NextTxnNumber(tx, storeID)
		if err != nil {
			return err
		}

		lines := make([]TransactionLine, 0, len(in.Lines))
		var subtotal, costTotal int64
		for _, l := range in.Lines {
			p, ok := byID[l.ProductID]
			if !ok {
				continue
			}
			if !p.IsActive {
				return apierr.Conflict("Product is inactive").WithField("product", p.Name)
			}
			if !p.IsService && p.Stock < l.Qty {
				return apierr.Conflict("Insufficient stock").WithField("product", fmt.Sprintf("%s only has %d units", p.Name, p.Stock))
			}
			if !p.IsService {
				if err := tx.Model(&inventory.Product{}).Where("id = ?", p.ID).
					Update("stock", gorm.Expr("stock - ?", l.Qty)).Error; err != nil {
					return err
				}
				adj := &inventory.StockAdjustment{
					ProductID: p.ID, QtyChange: -l.Qty, Reason: inventory.ReasonSale,
					Note: fmt.Sprintf("Sale %04d", number), CreatedBy: cashierID,
				}
				if err := tx.Create(adj).Error; err != nil {
					return err
				}
			}
			sku := ""
			if p.SKU != nil {
				sku = *p.SKU
			}
			lines = append(lines, TransactionLine{
				ProductID:   &p.ID,
				ProductName: p.Name,
				ProductSKU:  sku,
				UnitPrice:   p.SellingPrice,
				UnitCost:    p.CostPrice,
				Qty:         l.Qty,
			})
			subtotal += p.SellingPrice * int64(l.Qty)
			costTotal += p.CostPrice * int64(l.Qty)
		}

		discountAmount := int64(float64(subtotal) * in.DiscountPct / 100)
		taxable := subtotal - discountAmount
		if taxable < 0 {
			taxable = 0
		}
		tax := taxable * taxRatePct / 100
		total := taxable + tax
		profit := total - costTotal - tax

		customerName := strings.TrimSpace(in.CustomerName)
		if customerName == "" {
			customerName = "Walk-in"
		}
		till := strings.TrimSpace(in.TillNumber)
		if till == "" {
			till = "Till #1"
		}
		status := StatusPaid
		if method == MethodCredit {
			status = StatusCredit
		}

		txn = &Transaction{
			OrganisationID:   orgID,
			StoreID:          storeID,
			TxnNumber:        fmt.Sprintf("TXN-%04d", number),
			PaymentMethod:    method,
			PaymentReference: strings.TrimSpace(in.PaymentReference),
			Status:           status,
			Subtotal:         subtotal,
			DiscountPct:      in.DiscountPct,
			DiscountAmount:   discountAmount,
			TaxAmount:        tax,
			Total:            total,
			CostTotal:        costTotal,
			Profit:           profit,
			CashierID:        cashierID,
			TillNumber:       till,
			CustomerName:     customerName,
			CustomerPhone:    strings.TrimSpace(in.CustomerPhone),
			CustomerID:       in.CustomerID,
			Channel:          "counter",
			Notes:            in.Notes,
			Lines:            lines,
		}
		return s.repo.CreateSaleTx(tx, txn)
	})
	if err != nil {
		return nil, err
	}

	s.invalidateStoreCaches(storeID, orgID, ids...)
	s.bus.Publish("transaction.created", map[string]interface{}{
		"transaction_id": txn.ID, "store_id": storeID, "org_id": orgID,
		"total": txn.Total, "method": method, "status": txn.Status,
	})
	return txn, nil
}

// ListTransactions returns filtered, paginated transactions with lines.
func (s *Service) ListTransactions(ctx context.Context, storeID uuid.UUID, f Filter) (map[string]interface{}, error) {
	if f.Page <= 0 {
		f.Page = 1
	}
	if f.PageSize <= 0 {
		f.PageSize = 20
	}
	if f.PageSize > 100 {
		f.PageSize = 100
	}
	f.StoreID = storeID
	txns, total, err := s.repo.ListTransactions(f)
	if err != nil {
		return nil, err
	}
	_ = s.repo.LoadLines(txns)

	items := make([]map[string]interface{}, 0, len(txns))
	for i := range txns {
		items = append(items, txnResponse(&txns[i]))
	}
	out := map[string]interface{}{
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
	return out, nil
}

// GetTransaction returns one transaction with its lines.
func (s *Service) GetTransaction(ctx context.Context, storeID, id uuid.UUID) (*Transaction, error) {
	t, err := s.repo.GetTransaction(storeID, id)
	if err != nil {
		return nil, apierr.NotFound("Transaction not found")
	}
	return t, nil
}

// Refund reverses a paid/credit transaction: restores stock and marks it
// refunded. Requires a user with refund permission (owner/manager).
func (s *Service) Refund(ctx context.Context, orgID, storeID, userID, id uuid.UUID, reason string) (*Transaction, error) {
	if err := s.requireRefundPermission(userID); err != nil {
		return nil, err
	}
	status, err := s.repo.GetStatus(storeID, id)
	if err != nil {
		return nil, apierr.NotFound("Transaction not found")
	}
	if status != StatusPaid && status != StatusCredit {
		return nil, apierr.Conflict("Only paid or credit transactions can be refunded")
	}

	txn, err := s.repo.GetTransaction(storeID, id)
	if err != nil {
		return nil, apierr.NotFound("Transaction not found")
	}

	err = s.repo.WithTx(func(tx *gorm.DB) error {
		for _, l := range txn.Lines {
			if l.ProductID == nil {
				continue
			}
			if err := tx.Model(&inventory.Product{}).Where("id = ?", *l.ProductID).
				Update("stock", gorm.Expr("stock + ?", l.Qty)).Error; err != nil {
				return err
			}
			adj := &inventory.StockAdjustment{
				ProductID: *l.ProductID, QtyChange: l.Qty, Reason: inventory.ReasonRefund,
				Note: "Refund " + txn.TxnNumber, CreatedBy: userID,
			}
			if err := tx.Create(adj).Error; err != nil {
				return err
			}
		}
		return s.repo.UpdateStatus(tx, id, StatusRefunded, reason)
	})
	if err != nil {
		return nil, err
	}
	txn.Status = StatusRefunded
	txn.Notes = reason

	productIDs := make([]uuid.UUID, 0, len(txn.Lines))
	for _, l := range txn.Lines {
		if l.ProductID != nil {
			productIDs = append(productIDs, *l.ProductID)
		}
	}
	s.invalidateStoreCaches(storeID, orgID, productIDs...)
	s.bus.Publish("transaction.refunded", map[string]interface{}{
		"transaction_id": id, "store_id": storeID, "reason": reason,
	})
	return txn, nil
}

// Void cancels a transaction and restores stock (admin only).
func (s *Service) Void(ctx context.Context, orgID, storeID, id uuid.UUID, reason string) (*Transaction, error) {
	status, err := s.repo.GetStatus(storeID, id)
	if err != nil {
		return nil, apierr.NotFound("Transaction not found")
	}
	if status == StatusRefunded || status == StatusVoid {
		return nil, apierr.Conflict("Transaction already closed")
	}
	txn, err := s.repo.GetTransaction(storeID, id)
	if err != nil {
		return nil, apierr.NotFound("Transaction not found")
	}

	err = s.repo.WithTx(func(tx *gorm.DB) error {
		for _, l := range txn.Lines {
			if l.ProductID == nil {
				continue
			}
			if err := tx.Model(&inventory.Product{}).Where("id = ?", *l.ProductID).
				Update("stock", gorm.Expr("stock + ?", l.Qty)).Error; err != nil {
				return err
			}
			adj := &inventory.StockAdjustment{
				ProductID: *l.ProductID, QtyChange: l.Qty, Reason: inventory.ReasonCorrection,
				Note: "Void " + txn.TxnNumber, CreatedBy: uuid.Nil,
			}
			if err := tx.Create(adj).Error; err != nil {
				return err
			}
		}
		return s.repo.UpdateStatus(tx, id, StatusVoid, reason)
	})
	if err != nil {
		return nil, err
	}
	txn.Status = StatusVoid
	txn.Notes = reason

	productIDs := make([]uuid.UUID, 0, len(txn.Lines))
	for _, l := range txn.Lines {
		if l.ProductID != nil {
			productIDs = append(productIDs, *l.ProductID)
		}
	}
	s.invalidateStoreCaches(storeID, orgID, productIDs...)
	return txn, nil
}

// Summary returns KPIs for the dashboard for a date range.
func (s *Service) Summary(ctx context.Context, storeID uuid.UUID, from, to time.Time) (map[string]interface{}, error) {
	day := from.Format("2006-01-02")
	key := s.summaryKey(storeID, day)
	var cached map[string]interface{}
	if s.cache != nil {
		if err := s.cache.GetJSON(ctx, key, &cached); err == nil && cached != nil {
			return cached, nil
		}
	}

	rows, err := s.repo.RangeRows(storeID, from, to)
	if err != nil {
		return nil, err
	}
	var gross, refunded, net, profit, tax int64
	var txns int64
	for _, r := range rows {
		switch r.Status {
		case StatusPaid, StatusCredit:
			gross += r.Total
			profit += r.Profit
			tax += r.Tax
			txns++
		case StatusRefunded:
			refunded += r.Total
		}
	}
	net = gross - refunded

	methods, err := s.repo.MethodTotals(storeID, from, to)
	if err != nil {
		return nil, err
	}
	top, err := s.repo.TopProducts(storeID, from, to, 5)
	if err != nil {
		return nil, err
	}

	out := map[string]interface{}{
		"date_from":        from,
		"date_to":          to,
		"gross_sales":      gross,
		"refunded":         refunded,
		"net_sales":        net,
		"profit":           profit,
		"tax":              tax,
		"transactions":     txns,
		"by_method":        methods,
		"top_products":     top,
	}
	if s.cache != nil {
		_ = s.cache.SetJSONTTL(ctx, key, out, 30*time.Second)
	}
	return out, nil
}

// HourlySales returns per-hour totals for the dashboard chart.
func (s *Service) HourlySales(ctx context.Context, storeID uuid.UUID, day time.Time) ([]map[string]interface{}, error) {
	return s.repo.HourlySales(storeID, day)
}

func (s *Service) requireRefundPermission(userID uuid.UUID) error {
	var u accounts.User
	if err := s.accountsDB.Where("id = ?", userID).First(&u).Error; err != nil {
		return apierr.Unauthorized("User not found")
	}
	if u.CanRefund || u.Role == accounts.RoleOwner || u.Role == accounts.RoleAdmin {
		return nil
	}
	return apierr.Forbidden("You do not have refund permission")
}

func contains(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

func txnResponse(t *Transaction) map[string]interface{} {
	lines := make([]map[string]interface{}, 0, len(t.Lines))
	for _, l := range t.Lines {
		lines = append(lines, map[string]interface{}{
			"id": l.ID, "product_id": l.ProductID, "name": l.ProductName,
			"sku": l.ProductSKU, "unit_price": l.UnitPrice, "unit_cost": l.UnitCost,
			"qty": l.Qty, "line_total": l.UnitPrice * int64(l.Qty),
		})
	}
	itemCount := 0
	for _, l := range t.Lines {
		itemCount += l.Qty
	}
	return map[string]interface{}{
		"id":                t.ID,
		"txn_number":        t.TxnNumber,
		"created_at":        t.CreatedAt,
		"payment_method":    t.PaymentMethod,
		"payment_reference": t.PaymentReference,
		"status":            t.Status,
		"subtotal":          t.Subtotal,
		"discount_pct":      t.DiscountPct,
		"discount_amount":   t.DiscountAmount,
		"tax_amount":        t.TaxAmount,
		"total":             t.Total,
		"cost_total":        t.CostTotal,
		"profit":            t.Profit,
		"cashier_id":        t.CashierID,
		"till_number":       t.TillNumber,
		"customer_name":     t.CustomerName,
		"customer_phone":    t.CustomerPhone,
		"customer_id":       t.CustomerID,
		"item_count":        itemCount,
		"sku_count":         len(t.Lines),
		"notes":             t.Notes,
		"lines":             lines,
	}
}
