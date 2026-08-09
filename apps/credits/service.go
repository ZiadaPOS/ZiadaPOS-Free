package credits

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/troubleman96/ziadapos/apps/customers"
	"github.com/troubleman96/ziadapos/internal/apierr"
	"github.com/troubleman96/ziadapos/internal/cache"
	"github.com/troubleman96/ziadapos/internal/events"
	"github.com/troubleman96/ziadapos/internal/integrations/sms"
)

const defaultDueDays = 30

// Service implements credit business rules.
type Service struct {
	repo  *Repo
	cache *cache.Cache
	bus   *events.Bus
	cust  *customers.Service
	db    *gorm.DB
	sms   *sms.Client
}

// NewService wires a credits Service.
func NewService(repo *Repo, cache *cache.Cache, bus *events.Bus, cust *customers.Service, db *gorm.DB, sms *sms.Client) *Service {
	return &Service{repo: repo, cache: cache, bus: bus, cust: cust, db: db, sms: sms}
}

// CreateTabFromSale creates a credit tab after a credit POS sale and bumps
// the customer's open credit.
func (s *Service) CreateTabFromSale(ctx context.Context, orgID, storeID, customerID uuid.UUID, txnID uuid.UUID, amount int64, cashierID uuid.UUID, till string) error {
	due := time.Now().Add(defaultDueDays * 24 * time.Hour)
	tab := &CreditTab{
		CustomerID:    customerID,
		StoreID:       storeID,
		TransactionID: &txnID,
		Amount:        amount,
		Status:        StatusOpen,
		DueDate:       &due,
		CashierID:     cashierID,
		TillNumber:    till,
	}
	err := s.repo.WithTx(func(tx *gorm.DB) error {
		if err := s.repo.CreateTab(tx, tab); err != nil {
			return err
		}
		balance, err := s.repo.TabBalanceByCustomer(tx, customerID)
		if err != nil {
			return err
		}
		return s.cust.RecomputeCreditBalance(ctx, customerID, balance)
	})
	if err != nil {
		return err
	}
	s.bus.Publish("credit.tab_created", map[string]interface{}{
		"tab_id": tab.ID, "customer_id": customerID, "store_id": storeID,
		"org_id": orgID, "amount": amount,
	})
	return nil
}

// SettleTabForRefund closes the tab linked to a refunded credit transaction
// and drops the customer's open credit.
func (s *Service) SettleTabForRefund(ctx context.Context, txnID uuid.UUID) error {
	tab, err := s.repo.GetTabByTxn(txnID)
	if err != nil {
		return nil // not a credit sale
	}
	if tab.Status == StatusSettled || tab.Status == StatusWrittenOff {
		return nil
	}
	err = s.repo.WithTx(func(tx *gorm.DB) error {
		tab.AmountPaid = tab.Amount
		tab.Status = StatusSettled
		tab.WriteOffReason = "Refunded sale"
		if err := s.repo.UpdateTab(tx, tab); err != nil {
			return err
		}
		balance, err := s.repo.TabBalanceByCustomer(tx, tab.CustomerID)
		if err != nil {
			return err
		}
		return s.cust.RecomputeCreditBalance(ctx, tab.CustomerID, balance)
	})
	return err
}

