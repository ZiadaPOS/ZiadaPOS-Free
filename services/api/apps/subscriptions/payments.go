package subscriptions

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/troubleman96/ziadapos/internal/response"
	"gorm.io/gorm"
)

type PaymentOrder struct {
	ID              uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	OrganisationID  uuid.UUID `gorm:"type:uuid;not null;index" json:"organisation_id"`
	SubscriptionID  uuid.UUID `gorm:"type:uuid;not null;index" json:"subscription_id"`
	Reference       string    `gorm:"size:200;uniqueIndex;not null" json:"reference"`
	ProviderEventID string    `gorm:"size:200;uniqueIndex" json:"provider_event_id"`
	Amount          int64     `gorm:"not null" json:"amount"`
	Currency        string    `gorm:"size:10;not null;default:TZS" json:"currency"`
	Phone           string    `gorm:"size:30" json:"phone"`
	Status          string    `gorm:"size:30;not null;default:pending;index" json:"status"`
	CheckoutURL     string    `gorm:"type:text" json:"checkout_url"`
	RawEvent        string    `gorm:"type:text" json:"-"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

func (p *PaymentOrder) BeforeCreate(tx *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	now := time.Now()
	p.CreatedAt = now
	p.UpdatedAt = now
	return nil
}
func (p *PaymentOrder) BeforeUpdate(tx *gorm.DB) error { p.UpdatedAt = time.Now(); return nil }

type snippePaymentRequest struct {
	Amount      int64             `json:"amount"`
	Currency    string            `json:"currency"`
	Phone       string            `json:"phone,omitempty"`
	Description string            `json:"description"`
	CallbackURL string            `json:"webhook_url,omitempty"`
	RedirectURL string            `json:"redirect_url,omitempty"`
	Metadata    map[string]string `json:"metadata"`
}
type snippePaymentResponse struct {
	Data struct {
		Reference   string `json:"reference"`
		CheckoutURL string `json:"checkout_url"`
	} `json:"data"`
	Reference   string `json:"reference"`
	CheckoutURL string `json:"checkout_url"`
}

func (h *Handler) CreatePayment(c *gin.Context) {
	if !h.svc.deps.Cfg.EnableSnippe || h.svc.deps.Cfg.SnippeAPIKey == "" {
		response.BadRequest(c, "Snippe payments are not enabled", nil)
		return
	}
	orgID, ok := h.orgID(c)
	if !ok {
		return
	}
	var in struct {
		PlanID uuid.UUID `json:"plan_id"`
		Phone  string    `json:"phone"`
	}
	if c.ShouldBindJSON(&in) != nil || in.PlanID == uuid.Nil {
		response.BadRequest(c, "plan_id is required", nil)
		return
	}
	plan, err := h.svc.repo.GetPlan(in.PlanID)
	if err != nil || !plan.IsActive {
		response.BadRequest(c, "Active plan not found", nil)
		return
	}
	sub, err := h.svc.repo.LatestSubscription(orgID)
	if err != nil {
		response.NotFound(c, "Subscription not found")
		return
	}
	ref := "ZP-" + strings.ToUpper(strings.ReplaceAll(uuid.NewString(), "-", ""))
	order := &PaymentOrder{OrganisationID: orgID, SubscriptionID: sub.ID, Reference: ref, Amount: plan.TotalPrice(), Currency: "TZS", Phone: in.Phone, Status: "pending"}
	payload := snippePaymentRequest{Amount: order.Amount, Currency: "TZS", Phone: in.Phone, Description: "ZiadaPOS subscription - " + plan.Name, CallbackURL: h.svc.deps.Cfg.SnippeSubscriptionWebhookURL, RedirectURL: h.svc.deps.Cfg.SnippeCheckoutRedirectURL, Metadata: map[string]string{"ziadapos_order_id": ref, "organisation_id": orgID.String(), "subscription_id": sub.ID.String()}}
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequestWithContext(c.Request.Context(), http.MethodPost, strings.TrimRight(h.svc.deps.Cfg.SnippeBaseURL, "/")+"/v1/payments", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+h.svc.deps.Cfg.SnippeAPIKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", ref)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		response.BadRequest(c, "Unable to reach payment provider", nil)
		return
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		response.BadRequest(c, "Payment provider rejected the request", nil)
		return
	}
	var out snippePaymentResponse
	if json.Unmarshal(raw, &out) != nil {
		response.BadRequest(c, "Invalid provider response", nil)
		return
	}
	if out.Data.Reference != "" {
		order.Reference = out.Data.Reference
	}
	order.CheckoutURL = out.Data.CheckoutURL
	if order.CheckoutURL == "" {
		order.CheckoutURL = out.CheckoutURL
	}
	if h.svc.repo.db.Create(order).Error != nil {
		response.BadRequest(c, "Payment could not be recorded", nil)
		return
	}
	response.Created(c, gin.H{"reference": order.Reference, "checkout_url": order.CheckoutURL, "amount": order.Amount, "currency": order.Currency, "status": order.Status}, "Payment initiated")
}

func (h *Handler) SnippeWebhook(c *gin.Context) {
	raw, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.Status(http.StatusBadRequest)
		return
	}
	if !verifySnippe(c.GetHeader("X-Webhook-Timestamp"), c.GetHeader("X-Webhook-Signature"), raw, h.svc.deps.Cfg.SnippeWebhookSecret) {
		c.Status(http.StatusUnauthorized)
		return
	}
	var event struct {
		ID   string `json:"id"`
		Type string `json:"type"`
		Data struct {
			Reference string `json:"reference"`
			Status    string `json:"status"`
		} `json:"data"`
	}
	if json.Unmarshal(raw, &event) != nil || event.Data.Reference == "" {
		c.Status(http.StatusBadRequest)
		return
	}
	var order PaymentOrder
	if h.svc.repo.db.Where("reference = ?", event.Data.Reference).First(&order).Error != nil {
		c.Status(http.StatusOK)
		return
	}
	if order.ProviderEventID == event.ID {
		c.Status(http.StatusOK)
		return
	}
	status := strings.ToLower(event.Data.Status)
	if status == "completed" || strings.Contains(strings.ToLower(event.Type), "completed") {
		status = "completed"
	} else if status == "failed" || status == "voided" || status == "expired" {
		status = "failed"
	} else {
		c.Status(http.StatusOK)
		return
	}
	tx := h.svc.repo.db.Begin()
	order.Status = status
	order.ProviderEventID = event.ID
	order.RawEvent = string(raw)
	if tx.Save(&order).Error != nil {
		tx.Rollback()
		c.Status(http.StatusInternalServerError)
		return
	}
	if status == "completed" {
		var sub Subscription
		if tx.First(&sub, "id = ?", order.SubscriptionID).Error == nil {
			now := time.Now()
			end := now.AddDate(0, 1, 0)
			sub.Status = StatusActive
			sub.StartDate = &now
			sub.EndDate = &end
			sub.AmountPaid = order.Amount
			sub.PaymentReference = order.Reference
			sub.PaymentDate = &now
			if tx.Save(&sub).Error != nil {
				tx.Rollback()
				c.Status(http.StatusInternalServerError)
				return
			}
		}
	}
	if tx.Commit().Error != nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	c.Status(http.StatusOK)
}

func verifySnippe(ts, signature string, body []byte, secret string) bool {
	if secret == "" || ts == "" || signature == "" {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(ts + "."))
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))
	signature = strings.TrimPrefix(signature, "sha256=")
	return hmac.Equal([]byte(expected), []byte(signature))
}
