package main

import (
	"bytes"
	"image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewLoopbackProxyRejectsRemoteFrontend(t *testing.T) {
	for _, target := range []string{"http://example.com", "https://127.0.0.1:3000", "file:///tmp/app"} {
		t.Run(target, func(t *testing.T) {
			if _, err := newLoopbackProxy(target); err == nil {
				t.Fatalf("newLoopbackProxy(%q) succeeded; want error", target)
			}
		})
	}
}

func TestNewLoopbackProxyForwardsMethodPathAndBody(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read body: %v", err)
		}
		w.Header().Set("X-Forwarded-Test", "ok")
		_, _ = w.Write([]byte(r.Method + " " + r.URL.RequestURI() + " " + string(body)))
	}))
	defer backend.Close()

	proxy, err := newLoopbackProxy(backend.URL)
	if err != nil {
		t.Fatalf("newLoopbackProxy: %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "http://wails.localhost/api/pos?store=1", strings.NewReader("payload"))
	proxy.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if got, want := recorder.Body.String(), "POST /api/pos?store=1 payload"; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
	if got := recorder.Header().Get("X-Forwarded-Test"); got != "ok" {
		t.Fatalf("forwarded response header = %q, want %q", got, "ok")
	}
}

func TestMergeEnvironmentUsesOverrides(t *testing.T) {
	got := mergeEnvironment([]string{"A=old", "B=kept"}, map[string]string{"A": "new", "C": "added"})
	values := make(map[string]string)
	for _, item := range got {
		key, value, ok := strings.Cut(item, "=")
		if ok {
			values[key] = value
		}
	}
	for key, want := range map[string]string{"A": "new", "B": "kept", "C": "added"} {
		if values[key] != want {
			t.Errorf("%s = %q, want %q", key, values[key], want)
		}
	}
}

func TestAppIconIsEmbeddedPNG(t *testing.T) {
	config, err := png.DecodeConfig(bytes.NewReader(appIcon))
	if err != nil {
		t.Fatalf("decode embedded app icon: %v", err)
	}
	if config.Width != 1024 || config.Height != 1024 {
		t.Fatalf("embedded app icon size = %dx%d, want 1024x1024", config.Width, config.Height)
	}
}
