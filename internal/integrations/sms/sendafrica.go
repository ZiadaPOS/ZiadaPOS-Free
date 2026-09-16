// Package sms provides a client for the SendAfrica SMS API
// (https://docs.sendafrica.online). Auth is the X-API-Key header only.
package sms

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/troubleman96/ziadapos/internal/config"
)

// Client sends SMS through SendAfrica.
type Client struct {
	baseURL  string
	apiKey   string
	senderID string
	http     *http.Client
}

// New builds a SendAfrica client. When no API key is configured the client
// runs in dry-run mode (message logged, nothing sent).
func New(cfg *config.Config) *Client {
	return &Client{
		baseURL:  strings.TrimRight(cfg.SendAfricaBaseURL, "/"),
		apiKey:   cfg.SendAfricaAPIKey,
		senderID: cfg.SendAfricaSenderID,
		http:     &http.Client{Timeout: 10 * time.Second},
	}
}

// Error classifies SMS failures.
type Error struct {
	Code    string
	Message string
	Status  int
}

func (e *Error) Error() string {
	return fmt.Sprintf("sms: %s (%d) %s", e.Code, e.Status, e.Message)
}

// Response is the successful SendAfrica response envelope.
type Response struct {
	Success   bool `json:"success"`
	Data      struct {
		MessageID   string `json:"message_id"`
		Status      string `json:"status"`
		Cost        string `json:"cost"`
		CreditsUsed int    `json:"credits_used"`
	} `json:"data"`
	RequestID string `json:"request_id"`
	Error     string `json:"error,omitempty"`
	Errors    struct {
		To      []string `json:"to,omitempty"`
		Message []string `json:"message,omitempty"`
		Detail  string   `json:"detail,omitempty"`
	} `json:"errors,omitempty"`
}

// Send delivers a single SMS. `from` is optional; empty uses the account's
// default sender ID ("ZIADA POS").
func (c *Client) Send(ctx context.Context, to, message, from string) (*Response, error) {
	if c.apiKey == "" {
		return nil, &Error{Code: "not_configured", Message: "SendAfrica API key not configured"}
	}

	normalized, err := NormalizePhone(to)
	if err != nil {
		return nil, err
	}

	payload := map[string]string{
		"to":      normalized,
		"message": message,
	}
	if from != "" {
		payload["from"] = from
	} else if c.senderID != "" {
		payload["from"] = c.senderID
	}

	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/sms/", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-API-Key", c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, &Error{Code: "network_error", Message: err.Error()}
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)

	var out Response
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, &Error{Code: "server_error", Message: string(data)}
	}

	if resp.StatusCode >= 400 {
		msg := out.Error
		if msg == "" {
			msg = strings.TrimSpace(string(data))
		}
		return nil, &Error{Code: fmt.Sprintf("http_%d", resp.StatusCode), Message: msg, Status: resp.StatusCode}
	}
	if !out.Success {
		return nil, &Error{Code: "server_error", Message: string(data)}
	}
	return &out, nil
}

// Balance queries the account's remaining SMS credits.
func (c *Client) Balance(ctx context.Context) (float64, error) {
	if c.apiKey == "" {
		return 0, &Error{Code: "not_configured", Message: "SendAfrica API key not configured"}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/v1/credits/balance", nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("X-API-Key", c.apiKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	var out struct {
		Success bool `json:"success"`
		Data    struct {
			Balance float64 `json:"balance"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return 0, err
	}
	if resp.StatusCode >= 400 {
		return 0, &Error{Code: fmt.Sprintf("http_%d", resp.StatusCode), Message: "balance request failed", Status: resp.StatusCode}
	}
	return out.Data.Balance, nil
}

// NormalizePhone converts Tanzanian phone numbers to the 10-digit form the
// SendAfrica API expects (e.g. "+255712345678", "255712345678" or
// "0712345678" -> "0712345678"). Valid local prefixes are 071-079.
func NormalizePhone(phone string) (string, error) {
	digits := ""
	for _, r := range phone {
		if r >= '0' && r <= '9' {
			digits += string(r)
		}
	}
	switch {
	case len(digits) == 13 && strings.HasPrefix(digits, "255"):
		return "0" + digits[3:], nil
	case len(digits) == 12 && strings.HasPrefix(digits, "255"):
		return "0" + digits[3:], nil
	case len(digits) == 10 && strings.HasPrefix(digits, "0"):
		return digits, nil
	case len(digits) == 9 && strings.HasPrefix(digits, "7"):
		return "0" + digits, nil
	}
	return "", errors.New("sms: invalid_phone")
}
