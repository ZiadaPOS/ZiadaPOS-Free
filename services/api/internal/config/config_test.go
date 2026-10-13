package config

import "testing"

func TestListenAddrDefaultsToPort(t *testing.T) {
	t.Setenv("PORT", "8091")
	t.Setenv("API_LISTEN_ADDR", "")

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got, want := cfg.ListenAddr, ":8091"; got != want {
		t.Fatalf("ListenAddr = %q, want %q", got, want)
	}
}

func TestListenAddrCanBeRestrictedToLoopback(t *testing.T) {
	t.Setenv("PORT", "8080")
	t.Setenv("API_LISTEN_ADDR", "127.0.0.1:8080")

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got, want := cfg.ListenAddr, "127.0.0.1:8080"; got != want {
		t.Fatalf("ListenAddr = %q, want %q", got, want)
	}
}
