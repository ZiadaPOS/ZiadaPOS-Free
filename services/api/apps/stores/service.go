package stores

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/troubleman96/ziadapos/apps/subscriptions"
	"github.com/troubleman96/ziadapos/internal/apierr"
)

// Service implements store management business rules.
type Service struct {
	repo *Repo
}

// NewService wires a stores Service.
func NewService(repo *Repo) *Service { return &Service{repo: repo} }

// StoreInput is the create/update payload.
type StoreInput struct {
	Name       string  `json:"name"`
	Code       string  `json:"code"`
	Phone      string  `json:"phone"`
	Email      string  `json:"email"`
	Area       string  `json:"area"`
	Address    string  `json:"address"`
	Latitude   float64 `json:"latitude"`
	Longitude  float64 `json:"longitude"`
	OpenHours  string  `json:"open_hours"`
	Color      string  `json:"color"`
	Status     string  `json:"status"`
	VATEnabled *bool   `json:"vat_enabled"`
}

// List returns stores with live KPIs and week sparkline data.
func (s *Service) List(ctx context.Context, orgID uuid.UUID, status string) ([]map[string]interface{}, error) {
	stores, err := s.repo.ListStores(ctx, orgID, status)
	if err != nil {
		return nil, err
	}
	out := make([]map[string]interface{}, 0, len(stores))
	for i := range stores {
		item, err := s.StoreCard(ctx, &stores[i])
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, nil
}

// Get returns the full store detail with week breakdown and staff roster.
func (s *Service) Get(ctx context.Context, orgID, id uuid.UUID) (map[string]interface{}, error) {
	store, err := s.repo.GetStore(ctx, orgID, id)
	if err != nil {
		return nil, apierr.NotFound("Store not found")
	}
	card, err := s.StoreCard(ctx, store)
	if err != nil {
		return nil, err
	}
	week, err := s.WeekBreakdown(ctx, store.ID)
	if err != nil {
		return nil, err
	}
	roster, err := s.StaffRoster(ctx, store.ID)
	if err != nil {
		return nil, err
	}
	card["week_breakdown"] = week
	card["staff_roster"] = roster
	return card, nil
}

// StoreCard builds the list-card representation (KPIs + sparkline).
func (s *Service) StoreCard(ctx context.Context, store *Store) (map[string]interface{}, error) {
	todayRev, todayTxns, err := s.repo.TodayKPIs(ctx, store.ID)
	if err != nil {
		return nil, err
	}
	staffCount, err := s.repo.CountStaff(ctx, store.ID)
	if err != nil {
		return nil, err
	}
	manager, err := s.repo.Manager(ctx, store.ID)
	if err != nil {
		return nil, err
	}
	weekData, err := s.WeekSparkline(ctx, store.ID)
	if err != nil {
		return nil, err
	}

	managerName := interface{}(nil)
	if manager != nil {
		managerName = fullName(manager)
	}
	return map[string]interface{}{
		"id":               store.ID,
		"organisation":     store.OrganisationID,
		"name":             store.Name,
		"code":             store.Code,
		"area":             store.Area,
		"address":          store.Address,
		"phone":            store.Phone,
		"email":            store.Email,
		"color":            store.Color,
		"open_hours":       store.OpenHours,
		"status":           store.Status,
		"is_active":        store.IsActive,
		"vat_enabled":      store.VATEnabled,
		"is_main_store":    store.IsMainStore,
		"manager_name":     managerName,
		"staff_count":      staffCount,
		"staff_on_duty":    staffCount,
		"today_revenue":    todayRev,
		"today_txns":       todayTxns,
		"week_data":        weekData,
		"created_at":       store.CreatedAt,
		"updated_at":       store.UpdatedAt,
	}, nil
}

// WeekSparkline returns a 7-element Mon–Sun revenue array for the current week.
func (s *Service) WeekSparkline(ctx context.Context, storeID uuid.UUID) ([]int64, error) {
	weekStart := currentWeekStart()
	rows, err := s.repo.WeekDays(ctx, storeID, weekStart.UTC(), weekStart.AddDate(0, 0, 7).UTC())
	if err != nil {
		return nil, err
	}
	byDay := map[string]int64{}
	for _, row := range rows {
		byDay[row.Day.In(eat).Format("2006-01-02")] = row.Total
	}
	out := make([]int64, 7)
	for i := 0; i < 7; i++ {
		out[i] = byDay[weekStart.AddDate(0, 0, i).Format("2006-01-02")]
	}
	return out, nil
}

// WeekBreakdown returns per-day detail for the current ISO week.
func (s *Service) WeekBreakdown(ctx context.Context, storeID uuid.UUID) ([]map[string]interface{}, error) {
	weekStart := currentWeekStart()
	rows, err := s.repo.WeekDays(ctx, storeID, weekStart.UTC(), weekStart.AddDate(0, 0, 7).UTC())
	if err != nil {
		return nil, err
	}
	byDay := map[string]dayRow{}
	for _, row := range rows {
		byDay[row.Day.In(eat).Format("2006-01-02")] = row
	}
	labels := []string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"}
	today := time.Now().In(eat)
	todayDate := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, eat)

	var activeSum int64
	activeCount := 0
	rows2 := make([]map[string]interface{}, 7)
	for i, label := range labels {
		d := weekStart.AddDate(0, 0, i)
		row, ok := byDay[d.Format("2006-01-02")]
		revenue := int64(0)
		count := int64(0)
		if ok {
			revenue = row.Total
			count = row.Count
		}
		isFuture := d.After(todayDate)
		if !isFuture && revenue > 0 {
			activeSum += revenue
			activeCount++
		}
		rows2[i] = map[string]interface{}{
			"date":         d.Format("2006-01-02"),
			"day":          label,
			"revenue":      revenue,
			"transactions": count,
			"is_future":    isFuture,
		}
	}
	var avg int64
	if activeCount > 0 {
		avg = activeSum / int64(activeCount)
	}
	for i := range rows2 {
		rev := rows2[i]["revenue"].(int64)
		isFuture := rows2[i]["is_future"].(bool)
		vsAvg := interface{}(nil)
		if avg > 0 && !isFuture && rev > 0 {
			vsAvg = round1(float64(rev-avg) / float64(avg) * 100)
		}
		rows2[i]["vs_avg_pct"] = vsAvg
	}
	return rows2, nil
}

