package analytics

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	eatOffset = 3 * time.Hour
	dayLayout = "2006-01-02"
)

var eatZone = time.FixedZone("EAT", int(eatOffset/time.Hour)*3600)

// Range describes a parsed analytics period.
type Range struct {
	From     time.Time
	To       time.Time
	FromDate time.Time // EAT calendar start (for labels)
	ToDate   time.Time // EAT calendar end
	Days     int
}

// ParseRange converts query params (range=7d|30d|90d|ytd, date_from, date_to)
// into a UTC time window. Defaults to the last 30 days.
func ParseRange(query func(string) string) Range {
	now := time.Now().In(eatZone)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, eatZone)

	if df := query("date_from"); df != "" {
		if dt := query("date_to"); dt != "" {
			from, err1 := time.ParseInLocation(dayLayout, df, eatZone)
			to, err2 := time.ParseInLocation(dayLayout, dt, eatZone)
			if err1 == nil && err2 == nil {
				return buildRange(from, to.AddDate(0, 0, 1))
			}
		}
	}

	switch query("range") {
	case "7d":
		return buildRange(today.AddDate(0, 0, -6), today.AddDate(0, 0, 1))
	case "90d":
		return buildRange(today.AddDate(0, 0, -89), today.AddDate(0, 0, 1))
	case "ytd":
		yearStart := time.Date(today.Year(), 1, 1, 0, 0, 0, 0, eatZone)
		return buildRange(yearStart, today.AddDate(0, 0, 1))
	}
	return buildRange(today.AddDate(0, 0, -29), today.AddDate(0, 0, 1))
}

func buildRange(fromEAT, toEATExclusive time.Time) Range {
	days := int(toEATExclusive.Sub(fromEAT).Hours()/24) + 1
	if days < 1 {
		days = 1
	}
	return Range{
		From:     fromEAT.UTC(),
		To:       toEATExclusive.UTC(),
		FromDate: fromEAT,
		ToDate:   toEATExclusive.AddDate(0, 0, -1),
		Days:     days,
	}
}

// Round1 rounds a float to 1 decimal place.
func Round1(v float64) float64 { return math.Round(v*10) / 10 }

// Service orchestrates analytics queries and shapes responses.
type Service struct {
	repo *Repo
}

// NewService builds an analytics Service.
func NewService(repo *Repo) *Service { return &Service{repo: repo} }

// ── KPIs ─────────────────────────────────────────────────────────────────────

// KPISummary computes the analytics header KPI strip for a range.
func (s *Service) KPISummary(ctx context.Context, storeID uuid.UUID, rng Range) (map[string]interface{}, error) {
	agg, err := s.repo.KPIAgg(ctx, storeID, rng.From, rng.To)
	if err != nil {
		return nil, err
	}
	totalRevenue := agg.Revenue + agg.CreditRevenue

	refunds, err := s.repo.RefundTotal(ctx, storeID, rng.From, rng.To)
	if err != nil {
		return nil, err
	}
	discTotal, discCount, err := s.repo.DiscountAgg(ctx, storeID, rng.From, rng.To)
	if err != nil {
		return nil, err
	}
	expTotal, expCount, err := s.repo.ExpenseAgg(ctx, storeID, rng.From, rng.To)
	if err != nil {
		return nil, err
	}

	priorEnd := rng.From.Add(-1 * time.Second)
	priorStart := priorEnd.Add(-time.Duration(rng.Days) * 24 * time.Hour)
	priorRev, priorTxns, err := s.repo.PriorRevenue(ctx, storeID, priorStart, priorEnd)
	if err != nil {
		return nil, err
	}

	out := map[string]interface{}{
		"revenue":               totalRevenue,
		"profit":                agg.Profit,
		"margin_pct":            pct(agg.Profit, totalRevenue),
		"transaction_count":     agg.Transactions,
		"customer_count":        agg.Customers,
		"avg_ticket":            div(totalRevenue, agg.Transactions),
		"tax_collected":         agg.Tax,
		"refund_total":          refunds,
		"discount_amount":       discTotal,
		"discounted_count":      discCount,
		"expense_amount":        expTotal,
		"expense_count":         expCount,
		"revenue_delta_pct":     deltaPct(totalRevenue, priorRev),
		"transaction_delta_pct": deltaPct(agg.Transactions, priorTxns),
	}
	return out, nil
}

