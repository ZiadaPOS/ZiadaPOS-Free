package pharmacy

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/troubleman96/ziadapos/apps/accounts"
	"github.com/troubleman96/ziadapos/apps/inventory"
	"github.com/troubleman96/ziadapos/internal/apierr"
)

type Service struct{ db *gorm.DB }

func NewService(db *gorm.DB) *Service { return &Service{db: db} }

type MedicineProfileInput struct {
	GenericName          string `json:"generic_name"`
	BrandName            string `json:"brand_name"`
	Unit                 string `json:"unit"`
	PackSize             string `json:"pack_size"`
	RequiresPrescription bool   `json:"requires_prescription"`
	MedicineSchedule     string `json:"medicine_schedule"`
	RegistrationNumber   string `json:"registration_number"`
}

func (s *Service) requirePharmacy(ctx context.Context, storeID uuid.UUID) error {
	var store accounts.Store
	if err := s.db.WithContext(ctx).Select("id", "organisation_id").First(&store, "id = ?", storeID).Error; err != nil {
		return apierr.NotFound("Store not found")
	}
	var org accounts.Organisation
	if err := s.db.WithContext(ctx).Select("id", "business_type").First(&org, "id = ?", store.OrganisationID).Error; err != nil {
		return err
	}
	if org.BusinessType != "pharmacy" {
		return apierr.Conflict("This workflow is available for pharmacy accounts")
	}
	return nil
}

func (s *Service) UpdateMedicineProfile(ctx context.Context, storeID, productID uuid.UUID, in MedicineProfileInput) (*MedicineProfile, error) {
	if err := s.requirePharmacy(ctx, storeID); err != nil {
		return nil, err
	}
	var product inventory.Product
	if err := s.db.WithContext(ctx).Where("id = ? AND store_id = ?", productID, storeID).First(&product).Error; err != nil {
		return nil, apierr.NotFound("Product not found")
	}
	schedule := strings.TrimSpace(in.MedicineSchedule)
	if schedule == "" {
		schedule = "otc"
	}
	switch schedule {
	case "otc", "addo_permitted", "prescription_only", "unscheduled":
	default:
		return nil, apierr.BadRequest("Unknown medicine schedule").WithField("medicine_schedule", "unsupported")
	}
	profile := MedicineProfile{StoreID: storeID, ProductID: productID, GenericName: strings.TrimSpace(in.GenericName), BrandName: strings.TrimSpace(in.BrandName), Unit: strings.TrimSpace(in.Unit), PackSize: strings.TrimSpace(in.PackSize), RequiresPrescription: in.RequiresPrescription, MedicineSchedule: schedule, RegistrationNumber: strings.TrimSpace(in.RegistrationNumber)}
	err := s.db.WithContext(ctx).Where("product_id = ?", productID).Assign(profile).FirstOrCreate(&profile).Error
	if err != nil {
		return nil, err
	}
	return &profile, nil
}

func (s *Service) GetMedicineProfile(ctx context.Context, storeID, productID uuid.UUID) (*MedicineProfile, error) {
	if err := s.requirePharmacy(ctx, storeID); err != nil {
		return nil, err
	}
	var profile MedicineProfile
	if err := s.db.WithContext(ctx).Where("store_id = ? AND product_id = ?", storeID, productID).First(&profile).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return &MedicineProfile{StoreID: storeID, ProductID: productID, MedicineSchedule: "unscheduled"}, nil
		}
		return nil, err
	}
	return &profile, nil
}

func (s *Service) ListMedicineProfiles(ctx context.Context, storeID uuid.UUID) ([]MedicineProfile, error) {
	if err := s.requirePharmacy(ctx, storeID); err != nil {
		return nil, err
	}
	profiles := make([]MedicineProfile, 0)
	if err := s.db.WithContext(ctx).Where("store_id = ?", storeID).Order("generic_name asc").Find(&profiles).Error; err != nil {
		return nil, err
	}
	return profiles, nil
}

type DisposeBatchInput struct {
	Quantity int    `json:"quantity"`
	Reason   string `json:"reason"`
}

type ReleaseReturnedBatchInput struct {
	InspectionNote string `json:"inspection_note"`
}

