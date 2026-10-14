package ai

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/troubleman96/ziadapos/internal/apierr"
	"github.com/troubleman96/ziadapos/internal/config"
	"github.com/troubleman96/ziadapos/internal/integrations/ai"
	"github.com/troubleman96/ziadapos/internal/security"
)

// Service implements the AI assistant business rules.
type Service struct {
	repo    *Repo
	ai      *ai.Client
	cfg     *config.Config
	secrets *security.SecretBox
}

// NewService wires an ai Service.
func NewService(repo *Repo, aiClient *ai.Client, cfg *config.Config, secrets *security.SecretBox) *Service {
	return &Service{repo: repo, ai: aiClient, cfg: cfg, secrets: secrets}
}

var eatZone = time.FixedZone("EAT", 3*3600)

func dayRangeUTC(t time.Time) (time.Time, time.Time) {
	local := t.In(eatZone)
	day := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, eatZone)
	return day.UTC(), day.AddDate(0, 0, 1).UTC()
}

// ── Conversations ────────────────────────────────────────────────────────────

// List returns active conversations for the sidebar.
func (s *Service) List(ctx context.Context, userID, storeID uuid.UUID) ([]map[string]interface{}, error) {
	convos, err := s.repo.ListConversations(ctx, userID, storeID)
	if err != nil {
		return nil, err
	}
	out := make([]map[string]interface{}, 0, len(convos))
	for i := range convos {
		count, _ := s.repo.CountMessages(ctx, convos[i].ID)
		out = append(out, map[string]interface{}{
			"id":                    convos[i].ID,
			"title":                 convos[i].Title,
			"first_message_preview": convos[i].FirstMessage,
			"message_count":         count,
			"created_at":            convos[i].CreatedAt,
			"updated_at":            convos[i].UpdatedAt,
		})
	}
	return out, nil
}

// Get returns a conversation with its full message thread.
func (s *Service) Get(ctx context.Context, userID, id uuid.UUID) (map[string]interface{}, error) {
	conv, err := s.repo.GetConversation(ctx, userID, id)
	if err != nil {
		return nil, apierr.NotFound("Conversation not found")
	}
	msgs, err := s.repo.ListMessages(ctx, conv.ID, 500)
	if err != nil {
		return nil, err
	}
	out := make([]map[string]interface{}, 0, len(msgs))
	for i := range msgs {
		out = append(out, messageResponse(&msgs[i]))
	}
	return map[string]interface{}{
		"id":                    conv.ID,
		"title":                 conv.Title,
		"first_message_preview": conv.FirstMessage,
		"is_active":             conv.IsActive,
		"message_count":         len(out),
		"messages":              out,
		"created_at":            conv.CreatedAt,
		"updated_at":            conv.UpdatedAt,
	}, nil
}

// Update renames or archives a conversation.
func (s *Service) Update(ctx context.Context, userID, id uuid.UUID, title string, isActive *bool) (map[string]interface{}, error) {
	conv, err := s.repo.GetConversation(ctx, userID, id)
	if err != nil {
		return nil, apierr.NotFound("Conversation not found")
	}
	if title != "" {
		conv.Title = title
	}
	if isActive != nil {
		conv.IsActive = *isActive
	}
	if err := s.repo.UpdateConversation(ctx, conv); err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"id":        conv.ID,
		"title":     conv.Title,
		"is_active": conv.IsActive,
	}, nil
}

// ── Chat ─────────────────────────────────────────────────────────────────────

// StartChat creates a conversation and returns the first assistant reply.
func (s *Service) StartChat(ctx context.Context, orgID, storeID, userID uuid.UUID, message, title string) (map[string]interface{}, error) {
	if strings.TrimSpace(message) == "" {
		return nil, apierr.BadRequest("Message is required")
	}
	conv := &Conversation{
		UserID:       userID,
		StoreID:      storeID,
		OrgID:        orgID,
		Title:        truncate(title, 60),
		FirstMessage: truncate(message, 200),
		IsActive:     true,
	}
	if conv.Title == "" {
		conv.Title = "New conversation"
	}
	if err := s.repo.CreateConversation(ctx, conv); err != nil {
		return nil, err
	}
	msg, err := s.chatTurn(ctx, conv, message)
	if err != nil {
		return nil, err
	}
	if conv.FirstMessage == "" {
		conv.FirstMessage = truncate(message, 200)
	}
	return map[string]interface{}{
		"conversation_id": conv.ID,
		"message":         messageResponse(msg),
	}, nil
}