// RevenueTrend returns a zero-filled day-by-day revenue series.
func (s *Service) RevenueTrend(ctx context.Context, storeID uuid.UUID, rng Range) ([]map[string]interface{}, error) {
	rows, err := s.repo.RevenueTrend(ctx, storeID, rng.From, rng.To)
	if err != nil {
		return nil, err
	}
	byDay := make(map[string]trendRow, len(rows))
	for _, row := range rows {
		byDay[row.Day.In(eatZone).Format(dayLayout)] = row
	}

	out := make([]map[string]interface{}, 0, rng.Days)
	for d := rng.FromDate; !d.After(rng.ToDate); d = d.AddDate(0, 0, 1) {
		key := d.Format(dayLayout)
		row, ok := byDay[key]
		if !ok {
			row = trendRow{}
		}
		out = append(out, map[string]interface{}{
			"date":         key,
			"label":        d.Format("2 Jan"),
			"revenue":      row.Revenue + row.CreditRevenue,
			"profit":       row.Profit,
			"transactions": row.Transactions,
		})
	}
	return out, nil
}

// PaymentMix returns revenue by payment method with percentage shares.
func (s *Service) PaymentMix(ctx context.Context, storeID uuid.UUID, rng Range) ([]map[string]interface{}, error) {
	rows, err := s.repo.PaymentMethodTotals(ctx, storeID, rng.From, rng.To)
	if err != nil {
		return nil, err
	}
	totals := map[string]int64{}
	for _, row := range rows {
		totals[row.Method] = row.Amount
	}
	credit, err := s.repo.CreditIssued(ctx, storeID, rng.From, rng.To)
	if err != nil {
		return nil, err
	}
	if credit > 0 {
		totals["Credit"] += credit
	}

	order := []string{"Cash", "M-Pesa", "Tigo Pesa", "Bank", "Credit"}
	grand := int64(0)
	for _, m := range order {
		grand += totals[m]
	}
	if grand <= 0 {
		grand = 1
	}
	out := []map[string]interface{}{}
	for _, m := range order {
		amount := totals[m]
		if amount <= 0 {
			continue
		}
		out = append(out, map[string]interface{}{
			"method": m,
			"amount": amount,
			"pct":    Round1(float64(amount) / float64(grand) * 100),
		})
	}
	return out, nil
}

// TopProducts returns the top products by revenue.
func (s *Service) TopProducts(ctx context.Context, storeID uuid.UUID, rng Range, limit int) ([]map[string]interface{}, error) {
	rows, err := s.repo.TopProducts(ctx, storeID, rng.From, rng.To, limit)
	if err != nil {
		return nil, err
	}
	out := make([]map[string]interface{}, 0, len(rows))
	for _, row := range rows {
		profit := row.Revenue - row.Cost
		out = append(out, map[string]interface{}{
			"product_id":   row.ProductID,
			"product_name": row.ProductName,
			"product_sku":  row.ProductSKU,
			"qty_sold":     row.Units,
			"revenue":      row.Revenue,
			"cost":         row.Cost,
			"profit":       profit,
		})
	}
	return out, nil
}

// ── Sales sub-page ───────────────────────────────────────────────────────────