// ReleaseReturnedBatch makes a returned lot sellable only after an owner or
// administrator records an inspection outcome. The HTTP route enforces role.
func (s *Service) ReleaseReturnedBatch(ctx context.Context, storeID, batchID, userID uuid.UUID, in ReleaseReturnedBatchInput) error {
	if err := s.requirePharmacy(ctx, storeID); err != nil {
		return err
	}
	in.InspectionNote = strings.TrimSpace(in.InspectionNote)
	if in.InspectionNote == "" || len(in.InspectionNote) > 500 {
		return apierr.BadRequest("An inspection note is required").WithField("inspection_note", "must be 1 to 500 characters")
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var batch inventory.MedicineBatch
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND store_id = ?", batchID, storeID).First(&batch).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return apierr.NotFound("Medicine batch not found")
			}
			return err
		}
		if batch.StockStatus != "quarantined" {
			return apierr.Conflict("Only quarantined returned stock can be released")
		}
		if _, err := inventory.ParseMedicineExpiryDate(batch.ExpiryDate.Format("2006-01-02"), time.Now()); err != nil {
			return apierr.Conflict("Expired medicine cannot be released for sale")
		}
		result := tx.Model(&inventory.MedicineBatch{}).Where("id = ? AND store_id = ? AND stock_status = ?", batchID, storeID, "quarantined").Update("stock_status", "available")
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return apierr.Conflict("Batch status changed; reload and try again")
		}
		return tx.Create(&inventory.MedicineStockMovement{
			StoreID: storeID, ProductID: batch.ProductID, BatchID: batch.ID,
			MovementType: "quarantine_release", ReferenceID: batch.ID.String(),
			Reason: in.InspectionNote, PerformedBy: userID,
		}).Error
	})
}

// DisposeBatch removes a damaged, expired, recalled or otherwise unusable
// quantity from a specific lot while preserving its movement history.
func (s *Service) DisposeBatch(ctx context.Context, storeID, batchID, userID uuid.UUID, in DisposeBatchInput) error {
	if err := s.requirePharmacy(ctx, storeID); err != nil {
		return err
	}
	in.Reason = strings.TrimSpace(in.Reason)
	if in.Quantity <= 0 || in.Reason == "" || len(in.Reason) > 500 {
		return apierr.BadRequest("A positive quantity and disposal reason are required").WithField("quantity", "quantity must be positive and reason must be 1 to 500 characters")
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var batch inventory.MedicineBatch
		if err := tx.Where("id = ? AND store_id = ?", batchID, storeID).First(&batch).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return apierr.NotFound("Medicine batch not found")
			}
			return err
		}
		var product inventory.Product
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND store_id = ?", batch.ProductID, storeID).First(&product).Error; err != nil {
			return err
		}
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND store_id = ?", batchID, storeID).First(&batch).Error; err != nil {
			return err
		}
		if in.Quantity > batch.Quantity {
			return apierr.Conflict("Disposal quantity exceeds the batch quantity").WithField("quantity", "reduce quantity to the available batch balance")
		}
		if err := tx.Create(&inventory.MedicineStockMovement{StoreID: storeID, ProductID: batch.ProductID, BatchID: batch.ID, Quantity: -in.Quantity, MovementType: "disposal", Reason: in.Reason, PerformedBy: userID}).Error; err != nil {
			return err
		}
		if err := tx.Create(&inventory.StockAdjustment{ProductID: batch.ProductID, MedicineBatchID: &batch.ID, QtyChange: -in.Quantity, Reason: inventory.ReasonDisposal, Note: in.Reason, CreatedBy: userID}).Error; err != nil {
			return err
		}
		return inventory.ReconcilePharmacyStock(tx, storeID, batch.ProductID, 0)
	})
}

type ExpiryItem struct {
	BatchID         uuid.UUID `json:"batch_id"`
	ProductID       uuid.UUID `json:"product_id"`
	ProductName     string    `json:"product_name"`
	BatchNumber     string    `json:"batch_number"`
	Quantity        int       `json:"quantity"`
	ExpiryDate      time.Time `json:"expiry_date"`
	DaysUntilExpiry int       `json:"days_until_expiry"`
	State           string    `json:"state"`
}

