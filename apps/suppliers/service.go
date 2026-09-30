package suppliers

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/troubleman96/ziadapos/apps/inventory"
	"github.com/troubleman96/ziadapos/internal/apierr"
	"github.com/troubleman96/ziadapos/internal/cache"
	"github.com/troubleman96/ziadapos/internal/events"
)

const supplierCacheKey = "sup:suppliers:%s"

// Service implements supplier business rules.
type Service struct {
	repo  *Repo
	cache *cache.Cache
	bus   *events.Bus
}

// NewService wires a suppliers Service.
func NewService(repo *Repo, cache *cache.Cache, bus *events.Bus) *Service {
	return &Service{repo: repo, cache: cache, bus: bus}
}

// SupplierInput is the create/update payload.
type SupplierInput struct {
	Name         string `json:"name" binding:"required"`
	Category     string `json:"category"`
	Rep          string `json:"rep"`
	Phone        string `json:"phone"`
	Email        string `json:"email"`
	Address      string `json:"address"`
	PaymentTerms int    `json:"payment_terms"`
	LeadTimeDays int    `json:"lead_time_days"`
	Status       string `json:"status"`
	Notes        string `json:"notes"`
}

// DeliveryInput is the create/update delivery payload.
type DeliveryInput struct {
	InvoiceNumber string              `json:"invoice_number"`
	DeliveryDate  time.Time           `json:"delivery_date"`
	ItemsCount    int                 `json:"items_count"`
	TotalValue    int64               `json:"total_value"`
	Status        string              `json:"status"`
	IsPaid        bool                `json:"is_paid"`
	Notes         string              `json:"notes"`
	Lines         []DeliveryLineInput `json:"lines,omitempty"`
}

type DeliveryLineInput struct {
	ProductID   uuid.UUID `json:"product_id" binding:"required"`
	Quantity    int       `json:"quantity" binding:"required,gt=0"`
	UnitCost    int64     `json:"unit_cost" binding:"gte=0"`
	BatchNumber string    `json:"batch_number"`
	ExpiryDate  string    `json:"expiry_date"`
}

// PaymentInput is a payment against one supplier (optionally one delivery).
type PaymentInput struct {
	Amount        int64  `json:"amount" binding:"required,gt=0"`
	PaymentMethod string `json:"payment_method"`
	Reference     string `json:"reference"`
	DeliveryID    string `json:"delivery_id"`
	Notes         string `json:"notes"`
}

func (s *Service) Create(ctx context.Context, orgID, storeID uuid.UUID, in *SupplierInput) (*Supplier, error) {
	if strings.TrimSpace(in.Name) == "" {
		return nil, apierr.BadRequest("Supplier name is required")
	}
	terms := in.PaymentTerms
	if terms == 0 {
		terms = Terms30
	}
	if !containsInt(ValidTerms, terms) {
		return nil, apierr.BadRequest("Invalid payment terms").WithField("payment_terms", "must be 30, 60 or 90")
	}
	leadTime := in.LeadTimeDays
	if leadTime == 0 {
		leadTime = 7
	}
	if leadTime < 0 || leadTime > 365 {
		return nil, apierr.BadRequest("Invalid supplier lead time").WithField("lead_time_days", "must be between 1 and 365")
	}
	status := in.Status
	if status == "" {
		status = StatusActive
	}
	if !contains(ValidStatuses, status) {
		return nil, apierr.BadRequest("Invalid status").WithField("status", "must be active, inactive or on_hold")
	}
	supplier := &Supplier{
		OrganisationID: orgID,
		StoreID:        storeID,
		Name:           strings.TrimSpace(in.Name),
		Category:       strings.TrimSpace(in.Category),
		Rep:            strings.TrimSpace(in.Rep),
		Phone:          strings.TrimSpace(in.Phone),
		Email:          strings.TrimSpace(in.Email),
		Address:        strings.TrimSpace(in.Address),
		PaymentTerms:   terms,
		LeadTimeDays:   leadTime,
		Status:         status,
		Notes:          in.Notes,
	}
	if err := s.repo.Create(supplier); err != nil {
		return nil, err
	}
	s.invalidateCache(storeID)
	return supplier, nil
}

func (s *Service) Get(ctx context.Context, storeID, id uuid.UUID) (*Supplier, error) {
	supplier, err := s.repo.Get(storeID, id)
	if err != nil {
		return nil, apierr.NotFound("Supplier not found")
	}
	return supplier, nil
}

