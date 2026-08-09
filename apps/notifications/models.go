// Package notifications records in-app notifications for staff and owners.
// Notifications are created automatically by domain events (low stock, credit
// activity, big sales, new team members) and surfaced to every client.
package notifications

import (
	"time"

	"github.com/google/uuid"

	"github.com/troubleman96/ziadapos/internal/models"
)

// Notification categories (match the client icon/label map).
const (
	CatStock    = "stock"
	CatCredit   = "credit"
	CatSales    = "sales"
	CatAI       = "ai"
	CatSystem   = "system"
	CatSupplier = "supplier"
)

// ValidCategories lists the notification categories clients understand.
var ValidCategories = []string{CatStock, CatCredit, CatSales, CatAI, CatSystem, CatSupplier}

// Notification is an in-app alert scoped to an organisation (optionally to a
// single store and/or a specific user).
type Notification struct {
	models.BaseModel
	OrganisationID uuid.UUID  `gorm:"type:uuid;not null;index" json:"organisation_id"`
	StoreID        uuid.UUID  `gorm:"type:uuid;index" json:"store_id"`
	UserID         uuid.UUID  `gorm:"type:uuid;index" json:"user_id"`
	Category       string     `gorm:"size:20;not null;index" json:"category"`
	Title          string     `gorm:"size:255;not null" json:"title"`
	Body           string     `gorm:"type:text" json:"body"`
	Link           string     `gorm:"size:255" json:"link"`
	Read           bool       `gorm:"not null;default:false;index" json:"read"`
	ReadAt         *time.Time `json:"read_at"`
}

// Filter is the list input for Get.
type Filter struct {
	Category string
	Unread   bool
	Page     int
	PageSize int
}