// SalesBreakdown returns category, day-of-week and hourly patterns.
func (s *Service) SalesBreakdown(ctx context.Context, storeID uuid.UUID, rng Range) (map[string]interface{}, error) {
	priorEnd := rng.From.Add(-1 * time.Second)
	priorStart := priorEnd.Add(-time.Duration(rng.Days) * 24 * time.Hour)

	// Category breakdown with prior-period trend.
	cur, err := s.repo.CategoryRevenue(ctx, storeID, rng.From, rng.To)
	if err != nil {
		return nil, err
	}
	prior, err := s.repo.CategoryRevenue(ctx, storeID, priorStart, priorEnd)
	if err != nil {
		return nil, err
	}
	priorByCat := map[string]int64{}
	for _, row := range prior {
		priorByCat[row.Name] = row.Revenue
	}
	totalRev := int64(0)
	for _, row := range cur {
		totalRev += row.Revenue
	}
	sort.Slice(cur, func(i, j int) bool { return cur[i].Revenue > cur[j].Revenue })
	categories := make([]map[string]interface{}, 0, len(cur))
	for _, row := range cur {
		prev := priorByCat[row.Name]
		categories = append(categories, map[string]interface{}{
			"name":      row.Name,
			"revenue":   row.Revenue,
			"pct":       Round1(pctVal(row.Revenue, totalRev)),
			"trend_pct": trendPct(row.Revenue, prev),
		})
	}

	// Day of week averages.
	dows, err := s.repo.DayOfWeekTotals(ctx, storeID, rng.From, rng.To)
	if err != nil {
		return nil, err
	}
	dowTotals := map[int]int64{}
	for _, row := range dows {
		dowTotals[row.Dow] = row.Total
	}
	labels := []string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"}
	occurrences := make([]int, 7)
	for d := rng.FromDate; !d.After(rng.ToDate); d = d.AddDate(0, 0, 1) {
		occurrences[int(d.Weekday()+6)%7]++
	}
	dayOfWeek := make([]map[string]interface{}, 7)
	for i := 0; i < 7; i++ {
		occ := occurrences[i]
		if occ == 0 {
			occ = 1
		}
		dayOfWeek[i] = map[string]interface{}{
			"dow":         i,
			"label":       labels[i],
			"avg_revenue": dowTotals[i] / int64(occ),
		}
	}

	// Hourly pattern (07:00–20:00) averaged per day.
	hours, err := s.repo.HourlyTotals(ctx, storeID, rng.From, rng.To)
	if err != nil {
		return nil, err
	}
	hourMap := map[int]int64{}
	for _, row := range hours {
		hourMap[row.Hour] = row.Total
	}
	numDays := rng.Days
	if numDays < 1 {
		numDays = 1
	}
	hourlyPattern := make([]map[string]interface{}, 0, 14)
	for hr := 7; hr <= 20; hr++ {
		hourlyPattern = append(hourlyPattern, map[string]interface{}{
			"hour":        hr,
			"label":       fmt.Sprintf("%02d:00", hr),
			"avg_revenue": hourMap[hr] / int64(numDays),
		})
	}

	return map[string]interface{}{
		"category_breakdown": categories,
		"day_of_week":        dayOfWeek,
		"hourly_pattern":     hourlyPattern,
	}, nil
}

// ── Products sub-page ────────────────────────────────────────────────────────