func (s *Service) Update(ctx context.Context, storeID, id uuid.UUID, in *SupplierInput) (*Supplier, error) {
	supplier, err := s.repo.Get(storeID, id)
	if err != nil {
		return nil, apierr.NotFound("Supplier not found")
	}
	if strings.TrimSpace(in.Name) != "" {
		supplier.Name = strings.TrimSpace(in.Name)
	}
	supplier.Category = in.Category
	supplier.Rep = in.Rep
	supplier.Phone = in.Phone
	supplier.Email = in.Email
	supplier.Address = in.Address
	if in.PaymentTerms > 0 {
		if !containsInt(ValidTerms, in.PaymentTerms) {
			return nil, apierr.BadRequest("Invalid payment terms").WithField("payment_terms", "must be 30, 60 or 90")
		}
		supplier.PaymentTerms = in.PaymentTerms
	}
	if in.LeadTimeDays > 0 {
		if in.LeadTimeDays > 365 {
			return nil, apierr.BadRequest("Invalid supplier lead time").WithField("lead_time_days", "must be between 1 and 365")
		}
		supplier.LeadTimeDays = in.LeadTimeDays
	}
	if in.Status != "" {
		if !contains(ValidStatuses, in.Status) {
			return nil, apierr.BadRequest("Invalid status").WithField("status", "must be active, inactive or on_hold")
		}
		supplier.Status = in.Status
	}
	supplier.Notes = in.Notes
	if err := s.repo.Update(supplier); err != nil {
		return nil, err
	}
	s.invalidateCache(storeID)
	return supplier, nil
}

// Deactivate soft-deactivates a supplier by flipping its status.
func (s *Service) Deactivate(ctx context.Context, storeID, id uuid.UUID) (*Supplier, error) {
	supplier, err := s.repo.Get(storeID, id)
	if err != nil {
		return nil, apierr.NotFound("Supplier not found")
	}
	if supplier.Status != StatusInactive {
		supplier.Status = StatusInactive
		if err := s.repo.Update(supplier); err != nil {
			return nil, err
		}
	}
	s.invalidateCache(storeID)
	return supplier, nil
}

func (s *Service) List(ctx context.Context, storeID uuid.UUID, f SupplierFilter) (map[string]interface{}, error) {
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
	items, total, err := s.repo.List(f)
	if err != nil {
		return nil, err
	}
	balances, err := s.repo.OutstandingBySupplier(storeID)
	if err != nil {
		return nil, err
	}
	out := make([]map[string]interface{}, 0, len(items))
	for i := range items {
		out = append(out, supplierResponse(&items[i], balances[items[i].ID]))
	}
	return map[string]interface{}{
		"items": out, "total": total, "page": f.Page, "page_size": f.PageSize,
	}, nil
}

func (s *Service) Stats(ctx context.Context, storeID uuid.UUID) (map[string]interface{}, error) {
	if s.cache != nil {
		key := "sup:stats:" + storeID.String()
		var cached map[string]interface{}
		if err := s.cache.GetJSON(ctx, key, &cached); err == nil {
			return cached, nil
		}
	}
	st, err := s.repo.Stats(storeID)
	if err != nil {
		return nil, err
	}
	out := map[string]interface{}{
		"total_suppliers":   st.TotalSuppliers,
		"active_count":      st.ActiveCount,
		"pending_invoices":  st.PendingInvoices,
		"total_outstanding": st.TotalOutstanding,
	}
	if s.cache != nil {
		_ = s.cache.SetJSONTTL(ctx, "sup:stats:"+storeID.String(), out, 30*time.Second)
	}
	return out, nil
}

