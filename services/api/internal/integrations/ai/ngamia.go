// Package ai provides a client for the Ngamia AI gateway
// (https://docs.ngamia.cc). Ngamia is OpenAI-compatible; the chat
// completions endpoint mirrors OpenAI's wire format exactly.
package ai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Client talks to the Ngamia gateway.
type Client struct {
	baseURL string
	apiKey  string
	model   string
	http    *http.Client
}

// New builds a Ngamia client. baseURL should include the /v1 suffix.
func New(baseURL, apiKey, model string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  apiKey,
		model:   model,
		http:    &http.Client{Timeout: 60 * time.Second},
	}
}

// Message is one chat message in OpenAI format.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatRequest mirrors OpenAI's chat completions request.
type ChatRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Temperature *float64  `json:"temperature,omitempty"`
	MaxTokens   *int      `json:"max_tokens,omitempty"`
	Stream      bool      `json:"stream"`
	ID          string    `json:"-"` // internal correlation id (not sent)
}

// Usage reports token counts returned by the gateway.
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ChatCompletion is the OpenAI-shaped non-streaming response.
type ChatCompletion struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index        int    `json:"index"`
		FinishReason string `json:"finish_reason"`
		Message      struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Usage Usage `json:"usage"`
}

// ModelInfo is one row of GET /v1/models.
type ModelInfo struct {
	ID                        string  `json:"id"`
	Model                     string  `json:"model"`
	DisplayName               string  `json:"display_name"`
	Modality                  string  `json:"modality"`
	ContextWindow             int     `json:"context_window"`
	InputPricePer1kCredits    float64 `json:"input_price_per_1k_credits"`
	OutputPricePer1kCredits   float64 `json:"output_price_per_1k_credits"`
}

// GatewayError is the OpenAI-shaped error body.
type GatewayError struct {
	Err struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error"`
}

func (e *GatewayError) Error() string {
	return fmt.Sprintf("ai: [%s] %s", e.Err.Type, e.Err.Message)
}

// Chat performs a non-streaming chat completion.
func (c *Client) Chat(ctx context.Context, req ChatRequest) (*ChatCompletion, error) {
	if c.apiKey == "" {
		return nil, errors.New("ai: NGAMIA_API_KEY not configured")
	}
	if req.Model == "" {
		req.Model = c.model
	}
	if req.Temperature == nil {
		t := 0.7
		req.Temperature = &t
	}

	body, _ := json.Marshal(req)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 400 {
		var ge GatewayError
		if err := json.Unmarshal(data, &ge); err == nil && ge.Err.Message != "" {
			return nil, &ge
		}
		return nil, fmt.Errorf("ai: http_%d %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}

	var out ChatCompletion
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, fmt.Errorf("ai: invalid response: %w", err)
	}
	return &out, nil
}

// StreamChunk is one SSE chunk. Done is true for the [DONE] sentinel.
type StreamChunk struct {
	Delta string
	Done  bool
	Usage Usage
}

// ChatStream streams a chat completion token by token over SSE.
func (c *Client) ChatStream(ctx context.Context, req ChatRequest) (<-chan StreamChunk, error) {
	if c.apiKey == "" {
		return nil, errors.New("ai: NGAMIA_API_KEY not configured")
	}
	req.Stream = true
	if req.Model == "" {
		req.Model = c.model
	}

	body, _ := json.Marshal(req)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		defer resp.Body.Close()
		data, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ai: http_%d %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}

	ch := make(chan StreamChunk)
	go func() {
		defer close(ch)
		defer resp.Body.Close()
		sc := bufio.NewScanner(resp.Body)
		sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		for sc.Scan() {
			line := strings.TrimSpace(sc.Text())
			if !strings.HasPrefix(line, "data:") {
				continue
			}
			payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
			if payload == "[DONE]" {
				ch <- StreamChunk{Done: true}
				return
			}
			var chunk struct {
				Choices []struct {
					Delta struct {
						Content string `json:"content"`
					} `json:"delta"`
				} `json:"choices"`
				Usage Usage `json:"usage"`
			}
			if err := json.Unmarshal([]byte(payload), &chunk); err != nil {
				continue
			}
			ch <- StreamChunk{Delta: chunk.Choices[0].Delta.Content, Usage: chunk.Usage}
		}
	}()
	return ch, nil
}

// ListModels returns the enabled model catalog.
func (c *Client) ListModels(ctx context.Context) ([]ModelInfo, error) {
	if c.apiKey == "" {
		return nil, errors.New("ai: NGAMIA_API_KEY not configured")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/models", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("ai: models http_%d %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}

	// Endpoint may wrap in {data: [...]} or return a bare array.
	var envelope struct {
		Data []ModelInfo `json:"data"`
	}
	if err := json.Unmarshal(data, &envelope); err == nil && envelope.Data != nil {
		return envelope.Data, nil
	}
	var bare []ModelInfo
	if err := json.Unmarshal(data, &bare); err != nil {
		return nil, fmt.Errorf("ai: invalid models response: %w", err)
	}
	return bare, nil
}

// Balance queries the Ngamia credit balance.
func (c *Client) Balance(ctx context.Context) (float64, error) {
	if c.apiKey == "" {
		return 0, errors.New("ai: NGAMIA_API_KEY not configured")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/billing/balance", nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return 0, fmt.Errorf("ai: balance http_%d %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}

	var out struct {
		Balance float64 `json:"balance"`
		Data    *struct {
			Balance float64 `json:"balance"`
		} `json:"data"`
	}
	if err := json.Unmarshal(data, &out); err != nil {
		return 0, err
	}
	if out.Data != nil {
		return out.Data.Balance, nil
	}
	return out.Balance, nil
}

// Ping returns the gateway health (GET /v1/gateway/health).
func (c *Client) Ping(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/gateway/health", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("ai: health http_%d", resp.StatusCode)
	}
	return nil
}
