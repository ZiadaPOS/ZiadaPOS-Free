// Package customers manages the store's registered buyers and their spend
// statistics, which feed segments and the credit system.
package customers

import (
	"math/rand"
	"time"

	"github.com/google/uuid"
	"github.com/troubleman96/ziadapos/internal/models"
)

// Segments assigned to customers based on spend.
const (
	SegmentVIP        = "VIP"
	SegmentRegular    = "Regular"
	SegmentOccasional = "Occasional"
	SegmentNew        = "New"
)

// Spend thresholds (TZS) used by RecomputeSegment.
const (
	VIPThreshold     = 1_000_000
	RegularThreshold = 200_000
)

// Customer is a registered buyer in a store.
type Customer struct {
	models.BaseModel
	OrganisationID uuid.UUID  `gorm:"type:uuid;not null;index" json:"organisation_id"`
	StoreID        uuid.UUID  `gorm:"type:uuid;not null;index" json:"store_id"`
	Name           string     `gorm:"size:200;not null;index" json:"name"`
	Phone          *string    `gorm:"size:30;uniqueIndex:idx_customer_store_phone" json:"phone"`
	Email          string     `gorm:"size:150" json:"email"`
	Segment        string     `gorm:"size:20;default:New;index" json:"segment"`
	TotalSpent     int64      `gorm:"not null;default:0" json:"total_spent"`
	LastVisit      *time.Time `json:"last_visit,omitempty"`
	AvgTicket      int64      `gorm:"not null;default:0" json:"avg_ticket"`
	OpenCredit     int64      `gorm:"not null;default:0" json:"open_credit"`
	CreditLimit    *int64     `json:"credit_limit,omitempty"`
	AvatarHue      int        `gorm:"not null;default:200" json:"avatar_hue"`
	Notes          string     `gorm:"size:1000" json:"notes"`
	IsActive       bool       `gorm:"not null;default:true;index" json:"is_active"`
}

// HasOpenCredit reports whether the customer owes money.
func (c *Customer) HasOpenCredit() bool { return c.OpenCredit > 0 }

// Initials returns up to two initials for the avatar fallback.
func (c *Customer) Initials() string {
	out := make([]byte, 0, 2)
	words := splitWords(c.Name)
	for _, w := range words {
		if len(out) >= 2 {
			break
		}
		if w != "" {
			out = append(out, w[0])
		}
	}
	for i := range out {
		if out[i] >= 'a' && out[i] <= 'z' {
			out[i] -= 32
		}
	}
	return string(out)
}

// RecomputeSegment assigns a loyalty segment from current spend.
func (c *Customer) RecomputeSegment() {
	switch {
	case c.TotalSpent >= VIPThreshold:
		c.Segment = SegmentVIP
	case c.TotalSpent >= RegularThreshold:
		c.Segment = SegmentRegular
	case c.TotalSpent > 0:
		c.Segment = SegmentOccasional
	default:
		c.Segment = SegmentNew
	}
}

func splitWords(s string) []string {
	var words []string
	cur := ""
	for _, r := range s {
		if r == ' ' || r == '\t' {
			if cur != "" {
				words = append(words, cur)
				cur = ""
			}
			continue
		}
		cur += string(r)
	}
	if cur != "" {
		words = append(words, cur)
	}
	return words
}

func randomHue() int { return rand.Intn(360) }