// ProductPerformance returns the full product table with margins and trends.
func (s *Service) ProductPerformance(ctx context.Context, storeID uuid.UUID, rng Range, category, sortBy string) (map[string]interface{}, error) {
	rows, err := s.repo.Products(ctx, storeID, rng.From, rng.To, category)
	if err != nil {
		return nil, err
	}
	priorEnd := rng.From.Add(-1 * time.Second)
	priorStart := priorEnd.Add(-time.Duration(rng.Days) * 24 * time.Hour)
	prior, err := s.repo.PriorProductRevenue(ctx, storeID, priorStart, priorEnd)
	if err != nil {
		return nil, err
	}

	totalRev := int64(0)
	for _, row := range rows {
		totalRev += row.Revenue
	}

	products := make([]map[string]interface{}, 0, len(rows))
	for _, row := range rows {
		profit := row.Revenue - row.Cost
		key := row.ProductID.String()
		if row.ProductID == uuid.Nil {
			key = row.ProductName
		}
		products = append(products, map[string]interface{}{
			"product_id":        row.ProductID,
			"product_name":      row.ProductName,
			"product_sku":       row.ProductSKU,
			"category":          row.Category,
			"units_sold":        row.Units,
			"revenue":           row.Revenue,
			"cost":              row.Cost,
			"profit":            profit,
			"margin_pct":        pctVal(profit, row.Revenue),
			"trend_pct":         trendPct(row.Revenue, prior[key]),
			"revenue_share_pct": Round1(pctVal(row.Revenue, totalRev)),
		})
	}

	switch strings.ToLower(sortBy) {
	case "units":
		sort.Slice(products, func(i, j int) bool { return products[i]["units_sold"].(int) > products[j]["units_sold"].(int) })
	case "margin":
		sort.Slice(products, func(i, j int) bool { return products[i]["margin_pct"].(float64) > products[j]["margin_pct"].(float64) })
	case "profit":
		sort.Slice(products, func(i, j int) bool { return products[i]["profit"].(int64) > products[j]["profit"].(int64) })
	default:
		sort.Slice(products, func(i, j int) bool { return products[i]["revenue"].(int64) > products[j]["revenue"].(int64) })
	}

	totalProfit := int64(0)
	totalUnits := 0
	for _, p := range products {
		totalProfit += p["profit"].(int64)
		totalUnits += p["units_sold"].(int)
	}

	return map[string]interface{}{
		"totals": map[string]interface{}{
			"revenue":    totalRev,
			"profit":     totalProfit,
			"margin_pct": Round1(pctVal(totalProfit, totalRev)),
			"units_sold": totalUnits,
			"sku_count":  len(products),
		},
		"products": products,
	}, nil
}

// ── Customers sub-page ───────────────────────────────────────────────────────