type PrescriptionAuditItem struct {
	ID                    uuid.UUID `json:"id"`
	RegisterNumber        string    `json:"register_number"`
	TransactionLineID     uuid.UUID `json:"transaction_line_id"`
	TransactionNumber     string    `json:"transaction_number"`
	TransactionStatus     string    `json:"transaction_status"`
	ProductName           string    `json:"product_name"`
	Quantity              int       `json:"quantity"`
	PrescriberName        string    `json:"prescriber_name"`
	PrescriberAddress     string    `json:"prescriber_address"`
	PrescriberRegNumber   string    `json:"prescriber_registration_number"`
	PrescriberVerified    bool      `json:"prescriber_authenticity_confirmed"`
	FacilityName          string    `json:"prescribing_facility_name"`
	FacilityAddress       string    `json:"prescribing_facility_address"`
	PrescriptionReference string    `json:"prescription_reference"`
	PatientName           string    `json:"patient_name"`
	PatientAddress        string    `json:"patient_address"`
	MedicineStrength      string    `json:"medicine_strength"`
	DosageForm            string    `json:"dosage_form"`
	BatchNumbers          string    `json:"batch_numbers"`
	RecordedBy            uuid.UUID `json:"recorded_by"`
	RecordedAt            time.Time `json:"recorded_at"`
	SaleAt                time.Time `json:"sale_at"`
}

type StockMovementAuditItem struct {
	ID           uuid.UUID `json:"id"`
	MovementType string    `json:"movement_type"`
	Quantity     int       `json:"quantity"`
	ReferenceID  string    `json:"reference_id,omitempty"`
	Reason       string    `json:"reason,omitempty"`
	PerformedBy  uuid.UUID `json:"performed_by"`
	CreatedAt    time.Time `json:"created_at"`
}

func (s *Service) ListBatchMovements(ctx context.Context, storeID, batchID uuid.UUID, page, pageSize int) ([]StockMovementAuditItem, int64, error) {
	if err := s.requirePharmacy(ctx, storeID); err != nil {
		return nil, 0, err
	}
	var batch inventory.MedicineBatch
	if err := s.db.WithContext(ctx).Select("id").Where("id = ? AND store_id = ?", batchID, storeID).First(&batch).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, 0, apierr.NotFound("Medicine batch not found")
		}
		return nil, 0, err
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 50
	}
	if pageSize > 200 {
		pageSize = 200
	}
	query := s.db.WithContext(ctx).Model(&inventory.MedicineStockMovement{}).Where("store_id = ? AND batch_id = ?", storeID, batchID)
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	items := make([]StockMovementAuditItem, 0)
	err := query.Select("id, movement_type, quantity, reference_id, reason, performed_by, created_at").
		Order("created_at DESC, id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Scan(&items).Error
	return items, total, err
}

