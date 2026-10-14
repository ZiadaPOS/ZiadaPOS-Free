package customers

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/troubleman96/ziadapos/apps/transactions"
	"github.com/troubleman96/ziadapos/internal/apierr"
	"github.com/troubleman96/ziadapos/internal/cache"
	"github.com/troubleman96/ziadapos/internal/integrations/sms"
	"github.com/troubleman96/ziadapos/internal/security"
)

// Service implements customer business rules.
type Service struct {
	repo    *Repo
	cache   *cache.Cache
	db      *gorm.DB
	sms     *sms.Client
	secrets *security.SecretBox
}

// NewService wires a customers Service.
func NewService(repo *Repo, cache *cache.Cache, db *gorm.DB, sms *sms.Client, secrets *security.SecretBox) *Service {
	return &Service{repo: repo, cache: cache, db: db, sms: sms, secrets: secrets}
}

// Input is the create/update payload.
type Input struct {
	Name        string `json:"name" binding:"required"`
	Phone       string `json:"phone"`
	Email       string `json:"email"`
	Segment     string `json:"segment"`
	CreditLimit *int64 `json:"credit_limit"`
	Notes       string `json:"notes"`
	IsActive    *bool  `json:"is_active"`
}

func (s *Service) Create(ctx context.Context, orgID, storeID uuid.UUID, in *Input) (*Customer, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return nil, apierr.BadRequest("Customer name is required")
	}
	phone, err := normalizePhone(in.Phone)
	if err != nil {
		// Silently ignore invalid phone numbers (e.g. from device contacts
		// that use non-Tanzanian formats) rather than rejecting the customer.
		phone = nil
	}
	if phone != nil {
		if _, err := s.repo.FindByPhone(storeID, *phone); err == nil {
			return nil, apierr.Conflict("A customer with this phone already exists")
		}
	}

	c := &Customer{
		OrganisationID: orgID,
		StoreID:        storeID,
		Name:           name,
		Phone:          phone,
		Email:          strings.TrimSpace(in.Email),
		Segment:        SegmentNew,
		AvatarHue:      randomHue(),
		Notes:          in.Notes,
		IsActive:       true,
		CreditLimit:    in.CreditLimit,
	}
	if err := s.repo.Create(c); err != nil {
		if isUniqueViolation(err) {
			return nil, apierr.Conflict("A customer with this phone already exists")
		}
		return nil, err
	}
	return c, nil
}

func (s *Service) Get(ctx context.Context, storeID, id uuid.UUID) (*Customer, error) {
	c, err := s.repo.Get(storeID, id)
	if err != nil {
		return nil, apierr.NotFound("Customer not found")
	}
	return c, nil
}

func (s *Service) Update(ctx context.Context, storeID, id uuid.UUID, in *Input) (*Customer, error) {
	c, err := s.repo.Get(storeID, id)
	if err != nil {
		return nil, apierr.NotFound("Customer not found")
	}
	if strings.TrimSpace(in.Name) != "" {
		c.Name = strings.TrimSpace(in.Name)
	}
	if phone, err := normalizePhone(in.Phone); err == nil {
		c.Phone = phone
	}
	c.Email = strings.TrimSpace(in.Email)
	if in.Segment != "" {
		c.Segment = in.Segment
	}
	if in.CreditLimit != nil {
		c.CreditLimit = in.CreditLimit
	}
	c.Notes = in.Notes
	if in.IsActive != nil {
		c.IsActive = *in.IsActive
	}
	if err := s.repo.Update(c); err != nil {
		return nil, err
	}
	return c, nil
}

// SendResult is what SendSMS returns after the SMS provider accepts the send.
type SendResult struct {
	MessageID        string `json:"message_id"`
	Status           string `json:"status"`
	CreditsUsed      int    `json:"credits_used"`
	ProviderAccepted bool   `json:"provider_accepted"`
}

// SendSMS sends a free-form SMS to a customer's phone via SendAfrica.
func (s *Service) SendSMS(ctx context.Context, orgID, storeID, id uuid.UUID, message string) (*SendResult, error) {
	if s.sms == nil {
		return nil, apierr.Internal("SMS service is not configured")
	}
	msg := strings.TrimSpace(message)
	if msg == "" {
		return nil, apierr.BadRequest("Message is required")
	}
	c, err := s.repo.Get(storeID, id)
	if err != nil {
		return nil, apierr.NotFound("Customer not found")
	}
	if c.Phone == nil || strings.TrimSpace(*c.Phone) == "" {
		return nil, apierr.BadRequest("Customer has no phone number").WithField("phone", "required for SMS")
	}
	var integration struct {
		APIKey   string `gorm:"column:send_africa_api_key"`
		SenderID string `gorm:"column:sms_sender_id"`
	}
	if err := s.db.WithContext(ctx).Table("organisations").Select("send_africa_api_key", "sms_sender_id").Where("id = ?", orgID).Take(&integration).Error; err != nil {
		return nil, apierr.NotFound("Organisation not found")
	}
	if integration.APIKey == "" {
		return nil, apierr.BadRequest("Connect a SendAfrica API key for this organisation before sending SMS")
	}
	if s.secrets == nil {
		return nil, apierr.Internal("SendAfrica credentials cannot be opened; check INTEGRATION_ENCRYPTION_KEY")
	}
	apiKey, err := s.secrets.Decrypt(integration.APIKey)
	if err != nil {
		return nil, apierr.Internal("SendAfrica credentials cannot be opened; check INTEGRATION_ENCRYPTION_KEY")
	}
	client := s.sms.ForAccount(apiKey, integration.SenderID)
	resp, err := client.Send(ctx, *c.Phone, msg, "")
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, apierr.Internal("SMS provider returned an empty response")
	}
	return &SendResult{
		MessageID:        resp.Data.MessageID,
		Status:           resp.Data.Status,
		CreditsUsed:      resp.Data.CreditsUsed,
		ProviderAccepted: true,
	}, nil
}