// ContinueChat appends a message to an existing conversation.
func (s *Service) ContinueChat(ctx context.Context, userID, conversationID uuid.UUID, message string) (map[string]interface{}, error) {
	if strings.TrimSpace(message) == "" {
		return nil, apierr.BadRequest("Message is required")
	}
	conv, err := s.repo.GetConversation(ctx, userID, conversationID)
	if err != nil {
		return nil, apierr.NotFound("Conversation not found")
	}
	msg, err := s.chatTurn(ctx, conv, message)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{"message": messageResponse(msg)}, nil
}

// chatTurn saves the user message, calls the LLM and stores the reply.
func (s *Service) chatTurn(ctx context.Context, conv *Conversation, userMessage string) (*Message, error) {
	orgID := conv.OrgID
	client, tenantKey, model, err := s.clientForOrg(ctx, orgID)
	if err != nil {
		return nil, apierr.Internal("Ngamia credentials could not be opened; check the server encryption key")
	}

	// Use the organisation's own Ngamia balance when configured. Platform credits
	// apply only when the centrally configured provider key is being used.
	credit, err := s.repo.Credit(ctx, orgID)
	if err != nil {
		return nil, err
	}
	if !tenantKey && credit.Remaining() <= 0 {
		fallback := &Message{
			ConversationID: conv.ID,
			Role:           "assistant",
			Content:        creditExhaustedMsg(),
		}
		if err := s.repo.CreateMessage(ctx, fallback); err != nil {
			return nil, err
		}
		_ = s.repo.TouchConversation(ctx, conv.ID)
		return fallback, nil
	}

	if err := s.repo.CreateMessage(ctx, &Message{
		ConversationID: conv.ID,
		Role:           "user",
		Content:        strings.TrimSpace(userMessage),
	}); err != nil {
		return nil, err
	}

	systemPrompt, sources := s.buildSystemPrompt(ctx, conv.StoreID)
	history, err := s.repo.ListMessages(ctx, conv.ID, 10)
	if err != nil {
		return nil, err
	}
	msgs := []ai.Message{{Role: "system", Content: systemPrompt}}
	for i := range history {
		msgs = append(msgs, ai.Message{Role: history[i].Role, Content: history[i].Content})
	}

	completion, err := client.Chat(ctx, ai.ChatRequest{
		Model:    model,
		Messages: msgs,
		Stream:   false,
	})
	if err != nil {
		errorMsg := &Message{
			ConversationID: conv.ID,
			Role:           "assistant",
			Content:        "Samahani, kulikuwa na hitilafu ya kiufundi. Tafadhali jaribu tena.\n\n_Sorry, there was a technical error. Please try again._",
		}
		_ = s.repo.CreateMessage(ctx, errorMsg)
		_ = s.repo.TouchConversation(ctx, conv.ID)
		return errorMsg, apierr.Internal("Ngamia could not complete this request; check the model, API key, and provider balance")
	}

	reply := ""
	if len(completion.Choices) > 0 {
		reply = completion.Choices[0].Message.Content
	}
	assistant := &Message{
		ConversationID: conv.ID,
		Role:           "assistant",
		Content:        reply,
		ModelUsed:      completion.Model,
		PromptTokens:   completion.Usage.PromptTokens,
		Completion:     completion.Usage.CompletionTokens,
		SourcesUsed:    strings.Join(sources, ","),
	}
	if err := s.repo.CreateMessage(ctx, assistant); err != nil {
		return nil, err
	}
	if !tenantKey {
		_ = s.repo.IncrementCredit(ctx, orgID)
	}
	_ = s.repo.TouchConversation(ctx, conv.ID)
	return assistant, nil
}

func (s *Service) clientForOrg(ctx context.Context, orgID uuid.UUID) (*ai.Client, bool, string, error) {
	storedKey, model, err := s.repo.NgamiaSettings(ctx, orgID)
	if err != nil {
		return nil, false, "", err
	}
	if model == "" && s.cfg != nil {
		model = s.cfg.NgamiaModel
	}
	if storedKey != "" {
		if s.secrets == nil {
			return nil, false, "", fmt.Errorf("integration encryption is unavailable")
		}
		apiKey, err := s.secrets.Decrypt(storedKey)
		if err != nil {
			return nil, false, "", fmt.Errorf("integration credential cannot be opened")
		}
		if s.cfg == nil {
			return nil, false, "", fmt.Errorf("Ngamia configuration is unavailable")
		}
		return ai.New(s.cfg.NgamiaBaseURL, apiKey, model), true, model, nil
	}
	if s.ai == nil {
		return nil, false, "", fmt.Errorf("Ngamia client is unavailable")
	}
	return s.ai, false, model, nil
}

