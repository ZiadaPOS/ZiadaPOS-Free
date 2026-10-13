package notifications

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Repo persists notifications.
type Repo struct {
	db *gorm.DB
}

// NewRepo builds a notifications Repo.
func NewRepo(db *gorm.DB) *Repo { return &Repo{db: db} }

// Create inserts a notification.
func (r *Repo) Create(ctx context.Context, n *Notification) error {
	return r.db.WithContext(ctx).Create(n).Error
}

// List returns notifications for an organisation and store, newest first,
// with the total count and number of unread items in the same scope.
func (r *Repo) List(ctx context.Context, orgID, storeID uuid.UUID, f Filter) ([]Notification, int64, int64, error) {
	q := r.db.WithContext(ctx).Model(&Notification{}).Where("organisation_id = ?", orgID)

	storeScoped := storeID != uuid.Nil
	if storeScoped {
		q = q.Where("store_id = ? OR store_id = '00000000-0000-0000-0000-000000000000'", storeID)
	}
	if f.Category != "" {
		q = q.Where("category = ?", f.Category)
	}
	if f.Unread {
		q = q.Where("read = ?", false)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, 0, err
	}

	var unread int64
	uc := r.db.WithContext(ctx).Model(&Notification{}).Where("organisation_id = ? AND read = ?", orgID, false)
	if storeScoped {
		uc = uc.Where("store_id = ? OR store_id = '00000000-0000-0000-0000-000000000000'", storeID)
	}
	if err := uc.Count(&unread).Error; err != nil {
		return nil, 0, 0, err
	}

	page, pageSize := f.Page, f.PageSize
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 30
	}
	if pageSize > 100 {
		pageSize = 100
	}

	var items []Notification
	if err := q.Order("created_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&items).Error; err != nil {
		return nil, 0, 0, err
	}
	return items, total, unread, nil
}

// UnreadCount counts unread notifications for the scope.
func (r *Repo) UnreadCount(ctx context.Context, orgID, storeID uuid.UUID) (int64, error) {
	q := r.db.WithContext(ctx).Model(&Notification{}).Where("organisation_id = ? AND read = ?", orgID, false)
	if storeID != uuid.Nil {
		q = q.Where("store_id = ? OR store_id = '00000000-0000-0000-0000-000000000000'", storeID)
	}
	var count int64
	if err := q.Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// Find returns a single notification by id, scoped to the organisation (and
// optionally the store). Returns gorm.ErrRecordNotFound when not found.
func (r *Repo) Find(ctx context.Context, orgID, storeID, id uuid.UUID) (Notification, error) {
	var n Notification
	q := r.db.WithContext(ctx).Where("id = ? AND organisation_id = ?", id, orgID)
	if storeID != uuid.Nil {
		q = q.Where("store_id = ? OR store_id = '00000000-0000-0000-0000-000000000000'", storeID)
	}
	if err := q.First(&n).Error; err != nil {
		return Notification{}, err
	}
	return n, nil
}

// MarkRead marks a single notification as read. Returns gorm.ErrRecordNotFound
// if it does not belong to the caller's scope.
func (r *Repo) MarkRead(ctx context.Context, orgID, storeID, id uuid.UUID) error {
	now := time.Now()
	res := r.db.WithContext(ctx).Model(&Notification{}).
		Where("id = ? AND organisation_id = ?", id, orgID).
		Where("read = ?", false).
		Updates(map[string]interface{}{"read": true, "read_at": &now})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		var count int64
		r.db.WithContext(ctx).Model(&Notification{}).Where("id = ? AND organisation_id = ?", id, orgID).Count(&count)
		if count == 0 {
			return gorm.ErrRecordNotFound
		}
	}
	return nil
}

// MarkAllRead marks every notification in the scope as read.
func (r *Repo) MarkAllRead(ctx context.Context, orgID, storeID uuid.UUID) error {
	now := time.Now()
	updates := map[string]interface{}{"read": true, "read_at": &now}
	q := r.db.WithContext(ctx).Model(&Notification{}).
		Where("organisation_id = ?", orgID).
		Where("read = ?", false)
	if storeID != uuid.Nil {
		q = q.Where("store_id = ? OR store_id = '00000000-0000-0000-0000-000000000000'", storeID)
	}
	return q.Updates(updates).Error
}

// Delete removes a notification from the caller's scope.
func (r *Repo) Delete(ctx context.Context, orgID, storeID, id uuid.UUID) error {
	q := r.db.WithContext(ctx).Where("id = ? AND organisation_id = ?", id, orgID)
	if storeID != uuid.Nil {
		q = q.Where("store_id = ? OR store_id = '00000000-0000-0000-0000-000000000000'", storeID)
	}
	res := q.Delete(&Notification{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return errors.New("not_found")
	}
	return nil
}
