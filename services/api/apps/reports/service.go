package reports

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/troubleman96/ziadapos/apps/analytics"
	"github.com/troubleman96/ziadapos/internal/apierr"
	"github.com/troubleman96/ziadapos/internal/mailer"
)

// Service implements report generation, CSV building, export history and
// scheduled-report management.
type Service struct {
	repo   *Repo
	anal   *analytics.Service
	mailer *mailer.Mailer
}

// NewService wires a reports Service.
func NewService(repo *Repo, mailer *mailer.Mailer) *Service {
	return &Service{
		repo:   repo,
		anal:   analytics.NewService(analytics.NewRepo(repo.db)),
		mailer: mailer,
	}
}

var eatZone = time.FixedZone("EAT", 3*3600)

// ── Date range helpers ───────────────────────────────────────────────────────

// Range holds an inclusive EAT calendar range plus its display label.
type Range struct {
	From  time.Time // inclusive EAT midnight
	To    time.Time // inclusive EAT midnight
	Label string
}

// ParseRangeForReport parses ?range=|date_from=+date_to= into a Range.
func ParseRangeForReport(query func(string) string) Range {
	now := time.Now().In(eatZone)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, eatZone)

	if df := query("date_from"); df != "" {
		if dt := query("date_to"); dt != "" {
			from, err1 := time.ParseInLocation("2006-01-02", df, eatZone)
			to, err2 := time.ParseInLocation("2006-01-02", dt, eatZone)
			if err1 == nil && err2 == nil {
				if to.Before(from) {
					to = from
				}
				return buildReportRange(from, to)
			}
		}
	}

	switch query("range") {
	case Range7D:
		return buildReportRange(today.AddDate(0, 0, -6), today)
	case Range90D:
		return buildReportRange(today.AddDate(0, 0, -89), today)
	case RangeMonth:
		return buildReportRange(time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, eatZone), today)
	case RangeYTD:
		return buildReportRange(time.Date(today.Year(), 1, 1, 0, 0, 0, 0, eatZone), today)
	}
	return buildReportRange(today.AddDate(0, 0, -29), today)
}

func buildReportRange(from, to time.Time) Range {
	return Range{From: from, To: to, Label: periodLabel(from, to)}
}

func lastDayOfMonth(t time.Time) time.Time {
	first := time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, eatZone)
	return first.AddDate(0, 1, -1)
}

func periodLabel(start, end time.Time) string {
	if start.Year() == end.Year() && start.Month() == end.Month() {
		if start.Day() == 1 && end.Equal(lastDayOfMonth(start)) {
			return start.Format("January 2006")
		}
		return fmt.Sprintf("%d–%d %s %d", start.Day(), end.Day(), end.Format("Jan"), end.Year())
	}
	if start.Year() == end.Year() {
		return fmt.Sprintf("%d %s – %d %s %d", start.Day(), start.Format("Jan"), end.Day(), end.Format("Jan"), end.Year())
	}
	return fmt.Sprintf("%d %s %d – %d %s %d", start.Day(), start.Format("Jan"), start.Year(), end.Day(), end.Format("Jan"), end.Year())
}

// ComputeNextSend returns the next send time for a frequency.
func ComputeNextSend(frequency string, from time.Time) time.Time {
	switch frequency {
	case FreqWeekly:
		return from.AddDate(0, 0, 7)
	case FreqMonthly:
		return from.AddDate(0, 1, 0)
	default:
		return from.AddDate(0, 0, 1)
	}
}

// asAnalyticsRange converts a report Range into the analytics range type.
func (r Range) asAnalyticsRange() analytics.Range {
	return analytics.Range{
		From:     r.From.UTC(),
		To:       r.To.AddDate(0, 0, 1).UTC(),
		FromDate: r.From,
		ToDate:   r.To,
		Days:     int(r.To.Sub(r.From).Hours()/24) + 1,
	}
}

// ── Report types catalogue ───────────────────────────────────────────────────

