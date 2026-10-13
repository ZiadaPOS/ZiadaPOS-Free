package ai

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Repo provides AI data access.
type Repo struct {
	db *gorm.DB
}

// NewRepo builds a Repo.
func NewRepo(db *gorm.DB) *Repo { return &Repo{db: db} }

// ── Conversations ────────────────────────────────────────────────────────────

// ListConversations returns active conversations for a user + store.
func (r *Repo) ListConversations(ctx context.Context, userID, storeID uuid.UUID) ([]Conversation, error) {
	var convos []Conversation
	err := r.db.
		Where("user_id = ? AND store_id = ? AND is_active = true", userID, storeID).
		Order("updated_at desc").
		Find(&convos).Error
	return convos, err
}

// GetConversation returns a conversation scoped to the user.
func (r *Repo) GetConversation(ctx context.Context, userID, id uuid.UUID) (*Conversation, error) {
	var c Conversation
	err := r.db.Where("id = ? AND user_id = ?", id, userID).First(&c).Error
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// CreateConversation inserts a new conversation.
func (r *Repo) CreateConversation(ctx context.Context, c *Conversation) error {
	return r.db.Create(c).Error
}

// UpdateConversation persists title/is_active.
func (r *Repo) UpdateConversation(ctx context.Context, c *Conversation) error {
	return r.db.Model(&Conversation{}).
		Where("id = ? AND user_id = ?", c.ID, c.UserID).
		Updates(map[string]interface{}{
			"title": c.Title, "is_active": c.IsActive, "updated_at": time.Now(),
		}).Error
}

// TouchConversation bumps updated_at after a chat turn.
func (r *Repo) TouchConversation(ctx context.Context, id uuid.UUID) error {
	return r.db.Model(&Conversation{}).
		Where("id = ?", id).
		Update("updated_at", time.Now()).Error
}

// ── Messages ─────────────────────────────────────────────────────────────────

// CreateMessage inserts one message.
func (r *Repo) CreateMessage(ctx context.Context, m *Message) error {
	return r.db.Create(m).Error
}

// ListMessages returns the last n user/assistant messages in order.
func (r *Repo) ListMessages(ctx context.Context, conversationID uuid.UUID, n int) ([]Message, error) {
	var msgs []Message
	err := r.db.
		Where("conversation_id = ? AND role IN ?", conversationID, []string{"user", "assistant"}).
		Order("created_at desc").
		Limit(n).
		Find(&msgs).Error
	if err != nil {
		return nil, err
	}
	// reverse to chronological order
	for i, j := 0, len(msgs)-1; i < j; i, j = i+1, j-1 {
		msgs[i], msgs[j] = msgs[j], msgs[i]
	}
	return msgs, nil
}

// CountMessages returns the message count for a conversation.
func (r *Repo) CountMessages(ctx context.Context, conversationID uuid.UUID) (int64, error) {
	var n int64
	err := r.db.Model(&Message{}).
		Where("conversation_id = ?", conversationID).
		Count(&n).Error
	return n, err
}

// UsageTotals returns aggregate message/token counts for assistant replies.
func (r *Repo) UsageTotals(ctx context.Context, storeID uuid.UUID) (int64, int64, int64, error) {
	var a struct {
		Count int64 `gorm:"column:count"`
		Prom  int64 `gorm:"column:prom"`
		Comp  int64 `gorm:"column:comp"`
	}
	err := r.db.Table("ai_messages AS m").
		Select(`
			COUNT(*) AS count,
			COALESCE(SUM(m.prompt_tokens),0) AS prom,
			COALESCE(SUM(m.completion),0) AS comp`).
		Joins("JOIN ai_conversations AS c ON c.id = m.conversation_id").
		Where("c.store_id = ? AND m.role = 'assistant'", storeID).
		Scan(&a).Error
	return a.Count, a.Prom, a.Comp, err
}

// RecentMessages returns the last n assistant replies with conversation titles.
func (r *Repo) RecentMessages(ctx context.Context, storeID uuid.UUID, n int) ([]map[string]interface{}, error) {
	var rows []struct {
		ID                uuid.UUID `gorm:"column:id"`
		ConversationTitle string    `gorm:"column:conversation_title"`
		ModelUsed         string    `gorm:"column:model_used"`
		PromptTokens      int       `gorm:"column:prompt_tokens"`
		Completion        int       `gorm:"column:completion"`
		CreatedAt         time.Time `gorm:"column:created_at"`
	}
	err := r.db.Table("ai_messages AS m").
		Select("m.id, c.title AS conversation_title, m.model_used, m.prompt_tokens, m.completion, m.created_at").
		Joins("JOIN ai_conversations AS c ON c.id = m.conversation_id").
		Where("c.store_id = ? AND m.role = 'assistant'", storeID).
		Order("m.created_at desc").
		Limit(n).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]map[string]interface{}, 0, len(rows))
	for _, row := range rows {
		out = append(out, map[string]interface{}{
			"id":                 row.ID,
			"conversation_title": row.ConversationTitle,
			"model_used":         row.ModelUsed,
			"prompt_tokens":      row.PromptTokens,
			"completion_tokens":  row.Completion,
			"created_at":         row.CreatedAt,
		})
	}
	return out, nil
}

// ── Store context ────────────────────────────────────────────────────────────

type dayAgg struct {
	Revenue int64 `gorm:"column:revenue"`
	Profit  int64 `gorm:"column:profit"`
	Count   int64 `gorm:"column:count"`
}

