package reviews

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Repo provides review data access.
type Repo struct {
	db *gorm.DB
}

// NewRepo builds a Repo.
func NewRepo(db *gorm.DB) *Repo { return &Repo{db: db} }

// Create persists a review.
func (r *Repo) Create(ctx context.Context, review *StoreReview) error {
	return r.db.Create(review).Error
}

// PublicReviews returns approved (4–5 star) reviews for the landing page.
func (r *Repo) PublicReviews(ctx context.Context, limit int) ([]reviewView, error) {
	var rows []reviewView
	err := r.db.Table("store_reviews AS rv").
		Select(`
			rv.id, rv.rating, rv.title, rv.body,
			rv.business_type_display, rv.region_display, rv.created_at,
			COALESCE(o.name, '') AS org_name,
			COALESCE(NULLIF(trim(u.first_name || ' ' || u.last_name), ''), COALESCE(u.phone, 'Anonymous'), 'Anonymous') AS reviewer_name`).
		Joins("LEFT JOIN organisations AS o ON o.id = rv.organisation_id").
		Joins("LEFT JOIN users AS u ON u.id = rv.reviewer_id").
		Where("rv.is_public = true AND rv.rating >= 4").
		Order("rv.created_at desc").
		Limit(limit).
		Scan(&rows).Error
	return rows, err
}

// OrgSnapshot reads the organisation display fields for a review.
func (r *Repo) OrgSnapshot(ctx context.Context, orgID uuid.UUID) (businessType, region string, err error) {
	var o struct {
		BusinessType string `gorm:"column:business_type"`
		Region       string `gorm:"column:region"`
	}
	err = r.db.Table("organisations").
		Select("COALESCE(business_type,''), COALESCE(region,'')").
		Where("id = ?", orgID).
		Scan(&o).Error
	return o.BusinessType, o.Region, err
}

// reviewView is a public review joined with org + reviewer display names.
type reviewView struct {
	ID                  uuid.UUID `gorm:"column:id"`
	Rating              int       `gorm:"column:rating"`
	Title               string    `gorm:"column:title"`
	Body                string    `gorm:"column:body"`
	BusinessTypeDisplay string    `gorm:"column:business_type_display"`
	RegionDisplay       string    `gorm:"column:region_display"`
	OrgName             string    `gorm:"column:org_name"`
	ReviewerName        string    `gorm:"column:reviewer_name"`
	CreatedAt           string    `gorm:"column:created_at"`
}