// ReportTypes returns the static catalogue shown on the Reports overview tab.
func (s *Service) ReportTypes() []map[string]interface{} {
	return []map[string]interface{}{
		{
			"id":                  ReportSales,
			"name":                "Sales Summary",
			"desc":                "Daily sales totals, top products, payment breakdown and revenue by category.",
			"color":               "accent",
			"supports_date_range": true,
		},
		{
			"id":                  ReportInventory,
			"name":                "Inventory Valuation",
			"desc":                "Cost and retail value of all stock on hand, sorted by category.",
			"color":               "good",
			"supports_date_range": false,
		},
		{
			"id":                  ReportTax,
			"name":                "Tax Statement",
			"desc":                "VAT collected, taxable revenue and TRA-ready summary for compliance.",
			"color":               "warn",
			"supports_date_range": true,
		},
		{
			"id":                  ReportCredit,
			"name":                "Credit Aged Debtors",
			"desc":                "Accounts receivable aged into buckets with contact details.",
			"color":               "bad",
			"supports_date_range": false,
		},
	}
}

// ── Report generation ────────────────────────────────────────────────────────

// GenerateReport builds report data, records an export audit row and returns
// the data plus the period label and date bounds.
func (s *Service) GenerateReport(ctx context.Context, reportType string, storeID, userID uuid.UUID, rng Range) (map[string]interface{}, error) {
	name, ok := ReportTypeNames[reportType]
	if !ok {
		return nil, apierr.BadRequest("Unknown report type")
	}

	storeName, _ := s.repo.storeName(ctx, storeID)

	header := map[string]interface{}{
		"report_type":  reportType,
		"report_name":  name,
		"store_name":   storeName,
		"date_from":    rng.From.Format("2006-01-02"),
		"date_to":      rng.To.Format("2006-01-02"),
		"generated_at": time.Now().Format("2006-01-02"),
	}

	var body map[string]interface{}
	var err error
	switch reportType {
	case ReportSales:
		body, err = s.generateSalesSummary(ctx, storeID, rng)
	case ReportInventory:
		body, err = s.generateInventoryValuation(ctx, storeID)
	case ReportTax:
		body, err = s.generateTaxStatement(ctx, storeID, rng)
	case ReportCredit:
		body, err = s.generateCreditAgedDebtors(ctx, storeID)
	default:
		return nil, apierr.BadRequest("Unknown report type")
	}
	if err != nil {
		return nil, err
	}

	data := make(map[string]interface{}, len(header)+len(body))
	for k, v := range header {
		data[k] = v
	}
	for k, v := range body {
		data[k] = v
	}
	return data, nil
}

func (s *Service) generateSalesSummary(ctx context.Context, storeID uuid.UUID, rng Range) (map[string]interface{}, error) {
	ar := rng.asAnalyticsRange()
	kpis, err := s.anal.KPISummary(ctx, storeID, ar)
	if err != nil {
		return nil, err
	}
	trend, err := s.anal.RevenueTrend(ctx, storeID, ar)
	if err != nil {
		return nil, err
	}
	top, err := s.anal.TopProducts(ctx, storeID, ar, 20)
	if err != nil {
		return nil, err
	}
	mix, err := s.anal.PaymentMix(ctx, storeID, ar)
	if err != nil {
		return nil, err
	}
	sales, err := s.anal.SalesBreakdown(ctx, storeID, ar)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"kpis":               kpis,
		"daily_trend":        trend,
		"top_products":       top,
		"payment_mix":        mix,
		"category_breakdown": sales["category_breakdown"],
		"day_of_week":        sales["day_of_week"],
	}, nil
}

