// Package staff manages a store's employees, their shifts, POS permissions
// and per-cashier sales performance. It reuses the accounts.User record as
// the staff identity and reads sales stats from the transactions ledger.
package staff

// Shift assignments.
const (
	ShiftMorning = "morning"
	ShiftEvening = "evening"
	ShiftFullDay = "full_day"
	ShiftWeekend = "weekend"
)

// ValidShifts lists accepted shift values.
var ValidShifts = []string{ShiftMorning, ShiftEvening, ShiftFullDay, ShiftWeekend}

// ShiftDisplay maps a shift code to its human-readable range.
func ShiftDisplay(shift string) string {
	switch shift {
	case ShiftMorning:
		return "Morning (7am–2pm)"
	case ShiftEvening:
		return "Evening (2pm–9pm)"
	case ShiftFullDay:
		return "Full day (7am–7pm)"
	case ShiftWeekend:
		return "Weekend (Sat–Sun)"
	}
	return ""
}

// Employment statuses.
const (
	EmpActive   = "active"
	EmpOnLeave  = "on_leave"
	EmpInactive = "inactive"
)

// ValidStatuses lists accepted employment statuses.
var ValidStatuses = []string{EmpActive, EmpOnLeave, EmpInactive}

// StaffStats aggregates a single cashier's performance.
type StaffStats struct {
	SalesToday           int64 `json:"sales_today"`
	TxnsToday            int64 `json:"txns_today"`
	TotalSales           int64 `json:"total_sales"`
	AvgTicket            int64 `json:"avg_ticket"`
	TxnsTotal            int64 `json:"txns_total"`
	SalesThisMonth       int64 `json:"sales_this_month"`
	TxnsThisMonth        int64 `json:"txns_this_month"`
	CreditCollected      int64 `json:"credit_collected"`
	CreditCollectedToday int64 `json:"credit_collected_today"`
}

// StoreKPIs is the store-level staff summary header.
type StoreKPIs struct {
	TotalStaff      int64  `json:"total_staff"`
	Active          int64  `json:"active"`
	OnLeave         int64  `json:"on_leave"`
	Inactive        int64  `json:"inactive"`
	OnShiftToday    int64  `json:"on_shift_today"`
	OnDuty          int64  `json:"on_duty"`
	SalesToday      int64  `json:"sales_today"`
	TotalSalesToday int64  `json:"total_sales_today"`
	TxnsToday       int64  `json:"txns_today"`
	AvgTicketToday  int64  `json:"avg_ticket_today"`
	SalesThisMonth  int64  `json:"sales_this_month"`
	TopCashier      string `json:"top_cashier"`
	TopCashierName  string `json:"top_cashier_name"`
	TopCashierSales int64  `json:"top_cashier_sales"`
}