// ListModels fetches the Ngamia catalog using this organisation's key when set,
// otherwise the server's configured platform key. Only public model metadata is returned.
func (s *Service) ListModels(ctx context.Context, orgID uuid.UUID) ([]ai.ModelInfo, error) {
	client, _, _, err := s.clientForOrg(ctx, orgID)
	if err != nil || client == nil {
		return nil, apierr.Internal("Ngamia model catalog could not be loaded; verify the server encryption configuration")
	}
	models, err := client.ListModels(ctx)
	if err != nil {
		return nil, apierr.Internal("Ngamia model catalog could not be loaded; verify the API key and provider service")
	}
	return models, nil
}

// buildSystemPrompt assembles only shop-level aggregates and inventory facts.
// Customer names, phone numbers, addresses and transaction identifiers are not
// queried for or sent to the AI provider.
func (s *Service) buildSystemPrompt(ctx context.Context, storeID uuid.UUID) (string, []string) {
	sources := []string{"Sales data"}
	now := time.Now().In(eatZone)
	todayStart, todayEnd := dayRangeUTC(now)
	today, _ := s.repo.SalesAgg(ctx, storeID, todayStart, todayEnd)

	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, eatZone).UTC()
	monthEnd := now.UTC().Add(time.Hour)
	month, _ := s.repo.SalesAgg(ctx, storeID, monthStart, monthEnd)
	monthExpenses, _ := s.repo.ExpenseTotal(ctx, storeID, monthStart, monthEnd)
	monthNet := month.Profit - monthExpenses
	margin := pctOf(month.Profit, month.Revenue)

	lowStock, _ := s.repo.LowStockItems(ctx, storeID, 10)
	if len(lowStock) > 0 {
		sources = append(sources, "Inventory")
	}
	creditCount, totalCredit, _ := s.repo.CreditSummary(ctx, storeID)
	if creditCount > 0 {
		sources = append(sources, "Credits")
	}

	lowStockText := "LOW STOCK: None — all products are above their reorder points."
	if len(lowStock) > 0 {
		lines := make([]string, 0, len(lowStock))
		for _, item := range lowStock {
			lines = append(lines, fmt.Sprintf("- %v (SKU %v): %v remaining; reorder point %v",
				item["name"], item["sku"], item["stock"], item["min_stock"]))
		}
		lowStockText = "LOW STOCK PRODUCTS:\n" + strings.Join(lines, "\n")
	}
	creditText := "OUTSTANDING CREDIT: No open customer credit balances."
	if creditCount > 0 {
		creditText = fmt.Sprintf("OUTSTANDING CREDIT: %d accounts; aggregate balance TZS %d. No customer identities or contact details are included.", creditCount, totalCredit)
	}

	prompt := fmt.Sprintf(`You are Ziada AI — a concise business intelligence assistant for African retail shops.
Today is %s.

Respond in the same language as the user's message (English or Swahili). Be helpful and business-focused. Do not infer or request customer identities from aggregate data.

MINIMIZED SHOP CONTEXT (no store name, customer names, phone numbers, addresses, or transaction identifiers):
TODAY (%s): paid revenue TZS %d; gross profit TZS %d; transaction count %d.
MONTH TO DATE (%s): revenue TZS %d; gross profit TZS %d; margin %.1f%%; transaction count %d; expenses TZS %d; net TZS %d.

%s
%s

Ground numeric answers only in the data above; do not invent figures. Use concise Markdown, with short paragraphs or bullets.`,
		now.Format("Monday, January 02, 2006"),
		now.Format("Monday, January 02, 2006"), today.Revenue, today.Profit, today.Count,
		now.Format("January 2006"), month.Revenue, month.Profit, margin, month.Count, monthExpenses, monthNet,
		lowStockText, creditText)
	return prompt, sources
}

// ── Suggestions & usage ──────────────────────────────────────────────────────