// StaffRoster lists staff with an on-duty flag derived from store status.
func (s *Service) StaffRoster(ctx context.Context, storeID uuid.UUID) ([]map[string]interface{}, error) {
	staff, err := s.repo.ListStaff(ctx, storeID)
	if err != nil {
		return nil, err
	}
	status, err := s.repo.StoreStatus(ctx, storeID)
	if err != nil {
		return nil, err
	}
	if status == StatusOpen {
		status = "on-duty"
	} else {
		status = "off-duty"
	}
	out := make([]map[string]interface{}, 0, len(staff))
	for i := range staff {
		out = append(out, map[string]interface{}{
			"id":         staff[i].ID,
			"name":       fullName(&staff[i]),
			"role":       strings.Title(staff[i].Role),
			"status":     status,
			"since":      nil,
			"phone":      staff[i].Phone,
			"avatar_hue": staff[i].AvatarHue,
		})
	}
	return out, nil
}

func (s *Service) isStoreOpen(ctx context.Context, storeID uuid.UUID) (bool, error) {
	status, err := s.repo.StoreStatus(ctx, storeID)
	if err != nil {
		return false, err
	}
	return status == StatusOpen, nil
}

// OrgStats aggregates the KPI header strip across all active stores.
func (s *Service) OrgStats(ctx context.Context, orgID uuid.UUID) (map[string]interface{}, error) {
	stores, err := s.repo.ListStores(ctx, orgID, "")
	if err != nil {
		return nil, err
	}
	stats := map[string]interface{}{
		"total_stores":  len(stores),
		"open_count":    0,
		"closed_count":  0,
		"paused_count":  0,
		"total_revenue": int64(0),
		"total_txns":    int64(0),
		"staff_on_duty": int64(0),
	}
	var revenue, txns, staff int64
	open, closed, paused := 0, 0, 0
	for i := range stores {
		switch stores[i].Status {
		case StatusOpen:
			open++
		case StatusClosed:
			closed++
		case StatusPaused:
			paused++
		}
		rev, cnt, err := s.repo.TodayKPIs(ctx, stores[i].ID)
		if err != nil {
			return nil, err
		}
		revenue += rev
		txns += cnt
		n, err := s.repo.CountStaff(ctx, stores[i].ID)
		if err != nil {
			return nil, err
		}
		staff += n
	}
	stats["open_count"] = open
	stats["closed_count"] = closed
	stats["paused_count"] = paused
	stats["total_revenue"] = revenue
	stats["total_txns"] = txns
	stats["staff_on_duty"] = staff
	return stats, nil
}

// Create creates a store, enforcing the subscription store limit.
func (s *Service) Create(ctx context.Context, orgID uuid.UUID, in *StoreInput) (map[string]interface{}, error) {
	if strings.TrimSpace(in.Name) == "" {
		return nil, apierr.BadRequest("Store name is required")
	}
	if err := s.checkStoreLimit(ctx, orgID); err != nil {
		return nil, err
	}
	store := &Store{
		ID:             uuid.New(),
		OrganisationID: orgID,
		Name:           strings.TrimSpace(in.Name),
		Code:           strings.ToUpper(strings.TrimSpace(in.Code)),
		Phone:          in.Phone,
		Email:          in.Email,
		Area:           in.Area,
		Address:        in.Address,
		Latitude:       in.Latitude,
		Longitude:      in.Longitude,
		OpenHours:      defaultStr(in.OpenHours, "7:00 AM - 9:00 PM"),
		Color:          defaultStr(in.Color, "#6366f1"),
		Status:         StatusOpen,
		IsActive:       true,
		VATEnabled:     true,
	}
	if in.VATEnabled != nil {
		store.VATEnabled = *in.VATEnabled
	}
	if err := s.repo.CreateStore(ctx, store); err != nil {
		return nil, err
	}
	card, err := s.StoreCard(ctx, store)
	if err != nil {
		return nil, err
	}
	return card, nil
}