func (s *Service) generateInventoryValuation(ctx context.Context, storeID uuid.UUID) (map[string]interface{}, error) {
	rows, err := s.repo.InventoryValuation(ctx, storeID)
	if err != nil {
		return nil, err
	}

	type catTotal struct {
		costValue   int64
		retailValue int64
		skuCount    int
	}
	catTotals := map[string]*catTotal{}
	order := []string{}

	out := make([]map[string]interface{}, 0, len(rows))
	for _, p := range rows {
		costV := int64(p.Stock) * p.CostPrice
		retailV := int64(p.Stock) * p.SellPrice
		margin := marginPct(retailV, costV)

		out = append(out, map[string]interface{}{
			"product_id":   p.ProductID,
			"name":         p.Name,
			"sku":          p.SKU,
			"category":     p.Category,
			"stock":        p.Stock,
			"unit_cost":    p.CostPrice,
			"unit_price":   p.SellPrice,
			"cost_value":   costV,
			"retail_value": retailV,
			"margin_pct":   margin,
		})

		if _, exists := catTotals[p.Category]; !exists {
			catTotals[p.Category] = &catTotal{}
			order = append(order, p.Category)
		}
		catTotals[p.Category].costValue += costV
		catTotals[p.Category].retailValue += retailV
		catTotals[p.Category].skuCount++
	}

	var grandCost, grandRetail int64
	categories := make([]map[string]interface{}, 0, len(order))
	for _, cat := range order {
		t := catTotals[cat]
		grandCost += t.costValue
		grandRetail += t.retailValue
		categories = append(categories, map[string]interface{}{
			"category":     cat,
			"sku_count":    t.skuCount,
			"cost_value":   t.costValue,
			"retail_value": t.retailValue,
			"margin_pct":   marginPct(t.retailValue, t.costValue),
		})
	}

	return map[string]interface{}{
		"products":           out,
		"category_subtotals": categories,
		"totals": map[string]interface{}{
			"sku_count":    len(out),
			"cost_value":   grandCost,
			"retail_value": grandRetail,
			"margin_pct":   marginPct(grandRetail, grandCost),
		},
		"valuation_date": time.Now().Format("2006-01-02"),
	}, nil
}

func (s *Service) generateTaxStatement(ctx context.Context, storeID uuid.UUID, rng Range) (map[string]interface{}, error) {
	rows, err := s.repo.TaxDaily(ctx, storeID, rng.From.UTC(), rng.To.AddDate(0, 0, 1).UTC())
	if err != nil {
		return nil, err
	}
	byDay := map[string]taxDay{}
	for _, row := range rows {
		byDay[row.Day.Format("2006-01-02")] = row
	}

	var grossTotal, taxTotal, refundTotal int64
	daily := make([]map[string]interface{}, 0)
	for d := rng.From; !d.After(rng.To); d = d.AddDate(0, 0, 1) {
		key := d.Format("2006-01-02")
		row := byDay[key]
		net := row.Gross - row.Tax
		grossTotal += row.Gross
		taxTotal += row.Tax
		refundTotal += row.Refunds
		daily = append(daily, map[string]interface{}{
			"date":          key,
			"label":         d.Format("2 Jan"),
			"gross_sales":   row.Gross,
			"tax_collected": row.Tax,
			"net_sales":     net,
			"refunds":       row.Refunds,
		})
	}

	taxableDays := 0
	for _, d := range daily {
		if d["gross_sales"].(int64) > 0 {
			taxableDays++
		}
	}

	return map[string]interface{}{
		"vat_rate": 0.18,
		"summary": map[string]interface{}{
			"gross_sales":          grossTotal,
			"tax_collected":        taxTotal,
			"net_sales_excl_tax":   grossTotal - taxTotal,
			"refunds":              refundTotal,
			"taxable_transactions": taxableDays,
		},
		"daily": daily,
		"note":  "This statement covers only paid (cash/mobile/bank) transactions. On-credit (tab) transactions are reported when payment is received.",
	}, nil
}