// CustomerAnalytics returns visits, segments, cohorts and top customers.
func (s *Service) CustomerAnalytics(ctx context.Context, storeID uuid.UUID, rng Range) (map[string]interface{}, error) {
	firstDates, err := s.repo.CustomerFirstDates(ctx, storeID)
	if err != nil {
		return nil, err
	}
	visits, err := s.repo.CustomerVisitDays(ctx, storeID, rng.From, rng.To)
	if err != nil {
		return nil, err
	}
	segments, err := s.repo.CustomerSegments(ctx, storeID, rng.From, rng.To)
	if err != nil {
		return nil, err
	}
	monthly, err := s.repo.CustomerMonthlyActivity(ctx, storeID)
	if err != nil {
		return nil, err
	}
	top, err := s.repo.TopCustomers(ctx, storeID, rng.From, rng.To, 20)
	if err != nil {
		return nil, err
	}

	// Daily visits: new vs returning.
	dayNew := map[string]int{}
	dayReturning := map[string]int{}
	for _, v := range visits {
		day := v.Day.In(eatZone).Format(dayLayout)
		first, ok := firstDates[v.CustomerID]
		if !ok {
			dayReturning[day]++
			continue
		}
		firstDay := first.In(eatZone).Format(dayLayout)
		if firstDay >= rng.FromDate.Format(dayLayout) {
			if firstDay == day {
				dayNew[day]++
			} else {
				dayReturning[day]++
			}
		} else {
			dayReturning[day]++
		}
	}
	dailyVisits := make([]map[string]interface{}, 0, rng.Days)
	totalVisits := 0
	newCustomers := 0
	returning := 0
	for d := rng.FromDate; !d.After(rng.ToDate); d = d.AddDate(0, 0, 1) {
		key := d.Format(dayLayout)
		n := dayNew[key]
		rt := dayReturning[key]
		totalVisits += n + rt
		newCustomers += n
		returning += rt
		dailyVisits = append(dailyVisits, map[string]interface{}{
			"date":          key,
			"label":         d.Format("2 Jan"),
			"total":         n + rt,
			"new_customers": n,
			"returning":     rt,
		})
	}

	// Segments.
	segTotal := 0
	totalSpend := int64(0)
	for _, row := range segments {
		segTotal += int(row.Count)
		totalSpend += row.Spend
	}
	if segTotal == 0 {
		segTotal = 1
	}
	segOut := make([]map[string]interface{}, 0, len(segments))
	for _, row := range segments {
		segOut = append(segOut, map[string]interface{}{
			"segment": row.Segment,
			"count":   row.Count,
			"spend":   row.Spend,
			"pct":     Round1(float64(row.Count) / float64(segTotal) * 100),
		})
	}

	// Retention cohorts (last 4 months).
	cohorts := buildCohorts(firstDates, monthly)

	// Top customers.
	now := time.Now().In(eatZone)
	topOut := make([]map[string]interface{}, 0, len(top))
	for _, row := range top {
		var lastSeenDays interface{}
		if !row.LastTxn.IsZero() {
			lastSeenDays = int(now.Sub(row.LastTxn.In(eatZone)).Hours() / 24)
		}
		topOut = append(topOut, map[string]interface{}{
			"customer_id":    row.CustomerID,
			"name":           row.Name,
			"visits":         row.Visits,
			"spent":          row.Spent,
			"avg_ticket":     div(row.Spent, row.Visits),
			"last_seen_days": lastSeenDays,
			"avatar_hue":     row.AvatarHue,
		})
	}

	return map[string]interface{}{
		"daily_visits":      dailyVisits,
		"segments":          segOut,
		"retention_cohorts": cohorts,
		"top_customers":     topOut,
		"kpis": map[string]interface{}{
			"total_visits":       totalVisits,
			"new_customers":      newCustomers,
			"returning":          returning,
			"retention_rate_pct": Round1(pctVal(int64(returning), int64(totalVisits))),
			"avg_daily_visits":   Round1(float64(totalVisits) / float64(rng.Days)),
			"total_spend":        totalSpend,
			"avg_visit_value":    div(totalSpend, int64(totalVisits)),
		},
	}, nil
}

func buildCohorts(firstDates map[uuid.UUID]time.Time, monthly []monthRow) []map[string]interface{} {
	now := time.Now().In(eatZone)
	currentMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, eatZone)

	starts := make([]time.Time, 0, 4)
	for i := 3; i >= 0; i-- {
		starts = append(starts, currentMonth.AddDate(0, -i, 0))
	}

	active := map[string]bool{}
	for _, row := range monthly {
		key := fmt.Sprintf("%s|%d-%d", row.CustomerID, row.Month.Year(), int(row.Month.Month()))
		active[key] = true
	}

	cohorts := make([]map[string]interface{}, 0, 4)
	for _, start := range starts {
		next := start.AddDate(0, 1, 0)
		ids := map[uuid.UUID]bool{}
		for cid, first := range firstDates {
			f := first.In(eatZone)
			if !f.Before(start) && f.Before(next) {
				ids[cid] = true
			}
		}
		ret := func(offset int) int {
			target := start.AddDate(0, offset, 0)
			if !target.Before(currentMonth) {
				return 0
			}
			count := 0
			for cid := range ids {
				if active[fmt.Sprintf("%s|%d-%d", cid, target.Year(), int(target.Month()))] {
					count++
				}
			}
			return count
		}
		cohorts = append(cohorts, map[string]interface{}{
			"month": start.Format("Jan"),
			"year":  start.Year(),
			"new":   len(ids),
			"m1":    ret(1),
			"m2":    ret(2),
			"m3":    ret(3),
		})
	}
	return cohorts
}

// ── Cashflow sub-page ────────────────────────────────────────────────────────

