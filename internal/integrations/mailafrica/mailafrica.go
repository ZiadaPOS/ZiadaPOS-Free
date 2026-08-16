// Package mailafrica provides a client for the MailAfrica email API
// (https://docs.mailafrica.online). Auth is the X-API-Key header only.
// Outbound sends are debited from the account wallet (5 TZS platform sender,
// 10 TZS custom domain, per recipient).
package mailafrica

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/troubleman96/ziadapos/internal/config"
)

// Client sends transactional email through MailAfrica.
type Client struct {
	baseURL     string
	apiKey      string
	fromDomain  string
	fromAddress string
	fromName    string
	http        *http.Client

	mu         sync.Mutex
	resolvedID int
}

// New builds a MailAfrica client. When no API key is configured the client
// reports not_configured on every send. MAILAFRICA_FROM_DOMAIN is the user's
// sending domain name (e.g. ziadapos.com); its numeric ID is resolved from
// the MailAfrica account on the first send and cached.
func New(cfg *config.Config) *Client {
	return &Client{
		baseURL:     strings.TrimRight(cfg.MailAfricaBaseURL, "/"),
		apiKey:      cfg.MailAfricaAPIKey,
		fromDomain:  strings.ToLower(strings.TrimSpace(cfg.MailAfricaFromDomain)),
		fromAddress: cfg.MailAfricaFromAddress,
		fromName:    cfg.MailAfricaFromName,
		http:        &http.Client{Timeout: 10 * time.Second},
	}
}

// Error classifies MailAfrica failures.
type Error struct {
	Code    string
	Message string
	Status  int
}

func (e *Error) Error() string {
	return fmt.Sprintf("mailafrica: %s (%d) %s", e.Code, e.Status, e.Message)
}