func (s *Service) generateCreditAgedDebtors(ctx context.Context, storeID uuid.UUID) (map[string]interface{}, error) {
	tabs, err := s.repo.OpenCreditTabs(ctx, storeID)
	if err != nil {
		return nil, err
	}
	today := time.Now().In(eatZone)

	type custAgg struct {
		CustomerID   uuid.UUID
		Name         string
		Phone        string
		TotalBalance int64
		Current      int64
		Days1_7      int64
		Days8_30     int64
		Days31_60    int64
		Days60Plus   int64
		OldestDue    *time.Time
		TabCount     int
	}
	customers := map[uuid.UUID]*custAgg{}
	order := []uuid.UUID{}

	for _, tab := range tabs {
		balance := tab.Amount - tab.AmountPaid
		if balance <= 0 {
			continue
		}
		agg, ok := customers[tab.CustomerID]
		if !ok {
			agg = &custAgg{CustomerID: tab.CustomerID, Name: tab.Name, Phone: tab.Phone}
			customers[tab.CustomerID] = agg
			order = append(order, tab.CustomerID)
		}
		agg.TotalBalance += balance
		agg.TabCount++

		switch ageBucket(tab.DueDate, today) {
		case 0:
			agg.Current += balance
		case 1:
			agg.Days1_7 += balance
		case 2:
			agg.Days8_30 += balance
		case 3:
			agg.Days31_60 += balance
		default:
			agg.Days60Plus += balance
		}
		if tab.DueDate != nil && (agg.OldestDue == nil || tab.DueDate.Before(*agg.OldestDue)) {
			agg.OldestDue = tab.DueDate
		}
	}

	// Sort by total balance descending.
	custList := make([]*custAgg, 0, len(order))
	for _, id := range order {
		custList = append(custList, customers[id])
	}
	for i := 1; i < len(custList); i++ {
		for j := i; j > 0 && custList[j].TotalBalance > custList[j-1].TotalBalance; j-- {
			custList[j], custList[j-1] = custList[j-1], custList[j]
		}
	}

	var grandTotal int64
	var overdueCount int
	rows := make([]map[string]interface{}, 0, len(custList))
	for _, c := range custList {
		grandTotal += c.TotalBalance
		if c.Days1_7+c.Days8_30+c.Days31_60+c.Days60Plus > 0 {
			overdueCount++
		}
		var oldest interface{}
		if c.OldestDue != nil {
			oldest = c.OldestDue.Format("2006-01-02")
		}
		rows = append(rows, map[string]interface{}{
			"customer_id":   c.CustomerID,
			"name":          c.Name,
			"phone":         c.Phone,
			"total_balance": c.TotalBalance,
			"current":       c.Current,
			"1_7d":          c.Days1_7,
			"8_30d":         c.Days8_30,
			"31_60d":        c.Days31_60,
			"60plus":        c.Days60Plus,
			"oldest_due":    oldest,
			"tab_count":     c.TabCount,
		})
	}

	bucketLabels := []struct {
		key   string
		label string
		val   func(*custAgg) int64
	}{
		{"current", "Current (not yet due)", func(c *custAgg) int64 { return c.Current }},
		{"1_7d", "1–7 days overdue", func(c *custAgg) int64 { return c.Days1_7 }},
		{"8_30d", "8–30 days overdue", func(c *custAgg) int64 { return c.Days8_30 }},
		{"31_60d", "31–60 days overdue", func(c *custAgg) int64 { return c.Days31_60 }},
		{"60plus", "60+ days overdue", func(c *custAgg) int64 { return c.Days60Plus }},
	}
	aging := make([]map[string]interface{}, 0, len(bucketLabels))
	for _, b := range bucketLabels {
		var amount int64
		count := 0
		for _, c := range custList {
			amount += b.val(c)
			if b.val(c) > 0 {
				count++
			}
		}
		aging = append(aging, map[string]interface{}{
			"bucket": b.key,
			"label":  b.label,
			"amount": amount,
			"count":  count,
		})
	}

	return map[string]interface{}{
		"customers":     rows,
		"aging_summary": aging,
		"totals": map[string]interface{}{
			"total":          grandTotal,
			"customer_count": len(rows),
			"overdue_count":  overdueCount,
		},
		"as_of_date": today.Format("2006-01-02"),
	}, nil
}

func ageBucket(due *time.Time, today time.Time) int {
	if due == nil {
		return 0
	}
	overdueDays := int(today.Sub(due.In(eatZone)).Hours() / 24)
	switch {
	case overdueDays <= 0:
		return 0
	case overdueDays <= 7:
		return 1
	case overdueDays <= 30:
		return 2
	case overdueDays <= 60:
		return 3
	default:
		return 4
	}
}

func marginPct(retail, cost int64) float64 {
	if retail == 0 {
		return 0
	}
	return float64(int((float64(retail-cost)/float64(retail)*1000)+0.5)) / 10
}

// ── CSV builder ──────────────────────────────────────────────────────────────

