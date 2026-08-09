package notebook

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Repo provides note data access.
type Repo struct {
	db *gorm.DB
}

// NewRepo builds a Repo.
func NewRepo(db *gorm.DB) *Repo { return &Repo{db: db} }

// List returns notes for a store with optional tag/search filters.
func (r *Repo) List(ctx context.Context, storeID uuid.UUID, tag, search, order string) ([]Note, error) {
	q := r.db.Where("store_id = ?", storeID)
	if tag != "" {
		q = q.Where("tags::text ILIKE ?", "%"+tag+"%")
	}
	if search != "" {
		like := "%" + search + "%"
		q = q.Where("(title ILIKE ? OR content ILIKE ?)", like, like)
	}
	switch order {
	case "created_at":
		q = q.Order("created_at asc")
	case "title":
		q = q.Order("title asc")
	case "-title":
		q = q.Order("title desc")
	default:
		q = q.Order("created_at desc")
	}
	var notes []Note
	if err := q.Find(&notes).Error; err != nil {
		return nil, err
	}
	return r.decorate(ctx, notes), nil
}

// Get returns one note scoped to a store.
func (r *Repo) Get(ctx context.Context, storeID, id uuid.UUID) (*Note, error) {
	var n Note
	if err := r.db.Where("id = ? AND store_id = ?", id, storeID).First(&n).Error; err != nil {
		return nil, err
	}
	notes := r.decorate(ctx, []Note{n})
	return &notes[0], nil
}

// Create inserts a note.
func (r *Repo) Create(ctx context.Context, n *Note) error {
	if n.Tags == nil {
		n.Tags = Tags{}
	}
	return r.db.Create(n).Error
}

// Update persists note fields.
func (r *Repo) Update(ctx context.Context, n *Note) error {
	if n.Tags == nil {
		n.Tags = Tags{}
	}
	return r.db.Model(&Note{}).
		Where("id = ? AND store_id = ?", n.ID, n.StoreID).
		Updates(map[string]interface{}{
			"title": n.Title, "content": n.Content, "tags": n.Tags,
		}).Error
}

// Delete removes a note.
func (r *Repo) Delete(ctx context.Context, storeID, id uuid.UUID) error {
	return r.db.Where("id = ? AND store_id = ?", id, storeID).Delete(&Note{}).Error
}

// decorate fills the author name and relative date label for a set of notes.
func (r *Repo) decorate(ctx context.Context, notes []Note) []Note {
	if len(notes) == 0 {
		return notes
	}
	authorIDs := make([]uuid.UUID, 0, len(notes))
	for i := range notes {
		if notes[i].CreatedBy != nil && !isNil(*notes[i].CreatedBy) {
			authorIDs = append(authorIDs, *notes[i].CreatedBy)
		}
	}
	names := map[uuid.UUID]string{}
	if len(authorIDs) > 0 {
		var users []struct {
			ID        uuid.UUID `gorm:"column:id"`
			FirstName string    `gorm:"column:first_name"`
			LastName  string    `gorm:"column:last_name"`
			Phone     string    `gorm:"column:phone"`
		}
		if err := r.db.Table("users").
			Select("id, first_name, last_name, phone").
			Where("id IN ?", authorIDs).
			Find(&users).Error; err == nil {
			for _, u := range users {
				name := strings.TrimSpace(u.FirstName + " " + u.LastName)
				if name == "" {
					name = u.Phone
				}
				names[u.ID] = name
			}
		}
	}
	for i := range notes {
		if notes[i].CreatedBy != nil {
			notes[i].CreatedByName = names[*notes[i].CreatedBy]
		}
		notes[i].buildDateLabel(time.Now())
	}
	return notes
}

func isNil(u uuid.UUID) bool { return u == uuid.Nil }
