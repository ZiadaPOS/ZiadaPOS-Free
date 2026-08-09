package notebook

import (
	"context"
	"strings"

	"github.com/google/uuid"

	"github.com/troubleman96/ziadapos/internal/apierr"
)

// Service implements notebook business rules.
type Service struct {
	repo *Repo
}

// NewService wires a notebook Service.
func NewService(repo *Repo) *Service { return &Service{repo: repo} }

// NoteInput is the create/update payload.
type NoteInput struct {
	Title   string   `json:"title"`
	Content string   `json:"content"`
	Tags    []string `json:"tags"`
}

// List returns notes for a store with filters applied.
func (s *Service) List(ctx context.Context, storeID uuid.UUID, tag, search, order string) ([]Note, error) {
	return s.repo.List(ctx, storeID, tag, search, order)
}

// Get returns one note.
func (s *Service) Get(ctx context.Context, storeID, id uuid.UUID) (*Note, error) {
	note, err := s.repo.Get(ctx, storeID, id)
	if err != nil {
		return nil, apierr.NotFound("Note not found")
	}
	return note, nil
}

// Create adds a new note authored by a staff member.
func (s *Service) Create(ctx context.Context, orgID, storeID, userID uuid.UUID, in *NoteInput) (*Note, error) {
	if strings.TrimSpace(in.Title) == "" {
		return nil, apierr.BadRequest("Note title is required")
	}
	tags, err := cleanTags(in.Tags)
	if err != nil {
		return nil, err
	}
	by := userID
	note := &Note{
		StoreID:        storeID,
		OrganisationID: orgID,
		Title:          strings.TrimSpace(in.Title),
		Content:        in.Content,
		Tags:           tags,
		CreatedBy:      &by,
	}
	if err := s.repo.Create(ctx, note); err != nil {
		return nil, err
	}
	return s.Get(ctx, storeID, note.ID)
}

// Update edits title/content/tags.
func (s *Service) Update(ctx context.Context, storeID, id uuid.UUID, in *NoteInput) (*Note, error) {
	note, err := s.repo.Get(ctx, storeID, id)
	if err != nil {
		return nil, apierr.NotFound("Note not found")
	}
	if in.Title != "" {
		note.Title = strings.TrimSpace(in.Title)
	}
	if in.Content != "" {
		note.Content = in.Content
	}
	if in.Tags != nil {
		tags, err := cleanTags(in.Tags)
		if err != nil {
			return nil, err
		}
		note.Tags = tags
	}
	if err := s.repo.Update(ctx, note); err != nil {
		return nil, err
	}
	return s.Get(ctx, storeID, note.ID)
}

// Delete removes a note.
func (s *Service) Delete(ctx context.Context, storeID, id uuid.UUID) error {
	note, err := s.repo.Get(ctx, storeID, id)
	if err != nil {
		return apierr.NotFound("Note not found")
	}
	return s.repo.Delete(ctx, storeID, note.ID)
}

// cleanTags validates and normalises a tag list (max 10, each ≤ 50 chars).
func cleanTags(tags []string) (Tags, error) {
	if tags == nil {
		return Tags{}, nil
	}
	cleaned := make(Tags, 0, len(tags))
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if len(tag) > 50 {
			tag = tag[:50]
		}
		if tag != "" {
			cleaned = append(cleaned, tag)
		}
	}
	if len(cleaned) > 10 {
		return nil, apierr.BadRequest("Too many tags").WithField("tags", "a note can have at most 10 tags")
	}
	return cleaned, nil
}
