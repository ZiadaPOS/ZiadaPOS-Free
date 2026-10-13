package notifications

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/uuid"

	"github.com/troubleman96/ziadapos/apps/accounts"
	"github.com/troubleman96/ziadapos/apps/inventory"
	"github.com/troubleman96/ziadapos/internal/events"
	"github.com/troubleman96/ziadapos/internal/uuidx"
)

// Service creates notifications and subscribes to domain events so the API
// surfaces in-app alerts without clients polling business logic.
type Service struct {
	repo *Repo
	bus  *events.Bus
}

// NewService builds a notifications Service.
func NewService(repo *Repo, bus *events.Bus) *Service { return &Service{repo: repo, bus: bus} }

// Create persists a notification for an organisation (and optionally a store).
func (s *Service) Create(ctx context.Context, orgID, storeID uuid.UUID, n *Notification) error {
	n.OrganisationID = orgID
	n.StoreID = storeID
	if n.Category == "" {
		n.Category = CatSystem
	}
	return s.repo.Create(ctx, n)
}

// CreateForUser persists a notification aimed at a specific user (org-scoped).
func (s *Service) CreateForUser(ctx context.Context, orgID, userID uuid.UUID, n *Notification) error {
	n.OrganisationID = orgID
	n.UserID = userID
	if n.Category == "" {
		n.Category = CatSystem
	}
	return s.repo.Create(ctx, n)
}

// ── Event wiring ────────────────────────────────────────────────────────────

// SubscribeHandlers attaches the event listeners that auto-create
// notifications. It must be called once at boot, after Migrate.
func (s *Service) SubscribeHandlers(bus *events.Bus) {
	bus.On("stock.adjusted", s.onStockAdjusted)
	bus.On("transaction.created", s.onTransactionCreated)
	bus.On("credit.payment_received", s.onCreditPayment)
	bus.On("credit.tab_created", s.onCreditTab)
	bus.On("staff.created", s.onStaffCreated)
}

// onStockAdjusted fires a low-stock alert whenever a stock change leaves a
// stocked product at or below its reorder point.
func (s *Service) onStockAdjusted(ctx context.Context, e events.Event) error {
	payload, ok := e.Payload.(map[string]interface{})
	if !ok {
		return nil
	}
	productID := parseUUID(payload["product_id"])
	storeID := parseUUID(payload["store_id"])
	if productID == nil || storeID == nil {
		return nil
	}

	var p inventory.Product
	if err := s.repo.db.WithContext(ctx).First(&p, "id = ?", *productID).Error; err != nil {
		return err
	}
	if !p.IsLowStock() {
		return nil
	}

	orgID := s.orgForStore(ctx, *storeID)
	if orgID == nil {
		return nil
	}

	title := "Low stock: " + p.Name
	body := fmt.Sprintf("Only %d units left — below your %d-unit threshold.", p.Stock, p.ReorderPoint)
	if p.Stock <= 0 {
		title = "Out of stock: " + p.Name
		body = "0 units remaining — mark as unavailable?"
	}
	return s.Create(ctx, *orgID, p.StoreID, &Notification{
		Category: CatStock,
		Title:    title,
		Body:     body,
		Link:     "inventory",
	})
}

// onTransactionCreated alerts staff to large completed sales.
func (s *Service) onTransactionCreated(ctx context.Context, e events.Event) error {
	payload, ok := e.Payload.(map[string]interface{})
	if !ok {
		return nil
	}
	orgID := parseUUID(payload["org_id"])
	storeID := parseUUID(payload["store_id"])
	if orgID == nil || storeID == nil {
		return nil
	}
	total, _ := payload["total"].(int64)
	if total < bigSaleThreshold {
		return nil
	}
	txnNumber, _ := payload["txn_number"].(string)
	if txnNumber == "" {
		txnNumber = "A sale"
	}
	return s.Create(ctx, *orgID, *storeID, &Notification{
		Category: CatSales,
		Title:    "Big sale recorded",
		Body:     fmt.Sprintf("%s · TZS %s.", txnNumber, money(total)),
		Link:     "transactions",
	})
}