func (s *Service) ListPrescriptionRecords(ctx context.Context, storeID uuid.UUID, search string, from, to *time.Time, page, pageSize int) ([]PrescriptionAuditItem, int64, error) {
	if err := s.requirePharmacy(ctx, storeID); err != nil {
		return nil, 0, err
	}
	if from != nil && to != nil && !from.Before(*to) {
		return nil, 0, apierr.BadRequest("The from date must be before the to date")
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 50
	}
	if pageSize > 200 {
		pageSize = 200
	}
	query := s.db.WithContext(ctx).Table("prescription_records AS rx").
		Joins("JOIN transaction_lines AS line ON line.id = rx.transaction_line_id").
		Joins("JOIN transactions AS txn ON txn.id = line.transaction_id").
		Where("rx.store_id = ? AND txn.store_id = ?", storeID, storeID)
	if search = strings.TrimSpace(search); search != "" {
		like := "%" + search + "%"
		query = query.Where("(rx.prescriber_name ILIKE ? OR rx.prescription_reference ILIKE ? OR rx.patient_name ILIKE ? OR line.product_name ILIKE ? OR txn.txn_number ILIKE ?)", like, like, like, like, like)
	}
	if from != nil {
		query = query.Where("txn.created_at >= ?", *from)
	}
	if to != nil {
		query = query.Where("txn.created_at < ?", *to)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	items := make([]PrescriptionAuditItem, 0)
	err := query.Select("rx.id, rx.register_number, rx.transaction_line_id, txn.txn_number AS transaction_number, txn.status AS transaction_status, line.product_name, line.qty AS quantity, rx.prescriber_name, rx.prescriber_address, rx.prescriber_reg_number, rx.prescriber_verified, rx.facility_name, rx.facility_address, rx.prescription_reference, rx.patient_name, rx.patient_address, rx.medicine_strength, rx.dosage_form, rx.recorded_by, rx.created_at AS recorded_at, txn.created_at AS sale_at").
		Order("txn.created_at DESC, rx.created_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Scan(&items).Error
	if err != nil {
		return nil, 0, err
	}
	if len(items) == 0 {
		return items, total, nil
	}
	lineIDs := make([]uuid.UUID, 0, len(items))
	for _, item := range items {
		lineIDs = append(lineIDs, item.TransactionLineID)
	}
	var batchRows []struct {
		TransactionLineID uuid.UUID `gorm:"column:transaction_line_id"`
		BatchNumbers      string    `gorm:"column:batch_numbers"`
	}
	if err := s.db.WithContext(ctx).Table("medicine_batch_allocations AS allocation").
		Select("allocation.transaction_line_id, string_agg(DISTINCT batch.batch_number, ', ' ORDER BY batch.batch_number) AS batch_numbers").
		Joins("JOIN medicine_batches AS batch ON batch.id = allocation.batch_id").
		Where("allocation.transaction_line_id IN ?", lineIDs).Group("allocation.transaction_line_id").Scan(&batchRows).Error; err != nil {
		return nil, 0, err
	}
	batchNumbers := make(map[uuid.UUID]string, len(batchRows))
	for _, row := range batchRows {
		batchNumbers[row.TransactionLineID] = row.BatchNumbers
	}
	for i := range items {
		items[i].BatchNumbers = batchNumbers[items[i].TransactionLineID]
	}
	return items, total, nil
}

func (s *Service) ExpiryOverview(ctx context.Context, storeID uuid.UUID, window int) ([]ExpiryItem, error) {
	if err := s.requirePharmacy(ctx, storeID); err != nil {
		return nil, err
	}
	if window <= 0 || window > 365 {
		window = 60
	}
	location, err := time.LoadLocation("Africa/Dar_es_Salaam")
	if err != nil {
		location = time.FixedZone("EAT", 3*60*60)
	}
	today := time.Now().In(location)
	today = time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, today.Location())
	var rows []struct {
		BatchID     uuid.UUID
		ProductID   uuid.UUID
		ProductName string
		BatchNumber string
		Quantity    int
		ExpiryDate  time.Time
	}
	if err := s.db.WithContext(ctx).Table("medicine_batches").Select("medicine_batches.id AS batch_id, medicine_batches.product_id, products.name AS product_name, medicine_batches.batch_number, medicine_batches.quantity, medicine_batches.expiry_date").Joins("JOIN products ON products.id = medicine_batches.product_id").Where("medicine_batches.store_id = ? AND medicine_batches.quantity > 0 AND medicine_batches.expiry_date <= ?", storeID, today.AddDate(0, 0, window)).Order("medicine_batches.expiry_date asc").Limit(500).Scan(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]ExpiryItem, 0, len(rows))
	for _, row := range rows {
		year, month, day := row.ExpiryDate.Date()
		expiry := time.Date(year, month, day, 0, 0, 0, 0, today.Location())
		days := int(expiry.Sub(today).Hours() / 24)
		state := "healthy"
		if days < 0 {
			state = "expired"
		} else if days <= 7 {
			state = "critical"
		} else if days <= 30 {
			state = "expiring_soon"
		} else if days <= 60 {
			state = "upcoming"
		}
		items = append(items, ExpiryItem{BatchID: row.BatchID, ProductID: row.ProductID, ProductName: row.ProductName, BatchNumber: row.BatchNumber, Quantity: row.Quantity, ExpiryDate: row.ExpiryDate, DaysUntilExpiry: days, State: state})
	}
	return items, nil
}
