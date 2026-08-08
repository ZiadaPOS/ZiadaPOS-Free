package expenses

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/troubleman96/ziadapos/internal/apierr"
	"github.com/troubleman96/ziadapos/internal/cache"
	"github.com/troubleman96/ziadapos/internal/events"
)

// Service implements expense business rules.
type Service struct {
	repo  *Repo
	cache *cache.Cache
	bus   *events.Bus
}

// NewService wires an expenses Service.
func NewService(repo *Repo, cache *cache.Cache, bus *events.Bus) *Service {
	return &Service{repo: repo, cache: cache, bus: bus}
}

// Input is the create/update payload.
type Input struct {
	Title            string `json:"title"`
	Category         string `json:"category"`
	Amount           int64  `json:"amount"`
	PaymentMethod    string `json:"payment_method"`
	PaymentReference string `json:"payment_reference"`
	Notes            string `json:"notes"`
	ReceiptURL       string `json:"receipt_url"`
}

func (s *Service) Create(ctx context.Context, orgID, storeID, userID uuid.UUID, in *Input) (*Expense, error) {
	if strings.TrimSpace(in.Title) == "" {
		return nil, apierr.BadRequest("Expense title is required")
	}
	if in.Amount <= 0 {
		return nil, apierr.BadRequest("Expense amount must be positive").WithField("amount", "greater than zero")
	}
	if in.Category == "" {
		in.Category = CatOther
	}
	if !contains(ValidCategories, in.Category) {
		return nil, apierr.BadRequest("Invalid category").WithField("category", "choose one of the known categories")
	}
	if in.PaymentMethod == "" {
		in.PaymentMethod = MethodCash
	}
	if !contains(ValidPaymentMethods, in.PaymentMethod) {
		return nil, apierr.BadRequest("Invalid payment method").WithField("payment_method", "must be Cash, M-Pesa, Tigo Pesa, Bank or Airtel Money")
	}
	e := &Expense{
		OrganisationID:   orgID,
		StoreID:          storeID,
		Title:            strings.TrimSpace(in.Title),
		Category:         in.Category,
		Amount:           in.Amount,
		PaymentMethod:    in.PaymentMethod,
		PaymentReference: strings.TrimSpace(in.PaymentReference),
		Notes:            in.Notes,
		ReceiptURL:       strings.TrimSpace(in.ReceiptURL),
		RecordedBy:       userID,
	}
	if err := s.repo.Create(e); err != nil {
		return nil, err
	}
	s.invalidateCache(storeID)
	s.bus.Publish("expense.created", map[string]interface{}{
		"expense_id": e.ID, "store_id": storeID, "org_id": orgID,
		"amount": e.Amount, "category": e.Category, "payment_method": e.PaymentMethod,
	})
	return e, nil
}

func (s *Service) Get(ctx context.Context, storeID, id uuid.UUID) (*Expense, error) {
	e, err := s.repo.Get(storeID, id)
	if err != nil {
		return nil, apierr.NotFound("Expense not found")
	}
	return e, nil
}

func (s *Service) Update(ctx context.Context, storeID, id uuid.UUID, in *Input) (*Expense, error) {
	e, err := s.repo.Get(storeID, id)
	if err != nil {
		return nil, apierr.NotFound("Expense not found")
	}
	if strings.TrimSpace(in.Title) != "" {
		e.Title = strings.TrimSpace(in.Title)
	}
	if in.Category != "" {
		if !contains(ValidCategories, in.Category) {
			return nil, apierr.BadRequest("Invalid category").WithField("category", "choose one of the known categories")
		}
		e.Category = in.Category
	}
	if in.Amount > 0 {
		e.Amount = in.Amount
	}
	if in.PaymentMethod != "" {
		if !contains(ValidPaymentMethods, in.PaymentMethod) {
			return nil, apierr.BadRequest("Invalid payment method").WithField("payment_method", "must be Cash, M-Pesa, Tigo Pesa, Bank or Airtel Money")
		}
		e.PaymentMethod = in.PaymentMethod
	}
	e.PaymentReference = in.PaymentReference
	e.Notes = in.Notes
	e.ReceiptURL = in.ReceiptURL
	if err := s.repo.Update(e); err != nil {
		return nil, err
	}
	s.invalidateCache(storeID)
	return e, nil
}