// RecordDelivery logs an invoice received from a supplier.
func (s *Service) RecordDelivery(ctx context.Context, orgID, storeID, supplierID, userID uuid.UUID, in *DeliveryInput) (*SupplierDelivery, error) {
	supplier, err := s.repo.Get(storeID, supplierID)
	if err != nil {
		return nil, apierr.NotFound("Supplier not found")
	}
	if in.TotalValue <= 0 {
		return nil, apierr.BadRequest("Delivery total value must be positive").WithField("total_value", "greater than zero")
	}
	status := in.Status
	if status == "" {
		status = DeliveryDelivered
	}
	if !contains(ValidDeliveryStatuses, status) {
		return nil, apierr.BadRequest("Invalid status").WithField("status", "must be delivered, pending or partial")
	}
	date := in.DeliveryDate
	if date.IsZero() {
		date = time.Now()
	}
	items := in.ItemsCount
	if items <= 0 {
		items = 1
	}
	d := &SupplierDelivery{
		SupplierID:     supplierID,
		StoreID:        storeID,
		OrganisationID: orgID,
		InvoiceNumber:  strings.TrimSpace(in.InvoiceNumber),
		DeliveryDate:   date,
		ItemsCount:     items,
		TotalValue:     in.TotalValue,
		Status:         status,
		IsPaid:         in.IsPaid,
		RecordedBy:     userID,
		Notes:          in.Notes,
	}
	var businessType string
	if err := s.repo.DB().WithContext(ctx).Table("organisations").Select("business_type").Where("id = ?", orgID).Scan(&businessType).Error; err != nil {
		return nil, err
	}
	if businessType == "pharmacy" && status == DeliveryDelivered && len(in.Lines) == 0 {
		return nil, apierr.BadRequest("Pharmacy deliveries require received product, lot and expiry details").WithField("lines", "add at least one received batch")
	}
	if len(in.Lines) > 0 && status != DeliveryDelivered {
		return nil, apierr.BadRequest("Inventory lines can only be recorded for delivered invoices").WithField("status", "must be delivered")
	}
	db := s.repo.DB().WithContext(ctx)
	lineRecords := make([]SupplierDeliveryLine, 0, len(in.Lines))
	productIDs := make([]uuid.UUID, 0, len(in.Lines))
	if err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(d).Error; err != nil {
			return err
		}
		for _, line := range in.Lines {
			if line.Quantity <= 0 || line.UnitCost < 0 {
				return apierr.BadRequest("Delivery line quantity and cost are invalid")
			}
			var product inventory.Product
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND store_id = ? AND is_active = true", line.ProductID, storeID).First(&product).Error; err != nil {
				return apierr.BadRequest("Delivery line product is not active in this store").WithField("lines", line.ProductID.String())
			}
			if product.IsService {
				return apierr.BadRequest("Service items cannot be received as stock")
			}
			persisted := SupplierDeliveryLine{DeliveryID: d.ID, ProductID: line.ProductID, Quantity: line.Quantity, UnitCost: line.UnitCost}
			if businessType == "pharmacy" {
				expiry, parseErr := time.Parse("2006-01-02", line.ExpiryDate)
				if strings.TrimSpace(line.BatchNumber) == "" || parseErr != nil {
					return apierr.BadRequest("Each pharmacy delivery line needs a batch number and expiry date").WithField("lines", product.Name)
				}
				persisted.BatchNumber = strings.TrimSpace(line.BatchNumber)
				persisted.ExpiryDate = &expiry
				batch := inventory.MedicineBatch{StoreID: storeID, ProductID: product.ID, BatchNumber: persisted.BatchNumber, Quantity: 0, ExpiryDate: expiry, CostPrice: line.UnitCost, SupplierName: supplier.Name}
				if err := tx.Create(&batch).Error; err != nil {
					return err
				}
				persisted.MedicineBatchID = &batch.ID
				if err := tx.Create(&inventory.MedicineStockMovement{StoreID: storeID, ProductID: product.ID, BatchID: batch.ID, Quantity: line.Quantity, MovementType: "purchase", ReferenceID: d.ID.String(), Reason: "Supplier delivery " + d.InvoiceNumber, PerformedBy: userID}).Error; err != nil {
					return err
				}
				if err := inventory.ReconcilePharmacyStock(tx, storeID, product.ID, 0); err != nil {
					return err
				}
			}
			if err := tx.Create(&persisted).Error; err != nil {
				return err
			}
			lineRecords = append(lineRecords, persisted)
			productIDs = append(productIDs, product.ID)
			if businessType != "pharmacy" {
				if err := tx.Model(&inventory.Product{}).Where("id = ?", product.ID).Update("stock", gorm.Expr("stock + ?", line.Quantity)).Error; err != nil {
					return err
				}
			}
			if err := tx.Create(&inventory.StockAdjustment{ProductID: product.ID, MedicineBatchID: persisted.MedicineBatchID, QtyChange: line.Quantity, Reason: inventory.ReasonRestock, Note: "Supplier delivery " + d.InvoiceNumber, CreatedBy: userID}).Error; err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}
	d.Lines = lineRecords
	if len(productIDs) > 0 {
		inventory.NewService(inventory.NewRepo(s.repo.DB()), s.cache, s.bus).InvalidateProductCache(&storeID, &orgID, productIDs...)
	}
	s.invalidateCache(storeID)
	s.bus.Publish("supplier.delivery_recorded", map[string]interface{}{
		"delivery_id": d.ID, "supplier_id": supplierID, "store_id": storeID,
		"total_value": d.TotalValue, "is_paid": d.IsPaid,
	})
	return d, nil
}