// Update edits store settings (owner/admin).
func (s *Service) Update(ctx context.Context, orgID, id uuid.UUID, in *StoreInput) (map[string]interface{}, error) {
	store, err := s.repo.GetStore(ctx, orgID, id)
	if err != nil {
		return nil, apierr.NotFound("Store not found")
	}
	if in.Name != "" {
		store.Name = strings.TrimSpace(in.Name)
	}
	if in.Code != "" {
		store.Code = strings.ToUpper(strings.TrimSpace(in.Code))
	}
	if in.Phone != "" {
		store.Phone = in.Phone
	}
	if in.Email != "" {
		store.Email = in.Email
	}
	if in.Area != "" {
		store.Area = in.Area
	}
	if in.Address != "" {
		store.Address = in.Address
	}
	if in.Latitude != 0 {
		store.Latitude = in.Latitude
	}
	if in.Longitude != 0 {
		store.Longitude = in.Longitude
	}
	if in.OpenHours != "" {
		store.OpenHours = in.OpenHours
	}
	if in.Color != "" {
		store.Color = in.Color
	}
	if in.Status != "" {
		if !validStatus(in.Status) {
			return nil, apierr.BadRequest("Invalid status").WithField("status", "must be open, closed or paused")
		}
		store.Status = in.Status
	}
	if in.VATEnabled != nil {
		store.VATEnabled = *in.VATEnabled
	}
	if err := s.repo.UpdateStore(ctx, store); err != nil {
		return nil, err
	}
	return s.Get(ctx, orgID, id)
}

// SetStatus toggles a store's operational status.
func (s *Service) SetStatus(ctx context.Context, orgID, id uuid.UUID, status string) (map[string]interface{}, error) {
	if !validStatus(status) {
		return nil, apierr.BadRequest("Invalid status").WithField("status", "must be open, closed or paused")
	}
	store, err := s.repo.GetStore(ctx, orgID, id)
	if err != nil {
		return nil, apierr.NotFound("Store not found")
	}
	if err := s.repo.SetStatus(ctx, orgID, id, status); err != nil {
		return nil, err
	}
	store.Status = status
	return map[string]interface{}{"status": store.Status}, nil
}

// Deactivate soft-deletes a store.
func (s *Service) Deactivate(ctx context.Context, orgID, id uuid.UUID) error {
	store, err := s.repo.GetStore(ctx, orgID, id)
	if err != nil {
		return apierr.NotFound("Store not found")
	}
	return s.repo.DeactivateStore(ctx, orgID, store.ID)
}

// checkStoreLimit enforces the subscription-driven max store slots.
func (s *Service) checkStoreLimit(ctx context.Context, orgID uuid.UUID) error {
	maxAllowed, err := s.repo.OrgMaxStores(ctx, orgID)
	if err != nil {
		return err
	}
	if maxAllowed <= 0 {
		maxAllowed = subscriptions.DefaultIncludedStores
	}
	current, err := s.repo.CountActiveStores(ctx, orgID)
	if err != nil {
		return err
	}
	if int(current) < maxAllowed {
		return nil
	}
	extraPrice := int64(subscriptions.DefaultExtraStorePrice)
	err = apierr.New(403, "store_limit_reached",
		"Store limit reached. Contact Ziada support to purchase more store slots.").
		WithField("can_add_store", false).
		WithField("current_active_stores", current).
		WithField("max_stores_allowed", maxAllowed).
		WithField("extra_store_price_per_month", extraPrice).
		WithField("action", "contact_support")
	return err
}

// ── helpers ──────────────────────────────────────────────────────────────────

var eat = time.FixedZone("EAT", 3*3600)

func currentWeekStart() time.Time {
	now := time.Now().In(eat)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, eat)
	offset := (int(today.Weekday()) + 6) % 7 // Monday = 0
	return today.AddDate(0, 0, -offset)
}

func validStatus(s string) bool {
	for _, v := range ValidStatuses {
		if v == s {
			return true
		}
	}
	return false
}

func defaultStr(v, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return v
}

func fullName(m *StaffMember) string {
	name := strings.TrimSpace(m.FirstName + " " + m.LastName)
	if name == "" {
		return m.Phone
	}
	return name
}

func round1(v float64) float64 {
	return float64(int(v*10+0.5)) / 10
}
