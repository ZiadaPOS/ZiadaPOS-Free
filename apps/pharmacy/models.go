package pharmacy

import (
	"time"

	"github.com/google/uuid"
	"github.com/troubleman96/ziadapos/internal/models"
)

// MedicineProfile stores product-specific dispensing and identification data.
// Schedule values are shop-entered scaffolding and are not legal classification.
type MedicineProfile struct {
	models.BaseModel
	StoreID              uuid.UUID `gorm:"type:uuid;not null;index" json:"store_id"`
	ProductID            uuid.UUID `gorm:"type:uuid;not null;uniqueIndex" json:"product_id"`
	GenericName          string    `gorm:"size:200" json:"generic_name"`
	BrandName            string    `gorm:"size:200" json:"brand_name"`
	Unit                 string    `gorm:"size:50" json:"unit"`
	PackSize             string    `gorm:"size:50" json:"pack_size"`
	RequiresPrescription bool      `gorm:"not null;default:false" json:"requires_prescription"`
	MedicineSchedule     string    `gorm:"size:30;not null;default:unscheduled" json:"medicine_schedule"`
	RegistrationNumber   string    `gorm:"size:100" json:"registration_number"`
}

// ExpiryAlertRun prevents a scheduler restart from creating duplicate daily
// digests for a store.
type ExpiryAlertRun struct {
	models.BaseModel
	StoreID  uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_store_expiry_alert_day" json:"store_id"`
	AlertDay time.Time `gorm:"type:date;not null;uniqueIndex:idx_store_expiry_alert_day" json:"alert_day"`
}

// PrescriptionRecord is attached to a sold line when the pharmacy requires a
// prescription reference. It records the supplied reference, not a legal review.
type PrescriptionRecord struct {
	models.BaseModel
	StoreID               uuid.UUID `gorm:"type:uuid;not null;index" json:"store_id"`
	RegisterNumber        string    `gorm:"size:50;not null;default:''" json:"register_number"`
	TransactionLineID     uuid.UUID `gorm:"type:uuid;not null;uniqueIndex" json:"transaction_line_id"`
	PrescriberName        string    `gorm:"size:200;not null" json:"prescriber_name"`
	PrescriberAddress     string    `gorm:"size:500" json:"prescriber_address"`
	PrescriberRegNumber   string    `gorm:"size:100" json:"prescriber_registration_number"`
	PrescriberVerified    bool      `gorm:"not null;default:false" json:"prescriber_authenticity_confirmed"`
	FacilityName          string    `gorm:"size:200" json:"prescribing_facility_name"`
	FacilityAddress       string    `gorm:"size:500" json:"prescribing_facility_address"`
	PrescriptionReference string    `gorm:"size:200;not null" json:"prescription_reference"`
	PatientName           string    `gorm:"size:200" json:"patient_name"`
	PatientAddress        string    `gorm:"size:500" json:"patient_address"`
	MedicineStrength      string    `gorm:"size:100" json:"medicine_strength"`
	DosageForm            string    `gorm:"size:100" json:"dosage_form"`
	RecordedBy            uuid.UUID `gorm:"type:uuid;not null" json:"recorded_by"`
}
