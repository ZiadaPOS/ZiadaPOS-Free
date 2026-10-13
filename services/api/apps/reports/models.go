// Package reports generates downloadable business reports (sales summary,
// inventory valuation, tax statement, credit aged debtors) and manages
// scheduled delivery plus the export audit trail.
package reports

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"

	"github.com/troubleman96/ziadapos/internal/models"
)

// Report types.
const (
	ReportSales     = "sales"
	ReportInventory = "inventory"
	ReportTax       = "tax"
	ReportCredit    = "credit"
)

// ReportTypeNames maps a report type to its human-readable display name.
var ReportTypeNames = map[string]string{
	ReportSales:     "Sales Summary",
	ReportInventory: "Inventory Valuation",
	ReportTax:       "Tax Statement",
	ReportCredit:    "Credit Aged Debtors",
}

// Export formats.
const (
	FormatCSV  = "csv"
	FormatJSON = "json"
)

// Delivery frequencies for scheduled reports.
const (
	FreqDaily   = "daily"
	FreqWeekly  = "weekly"
	FreqMonthly = "monthly"
)

// ValidFrequencies lists accepted schedule frequencies.
var ValidFrequencies = []string{FreqDaily, FreqWeekly, FreqMonthly}

// Date range presets for scheduled reports.
const (
	Range7D    = "7d"
	Range30D   = "30d"
	Range90D   = "90d"
	RangeMonth = "month"
	RangeYTD   = "ytd"
)

// ValidRangePresets lists accepted range presets.
var ValidRangePresets = []string{Range7D, Range30D, Range90D, RangeMonth, RangeYTD}

// Recipients is a JSONB list of email addresses.
type Recipients []string

// Value marshals recipients for storage.
func (r Recipients) Value() (driver.Value, error) {
	if r == nil {
		return "[]", nil
	}
	return json.Marshal(r)
}

// Scan unmarshals a stored recipients value.
func (r *Recipients) Scan(src interface{}) error {
	if src == nil {
		*r = nil
		return nil
	}
	var b []byte
	switch v := src.(type) {
	case []byte:
		b = v
	case string:
		b = []byte(v)
	default:
		return errors.New("reports: unsupported recipients scan type")
	}
	return json.Unmarshal(b, r)
}

// GormDBDataType maps Recipients to a JSONB column.
func (Recipients) GormDBDataType(db *gorm.DB, field *schema.Field) string {
	return "jsonb"
}

// GormValue writes the value using its driver Value.
func (r Recipients) GormValue(ctx *gorm.DB, _ *schema.Field) interface{} {
	v, err := r.Value()
	if err != nil {
		return nil
	}
	return v
}

// ScheduledReport is a recurring report configuration that emails a CSV/JSON
// report to a list of recipients on a daily/weekly/monthly cadence.
type ScheduledReport struct {
	models.BaseModel
	StoreID         uuid.UUID  `gorm:"type:uuid;not null;index:idx_sched_store_enabled,priority:1;index" json:"store_id"`
	OrganisationID  uuid.UUID  `gorm:"type:uuid;not null;index" json:"organisation_id"`
	CreatedBy       uuid.UUID  `gorm:"type:uuid;index" json:"created_by"`
	ReportType      string     `gorm:"size:20;not null;index" json:"report_type"`
	Name            string     `gorm:"size:200;not null" json:"name"`
	Frequency       string     `gorm:"size:10;not null;default:daily" json:"frequency"`
	DateRangePreset string     `gorm:"size:10;default:30d" json:"date_range_preset"`
	Recipients      Recipients `gorm:"type:jsonb;not null;default:'[]'" json:"recipients"`
	IsEnabled       bool       `gorm:"default:true;index:idx_sched_store_enabled,priority:2" json:"is_enabled"`
	LastSentAt      *time.Time `json:"last_sent_at"`
	NextSendAt      *time.Time `json:"next_send_at"`
}

// TableName maps to the scheduled_reports table.
func (ScheduledReport) TableName() string { return "scheduled_reports" }

// ReportExport is an audit record of every generated report. The file itself
// is regenerated on demand from the stored date bounds — nothing is persisted.
type ReportExport struct {
	models.BaseModel
	StoreID           uuid.UUID  `gorm:"type:uuid;not null;index:idx_export_store_type,priority:1;index:idx_export_store_created,priority:1" json:"store_id"`
	OrganisationID    uuid.UUID  `gorm:"type:uuid;not null;index" json:"organisation_id"`
	CreatedBy         uuid.UUID  `gorm:"type:uuid;index" json:"created_by"`
	ReportType        string     `gorm:"size:20;not null;index:idx_export_store_type,priority:2" json:"report_type"`
	Name              string     `gorm:"size:200;not null" json:"name"`
	PeriodLabel       string     `gorm:"size:100;not null" json:"period_label"`
	DateFrom          time.Time  `gorm:"type:date" json:"date_from"`
	DateTo            time.Time  `gorm:"type:date" json:"date_to"`
	Format            string     `gorm:"size:10;not null;default:csv" json:"format"`
	FileSizeBytes     int        `gorm:"default:0" json:"file_size_bytes"`
	ScheduledReportID *uuid.UUID `gorm:"type:uuid;index" json:"scheduled_report_id,omitempty"`
}

// TableName maps to the report_exports table.
func (ReportExport) TableName() string { return "report_exports" }