// RecordPayment distributes a lump-sum payment across the customer's open
// tabs (oldest first) and updates the cached open_credit.
func (s *Service) RecordPayment(ctx context.Context, orgID, storeID, customerID, cashierID uuid.UUID, amount int64, method, reference, note string) (*CreditPayment, error) {
	if amount <= 0 {
		return nil, apierr.BadRequest("Payment amount must be positive").WithField("amount", "greater than zero")
	}
	if !contains(ValidPaymentMethods, method) {
		return nil, apierr.BadRequest("Invalid payment method")
	}
	if _, err := s.cust.Get(ctx, storeID, customerID); err != nil {
		return nil, apierr.NotFound("Customer not found")
	}

	payment := &CreditPayment{
		CustomerID: customerID,
		StoreID:    storeID,
		Amount:     amount,
		Method:     method,
		Reference:  strings.TrimSpace(reference),
		CashierID:  cashierID,
		Note:       note,
	}
	remaining := amount

	err := s.repo.WithTx(func(tx *gorm.DB) error {
		tabs, err := s.repo.OpenTabs(tx, customerID)
		if err != nil {
			return err
		}
		var openTotal int64
		for i := range tabs {
			openTotal += tabs[i].Balance()
		}
		if amount > openTotal {
			return apierr.Conflict("Payment exceeds outstanding credit").WithField("amount", fmt.Sprintf("customer owes %d TZS", openTotal))
		}
		for i := range tabs {
			if remaining <= 0 {
				break
			}
			tab := &tabs[i]
			bal := tab.Balance()
			apply := remaining
			if apply > bal {
				apply = bal
			}
			tab.AmountPaid += apply
			remaining -= apply
			switch {
			case tab.AmountPaid >= tab.Amount:
				tab.Status = StatusSettled
			case tab.AmountPaid > 0:
				tab.Status = StatusPartial
			}
			if err := s.repo.UpdateTab(tx, tab); err != nil {
				return err
			}
		}
		if err := s.repo.CreatePayment(tx, payment); err != nil {
			return err
		}
		balance, err := s.repo.TabBalanceByCustomer(tx, customerID)
		if err != nil {
			return err
		}
		return s.cust.RecomputeCreditBalance(ctx, customerID, balance)
	})
	if err != nil {
		return nil, err
	}

	s.bus.Publish("credit.payment_received", map[string]interface{}{
		"payment_id": payment.ID, "customer_id": customerID, "store_id": storeID,
		"org_id": orgID, "amount": amount,
	})
	return payment, nil
}

// WriteOff closes an open tab and removes its balance.
func (s *Service) WriteOff(ctx context.Context, storeID, tabID uuid.UUID, reason string) (*CreditTab, error) {
	tab, err := s.repo.GetTab(storeID, tabID)
	if err != nil {
		return nil, apierr.NotFound("Credit tab not found")
	}
	if tab.Status == StatusSettled || tab.Status == StatusWrittenOff {
		return nil, apierr.Conflict("Tab is already closed")
	}
	err = s.repo.WithTx(func(tx *gorm.DB) error {
		tab.Status = StatusWrittenOff
		tab.WriteOffReason = strings.TrimSpace(reason)
		if err := s.repo.UpdateTab(tx, tab); err != nil {
			return err
		}
		balance, err := s.repo.TabBalanceByCustomer(tx, tab.CustomerID)
		if err != nil {
			return err
		}
		return s.cust.RecomputeCreditBalance(ctx, tab.CustomerID, balance)
	})
	if err != nil {
		return nil, err
	}
	return tab, nil
}

// ListTabs returns credit tabs with filters.
func (s *Service) ListTabs(ctx context.Context, storeID uuid.UUID, f TabFilter) (map[string]interface{}, error) {
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
	items, total, err := s.repo.ListTabs(f)
	if err != nil {
		return nil, err
	}
	out := make([]map[string]interface{}, 0, len(items))
	for i := range items {
		out = append(out, tabResponse(&items[i]))
	}
	return map[string]interface{}{
		"items": out, "total": total, "page": f.Page, "page_size": f.PageSize,
	}, nil
}

