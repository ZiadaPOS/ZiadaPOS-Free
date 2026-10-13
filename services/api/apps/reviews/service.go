package reviews

import (
	"context"

	"github.com/google/uuid"

	"github.com/troubleman96/ziadapos/internal/apierr"
)

// Service implements review business rules.
type Service struct {
	repo *Repo
}

// NewService wires a reviews Service.
func NewService(repo *Repo) *Service { return &Service{repo: repo} }

// SubmitInput is the authenticated review payload.
type SubmitInput struct {
	Rating  int    `json:"rating"`
	Title   string `json:"title"`
	Body    string `json:"body"`
	Trigger string `json:"trigger"`
}

// Submit records a review for an organisation.
func (s *Service) Submit(ctx context.Context, orgID, reviewerID uuid.UUID, in *SubmitInput) (map[string]interface{}, error) {
	if in.Rating < 1 || in.Rating > 5 {
		return nil, apierr.BadRequest("Invalid rating").WithField("rating", "must be between 1 and 5")
	}
	businessType, region, err := s.repo.OrgSnapshot(ctx, orgID)
	if err != nil {
		return nil, err
	}
	reviewer := reviewerID
	review := &StoreReview{
		OrganisationID:      orgID,
		ReviewerID:          &reviewer,
		Rating:              in.Rating,
		Title:               in.Title,
		Body:                in.Body,
		BusinessTypeDisplay: businessType,
		RegionDisplay:       region,
		Trigger:             defaultStr(in.Trigger, "manual"),
		IsPublic:            false,
	}
	if err := s.repo.Create(ctx, review); err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"id":     review.ID,
		"rating": review.Rating,
	}, nil
}

// Public returns approved reviews for the landing page.
func (s *Service) Public(ctx context.Context, limit int) ([]map[string]interface{}, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := s.repo.PublicReviews(ctx, limit)
	if err != nil {
		return nil, err
	}
	out := make([]map[string]interface{}, 0, len(rows))
	for _, row := range rows {
		out = append(out, map[string]interface{}{
			"id":                    row.ID,
			"rating":                row.Rating,
			"title":                 row.Title,
			"body":                  row.Body,
			"org_name":              row.OrgName,
			"reviewer_name":         row.ReviewerName,
			"business_type_display": row.BusinessTypeDisplay,
			"region_display":        row.RegionDisplay,
			"created_at":            row.CreatedAt,
		})
	}
	return out, nil
}

func defaultStr(v, fallback string) string {
	if v == "" {
		return fallback
	}
	return v
}