func (s *Service) GetDelivery(ctx context.Context, storeID, id uuid.UUID) (*SupplierDelivery, error) {
	d, err := s.repo.GetDelivery(storeID, id)
	if err != nil {
		return nil, apierr.NotFound("Delivery not found")
	}
	return d, nil
}

func (s *Service) UpdateDelivery(ctx context.Context, storeID, id uuid.UUID, in *DeliveryInput) (*SupplierDelivery, error) {
	d, err := s.repo.GetDelivery(storeID, id)
	if err != nil {
		return nil, apierr.NotFound("Delivery not found")
	}
	if in.InvoiceNumber != "" {
		d.InvoiceNumber = in.InvoiceNumber
	}
	if !in.DeliveryDate.IsZero() {
		d.DeliveryDate = in.DeliveryDate
	}
	if in.ItemsCount > 0 {
		d.ItemsCount = in.ItemsCount
	}
	if in.TotalValue > 0 {
		d.TotalValue = in.TotalValue
	}
	if in.Status != "" {
		if !contains(ValidDeliveryStatuses, in.Status) {
			return nil, apierr.BadRequest("Invalid status").WithField("status", "must be delivered, pending or partial")
		}
		d.Status = in.Status
	}
	d.IsPaid = in.IsPaid
	d.Notes = in.Notes
	if err := s.repo.UpdateDelivery(d); err != nil {
		return nil, err
	}
	s.invalidateCache(storeID)
	return d, nil
}

func (s *Service) DeleteDelivery(ctx context.Context, storeID, id uuid.UUID) error {
	delivery, err := s.repo.GetDelivery(storeID, id)
	if err != nil {
		return apierr.NotFound("Delivery not found")
	}
	if len(delivery.Lines) > 0 {
		return apierr.Conflict("A delivery with received inventory cannot be deleted; record a stock return or adjustment instead")
	}
	if err := s.repo.DeleteDelivery(storeID, id); err != nil {
		return apierr.NotFound("Delivery not found")
	}
	s.invalidateCache(storeID)
	return nil
}

func (s *Service) ListDeliveries(ctx context.Context, storeID uuid.UUID, f DeliveryFilter) (map[string]interface{}, error) {
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
	items, total, err := s.repo.ListDeliveries(f)
	if err != nil {
		return nil, err
	}
	names, err := s.supplierNames(storeID)
	if err != nil {
		return nil, err
	}
	out := make([]map[string]interface{}, 0, len(items))
	for i := range items {
		out = append(out, deliveryResponse(&items[i], names[items[i].SupplierID]))
	}
	return map[string]interface{}{
		"items": out, "total": total, "page": f.Page, "page_size": f.PageSize,
	}, nil
}