// CustomerProfile returns a customer's full credit picture: tabs, payments,
// messages and notes.
func (s *Service) CustomerProfile(ctx context.Context, storeID, customerID uuid.UUID) (map[string]interface{}, error) {
	cust, err := s.cust.Get(ctx, storeID, customerID)
	if err != nil {
		return nil, apierr.NotFound("Customer not found")
	}
	tabs, err := s.repo.TabsForCustomer(storeID, customerID)
	if err != nil {
		return nil, err
	}
	tabOut := make([]map[string]interface{}, 0, len(tabs))
	for i := range tabs {
		tabOut = append(tabOut, tabResponse(&tabs[i]))
	}
	payments, err := s.repo.ListPayments(storeID, customerID, 50)
	if err != nil {
		return nil, err
	}
	messages, err := s.repo.ListMessages(storeID, customerID, 50)
	if err != nil {
		return nil, err
	}
	notes, err := s.repo.ListNotes(storeID, customerID, 50)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"customer":    cust,
		"open_credit": cust.OpenCredit,
		"tabs":        tabOut,
		"payments":    payments,
		"messages":    messages,
		"notes":       notes,
	}, nil
}

// SendReminder sends an SMS debt reminder and logs it as a credit message.
func (s *Service) SendReminder(ctx context.Context, storeID, customerID uuid.UUID, who string) (*CreditMessage, error) {
	cust, err := s.cust.Get(ctx, storeID, customerID)
	if err != nil {
		return nil, apierr.NotFound("Customer not found")
	}
	if !cust.HasOpenCredit() {
		return nil, apierr.Conflict("Customer has no outstanding credit")
	}
	if cust.Phone == nil {
		return nil, apierr.BadRequest("Customer has no phone number").WithField("phone", "required for SMS reminder")
	}
	body := fmt.Sprintf("Habari %s, karibu ZiadaPOS. Unadaiwa TZS %d kwenye madeni yako. Tafadhali lipa au wasiliana na duka. Asante!", cust.Name, cust.OpenCredit)
	if _, err := s.sms.Send(ctx, *cust.Phone, body, ""); err != nil {
		return nil, err
	}
	msg := &CreditMessage{
		CustomerID: customerID,
		StoreID:    storeID,
		Kind:       KindSMS,
		Direction:  DirectionOut,
		Body:       body,
		Who:        who,
	}
	if err := s.repo.CreateMessage(msg); err != nil {
		return nil, err
	}
	return msg, nil
}

// AddMessage logs a manual outbound message (WhatsApp/call/SMS).
func (s *Service) AddMessage(ctx context.Context, storeID, customerID uuid.UUID, kind, body, who string) (*CreditMessage, error) {
	if body == "" {
		return nil, apierr.BadRequest("Message body is required")
	}
	if kind != KindWhatsApp && kind != KindCall && kind != KindSMS {
		return nil, apierr.BadRequest("Invalid kind").WithField("kind", "must be whatsapp, call or sms")
	}
	msg := &CreditMessage{
		CustomerID: customerID,
		StoreID:    storeID,
		Kind:       kind,
		Direction:  DirectionOut,
		Body:       body,
		Who:        who,
	}
	if err := s.repo.CreateMessage(msg); err != nil {
		return nil, err
	}
	return msg, nil
}

// AddNote writes an internal staff note.
func (s *Service) AddNote(ctx context.Context, storeID, customerID, by uuid.UUID, body string) (*CreditNote, error) {
	if strings.TrimSpace(body) == "" {
		return nil, apierr.BadRequest("Note body is required")
	}
	n := &CreditNote{CustomerID: customerID, StoreID: storeID, Body: body, By: by}
	if err := s.repo.CreateNote(n); err != nil {
		return nil, err
	}
	return n, nil
}

func tabResponse(t *CreditTab) map[string]interface{} {
	return map[string]interface{}{
		"id":             t.ID,
		"customer_id":    t.CustomerID,
		"transaction_id": t.TransactionID,
		"amount":         t.Amount,
		"amount_paid":    t.AmountPaid,
		"balance":        t.Balance(),
		"status":         t.Status,
		"due_date":       t.DueDate,
		"is_overdue":     t.IsOverdue(),
		"till_number":    t.TillNumber,
		"created_at":     t.CreatedAt,
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