// BuildCSV renders report data as a CSV string plus a safe filename.
func (s *Service) BuildCSV(reportType string, data map[string]interface{}) (string, string) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)

	writeRow := func(vals ...interface{}) {
		row := make([]string, len(vals))
		for i, v := range vals {
			row[i] = csvVal(v)
		}
		_ = w.Write(row)
	}

	writeRow(strings.ToUpper(strVal(data["report_name"])))
	writeRow("Store: " + strVal(data["store_name"]))
	writeRow("Period: " + strVal(data["date_from"]) + "  to  " + strVal(data["date_to"]))
	writeRow("Generated: " + strVal(data["generated_at"]))
	writeRow("")

	switch reportType {
	case ReportSales:
		s.writeSalesCSV(w, writeRow, data)
	case ReportInventory:
		s.writeInventoryCSV(w, writeRow, data)
	case ReportTax:
		s.writeTaxCSV(w, writeRow, data)
	case ReportCredit:
		s.writeCreditCSV(w, writeRow, data)
	}
	w.Flush()

	month := strVal(data["date_from"])
	if len(month) >= 7 {
		month = month[:7]
	}
	fname := fmt.Sprintf("%s-%s.csv", strings.ReplaceAll(reportType, "_", "-"), month)
	return buf.String(), fname
}

type rowWriter func(vals ...interface{})

func (s *Service) writeSalesCSV(w *csv.Writer, rw rowWriter, data map[string]interface{}) {
	k := mapVal(data["kpis"])
	rw("KPI SUMMARY")
	rw("Revenue (TZS)", "Profit (TZS)", "Transactions", "Avg Ticket (TZS)", "Margin %")
	rw(k["revenue"], k["profit"], k["transaction_count"], k["avg_ticket"], k["margin_pct"])
	rw("")

	rw("DAILY REVENUE")
	rw("Date", "Label", "Revenue (TZS)", "Profit (TZS)", "Transactions")
	for _, d := range sliceVal(data["daily_trend"]) {
		rw(d["date"], d["label"], d["revenue"], d["profit"], d["transactions"])
	}
	rw("")

	rw("TOP PRODUCTS")
	rw("Product", "SKU", "Units Sold", "Revenue (TZS)", "Cost (TZS)", "Profit (TZS)")
	for _, p := range sliceVal(data["top_products"]) {
		rw(p["product_name"], p["product_sku"], p["qty_sold"], p["revenue"], p["cost"], p["profit"])
	}
	rw("")

	rw("PAYMENT MIX")
	rw("Method", "Amount (TZS)", "Share %")
	for _, m := range sliceVal(data["payment_mix"]) {
		rw(m["method"], m["amount"], m["pct"])
	}
	rw("")

	rw("REVENUE BY CATEGORY")
	rw("Category", "Revenue (TZS)", "Share %", "Trend vs Prev Period %")
	for _, c := range sliceVal(data["category_breakdown"]) {
		rw(c["name"], c["revenue"], c["pct"], c["trend_pct"])
	}
}

func (s *Service) writeInventoryCSV(w *csv.Writer, rw rowWriter, data map[string]interface{}) {
	rw("Valuation date: " + strVal(data["valuation_date"]))
	rw("")

	rw("PRODUCT LIST")
	rw("Category", "Product", "SKU", "Stock", "Unit Cost (TZS)", "Unit Price (TZS)", "Cost Value (TZS)", "Retail Value (TZS)", "Margin %")
	for _, p := range sliceVal(data["products"]) {
		rw(p["category"], p["name"], p["sku"], p["stock"], p["unit_cost"], p["unit_price"], p["cost_value"], p["retail_value"], p["margin_pct"])
	}
	rw("")

	rw("CATEGORY SUBTOTALS")
	rw("Category", "SKUs", "Cost Value (TZS)", "Retail Value (TZS)", "Margin %")
	for _, c := range sliceVal(data["category_subtotals"]) {
		rw(c["category"], c["sku_count"], c["cost_value"], c["retail_value"], c["margin_pct"])
	}
	rw("")

	t := mapVal(data["totals"])
	rw("GRAND TOTAL")
	rw("Total SKUs", "Total Cost Value (TZS)", "Total Retail Value (TZS)", "Margin %")
	rw(t["sku_count"], t["cost_value"], t["retail_value"], t["margin_pct"])
}

