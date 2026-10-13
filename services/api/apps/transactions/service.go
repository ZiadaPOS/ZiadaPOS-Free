package transactions

import (
	"context"
	"fmt"
	"html"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/troubleman96/ziadapos/apps/accounts"
	"github.com/troubleman96/ziadapos/apps/inventory"
	"github.com/troubleman96/ziadapos/apps/pharmacy"
	"github.com/troubleman96/ziadapos/internal/apierr"
	"github.com/troubleman96/ziadapos/internal/cache"
	"github.com/troubleman96/ziadapos/internal/events"
	"github.com/troubleman96/ziadapos/internal/mailer"
)

const taxRatePct = 18

var lockForUpdate = clause.Locking{Strength: "UPDATE"}

func saleUnitPrice(product inventory.Product, quantity int, mode string) (int64, error) {
	if mode != "wholesale" {
		return product.SellingPrice, nil
	}
	if product.WholesalePrice == nil || *product.WholesalePrice <= 0 {
		return 0, fmt.Errorf("wholesale price is not configured")
	}
	minimum := product.WholesaleMinimum
	if minimum <= 0 {
		minimum = 1
	}
	if quantity < minimum {
		return 0, fmt.Errorf("wholesale minimum quantity is %d", minimum)
	}
	return *product.WholesalePrice, nil
}

// Service implements POS sale/refund flows with stock accounting.
type Service struct {
	repo       *Repo
	cache      *cache.Cache
	bus        *events.Bus
	invSvc     *inventory.Service
	accountsDB *gorm.DB
	mailer     *mailer.Mailer
}