// SalesAgg returns paid-sales aggregates between two timestamps.
func (r *Repo) SalesAgg(ctx context.Context, storeID uuid.UUID, from, to time.Time) (dayAgg, error) {
	var a dayAgg
	err := r.db.Table("transactions").
		Select(`
			COALESCE(SUM(total),0) AS revenue,
			COALESCE(SUM(profit),0) AS profit,
			COUNT(*) AS count`).
		Where("store_id = ? AND status = 'paid' AND created_at >= ? AND created_at < ?", storeID, from, to).
		Scan(&a).Error
	return a, err
}

// ExpenseTotal returns total expenses in range.
func (r *Repo) ExpenseTotal(ctx context.Context, storeID uuid.UUID, from, to time.Time) (int64, error) {
	var v int64
	err := r.db.Table("expenses").
		Select("COALESCE(SUM(amount),0)").
		Where("store_id = ? AND created_at >= ? AND created_at < ?", storeID, from, to).
		Scan(&v).Error
	return v, err
}

// LowStockItems returns active products at or below reorder point.
func (r *Repo) LowStockItems(ctx context.Context, storeID uuid.UUID, limit int) ([]map[string]interface{}, error) {
	var rows []map[string]interface{}
	err := r.db.Table("products").
		Select("name, COALESCE(sku,'') AS sku, stock, reorder_point AS min_stock").
		Where("store_id = ? AND is_active = true AND is_service = false AND stock <= reorder_point", storeID).
		Order("stock asc").
		Limit(limit).
		Scan(&rows).Error
	return rows, err
}

// TopProducts returns best sellers by revenue in the last 7 days.
func (r *Repo) TopProducts(ctx context.Context, storeID uuid.UUID, from, to time.Time, limit int) ([]map[string]interface{}, error) {
	var rows []map[string]interface{}
	err := r.db.Table("transaction_lines AS l").
		Select(`
			l.product_name AS name,
			MAX(l.unit_price) AS price,
			SUM(l.qty) AS weekly_sold,
			0 AS stock`).
		Joins("JOIN transactions AS t ON t.id = l.transaction_id").
		Where("t.store_id = ? AND t.status = 'paid' AND t.created_at >= ? AND t.created_at < ?", storeID, from, to).
		Group("l.product_name").
		Order("weekly_sold desc").
		Limit(limit).
		Scan(&rows).Error
	return rows, err
}

// CustomersWithCredit returns customers carrying open balances.
func (r *Repo) CustomersWithCredit(ctx context.Context, storeID uuid.UUID, limit int) ([]map[string]interface{}, int64, error) {
	var rows []map[string]interface{}
	err := r.db.Table("customers").
		Select("name, COALESCE(phone,'') AS phone, open_credit").
		Where("store_id = ? AND is_active = true AND open_credit > 0", storeID).
		Order("open_credit desc").
		Limit(limit).
		Scan(&rows).Error
	if err != nil {
		return nil, 0, err
	}
	var total int64
	err = r.db.Table("customers").
		Select("COALESCE(SUM(open_credit),0)").
		Where("store_id = ? AND is_active = true AND open_credit > 0", storeID).
		Scan(&total).Error
	return rows, total, err
}

// RecentTxns returns the latest sales for context.
func (r *Repo) RecentTxns(ctx context.Context, storeID uuid.UUID, limit int) ([]map[string]interface{}, error) {
	var rows []map[string]interface{}
	err := r.db.Table("transactions").
		Select("txn_number, total, payment_method, customer_name, status").
		Where("store_id = ?", storeID).
		Order("created_at desc").
		Limit(limit).
		Scan(&rows).Error
	return rows, err
}

// StoreInfo returns the store name and area.
func (r *Repo) StoreInfo(ctx context.Context, storeID uuid.UUID) (name, area string, err error) {
	var s struct {
		Name string `gorm:"column:name"`
		Area string `gorm:"column:area"`
	}
	err = r.db.Table("stores").
		Select("name, COALESCE(area,'') AS area").
		Where("id = ?", storeID).
		Scan(&s).Error
	return s.Name, s.Area, err
}

// ── AI credits ───────────────────────────────────────────────────────────────

// Credit returns the current-month credit row, creating it from the org
// allowance if it does not exist yet.
func (r *Repo) Credit(ctx context.Context, orgID uuid.UUID) (*AICredit, error) {
	now := time.Now()
	var c AICredit
	err := r.db.Where("organisation_id = ? AND year = ? AND month = ?", orgID, now.Year(), int(now.Month())).
		First(&c).Error
	if err == nil {
		return &c, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	alloc := 0
	var org struct {
		AICreditsMonthly int `gorm:"column:ai_credits_monthly"`
	}
	if rerr := r.db.Table("organisations").
		Select("ai_credits_monthly").
		Where("id = ?", orgID).
		Scan(&org).Error; rerr == nil {
		alloc = org.AICreditsMonthly
	}
	if alloc <= 0 {
		alloc = 5000
	}
	c = AICredit{
		ID:             uuid.New(),
		OrganisationID: orgID,
		Year:           now.Year(),
		Month:          int(now.Month()),
		Used:           0,
		Allocated:      alloc,
	}
	if err := r.db.Create(&c).Error; err != nil {
		return nil, err
	}
	return &c, nil
}

// IncrementCredit adds one used credit for the org's current month.
func (r *Repo) IncrementCredit(ctx context.Context, orgID uuid.UUID) error {
	return r.db.Model(&AICredit{}).
		Where("organisation_id = ? AND year = ? AND month = ?", orgID, time.Now().Year(), int(time.Now().Month())).
		UpdateColumn("used", gorm.Expr("used + 1")).Error
}