func (s *Service) writeTaxCSV(w *csv.Writer, rw rowWriter, data map[string]interface{}) {
	rw("VAT Rate: 18%")
	rw(strVal(data["note"]))
	rw("")

	sum := mapVal(data["summary"])
	rw("VAT SUMMARY")
	rw("Gross Sales (TZS)", "VAT Collected (TZS)", "Net Sales excl. VAT (TZS)", "Refunds (TZS)", "Taxable Days")
	rw(sum["gross_sales"], sum["tax_collected"], sum["net_sales_excl_tax"], sum["refunds"], sum["taxable_transactions"])
	rw("")

	rw("DAILY BREAKDOWN")
	rw("Date", "Label", "Gross Sales (TZS)", "VAT Collected (TZS)", "Net Sales (TZS)", "Refunds (TZS)")
	for _, d := range sliceVal(data["daily"]) {
		rw(d["date"], d["label"], d["gross_sales"], d["tax_collected"], d["net_sales"], d["refunds"])
	}
}

func (s *Service) writeCreditCSV(w *csv.Writer, rw rowWriter, data map[string]interface{}) {
	rw("As of: " + strVal(data["as_of_date"]))
	rw("")

	rw("DEBTOR LIST")
	rw("Customer", "Phone", "Total Balance (TZS)", "Current (TZS)", "1-7d (TZS)", "8-30d (TZS)", "31-60d (TZS)", "60+d (TZS)", "Oldest Due Date", "Open Tabs")
	for _, c := range sliceVal(data["customers"]) {
		rw(c["name"], c["phone"], c["total_balance"], c["current"], c["1_7d"], c["8_30d"], c["31_60d"], c["60plus"], c["oldest_due"], c["tab_count"])
	}
	rw("")

	rw("AGING SUMMARY")
	rw("Bucket", "Customers", "Amount (TZS)")
	for _, b := range sliceVal(data["aging_summary"]) {
		rw(b["label"], b["count"], b["amount"])
	}
	rw("")

	t := mapVal(data["totals"])
	rw("TOTALS")
	rw("Total Outstanding (TZS)", "Total Customers", "Overdue Customers")
	rw(t["total"], t["customer_count"], t["overdue_count"])
}

func csvVal(v interface{}) string {
	switch x := v.(type) {
	case nil:
		return "—"
	case string:
		return x
	case float64:
		return strconv.FormatFloat(x, 'f', -1, 64)
	case int:
		return strconv.Itoa(x)
	case int64:
		return strconv.FormatInt(x, 10)
	default:
		return fmt.Sprintf("%v", x)
	}
}

func strVal(v interface{}) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", v)
}

func mapVal(v interface{}) map[string]interface{} {
	if m, ok := v.(map[string]interface{}); ok {
		return m
	}
	return map[string]interface{}{}
}

func sliceVal(v interface{}) []map[string]interface{} {
	if s, ok := v.([]map[string]interface{}); ok {
		return s
	}
	return nil
}

// ── Export history ───────────────────────────────────────────────────────────

// RecordExport writes an audit row for a generated report.
func (s *Service) RecordExport(ctx context.Context, reportType string, storeID, orgID, userID uuid.UUID, rng Range, format string, size int) error {
	e := &ReportExport{
		StoreID:        storeID,
		OrganisationID: orgID,
		CreatedBy:      userID,
		ReportType:     reportType,
		Name:           ReportTypeNames[reportType],
		PeriodLabel:    rng.Label,
		DateFrom:       rng.From,
		DateTo:         rng.To,
		Format:         format,
		FileSizeBytes:  size,
	}
	return s.repo.CreateExport(ctx, e)
}

// ListExports returns the export history view.
func (s *Service) ListExports(ctx context.Context, storeID uuid.UUID, reportType, format string, limit int) (map[string]interface{}, error) {
	rows, err := s.repo.ListExports(ctx, storeID, reportType, format, limit)
	if err != nil {
		return nil, err
	}
	total, err := s.repo.CountExports(ctx, storeID, reportType, format)
	if err != nil {
		return nil, err
	}
	exports := make([]map[string]interface{}, 0, len(rows))
	for i := range rows {
		exports = append(exports, exportResponse(&rows[i]))
	}
	return map[string]interface{}{"exports": exports, "total": total}, nil
}