// NewService wires a transactions Service. invSvc is used only for cache
// invalidation after stock changes.
func NewService(repo *Repo, cache *cache.Cache, bus *events.Bus, invSvc *inventory.Service, db *gorm.DB, mailer *mailer.Mailer) *Service {
	return &Service{repo: repo, cache: cache, bus: bus, invSvc: invSvc, accountsDB: db, mailer: mailer}
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

func restoreMedicineBatches(tx *gorm.DB, lineID, storeID, productID, userID uuid.UUID, referenceID, movementType string, legacyPharmacy bool, fallbackQty int) error {
	var allocations []inventory.MedicineBatchAllocation
	if err := tx.Where("transaction_line_id = ?", lineID).Find(&allocations).Error; err != nil {
		return err
	}
	for _, allocation := range allocations {
		batchID := allocation.BatchID
		if movementType == "sale_return" {
			// Customer returns are physically separate from untouched supplier stock.
			// Hold them in quarantine until an authorized person inspects and releases
			// them; never silently make returned medicine available for resale.
			var original inventory.MedicineBatch
			if err := tx.Where("id = ? AND store_id = ? AND product_id = ?", allocation.BatchID, storeID, productID).First(&original).Error; err != nil {
				return err
			}
			returned := inventory.MedicineBatch{
				StoreID: storeID, ProductID: productID, SourceBatchID: &original.ID,
				BatchNumber: original.BatchNumber, Quantity: 0, ExpiryDate: original.ExpiryDate,
				CostPrice: original.CostPrice, SupplierName: original.SupplierName,
				StockStatus: "quarantined",
			}
			if err := tx.Create(&returned).Error; err != nil {
				return err
			}
			batchID = returned.ID
		}
		if err := tx.Create(&inventory.MedicineStockMovement{StoreID: storeID, ProductID: productID, BatchID: batchID, Quantity: allocation.Quantity, MovementType: movementType, ReferenceID: referenceID, Reason: "Sale reversal; source batch " + allocation.BatchID.String(), PerformedBy: userID}).Error; err != nil {
			return err
		}
	}
	if len(allocations) > 0 {
		return inventory.ReconcilePharmacyStock(tx, storeID, productID, 0)
	}
	if legacyPharmacy {
		// Transactions created before batch tracking have no allocations. Restore
		// their quantity as untracked stock so refunds and voids remain usable.
		return tx.Model(&inventory.Product{}).Where("id = ? AND store_id = ?", productID, storeID).Update("stock", gorm.Expr("stock + ?", fallbackQty)).Error
	}
	return nil
}

func (s *Service) isPharmacy(ctx context.Context, orgID uuid.UUID) (bool, error) {
	var businessType string
	err := s.accountsDB.WithContext(ctx).Table("organisations").Select("business_type").Where("id = ?", orgID).Scan(&businessType).Error
	return businessType == "pharmacy", err
}

// ── Sale DTOs ────────────────────────────────────────────────────────────────

// SaleLineInput is one item in a sale.
type SaleLineInput struct {
	ProductID                     uuid.UUID `json:"product_id" binding:"required"`
	Qty                           int       `json:"qty" binding:"required,gt=0"`
	PrescriberName                string    `json:"prescriber_name"`
	PrescriberAddress             string    `json:"prescriber_address"`
	PrescriberRegistrationNumber  string    `json:"prescriber_registration_number"`
	PrescriberAuthenticityChecked bool      `json:"prescriber_authenticity_confirmed"`
	PrescribingFacilityName       string    `json:"prescribing_facility_name"`
	PrescribingFacilityAddress    string    `json:"prescribing_facility_address"`
	PrescriptionReference         string    `json:"prescription_reference"`
	PatientName                   string    `json:"patient_name"`
	PatientAddress                string    `json:"patient_address"`
	MedicineStrength              string    `json:"medicine_strength"`
	DosageForm                    string    `json:"dosage_form"`
}

func medicineRequiresPrescription(profile pharmacy.MedicineProfile) bool {
	return profile.RequiresPrescription || profile.MedicineSchedule == "prescription_only"
}

func hasPrescriptionRegisterDetails(line SaleLineInput) bool {
	fields := []struct {
		value string
		limit int
	}{
		{line.PrescriberName, 200}, {line.PrescriberAddress, 500},
		{line.PrescriberRegistrationNumber, 100}, {line.PrescribingFacilityName, 200},
		{line.PrescribingFacilityAddress, 500}, {line.PrescriptionReference, 200},
		{line.PatientName, 200}, {line.PatientAddress, 500},
		{line.MedicineStrength, 100}, {line.DosageForm, 100},
	}
	if !line.PrescriberAuthenticityChecked {
		return false
	}
	for _, field := range fields {
		value := strings.TrimSpace(field.value)
		if value == "" || utf8.RuneCountInString(value) > field.limit {
			return false
		}
	}
	return true
}

// SaleInput is a complete sale payload.
type SaleInput struct {
	Lines            []SaleLineInput `json:"lines" binding:"required,min=1,dive"`
	ExpectedStoreID  *uuid.UUID      `json:"expected_store_id,omitempty"`
	PaymentMethod    string          `json:"payment_method" binding:"required"`
	PriceMode        string          `json:"price_mode"`
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
	if in.ExpectedStoreID != nil && *in.ExpectedStoreID != storeID {
		return nil, apierr.Conflict("Active store changed; refresh checkout before completing the sale")
	}
	method := strings.TrimSpace(in.PaymentMethod)
	if !contains(ValidMethods, method) {
		return nil, apierr.BadRequest("Invalid payment method").WithField("payment_method", "must be Cash, M-Pesa, Tigo Pesa, Bank or Credit")
	}
	if in.DiscountPct < 0 || in.DiscountPct > 100 {
		return nil, apierr.BadRequest("Discount must be between 0 and 100").WithField("discount_pct", "out of range")
	}
	var organisation struct {
		BusinessType string
		OutletType   string
	}
	if err := s.accountsDB.WithContext(ctx).Table("organisations").Select("business_type, outlet_type").Where("id = ?", orgID).Scan(&organisation).Error; err != nil {
		return nil, err
	}
	businessType := organisation.BusinessType
	isPharmacy := businessType == "pharmacy"
	priceMode := strings.TrimSpace(in.PriceMode)
	if priceMode == "" {
		priceMode = "retail"
	}
	if priceMode != "retail" && priceMode != "wholesale" {
		return nil, apierr.BadRequest("Invalid price mode").WithField("price_mode", "must be retail or wholesale")
	}
	if businessType == "wholesale" {
		priceMode = "wholesale"
	}
	if businessType == "retail" || isPharmacy {
		priceMode = "retail"
	}
	if businessType != "wholesale" && businessType != "retail_wholesale" && businessType != "retail" && !isPharmacy && priceMode == "wholesale" {
		return nil, apierr.Conflict("Wholesale pricing is not available for this business type")
	}

	// Load product snapshots with row locks.
	ids := make([]uuid.UUID, 0, len(in.Lines))
	seenProducts := make(map[uuid.UUID]struct{}, len(in.Lines))
	for _, l := range in.Lines {
		if _, seen := seenProducts[l.ProductID]; seen {
			return nil, apierr.BadRequest("Each product may appear only once per sale").WithField("lines", "combine quantities for duplicate products")
		}
		ids = append(ids, l.ProductID)
		seenProducts[l.ProductID] = struct{}{}
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
		medicineProfiles := make(map[uuid.UUID]pharmacy.MedicineProfile)
		if isPharmacy {
			var profiles []pharmacy.MedicineProfile
			if err := tx.Where("store_id = ? AND product_id IN ?", storeID, ids).Find(&profiles).Error; err != nil {
				return err
			}
			for _, profile := range profiles {
				medicineProfiles[profile.ProductID] = profile
			}
		}

		number, err := s.repo.NextTxnNumber(tx, storeID)
		if err != nil {
			return err
		}

		lines := make([]TransactionLine, 0, len(in.Lines))
		allocations := make([][]inventory.MedicineBatchAllocation, 0, len(in.Lines))
		var subtotal, costTotal int64
		for _, l := range in.Lines {
			p, ok := byID[l.ProductID]
			if !ok {
				continue
			}
			if !p.IsActive {
				return apierr.Conflict("Product is inactive").WithField("product", p.Name)
			}
			unitPrice, priceErr := saleUnitPrice(p, l.Qty, priceMode)
			if priceErr != nil {
				return apierr.Conflict(priceErr.Error()).WithField("product", p.Name)
			}
			if !isPharmacy && !p.IsService && p.Stock < l.Qty {
				return apierr.Conflict("Insufficient stock").WithField("product", fmt.Sprintf("%s only has %d units", p.Name, p.Stock))
			}
			if isPharmacy && !p.IsService && p.Stock < l.Qty {
				return apierr.Conflict("Product stock needs reconciliation before sale").WithField("product", p.Name)
			}
			if isPharmacy && !p.IsService {
				if _, configured := medicineProfiles[p.ID]; !configured {
					return apierr.Conflict("Medicine details must be configured before sale").WithField("product", p.Name+" needs a dispensing profile in inventory")
				}
			}
			if profile, ok := medicineProfiles[p.ID]; isPharmacy && ok {
				prescriptionOnly := medicineRequiresPrescription(profile)
				if organisation.OutletType == "addo" && profile.MedicineSchedule == "prescription_only" {
					return apierr.Conflict("This medicine is marked prescription-only and cannot be dispensed by an ADDO outlet").WithField("product", p.Name)
				}
				if prescriptionOnly && !hasPrescriptionRegisterDetails(l) {
					return apierr.BadRequest("Prescription register details and prescriber authenticity confirmation are required").WithField("lines", p.Name+": complete the prescription details")
				}
			}
			lineAllocations := []inventory.MedicineBatchAllocation{}
			lineCost := p.CostPrice * int64(l.Qty)
			if !p.IsService {
				if isPharmacy {
					lineCost = 0
					var batches []inventory.MedicineBatch
					location, err := time.LoadLocation("Africa/Dar_es_Salaam")
					if err != nil {
						location = time.FixedZone("EAT", 3*60*60)
					}
					today := time.Now().In(location)
					today = time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, location)
					if err := tx.Clauses(lockForUpdate).Where("store_id = ? AND product_id = ? AND stock_status = ? AND quantity > 0 AND expiry_date >= ?", storeID, p.ID, "available", today.Format("2006-01-02")).Order("expiry_date asc, created_at asc").Find(&batches).Error; err != nil {
						return err
					}
					fefoAllocations, err := pharmacy.AllocateFEFO(batches, l.Qty, today)
					if err != nil {
						available := 0
						for _, batch := range batches {
							available += batch.Quantity
						}
						return apierr.Conflict("Insufficient unexpired stock").WithField("product", fmt.Sprintf("%s has only %d unexpired units", p.Name, available))
					}
					batchByID := make(map[uuid.UUID]inventory.MedicineBatch, len(batches))
					for _, batch := range batches {
						batchByID[batch.ID] = batch
					}
					for _, allocation := range fefoAllocations {
						batch := batchByID[allocation.BatchID]
						if err := tx.Create(&inventory.MedicineStockMovement{StoreID: storeID, ProductID: p.ID, BatchID: batch.ID, Quantity: -allocation.Quantity, MovementType: "sale", ReferenceID: fmt.Sprintf("TXN-%04d", number), PerformedBy: cashierID}).Error; err != nil {
							return err
						}
						lineAllocations = append(lineAllocations, inventory.MedicineBatchAllocation{BatchID: batch.ID, Quantity: allocation.Quantity})
						lineCost += allocation.UnitCost * int64(allocation.Quantity)
					}
					if err := inventory.ReconcilePharmacyStock(tx, storeID, p.ID, 0); err != nil {
						return err
					}
				}
				if !isPharmacy {
					if err := tx.Model(&inventory.Product{}).Where("id = ?", p.ID).Update("stock", gorm.Expr("stock - ?", l.Qty)).Error; err != nil {
						return err
					}
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
				UnitPrice:   unitPrice,
				UnitCost: func() int64 {
					if isPharmacy && l.Qty > 0 {
						return lineCost / int64(l.Qty)
					}
					return p.CostPrice
				}(),
				Qty: l.Qty,
			})
			allocations = append(allocations, lineAllocations)
			subtotal += unitPrice * int64(l.Qty)
			costTotal += lineCost
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
		if err := s.repo.CreateSaleTx(tx, txn); err != nil {
			return err
		}
		if isPharmacy {
			rxSequence := 0
			for i := range txn.Lines {
				if profile, ok := medicineProfiles[*txn.Lines[i].ProductID]; ok && medicineRequiresPrescription(profile) {
					rxSequence++
					rx := &pharmacy.PrescriptionRecord{
						StoreID: storeID, RegisterNumber: fmt.Sprintf("RX-%s-%02d", txn.TxnNumber, rxSequence), TransactionLineID: txn.Lines[i].ID,
						PrescriberName:        strings.TrimSpace(in.Lines[i].PrescriberName),
						PrescriberAddress:     strings.TrimSpace(in.Lines[i].PrescriberAddress),
						PrescriberRegNumber:   strings.TrimSpace(in.Lines[i].PrescriberRegistrationNumber),
						PrescriberVerified:    in.Lines[i].PrescriberAuthenticityChecked,
						FacilityName:          strings.TrimSpace(in.Lines[i].PrescribingFacilityName),
						FacilityAddress:       strings.TrimSpace(in.Lines[i].PrescribingFacilityAddress),
						PrescriptionReference: strings.TrimSpace(in.Lines[i].PrescriptionReference),
						PatientName:           strings.TrimSpace(in.Lines[i].PatientName),
						PatientAddress:        strings.TrimSpace(in.Lines[i].PatientAddress),
						MedicineStrength:      strings.TrimSpace(in.Lines[i].MedicineStrength),
						DosageForm:            strings.TrimSpace(in.Lines[i].DosageForm), RecordedBy: cashierID,
					}
					if err := tx.Create(rx).Error; err != nil {
						return err
					}
				}
				for _, allocation := range allocations[i] {
					allocation.TransactionLineID = txn.Lines[i].ID
					if err := tx.Create(&allocation).Error; err != nil {
						return err
					}
				}
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	s.invalidateStoreCaches(storeID, orgID, ids...)
	s.bus.Publish("transaction.created", map[string]interface{}{
		"transaction_id": txn.ID,
		"store_id":       storeID,
		"org_id":         orgID,
		"total":          txn.Total,
		"method":         method,
		"status":         txn.Status,
		"customer_id":    txn.CustomerID,
		"customer_phone": txn.CustomerPhone,
		"customer_name":  txn.CustomerName,
		"cashier_id":     cashierID,
		"till_number":    txn.TillNumber,
		"txn_number":     txn.TxnNumber,
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
	isPharmacy, err := s.isPharmacy(ctx, orgID)
	if err != nil {
		return nil, err
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
			if !isPharmacy {
				if err := tx.Model(&inventory.Product{}).Where("id = ?", *l.ProductID).Update("stock", gorm.Expr("stock + ?", l.Qty)).Error; err != nil {
					return err
				}
			}
			if err := restoreMedicineBatches(tx, l.ID, storeID, *l.ProductID, userID, txn.TxnNumber, "sale_return", isPharmacy, l.Qty); err != nil {
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
		"total": txn.Total, "customer_id": txn.CustomerID,
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
	isPharmacy, err := s.isPharmacy(ctx, orgID)
	if err != nil {
		return nil, err
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
			if !isPharmacy {
				if err := tx.Model(&inventory.Product{}).Where("id = ?", *l.ProductID).Update("stock", gorm.Expr("stock + ?", l.Qty)).Error; err != nil {
					return err
				}
			}
			if err := restoreMedicineBatches(tx, l.ID, storeID, *l.ProductID, uuid.Nil, txn.TxnNumber, "sale_void", isPharmacy, l.Qty); err != nil {
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
	s.bus.Publish("transaction.voided", map[string]interface{}{
		"transaction_id": id, "store_id": storeID, "reason": reason,
		"total": txn.Total, "customer_id": txn.CustomerID,
	})
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
		"date_from":    from,
		"date_to":      to,
		"gross_sales":  gross,
		"refunded":     refunded,
		"net_sales":    net,
		"profit":       profit,
		"tax":          tax,
		"transactions": txns,
		"by_method":    methods,
		"top_products": top,
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

// EmailReceipt emails a copy of the receipt to an address supplied in the
// request, or to the customer's saved email address.
func (s *Service) EmailReceipt(ctx context.Context, orgID, storeID, txnID uuid.UUID, email string) error {
	if s.mailer == nil {
		return apierr.Internal("Email service is not configured")
	}
	txn, err := s.GetTransaction(ctx, storeID, txnID)
	if err != nil {
		return err
	}
	dest := strings.TrimSpace(email)
	if dest == "" && txn.CustomerID != nil {
		var c struct {
			Email string
		}
		if err := s.accountsDB.Table("customers").Select("email").Where("id = ?", *txn.CustomerID).Scan(&c).Error; err == nil && strings.TrimSpace(c.Email) != "" {
			dest = strings.TrimSpace(c.Email)
		}
	}
	if dest == "" {
		return apierr.BadRequest("No email address — supply one in the request or add an email to the customer")
	}

	var storeName, orgName string
	if err := s.accountsDB.Table("stores").Select("name").Where("id = ?", storeID).Scan(&storeName).Error; err != nil {
		storeName = ""
	}
	if err := s.accountsDB.Table("organisations").Select("name").Where("id = ?", orgID).Scan(&orgName).Error; err != nil {
		orgName = ""
	}

	body := buildReceiptHTML(txn, storeName, orgName)
	return s.mailer.Send(dest, "Ziada POS receipt "+txn.TxnNumber, s.mailer.Layout("Receipt "+txn.TxnNumber, body))
}

// buildReceiptHTML renders a printable email receipt for a transaction.
func buildReceiptHTML(t *Transaction, storeName, orgName string) string {
	var rows strings.Builder
	for _, l := range t.Lines {
		fmt.Fprintf(&rows,
			`<tr><td style="padding:6px 0;border-bottom:1px solid #f3f4f6;">%s <span style="color:#9ca3af;">&times; %d</span></td>`+
				`<td style="padding:6px 0;border-bottom:1px solid #f3f4f6;text-align:right;white-space:nowrap;">TZS %s</td></tr>`,
			html.EscapeString(l.ProductName), l.Qty, fmtTZS(l.UnitPrice*int64(l.Qty)))
	}
	summaryRow := func(label string, amount int64, strong bool) string {
		weight := "400"
		if strong {
			weight = "700"
		}
		return fmt.Sprintf(`<tr><td style="padding:3px 0;color:#6b7280;">%s</td><td style="padding:3px 0;text-align:right;white-space:nowrap;font-weight:%s;">TZS %s</td></tr>`, label, weight, fmtTZS(amount))
	}

	return `<table width="100%" cellpadding="0" cellspacing="0" style="width:100%;">
  <tr><td style="font-size:16px;font-weight:700;color:#111827;">` + html.EscapeString(orgName) + `</td></tr>
  <tr><td style="font-size:13px;color:#6b7280;padding-top:2px;">` + html.EscapeString(storeName) + `</td></tr>
  <tr><td style="padding:14px 0;border-bottom:1px solid #e5e7eb;"></td></tr>
  <tr><td style="padding:10px 0;">
    <table width="100%" cellpadding="0" cellspacing="0" style="font-size:13px;">
      <tr><td style="color:#6b7280;padding:2px 0;">Receipt</td><td style="text-align:right;font-weight:600;">` + html.EscapeString(t.TxnNumber) + `</td></tr>
      <tr><td style="color:#6b7280;padding:2px 0;">Date</td><td style="text-align:right;">` + t.CreatedAt.Format("02 Jan 2006, 15:04") + `</td></tr>
      <tr><td style="color:#6b7280;padding:2px 0;">Customer</td><td style="text-align:right;">` + html.EscapeString(t.CustomerName) + `</td></tr>
      <tr><td style="color:#6b7280;padding:2px 0;">Payment</td><td style="text-align:right;">` + html.EscapeString(t.PaymentMethod) + `</td></tr>
    </table>
  </td></tr>
  <tr><td style="padding:6px 0;border-top:1px solid #e5e7eb;border-bottom:1px solid #e5e7eb;">
    <table width="100%" cellpadding="0" cellspacing="0" style="font-size:14px;">` + rows.String() + `</table>
  </td></tr>
  <tr><td>
    <table width="100%" cellpadding="0" cellspacing="0" style="font-size:13px;padding-top:6px;">
      ` + summaryRow("Subtotal", t.Subtotal, false) + `
      ` + summaryRow(fmt.Sprintf("Discount (%.0f%%)", t.DiscountPct), -t.DiscountAmount, false) + `
      ` + summaryRow("VAT (18%)", -t.TaxAmount, false) + `
      ` + summaryRow("Total", t.Total, true) + `
    </table>
  </td></tr>
</table>`
}

// fmtTZS renders an integer TZS amount with thousands separators.
func fmtTZS(n int64) string {
	sign := ""
	if n < 0 {
		sign = "-"
		n = -n
	}
	s := fmt.Sprintf("%d", n)
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	return sign + s
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
