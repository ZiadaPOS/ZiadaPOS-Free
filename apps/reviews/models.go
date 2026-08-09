// Package reviews implements the landing-page store reviews feature.
package reviews

import (
	"github.com/google/uuid"

	"github.com/troubleman96/ziadapos/internal/models"
)

// StoreReview is a rating submitted by an organisation owner/staff member.
type StoreReview struct {
	models.BaseModel
	OrganisationID uuid.UUID  `gorm:"type:uuid;not null;index" json:"organisation_id"`
	ReviewerID     *uuid.UUID `gorm:"type:uuid;index" json:"reviewer_id,omitempty"`

	Rating int    `gorm:"not null" json:"rating"`
	Title  string `gorm:"size:200;default:''" json:"title"`
	Body   string `gorm:"type:text" json:"body"`

	BusinessTypeDisplay string `gorm:"size:100;default:''" json:"business_type_display"`
	RegionDisplay       string `gorm:"size:100;default:''" json:"region_display"`
	Trigger             string `gorm:"size:20;default:manual" json:"trigger"`

	IsPublic bool `gorm:"default:false;index" json:"is_public"`
}
