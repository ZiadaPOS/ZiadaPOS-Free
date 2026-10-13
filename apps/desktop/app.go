package main

import (
	"context"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"sync"
)

const devServerEnvironmentVariable = "ZIADAPOS_DESKTOP_DEV_URL"

type desktopApp struct {
	resources fs.FS

	mu           sync.RWMutex
	frontend     *frontendServer
	proxy        *httputil.ReverseProxy
	startupError error
}

func newDesktopApp(resources fs.FS) *desktopApp {
	return &desktopApp{resources: resources}
}

func (a *desktopApp) startup(_ context.Context) {
	if devURL := strings.TrimSpace(os.Getenv(devServerEnvironmentVariable)); devURL != "" {
		proxy, err := newLoopbackProxy(devURL)
		a.mu.Lock()
		a.proxy = proxy
		a.startupError = err
		a.mu.Unlock()
		return
	}

	frontend, err := startFrontendServer(a.resources)
	if err != nil {
		a.mu.Lock()
		a.startupError = err
		a.mu.Unlock()
		return
	}

	proxy, err := newLoopbackProxy(frontend.url)
	if err != nil {
		frontend.Close()
		a.mu.Lock()
		a.startupError = err
		a.mu.Unlock()
		return
	}

	a.mu.Lock()
	a.frontend = frontend
	a.proxy = proxy
	a.mu.Unlock()
}

func (a *desktopApp) shutdown(_ context.Context) {
	a.mu.Lock()
	frontend := a.frontend
	a.frontend = nil
	a.mu.Unlock()
	if frontend != nil {
		frontend.Close()
	}
}

func (a *desktopApp) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.mu.RLock()
	proxy := a.proxy
	startupError := a.startupError
	a.mu.RUnlock()

	if startupError != nil {
		http.Error(w, fmt.Sprintf("ZiadaPOS could not start its local web UI: %v", startupError), http.StatusServiceUnavailable)
		return
	}
	if proxy == nil {
		http.Error(w, "ZiadaPOS is starting its local web UI. Please try again shortly.", http.StatusServiceUnavailable)
		return
	}
	proxy.ServeHTTP(w, r)
}

func newLoopbackProxy(rawURL string) (*httputil.ReverseProxy, error) {
	target, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid local frontend URL: %w", err)
	}
	if target.Scheme != "http" || target.Hostname() == "" {
		return nil, fmt.Errorf("local frontend URL must be an http loopback address")
	}
	hostname := strings.Trim(target.Hostname(), "[]")
	if hostname != "localhost" {
		ip := net.ParseIP(hostname)
		if ip == nil || !ip.IsLoopback() {
			return nil, fmt.Errorf("local frontend URL must resolve to loopback, got %q", target.Hostname())
		}
	}

	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.ErrorHandler = func(w http.ResponseWriter, _ *http.Request, err error) {
		log.Printf("local frontend request failed: %v", err)
		http.Error(w, "The local ZiadaPOS web UI is temporarily unavailable.", http.StatusBadGateway)
	}
	return proxy, nil
}