// Envelope is the MailAfrica response envelope.
type Envelope struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    struct {
		ID            int64    `json:"id"`
		FromAddress   string   `json:"from_address"`
		ToAddresses   []string `json:"to_addresses"`
		Subject       string   `json:"subject"`
		Status        string   `json:"status"`
		ProviderMsgID string   `json:"provider_message_id"`
		AmountTZS     int64    `json:"amount_tzs"`
		CreatedAt     string   `json:"created_at"`
	} `json:"data"`
	Errors []struct {
		Field   string `json:"field"`
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"errors"`
	RequestID string `json:"request_id"`
}

// SendRequest is the outbound email payload.
type SendRequest struct {
	To           []string `json:"to"`
	Subject      string   `json:"subject"`
	HTMLBody     string   `json:"html_body,omitempty"`
	TextBody     string   `json:"text_body,omitempty"`
	FromName     string   `json:"from_name,omitempty"`
	FromDomainID *int     `json:"from_domain_id,omitempty"`
	FromAddress  string   `json:"from_address,omitempty"`
}

// Send delivers a transactional email. When a from_domain is configured the
// from_address must match one of that domain's sender addresses; otherwise the
// mail goes out from the MailAfrica platform sender.
func (c *Client) Send(ctx context.Context, to, subject, html string) error {
	if c.apiKey == "" {
		return &Error{Code: "not_configured", Message: "MailAfrica API key not configured"}
	}

	req := SendRequest{
		To:       []string{to},
		Subject:  subject,
		HTMLBody: html,
	}
	if c.fromName != "" {
		req.FromName = c.fromName
	}
	if c.fromDomain != "" {
		id, err := c.resolveFromDomainID(ctx)
		if err != nil {
			return err
		}
		req.FromDomainID = &id
		if c.fromAddress != "" {
			req.FromAddress = c.fromAddress
		}
		if err := c.post(ctx, "/api/outbound/emails", req); err != nil {
			if !isUpstreamError(err) {
				return err
			}
			// The mail provider refused the own-domain send (relay not
			// authorised, or SMTP credential threshold reached). Fall back to
			// the MailAfrica platform sender so transactional mail still goes
			// out. The own-domain path is always tried first, so this reverts
			// automatically once the provider is sorted.
			log.Printf("mailafrica: own-domain send failed (%v); retrying via platform sender", err)
			req.FromDomainID = nil
			req.FromAddress = ""
			if fbErr := c.post(ctx, "/api/outbound/emails", req); fbErr != nil {
				return &Error{Code: "send_failed", Status: 502, Message: fmt.Sprintf("own-domain send failed (%v) and platform-sender fallback also failed (%v)", err, fbErr)}
			}
		}
		return nil
	}

	return c.post(ctx, "/api/outbound/emails", req)
}

// isUpstreamError reports whether err comes from MailAfrica's mail provider
// (relay refusal, credential threshold, ...) as opposed to a request
// validation or auth failure. Only provider errors benefit from the
// platform-sender fallback.
func isUpstreamError(err error) bool {
	e, ok := err.(*Error)
	if !ok {
		return false
	}
	return e.Status >= 500 || e.Code == "PROVIDER_ERROR"
}

// resolveFromDomainID maps the configured sending domain name to its numeric
// ID on the MailAfrica account, caching the result after the first success.
// Failures are not cached so a later send retries once the domain is added or
// verified.
func (c *Client) resolveFromDomainID(ctx context.Context) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.resolvedID > 0 {
		return c.resolvedID, nil
	}

	var out struct {
		Success bool `json:"success"`
		Data    []struct {
			ID     int64  `json:"id"`
			Domain string `json:"domain"`
			Status string `json:"status"`
		} `json:"data"`
	}
	if err := c.get(ctx, "/api/domains", &out); err != nil {
		return 0, &Error{Code: "domain_lookup_failed", Message: fmt.Sprintf("resolve from domain %q: %v", c.fromDomain, err)}
	}

	want := c.fromDomain
	for _, d := range out.Data {
		if strings.EqualFold(d.Domain, want) {
			if d.Status != "verified" {
				return 0, &Error{Code: "domain_not_verified", Message: fmt.Sprintf("sending domain %q is not verified (status %q) - publish the DKIM/CNAME records and click Verify in the MailAfrica dashboard", d.Domain, d.Status)}
			}
			c.resolvedID = int(d.ID)
			return c.resolvedID, nil
		}
	}
	return 0, &Error{Code: "domain_not_found", Message: fmt.Sprintf("sending domain %q not found on this MailAfrica account - add it under Domains -> Sending and verify it", c.fromDomain)}
}

// Balance queries the wallet balance in TZS.
func (c *Client) Balance(ctx context.Context) (int64, error) {
	if c.apiKey == "" {
		return 0, &Error{Code: "not_configured", Message: "MailAfrica API key not configured"}
	}
	var out struct {
		Success bool `json:"success"`
		Data    struct {
			BalanceTZS int64 `json:"balance_tzs"`
		} `json:"data"`
	}
	if err := c.get(ctx, "/api/billing/balance", &out); err != nil {
		return 0, err
	}
	return out.Data.BalanceTZS, nil
}

func (c *Client) post(ctx context.Context, path string, payload interface{}) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("X-API-Key", c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return &Error{Code: "network_error", Message: err.Error()}
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)

	var out Envelope
	if err := json.Unmarshal(data, &out); err != nil {
		return &Error{Code: "server_error", Message: strings.TrimSpace(string(data))}
	}
	if resp.StatusCode >= 400 || !out.Success {
		return envelopeError(&out, resp.StatusCode)
	}
	return nil
}

func (c *Client) get(ctx context.Context, path string, out interface{}) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("X-API-Key", c.apiKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return &Error{Code: "network_error", Message: err.Error()}
	}
	defer resp.Body.Close()

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return &Error{Code: "server_error", Message: err.Error()}
	}
	if resp.StatusCode >= 400 {
		return &Error{Code: fmt.Sprintf("http_%d", resp.StatusCode), Message: "request failed", Status: resp.StatusCode}
	}
	return nil
}

func envelopeError(e *Envelope, status int) *Error {
	code := e.Message
	msg := ""
	for _, err := range e.Errors {
		if err.Code != "" {
			code = err.Code
			msg = err.Message
			break
		}
	}
	if code == "" {
		code = fmt.Sprintf("http_%d", status)
	}
	if msg == "" {
		msg = e.Message
	}
	return &Error{Code: code, Message: msg, Status: status}
}
