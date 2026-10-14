package sms

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSendUsesAccountKeySenderAndOfficialRequestShape(t *testing.T) {
	const apiKey = "test-key-not-a-real-secret"
	var got struct {
		To      string `json:"to"`
		Message string `json:"message"`
		From    string `json:"from"`
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/sms/" {
			t.Errorf("request = %s %s, want POST /v1/sms/", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("X-API-Key"); got != apiKey {
			t.Errorf("X-API-Key was not the tenant key")
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", r.Header.Get("Content-Type"))
		}
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Errorf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"data":{"message_id":"msg-1","status":"Success","credits_used":1}}`))
	}))
	defer server.Close()

	client := NewForCredentials(server.URL, apiKey, "SHOP-APPROVED")
	resp, err := client.Send(context.Background(), "0000000000", "Hello", "")
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	if got.To != "0000000000" || got.Message != "Hello" || got.From != "SHOP-APPROVED" {
		t.Fatalf("request payload = %+v", got)
	}
	if !resp.Success || resp.Data.MessageID != "msg-1" {
		t.Fatalf("response = %+v", resp)
	}
}

func TestProviderErrorBodyIsNotReturned(t *testing.T) {
	const sensitiveBody = `{"success":false,"error":"echoed-key-test-key-not-a-real-secret"}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(sensitiveBody))
	}))
	defer server.Close()

	client := NewForCredentials(server.URL, "test-key-not-a-real-secret", "")
	_, err := client.Send(context.Background(), "0000000000", "Hello", "")
	if err == nil {
		t.Fatal("expected provider error")
	}
	if strings.Contains(err.Error(), "test-key-not-a-real-secret") || strings.Contains(err.Error(), "echoed-key") {
		t.Fatalf("provider response body leaked through error: %v", err)
	}
}
