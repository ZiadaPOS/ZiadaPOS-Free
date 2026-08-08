package accounts

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Repo provides data access for accounts entities.
type Repo struct {
	db *gorm.DB
}

// NewRepo builds a Repo.
func NewRepo(db *gorm.DB) *Repo { return &Repo{db: db} }

// DB exposes the underlying *gorm.DB for cross-app transactions.
func (r *Repo) DB() *gorm.DB { return r.db }

// ── Users ───────────────────────────────────────────────────────────────────

func (r *Repo) CreateUser(u *User) error { return r.db.Create(u).Error }

func (r *Repo) GetUserByPhone(phone string) (*User, error) {
	var u User
	if err := r.db.Where("phone = ?", phone).First(&u).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *Repo) GetUserByEmail(email string) (*User, error) {
	var u User
	if err := r.db.Where("email = ?", email).First(&u).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *Repo) GetUserByID(id uuid.UUID) (*User, error) {
	var u User
	if err := r.db.Where("id = ?", id).First(&u).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

// GetUserByIdentifier matches phone or email (case-insensitive).
func (r *Repo) GetUserByIdentifier(identifier string) (*User, error) {
	var u User
	err := r.db.Where("phone = ?", identifier).
		Or("lower(email) = lower(?)", identifier).
		First(&u).Error
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *Repo) UpdateUser(u *User) error { return r.db.Save(u).Error }

func (r *Repo) ListUsersByOrg(orgID uuid.UUID) ([]User, error) {
	var users []User
	err := r.db.Where("organisation_id = ?", orgID).Order("created_at desc").Find(&users).Error
	return users, err
}

func (r *Repo) CountUsersByOrg(orgID uuid.UUID) (int64, error) {
	var n int64
	err := r.db.Model(&User{}).Where("organisation_id = ?", orgID).Count(&n).Error
	return n, err
}

// ── Organisations ───────────────────────────────────────────────────────────

func (r *Repo) CreateOrganisation(o *Organisation) error { return r.db.Create(o).Error }

func (r *Repo) GetOrgByID(id uuid.UUID) (*Organisation, error) {
	var o Organisation
	if err := r.db.Where("id = ?", id).First(&o).Error; err != nil {
		return nil, err
	}
	return &o, nil
}

func (r *Repo) GetOrgBySlug(slug string) (*Organisation, error) {
	var o Organisation
	if err := r.db.Where("slug = ?", slug).First(&o).Error; err != nil {
		return nil, err
	}
	return &o, nil
}

func (r *Repo) UpdateOrg(o *Organisation) error { return r.db.Save(o).Error }

// ── Stores ──────────────────────────────────────────────────────────────────

func (r *Repo) CreateStore(s *Store) error { return r.db.Create(s).Error }

func (r *Repo) GetStoreByID(id uuid.UUID) (*Store, error) {
	var s Store
	if err := r.db.Where("id = ?", id).First(&s).Error; err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *Repo) GetStoreByIDAndOrg(id, orgID uuid.UUID) (*Store, error) {
	var s Store
	if err := r.db.Where("id = ? AND organisation_id = ?", id, orgID).First(&s).Error; err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *Repo) ListStoresByOrg(orgID uuid.UUID) ([]Store, error) {
	var stores []Store
	err := r.db.Where("organisation_id = ?", orgID).Order("created_at asc").Find(&stores).Error
	return stores, err
}

func (r *Repo) CountStoresByOrg(orgID uuid.UUID) (int64, error) {
	var n int64
	err := r.db.Model(&Store{}).Where("organisation_id = ?", orgID).Count(&n).Error
	return n, err
}

func (r *Repo) UpdateStore(s *Store) error { return r.db.Save(s).Error }

// ── OTP ─────────────────────────────────────────────────────────────────────

func (r *Repo) CreateOTP(o *PhoneOTP) error { return r.db.Create(o).Error }

func (r *Repo) GetActiveOTP(phone, purpose string) (*PhoneOTP, error) {
	var o PhoneOTP
	err := r.db.Where("phone = ? AND purpose = ? AND is_used = false AND expires_at > ?",
		phone, purpose, time.Now()).
		Order("created_at desc").First(&o).Error
	return &o, err
}

func (r *Repo) MarkOTPUsed(id uuid.UUID) error {
	return r.db.Model(&PhoneOTP{}).Where("id = ?", id).Update("is_used", true).Error
}

func (r *Repo) IncrementOTPAttempts(id uuid.UUID) error {
	return r.db.Model(&PhoneOTP{}).Where("id = ?", id).
		UpdateColumn("attempts", gorm.Expr("attempts + 1")).Error
}

func (r *Repo) InvalidateOTPs(phone, purpose string) error {
	return r.db.Model(&PhoneOTP{}).
		Where("phone = ? AND purpose = ? AND is_used = false", phone, purpose).
		Update("is_used", true).Error
}

// ── Email verification ──────────────────────────────────────────────────────

func (r *Repo) CreateEmailToken(t *EmailVerificationToken) error { return r.db.Create(t).Error }

func (r *Repo) GetEmailToken(token string) (*EmailVerificationToken, error) {
	var t EmailVerificationToken
	err := r.db.Where("token = ? AND expires_at > ?", token, time.Now()).First(&t).Error
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// ── AI credits ──────────────────────────────────────────────────────────────

func (r *Repo) GetAICredit(orgID uuid.UUID, year, month int) (*AICredit, error) {
	var c AICredit
	err := r.db.Where("organisation_id = ? AND year = ? AND month = ?", orgID, year, month).
		First(&c).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	return &c, nil
}

func (r *Repo) UpsertAICredit(c *AICredit) error {
	return r.db.Save(c).Error
}

// WithTx runs fn inside a database transaction.
func (r *Repo) WithTx(fn func(tx *gorm.DB) error) error {
	return r.db.Transaction(fn)
}