// GetExport fetches one export scoped to the store.
func (s *Service) GetExport(ctx context.Context, storeID, id uuid.UUID) (*ReportExport, error) {
	e, err := s.repo.GetExport(ctx, storeID, id)
	if err != nil {
		return nil, apierr.NotFound("Export not found")
	}
	return e, nil
}

func exportResponse(e *ReportExport) map[string]interface{} {
	size := "0 B"
	switch {
	case e.FileSizeBytes < 1024:
		size = fmt.Sprintf("%d B", e.FileSizeBytes)
	case e.FileSizeBytes < 1_048_576:
		size = fmt.Sprintf("%d KB", e.FileSizeBytes/1024)
	default:
		size = fmt.Sprintf("%.1f MB", float64(e.FileSizeBytes)/1_048_576)
	}
	return map[string]interface{}{
		"id":              e.ID,
		"report_type":     e.ReportType,
		"name":            e.Name,
		"period_label":    e.PeriodLabel,
		"date_from":       e.DateFrom.Format("2006-01-02"),
		"date_to":         e.DateTo.Format("2006-01-02"),
		"format":          e.Format,
		"file_size_bytes": e.FileSizeBytes,
		"file_size":       size,
		"is_manual":       e.ScheduledReportID == nil,
		"created_at":      e.CreatedAt,
	}
}

// ── Scheduled reports ────────────────────────────────────────────────────────

// ListScheduled returns the store's schedules.
func (s *Service) ListScheduled(ctx context.Context, storeID uuid.UUID) (map[string]interface{}, error) {
	rows, err := s.repo.ListScheduled(ctx, storeID)
	if err != nil {
		return nil, err
	}
	active, err := s.repo.CountActiveScheduled(ctx, storeID)
	if err != nil {
		return nil, err
	}
	out := make([]map[string]interface{}, 0, len(rows))
	for i := range rows {
		out = append(out, scheduledResponse(&rows[i]))
	}
	return map[string]interface{}{
		"scheduled_reports": out,
		"total":             len(out),
		"active_count":      active,
	}, nil
}

// CreateScheduled validates and stores a new schedule.
func (s *Service) CreateScheduled(ctx context.Context, storeID, orgID, userID uuid.UUID, reportType, name, frequency, preset string, recipients []string) (map[string]interface{}, error) {
	if _, ok := ReportTypeNames[reportType]; !ok {
		return nil, apierr.BadRequest("Unknown report type")
	}
	if !contains(ValidFrequencies, frequency) {
		frequency = FreqDaily
	}
	if !contains(ValidRangePresets, preset) {
		preset = Range30D
	}
	if strings.TrimSpace(name) == "" {
		name = ReportTypeNames[reportType]
	}
	now := time.Now()
	recips := Recipients(cleanEmails(recipients))
	sched := &ScheduledReport{
		StoreID:         storeID,
		OrganisationID:  orgID,
		CreatedBy:       userID,
		ReportType:      reportType,
		Name:            strings.TrimSpace(name),
		Frequency:       frequency,
		DateRangePreset: preset,
		Recipients:      recips,
		IsEnabled:       true,
		NextSendAt:      ptrTime(ComputeNextSend(frequency, now)),
	}
	if err := s.repo.CreateScheduled(ctx, sched); err != nil {
		return nil, err
	}
	return scheduledResponse(sched), nil
}

// UpdateScheduled updates a schedule (rename, frequency, recipients, toggle).
func (s *Service) UpdateScheduled(ctx context.Context, storeID, id uuid.UUID, name, frequency, preset string, recipients []string, isEnabled *bool) (map[string]interface{}, error) {
	sched, err := s.repo.GetScheduled(ctx, storeID, id)
	if err != nil {
		return nil, apierr.NotFound("Scheduled report not found")
	}
	if name != "" {
		sched.Name = strings.TrimSpace(name)
	}
	if frequency != "" {
		if contains(ValidFrequencies, frequency) {
			sched.Frequency = frequency
			sched.NextSendAt = ptrTime(ComputeNextSend(frequency, time.Now()))
		}
	}
	if preset != "" && contains(ValidRangePresets, preset) {
		sched.DateRangePreset = preset
	}
	if recipients != nil {
		sched.Recipients = Recipients(cleanEmails(recipients))
	}
	if isEnabled != nil {
		sched.IsEnabled = *isEnabled
	}
	if err := s.repo.UpdateScheduled(ctx, sched); err != nil {
		return nil, err
	}
	return scheduledResponse(sched), nil
}