func (s *Service) Delete(ctx context.Context, storeID, id uuid.UUID) error {
	if err := s.repo.Delete(storeID, id); err != nil {
		return apierr.NotFound("Expense not found")
	}
	s.invalidateCache(storeID)
	return nil
}

func (s *Service) List(ctx context.Context, storeID uuid.UUID, f Filter) (map[string]interface{}, error) {
	if f.Page <= 0 {
		f.Page = 1
	}
	if f.PageSize <= 0 {
		f.PageSize = 20
	}
	if f.PageSize > 100 {
		f.PageSize = 100
	}
	f.StoreID = storeID
	items, total, err := s.repo.List(f)
	if err != nil {
		return nil, err
	}
	out := make([]map[string]interface{}, 0, len(items))
	for i := range items {
		out = append(out, expenseResponse(&items[i]))
	}
	return map[string]interface{}{
		"items": out, "total": total, "page": f.Page, "page_size": f.PageSize,
	}, nil
}

func (s *Service) Summary(ctx context.Context, storeID uuid.UUID) (map[string]interface{}, error) {
	if s.cache != nil {
		key := "exp:summary:" + storeID.String()
		var cached map[string]interface{}
		if err := s.cache.GetJSON(ctx, key, &cached); err == nil {
			return cached, nil
		}
	}
	agg, err := s.repo.SummaryAgg(storeID)
	if err != nil {
		return nil, err
	}
	byCat, err := s.repo.ByCategory(storeID)
	if err != nil {
		return nil, err
	}
	byMethod, err := s.repo.ByMethod(storeID)
	if err != nil {
		return nil, err
	}
	recent, err := s.repo.Recent(storeID, 5)
	if err != nil {
		return nil, err
	}

	denom := agg.TotalAmount
	if denom == 0 {
		denom = 1
	}
	cats := make([]map[string]interface{}, 0, len(byCat))
	for _, row := range byCat {
		cats = append(cats, map[string]interface{}{
			"category": row.Category,
			"amount":   row.Amount,
			"count":    row.Count,
			"pct":      round1(float64(row.Amount) / float64(denom) * 100),
		})
	}
	methods := make([]map[string]interface{}, 0, len(byMethod))
	for _, row := range byMethod {
		methods = append(methods, map[string]interface{}{
			"method": row.Method,
			"amount": row.Amount,
			"count":  row.Count,
			"pct":    round1(float64(row.Amount) / float64(denom) * 100),
		})
	}
	recentOut := make([]map[string]interface{}, 0, len(recent))
	for i := range recent {
		recentOut = append(recentOut, expenseResponse(&recent[i]))
	}
	out := map[string]interface{}{
		"total_amount": agg.TotalAmount,
		"total_count":  agg.TotalCount,
		"by_category":  cats,
		"by_method":    methods,
		"recent":       recentOut,
	}
	if s.cache != nil {
		_ = s.cache.SetJSONTTL(ctx, "exp:summary:"+storeID.String(), out, 30*time.Second)
	}
	return out, nil
}

func (s *Service) invalidateCache(storeID uuid.UUID) {
	if s.cache == nil {
		return
	}
	ctx := context.Background()
	_ = s.cache.DeletePattern(ctx, "exp:summary:"+storeID.String())
}

func expenseResponse(e *Expense) map[string]interface{} {
	return map[string]interface{}{
		"id":                e.ID,
		"title":             e.Title,
		"category":          e.Category,
		"amount":            e.Amount,
		"payment_method":    e.PaymentMethod,
		"payment_reference": e.PaymentReference,
		"notes":             e.Notes,
		"receipt_url":       e.ReceiptURL,
		"recorded_by":       e.RecordedBy,
		"created_at":        e.CreatedAt,
	}
}

func contains(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

func round1(v float64) float64 {
	return float64(int(v*10+0.5)) / 10
}