// onCreditPayment confirms credit payments on the customer's store.
func (s *Service) onCreditPayment(ctx context.Context, e events.Event) error {
	payload, ok := e.Payload.(map[string]interface{})
	if !ok {
		return nil
	}
	orgID := parseUUID(payload["org_id"])
	storeID := parseUUID(payload["store_id"])
	if orgID == nil || storeID == nil {
		return nil
	}
	amount, _ := payload["amount"].(int64)
	customer, _ := payload["customer_name"].(string)
	if customer == "" {
		customer = "Customer"
	}
	return s.Create(ctx, *orgID, *storeID, &Notification{
		Category: CatCredit,
		Title:    "Payment received",
		Body:     fmt.Sprintf("TZS %s received from %s.", money(amount), customer),
		Link:     "credits",
	})
}

// onCreditTab alerts when a credit tab is opened for a customer.
func (s *Service) onCreditTab(ctx context.Context, e events.Event) error {
	payload, ok := e.Payload.(map[string]interface{})
	if !ok {
		return nil
	}
	orgID := parseUUID(payload["org_id"])
	storeID := parseUUID(payload["store_id"])
	if orgID == nil || storeID == nil {
		return nil
	}
	amount, _ := payload["amount"].(int64)
	customer, _ := payload["customer_name"].(string)
	if customer == "" {
		customer = "A customer"
	}
	return s.Create(ctx, *orgID, *storeID, &Notification{
		Category: CatCredit,
		Title:    "New credit tab",
		Body:     fmt.Sprintf("%s now owes TZS %s on credit.", customer, money(amount)),
		Link:     "credits",
	})
}

// onStaffCreated welcomes new team members with an org-wide system alert.
func (s *Service) onStaffCreated(ctx context.Context, e events.Event) error {
	payload, ok := e.Payload.(map[string]interface{})
	if !ok {
		return nil
	}
	orgID := parseUUID(payload["org_id"])
	if orgID == nil {
		return nil
	}
	name := ""
	if uid := parseUUID(payload["user_id"]); uid != nil {
		var u accounts.User
		if err := s.repo.db.WithContext(ctx).First(&u, "id = ?", *uid).Error; err == nil {
			name = strings.TrimSpace(u.FirstName + " " + u.LastName)
		}
	}
	if name == "" {
		name = "A new team member"
	}
	n := &Notification{
		Category: CatSystem,
		Title:    "New team member added",
		Body:     name + " was added to your team.",
		Link:     "settings",
	}
	if uid := parseUUID(payload["user_id"]); uid != nil {
		return s.CreateForUser(ctx, *orgID, *uid, n)
	}
	return s.Create(ctx, *orgID, uuid.Nil, n)
}

// ── helpers ─────────────────────────────────────────────────────────────────

const bigSaleThreshold int64 = 100000

// orgForStore resolves an organisation from a store id. Used by events that
// only carry a store scope (e.g. stock.adjusted).
func (s *Service) orgForStore(ctx context.Context, storeID uuid.UUID) *uuid.UUID {
	var row struct {
		OrganisationID uuid.UUID `gorm:"column:organisation_id"`
	}
	if err := s.repo.db.WithContext(ctx).Table("stores").Select("organisation_id").
		Where("id = ?", storeID).Scan(&row).Error; err != nil {
		return nil
	}
	if row.OrganisationID == uuid.Nil {
		return nil
	}
	return &row.OrganisationID
}

func parseUUID(v interface{}) *uuid.UUID { return uuidx.Parse(v) }

func money(n int64) string {
	neg := n < 0
	if neg {
		n = -n
	}
	s := strconv.FormatInt(n, 10)
	if len(s) > 3 {
		out := ""
		for i, ch := range s {
			if i > 0 && (len(s)-i)%3 == 0 {
				out += ","
			}
			out += string(ch)
		}
		s = out
	}
	if neg {
		return "-" + s
	}
	return s
}