// RecordPayment pays a supplier. When linked to a delivery, the delivery is
// marked paid once fully covered.
func (s *Service) RecordPayment(ctx context.Context, orgID, storeID, supplierID, userID uuid.UUID, in *PaymentInput) (*SupplierPayment, error) {
	if _, err := s.repo.Get(storeID, supplierID); err != nil {
		return nil, apierr.NotFound("Supplier not found")
	}
	method := in.PaymentMethod
	if method == "" {
		method = MethodCash
	}
	if !contains(ValidPaymentMethods, method) {
		return nil, apierr.BadRequest("Invalid payment method").WithField("payment_method", "must be cash, bank or mobile")
	}
	var delivery *SupplierDelivery
	if in.DeliveryID != "" {
		id, err := uuid.Parse(in.DeliveryID)
		if err != nil {
			return nil, apierr.BadRequest("Invalid delivery_id")
		}
		var gerr error
		delivery, gerr = s.repo.GetDelivery(storeID, id)
		if gerr != nil || delivery.SupplierID != supplierID {
			return nil, apierr.NotFound("Delivery not found for this supplier")
		}
	}

	payment := &SupplierPayment{
		SupplierID:     supplierID,
		StoreID:        storeID,
		OrganisationID: orgID,
		Amount:         in.Amount,
		PaymentDate:    time.Now(),
		PaymentMethod:  method,
		Reference:      strings.TrimSpace(in.Reference),
		RecordedBy:     userID,
		Notes:          in.Notes,
	}
	if delivery != nil {
		payment.DeliveryID = &delivery.ID
	}

	err := s.repo.WithTx(func(tx *gorm.DB) error {
		if err := s.repo.CreatePaymentTx(tx, payment); err != nil {
			return err
		}
		if delivery != nil && !delivery.IsPaid {
			paid, err := s.repo.SumPaymentsForDelivery(tx, delivery.ID)
			if err != nil {
				return err
			}
			if paid >= delivery.TotalValue {
				delivery.IsPaid = true
				return s.repo.UpdateDeliveryTx(tx, delivery)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	s.invalidateCache(storeID)
	s.bus.Publish("supplier.payment_recorded", map[string]interface{}{
		"payment_id": payment.ID, "supplier_id": supplierID, "store_id": storeID, "amount": in.Amount,
	})
	return payment, nil
}

func (s *Service) ListPayments(ctx context.Context, storeID uuid.UUID, f PaymentFilter) (map[string]interface{}, error) {
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
	items, total, err := s.repo.ListPayments(f)
	if err != nil {
		return nil, err
	}
	names, err := s.supplierNames(storeID)
	if err != nil {
		return nil, err
	}
	out := make([]map[string]interface{}, 0, len(items))
	for i := range items {
		out = append(out, paymentResponse(&items[i], names[items[i].SupplierID]))
	}
	return map[string]interface{}{
		"items": out, "total": total, "page": f.Page, "page_size": f.PageSize,
	}, nil
}

// outstandingBalance returns the current balance owed to one supplier.
func (s *Service) outstandingBalance(ctx context.Context, storeID, supplierID uuid.UUID) (int64, error) {
	balances, err := s.repo.OutstandingBySupplier(storeID)
	if err != nil {
		return 0, err
	}
	return balances[supplierID], nil
}

func (s *Service) supplierNames(storeID uuid.UUID) (map[uuid.UUID]string, error) {
	var suppliers []Supplier
	if err := s.repo.db.Where("store_id = ?", storeID).Find(&suppliers).Error; err != nil {
		return nil, err
	}
	out := make(map[uuid.UUID]string, len(suppliers))
	for i := range suppliers {
		out[suppliers[i].ID] = suppliers[i].Name
	}
	return out, nil
}

func (s *Service) invalidateCache(storeID uuid.UUID) {
	if s.cache == nil {
		return
	}
	ctx := context.Background()
	_ = s.cache.DeletePattern(ctx, "sup:*"+storeID.String()+"*")
}

func supplierResponse(s *Supplier, balance int64) map[string]interface{} {
	return map[string]interface{}{
		"id":                  s.ID,
		"name":                s.Name,
		"category":            s.Category,
		"rep":                 s.Rep,
		"phone":               s.Phone,
		"email":               s.Email,
		"address":             s.Address,
		"payment_terms":       s.PaymentTerms,
		"lead_time_days":      s.LeadTimeDays,
		"status":              s.Status,
		"is_active":           s.IsActive(),
		"outstanding_balance": balance,
		"notes":               s.Notes,
		"created_at":          s.CreatedAt,
	}
}

func deliveryResponse(d *SupplierDelivery, supplierName string) map[string]interface{} {
	return map[string]interface{}{
		"id":             d.ID,
		"supplier_id":    d.SupplierID,
		"supplier_name":  supplierName,
		"invoice_number": d.InvoiceNumber,
		"delivery_date":  d.DeliveryDate,
		"items_count":    d.ItemsCount,
		"total_value":    d.TotalValue,
		"status":         d.Status,
		"is_paid":        d.IsPaid,
		"recorded_by":    d.RecordedBy,
		"notes":          d.Notes,
		"created_at":     d.CreatedAt,
	}
}

func paymentResponse(p *SupplierPayment, supplierName string) map[string]interface{} {
	return map[string]interface{}{
		"id":             p.ID,
		"supplier_id":    p.SupplierID,
		"supplier_name":  supplierName,
		"delivery_id":    p.DeliveryID,
		"amount":         p.Amount,
		"payment_date":   p.PaymentDate,
		"payment_method": p.PaymentMethod,
		"reference":      p.Reference,
		"recorded_by":    p.RecordedBy,
		"notes":          p.Notes,
		"created_at":     p.CreatedAt,
	}
}

func contains(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

func containsInt(list []int, v int) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}