// Cashflow returns daily cash in/out, running balance and expense breakdown.
func (s *Service) Cashflow(ctx context.Context, storeID uuid.UUID, rng Range) (map[string]interface{}, error) {
	days, err := s.repo.CashflowDays(ctx, storeID, rng.From, rng.To)
	if err != nil {
		return nil, err
	}
	expenseDays, err := s.repo.ExpenseDays(ctx, storeID, rng.From, rng.To)
	if err != nil {
		return nil, err
	}
	expenseCats, err := s.repo.ExpenseByCategory(ctx, storeID, rng.From, rng.To)
	if err != nil {
		return nil, err
	}
	mix, err := s.PaymentMix(ctx, storeID, rng)
	if err != nil {
		return nil, err
	}
	credit, err := s.repo.CreditOutstanding(ctx, storeID)
	if err != nil {
		return nil, err
	}

	inflowByDay := map[string]int64{}
	cogsByDay := map[string]int64{}
	for _, row := range days {
		key := row.Day.In(eatZone).Format(dayLayout)
		inflowByDay[key] = row.Revenue + row.CreditRevenue
		cogsByDay[key] = row.Cogs
	}
	opexByDay := map[string]int64{}
	for _, row := range expenseDays {
		opexByDay[row.Day.In(eatZone).Format(dayLayout)] = row.Amount
	}

	daily := make([]map[string]interface{}, 0, rng.Days)
	running := make([]map[string]interface{}, 0, rng.Days)
	var totalInflow, totalCogs, totalOpex, totalNet int64
	balance := int64(0)
	for d := rng.FromDate; !d.After(rng.ToDate); d = d.AddDate(0, 0, 1) {
		key := d.Format(dayLayout)
		inflow := inflowByDay[key]
		cogs := cogsByDay[key]
		opex := opexByDay[key]
		net := inflow - cogs - opex

		totalInflow += inflow
		totalCogs += cogs
		totalOpex += opex
		totalNet += net
		balance += net

		daily = append(daily, map[string]interface{}{
			"date":   key,
			"label":  d.Format("2 Jan"),
			"inflow": inflow,
			"cogs":   cogs,
			"opex":   opex,
			"net":    net,
		})
		running = append(running, map[string]interface{}{
			"date":    key,
			"label":   d.Format("2 Jan"),
			"balance": balance,
		})
	}

	expenseBreakdown := []map[string]interface{}{
		{"name": "Cost of Goods", "amount": totalCogs, "pct": Round1(pctVal(totalCogs, totalInflow))},
	}
	for _, row := range expenseCats {
		expenseBreakdown = append(expenseBreakdown, map[string]interface{}{
			"name":   row.Name,
			"amount": row.Revenue,
			"pct":    Round1(pctVal(row.Revenue, totalInflow)),
		})
	}

	return map[string]interface{}{
		"totals": map[string]interface{}{
			"inflow":         totalInflow,
			"cogs":           totalCogs,
			"opex":           totalOpex,
			"net":            totalNet,
			"net_margin_pct": Round1(pctVal(totalNet, totalInflow)),
		},
		"daily":             daily,
		"running_balance":   running,
		"expense_breakdown": expenseBreakdown,
		"payment_inflow":    mix,
		"credit_outstanding": map[string]interface{}{
			"total":          credit.Total,
			"customer_count": credit.CustomerCount,
			"overdue_count":  credit.OverdueCount,
		},
	}, nil
}

// ── Dashboard ────────────────────────────────────────────────────────────────

// Dashboard returns all widgets for the main dashboard page in one call.
func (s *Service) Dashboard(ctx context.Context, storeID uuid.UUID) (map[string]interface{}, error) {
	now := time.Now().In(eatZone)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, eatZone)
	todayRange := buildRange(today, today.AddDate(0, 0, 1))
	sevenDays := buildRange(today.AddDate(0, 0, -6), today.AddDate(0, 0, 1))
	return s.DashboardForRange(ctx, storeID, todayRange, sevenDays)
}