// DeleteScheduled removes a schedule.
func (s *Service) DeleteScheduled(ctx context.Context, storeID, id uuid.UUID) error {
	if _, err := s.repo.GetScheduled(ctx, storeID, id); err != nil {
		return apierr.NotFound("Scheduled report not found")
	}
	return s.repo.DeleteScheduled(ctx, id)
}

// SendScheduled immediately generates and emails the report, then advances the
// send timestamps.
func (s *Service) SendScheduled(ctx context.Context, storeID, id uuid.UUID) (map[string]interface{}, error) {
	sched, err := s.repo.GetScheduled(ctx, storeID, id)
	if err != nil {
		return nil, apierr.NotFound("Scheduled report not found")
	}
	return s.dispatch(ctx, sched)
}

func (s *Service) dispatch(ctx context.Context, sched *ScheduledReport) (map[string]interface{}, error) {
	rng := s.presetRange(sched.DateRangePreset)
	data, err := s.GenerateReport(ctx, sched.ReportType, sched.StoreID, sched.CreatedBy, rng)
	if err != nil {
		return nil, err
	}

	sent, failed := 0, 0
	recipients := []string(sched.Recipients)
	if len(recipients) > 0 {
		csvStr, _ := s.BuildCSV(sched.ReportType, data)
		subject := fmt.Sprintf("%s — %s", sched.Name, rng.Label)
		body := "<p>Here is your scheduled report <b>" + sched.Name + "</b> for the period <b>" +
			rng.Label + "</b>.</p><p>The CSV data is included below:</p><pre style=\"font-size:11px\">" +
			htmlEscape(csvStr) + "</pre>"
		for _, to := range recipients {
			if err := s.mailer.Send(to, subject, s.mailer.Layout(subject, body)); err != nil {
				failed++
			} else {
				sent++
			}
		}
	}

	now := time.Now()
	sched.LastSentAt = ptrTime(now)
	sched.NextSendAt = ptrTime(ComputeNextSend(sched.Frequency, now))
	if err := s.repo.UpdateScheduled(ctx, sched); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"sent":         sent,
		"failed":       failed,
		"recipients":   len(recipients),
		"last_sent_at": sched.LastSentAt.Format(time.RFC3339),
	}, nil
}

func (s *Service) presetRange(preset string) Range {
	now := time.Now().In(eatZone)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, eatZone)
	switch preset {
	case Range7D:
		return buildReportRange(today.AddDate(0, 0, -6), today)
	case Range90D:
		return buildReportRange(today.AddDate(0, 0, -89), today)
	case RangeMonth:
		first := time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, eatZone)
		return buildReportRange(first, lastDayOfMonth(first))
	case RangeYTD:
		return buildReportRange(time.Date(today.Year(), 1, 1, 0, 0, 0, 0, eatZone), today)
	default:
		return buildReportRange(today.AddDate(0, 0, -29), today)
	}
}

func scheduledResponse(s *ScheduledReport) map[string]interface{} {
	out := map[string]interface{}{
		"id":                s.ID,
		"store_id":          s.StoreID,
		"report_type":       s.ReportType,
		"name":              s.Name,
		"frequency":         s.Frequency,
		"date_range_preset": s.DateRangePreset,
		"recipients":        []string(s.Recipients),
		"recipient_count":   len(s.Recipients),
		"is_enabled":        s.IsEnabled,
		"created_at":        s.CreatedAt,
	}
	if s.LastSentAt != nil {
		out["last_sent_at"] = s.LastSentAt
	}
	if s.NextSendAt != nil {
		out["next_send_at"] = s.NextSendAt
	}
	return out
}

func cleanEmails(list []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, raw := range list {
		e := strings.ToLower(strings.TrimSpace(raw))
		if e == "" || seen[e] {
			continue
		}
		seen[e] = true
		out = append(out, e)
	}
	return out
}

func contains(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

func ptrTime(t time.Time) *time.Time { return &t }

func htmlEscape(s string) string {
	repl := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;")
	return repl.Replace(s)
}
