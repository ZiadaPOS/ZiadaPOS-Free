// Package notebook implements the shared store notebook (staff memos).
package notebook

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

// Tags is a JSONB array of note tag strings.
type Tags []string

// Value marshals tags for storage.
func (t Tags) Value() (driver.Value, error) {
	if t == nil {
		return "[]", nil
	}
	return json.Marshal(t)
}

// Scan unmarshals a stored tags value.
func (t *Tags) Scan(src interface{}) error {
	if src == nil {
		*t = nil
		return nil
	}
	var b []byte
	switch v := src.(type) {
	case []byte:
		b = v
	case string:
		b = []byte(v)
	default:
		return errors.New("notebook: unsupported tags scan type")
	}
	return json.Unmarshal(b, t)
}

// GormDBDataType maps Tags to a JSONB column.
func (Tags) GormDBDataType(db *gorm.DB, field *schema.Field) string {
	return "jsonb"
}

// GormValue writes the value using its driver Value.
func (t Tags) GormValue(ctx *gorm.DB, _ *schema.Field) interface{} {
	v, err := t.Value()
	if err != nil {
		return nil
	}
	return v
}

// Note is a single notebook entry visible to all staff at a store.
type Note struct {
	models.BaseModel
	StoreID        uuid.UUID  `gorm:"type:uuid;not null;index:idx_note_store_created,priority:1" json:"store_id"`
	OrganisationID uuid.UUID  `gorm:"type:uuid;not null;index" json:"organisation_id"`
	Title          string     `gorm:"size:300;not null" json:"title"`
	Content        string     `gorm:"type:text" json:"content"`
	Tags           Tags       `gorm:"type:jsonb;not null;default:'[]'" json:"tags"`
	CreatedBy      *uuid.UUID `gorm:"type:uuid;index" json:"created_by,omitempty"`

	CreatedByName string `gorm:"-" json:"created_by_name"`
	DateLabel     string `gorm:"-" json:"date_label"`
}

// TableName maps to the notes table.
func (Note) TableName() string { return "notes" }

// buildDateLabel derives the relative UI label from the created time.
func (n *Note) buildDateLabel(now time.Time) {
	diff := now.Sub(n.CreatedAt)
	mins := int(diff.Minutes())
	hours := int(diff.Hours())
	days := int(diff.Hours() / 24)

	switch {
	case mins < 2:
		n.DateLabel = "Just now"
	case hours < 24 && days == 0:
		n.DateLabel = "Today"
	case days == 1:
		n.DateLabel = "Yesterday"
	case days < 7:
		n.DateLabel = daysLabel(days)
	case days < 14:
		n.DateLabel = "1w ago"
	default:
		n.DateLabel = n.CreatedAt.Format("2 Jan")
	}
}

func daysLabel(days int) string {
	return strDays(days) + "d ago"
}

func strDays(days int) string {
	if days == 0 {
		return "0"
	}
	n := days
	digits := []byte{}
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}