func (s *Service) List(ctx context.Context, storeID uuid.UUID, f Filter) (map[string]interface{}, error) {
	if f.Page <= 0 {
		f.Page = 1
	}
	if f.PageSize <= 0 {
		f.PageSize = 20
	}
	if f.PageSize > 200 {
		f.PageSize = 200
	}
	f.StoreID = storeID
	items, total, err := s.repo.List(f)
	if err != nil {
		return nil, err
	}
	out := make([]map[string]interface{}, 0, len(items))
	for i := range items {
		out = append(out, customerResponse(&items[i]))
	}
	return map[string]interface{}{
		"items": out, "total": total, "page": f.Page, "page_size": f.PageSize,
	}, nil
}

// CreditCustomers lists customers with open balances (optionally overdue).
func (s *Service) CreditCustomers(ctx context.Context, storeID uuid.UUID, overdue bool) ([]map[string]interface{}, error) {
	items, err := s.repo.CreditCustomers(storeID, overdue)
	if err != nil {
		return nil, err
	}
	out := make([]map[string]interface{}, 0, len(items))
	for i := range items {
		out = append(out, customerResponse(&items[i]))
	}
	return out, nil
}

// RecordSale bumps spend stats after a completed transaction.
func (s *Service) RecordSale(ctx context.Context, customerID *uuid.UUID, total int64) error {
	if customerID == nil {
		return nil
	}
	c, err := s.repo.GetByID(*customerID)
	if err != nil {
		return nil // customer deactivated
	}
	now := time.Now()
	c.TotalSpent += total
	c.LastVisit = &now
	c.AvgTicket = c.TotalSpent / s.txnCount(c.ID)
	c.RecomputeSegment()
	return s.repo.Update(c)
}

// RecordRefund subtracts a refunded total from spend stats.
func (s *Service) RecordRefund(ctx context.Context, customerID *uuid.UUID, total int64) error {
	if customerID == nil {
		return nil
	}
	c, err := s.repo.GetByID(*customerID)
	if err != nil {
		return nil
	}
	c.TotalSpent -= total
	if c.TotalSpent < 0 {
		c.TotalSpent = 0
	}
	c.AvgTicket = 0
	if c.TotalSpent > 0 {
		if n := s.txnCount(c.ID); n > 0 {
			c.AvgTicket = c.TotalSpent / n
		}
	}
	c.RecomputeSegment()
	return s.repo.Update(c)
}

// RecomputeCreditBalance refreshes a customer's open_credit from their tabs.
func (s *Service) RecomputeCreditBalance(ctx context.Context, customerID uuid.UUID, balance int64) error {
	c, err := s.repo.GetByID(customerID)
	if err != nil {
		return err
	}
	c.OpenCredit = balance
	return s.repo.Update(c)
}

func (s *Service) txnCount(customerID uuid.UUID) int64 {
	var n int64
	s.db.Model(&transactions.Transaction{}).
		Where("customer_id = ? AND status IN ?", customerID, []string{transactions.StatusPaid, transactions.StatusCredit}).
		Count(&n)
	if n == 0 {
		return 1
	}
	return n
}

func customerResponse(c *Customer) map[string]interface{} {
	var phone string
	if c.Phone != nil {
		phone = *c.Phone
	}
	var lastVisit *time.Time
	if c.LastVisit != nil {
		lv := *c.LastVisit
		lastVisit = &lv
	}
	return map[string]interface{}{
		"id":           c.ID,
		"name":         c.Name,
		"initials":     c.Initials(),
		"phone":        phone,
		"email":        c.Email,
		"segment":      c.Segment,
		"total_spent":  c.TotalSpent,
		"last_visit":   lastVisit,
		"avg_ticket":   c.AvgTicket,
		"open_credit":  c.OpenCredit,
		"has_credit":   c.HasOpenCredit(),
		"credit_limit": c.CreditLimit,
		"avatar_hue":   c.AvatarHue,
		"notes":        c.Notes,
		"is_active":    c.IsActive,
		"created_at":   c.CreatedAt,
	}
}

func normalizePhone(p string) (*string, error) {
	p = strings.TrimSpace(p)
	if p == "" {
		return nil, nil
	}
	n, err := sms.NormalizePhone(p)
	if err != nil {
		return nil, err
	}
	return &n, nil
}

func isUniqueViolation(err error) bool {
	return err != nil && (strings.Contains(strings.ToLower(err.Error()), "duplicate") ||
		strings.Contains(strings.ToLower(err.Error()), "unique"))
}