// DashboardForRange returns the complete dashboard payload for a selected
// period. The secondary range is used for the compact "best sellers" widget;
// callers can pass the same range when the dashboard is fully period-driven.
func (s *Service) DashboardForRange(ctx context.Context, storeID uuid.UUID, todayRange, secondaryRange Range) (map[string]interface{}, error) {

	kpis, err := s.KPISummary(ctx, storeID, todayRange)
	if err != nil {
		return nil, err
	}

	hourly, err := s.repo.HourlyBreakdown(ctx, storeID, todayRange.From, todayRange.To)
	if err != nil {
		return nil, err
	}
	expHours, err := s.repo.ExpenseHourly(ctx, storeID, todayRange.From, todayRange.To)
	if err != nil {
		return nil, err
	}
	hourMap := map[int]hourBreakdown{}
	for _, row := range hourly {
		hourMap[row.Hour] = row
	}
	expMap := map[int]int64{}
	for _, row := range expHours {
		expMap[row.Hour] = row.Total
	}
	hourlyToday := make([]map[string]interface{}, 0, 14)
	for hr := 7; hr <= 20; hr++ {
		h := hourMap[hr]
		hourlyToday = append(hourlyToday, map[string]interface{}{
			"hour":            hr,
			"label":           fmt.Sprintf("%02d:00", hr),
			"revenue":         h.Revenue,
			"profit":          h.Profit,
			"discount_amount": h.DiscountAmount,
			"credit_amount":   h.CreditAmount,
			"expense_amount":  expMap[hr],
			"txn_count":       h.TxnCount,
		})
	}

	mix, err := s.PaymentMix(ctx, storeID, todayRange)
	if err != nil {
		return nil, err
	}
	top, err := s.TopProducts(ctx, storeID, secondaryRange, 6)
	if err != nil {
		return nil, err
	}
	lowStock, err := s.repo.LowStock(ctx, storeID, 10)
	if err != nil {
		return nil, err
	}
	lowOut := make([]map[string]interface{}, 0, len(lowStock))
	for _, p := range lowStock {
		status := "low"
		if p.Stock <= 0 {
			status = "critical"
		}
		lowOut = append(lowOut, map[string]interface{}{
			"id":        p.ID,
			"name":      p.Name,
			"sku":       p.SKU,
			"stock":     p.Stock,
			"min_stock": p.Reorder,
			"status":    status,
		})
	}
	credit, err := s.repo.CreditOutstanding(ctx, storeID)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"date_from":    todayRange.FromDate.Format(dayLayout),
		"date_to":      todayRange.ToDate.Format(dayLayout),
		"days":         todayRange.Days,
		"kpis_today":   kpis,
		"hourly_today": hourlyToday,
		"payment_mix":  mix,
		"top_products": top,
		"low_stock":    lowOut,
		"credit_kpi": map[string]interface{}{
			"total":          credit.Total,
			"customer_count": credit.CustomerCount,
			"overdue_count":  credit.OverdueCount,
		},
	}, nil
}

// ── helpers ──────────────────────────────────────────────────────────────────

func pct(part, whole int64) float64 { return Round1(pctVal(part, whole)) }

func pctVal(part, whole int64) float64 {
	if whole == 0 {
		return 0
	}
	return float64(part) / float64(whole) * 100
}

func trendPct(cur, prior int64) interface{} {
	if prior == 0 {
		return nil
	}
	return Round1(float64(cur-prior) / float64(prior) * 100)
}

func deltaPct(cur, prior int64) interface{} {
	if prior == 0 {
		return nil
	}
	return Round1(float64(cur-prior) / float64(prior) * 100)
}

func div(n, d int64) int64 {
	if d == 0 {
		return 0
	}
	return n / d
}