// Suggestions returns contextual prompt chips plus the credit meter.
func (s *Service) Suggestions(ctx context.Context, storeID, orgID uuid.UUID) (map[string]interface{}, error) {
	credit, err := s.repo.Credit(ctx, orgID)
	if err != nil {
		return nil, err
	}
	suggestions := make([]map[string]interface{}, 0, 6)

	lowStock, _ := s.repo.LowStockItems(ctx, storeID, 10)
	if len(lowStock) > 0 {
		suggestions = append(suggestions, map[string]interface{}{
			"icon":  "📦",
			"label": fmt.Sprintf("Bidhaa %d zinahitaji kununuliwa — nionyeshe", len(lowStock)),
			"key":   "stock",
		})
	}
	creditCustomers, _, _ := s.repo.CustomersWithCredit(ctx, storeID, 10)
	if len(creditCustomers) > 0 {
		suggestions = append(suggestions, map[string]interface{}{
			"icon":  "💳",
			"label": fmt.Sprintf("Hali ya madeni — wateja %d wana deni", len(creditCustomers)),
			"key":   "credit",
		})
	}
	suggestions = append(suggestions,
		map[string]interface{}{"icon": "📊", "label": "Nionyeshe uchambuzi wa mauzo wiki hii", "key": "sales"},
		map[string]interface{}{"icon": "💰", "label": "Uchambuzi wa faida na margin mwezi huu", "key": "profit"},
		map[string]interface{}{"icon": "📈", "label": "Toa ripoti fupi ya leo kwa WhatsApp", "key": "default"},
		map[string]interface{}{"icon": "🔮", "label": "Tabiri mauzo ya wiki ijayo", "key": "default"},
	)
	if len(suggestions) > 6 {
		suggestions = suggestions[:6]
	}

	return map[string]interface{}{
		"suggestions": suggestions,
		"ai_credits": map[string]interface{}{
			"remaining": credit.Remaining(),
			"used":      credit.Used,
			"allocated": credit.Allocated,
			"pct_used":  pctOf(int64(credit.Used), int64(credit.Allocated)),
		},
	}, nil
}

// Usage returns the real usage log and credit status.
func (s *Service) Usage(ctx context.Context, storeID, orgID uuid.UUID) (map[string]interface{}, error) {
	now := time.Now()
	credit, err := s.repo.Credit(ctx, orgID)
	if err != nil {
		return nil, err
	}
	totalCount, totalProm, totalComp, err := s.repo.UsageTotals(ctx, storeID)
	if err != nil {
		return nil, err
	}
	recent, err := s.repo.RecentMessages(ctx, storeID, 50)
	if err != nil {
		return nil, err
	}
	// This month: recompute totals scoped to the current calendar month.
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	var monthCount, monthProm, monthComp int64
	for _, r := range recent {
		if created, ok := r["created_at"].(time.Time); ok && !created.Before(monthStart) {
			monthCount++
			monthProm += int64(r["prompt_tokens"].(int))
			monthComp += int64(r["completion_tokens"].(int))
		}
	}
	storedKey, _, settingsErr := s.repo.NgamiaSettings(ctx, orgID)
	if settingsErr != nil {
		return nil, settingsErr
	}
	ngamiaConfigured := storedKey != ""
	if s.cfg != nil && s.cfg.NgamiaAPIKey != "" {
		ngamiaConfigured = true
	}
	return map[string]interface{}{
		"totals": map[string]interface{}{
			"messages":          totalCount,
			"prompt_tokens":     totalProm,
			"completion_tokens": totalComp,
		},
		"this_month": map[string]interface{}{
			"messages":          monthCount,
			"prompt_tokens":     monthProm,
			"completion_tokens": monthComp,
		},
		"recent": recent,
		"ai_credits": map[string]interface{}{
			"remaining": credit.Remaining(),
			"used":      credit.Used,
			"allocated": credit.Allocated,
			"pct_used":  pctOf(int64(credit.Used), int64(credit.Allocated)),
		},
		"ngamia_configured": ngamiaConfigured,
	}, nil
}

// ── helpers ──────────────────────────────────────────────────────────────────

func messageResponse(m *Message) map[string]interface{} {
	return map[string]interface{}{
		"id":                m.ID,
		"role":              m.Role,
		"content":           m.Content,
		"model_used":        m.ModelUsed,
		"prompt_tokens":     m.PromptTokens,
		"completion_tokens": m.Completion,
		"sources_used":      splitSources(m.SourcesUsed),
		"created_at":        m.CreatedAt,
	}
}

func splitSources(s string) []string {
	if strings.TrimSpace(s) == "" {
		return []string{}
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func hasSource(sources []string, name string) bool {
	for _, s := range sources {
		if s == name {
			return true
		}
	}
	return false
}

func truncate(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return strings.TrimSpace(s[:n])
}

func pctOf(part, whole int64) float64 {
	if whole == 0 {
		return 0
	}
	return float64(part) / float64(whole) * 100
}

func creditExhaustedMsg() string {
	return "Samahani — AI credits za shirika lako zimekwisha kwa mwezi huu. Unaweza kubandika Ngamia API key yako mwenyewe (Settings → Integrations) ili uendelee kutumia bila kikomo, au wasiliana na msimamizi wako.\n\n" +
		"_Sorry — your organisation's free AI credits are exhausted for this month. Paste your own Ngamia API key in Settings → Integrations to keep going unlimited on your own balance, or contact your admin._"
}
