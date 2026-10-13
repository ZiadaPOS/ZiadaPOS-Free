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
	"path/filepath"
	"strings"
	"sync"

	"github.com/ZiadaPOS/ZiadaPOS/apps/desktop/localpos"
)

const (
	devServerEnvironmentVariable     = "ZIADAPOS_DESKTOP_DEV_URL"
	dataDirectoryEnvironmentVariable = "ZIADAPOS_DATA_DIR"
)

type desktopApp struct {
	resources fs.FS

	mu           sync.RWMutex
	frontend     *frontendServer
	proxy        *httputil.ReverseProxy
	localStore   *localpos.Store
	localHandler http.Handler
	startupError error
}

func newDesktopApp(resources fs.FS) *desktopApp {
	return &desktopApp{resources: resources}
}

func (a *desktopApp) startup(_ context.Context) {
	dataDir, err := localDataDirectory()
	if err != nil {
		a.setStartupError(err)
		return
	}
	store, err := localpos.Open(filepath.Join(dataDir, "pos.sqlite"))
	if err != nil {
		a.setStartupError(err)
		return
	}
	a.mu.Lock()
	a.localStore = store
	a.localHandler = localpos.NewHandler(store)
	a.mu.Unlock()

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
		a.setStartupError(err)
		return
	}

	proxy, err := newLoopbackProxy(frontend.url)
	if err != nil {
		frontend.Close()
		a.setStartupError(err)
		return
	}

	a.mu.Lock()
	a.frontend = frontend
	a.proxy = proxy
	a.mu.Unlock()
}

func (a *desktopApp) setStartupError(err error) {
	a.mu.Lock()
	a.startupError = err
	a.mu.Unlock()
}

func (a *desktopApp) shutdown(_ context.Context) {
	a.mu.Lock()
	frontend := a.frontend
	store := a.localStore
	a.frontend = nil
	a.localStore = nil
	a.localHandler = nil
	a.mu.Unlock()
	if frontend != nil {
		frontend.Close()
	}
	if store != nil {
		if err := store.Close(); err != nil {
			log.Printf("close local POS database: %v", err)
		}
	}
}

func (a *desktopApp) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.mu.RLock()
	proxy := a.proxy
	localHandler := a.localHandler
	startupError := a.startupError
	a.mu.RUnlock()

	if startupError != nil {
		http.Error(w, fmt.Sprintf("ZiadaPOS could not start its local app: %v", startupError), http.StatusServiceUnavailable)
		return
	}
	if localHandler != nil {
		if isLocalServiceRequest(r.URL.Path) {
			localHandler.ServeHTTP(w, r)
			return
		}
		if isPosAPIRequest(r.URL.Path) {
			localRequest := r.Clone(r.Context())
			localURL := *r.URL
			path := strings.TrimPrefix(r.URL.Path, "/api/pos")
			if path == "" {
				path = "/"
			}
			localURL.Path = "/api/v1" + path
			localURL.RawPath = ""
			localRequest.URL = &localURL
			localHandler.ServeHTTP(w, localRequest)
			return
		}
	}
	if proxy == nil {
		http.Error(w, "ZiadaPOS is starting its local web UI. Please try again shortly.", http.StatusServiceUnavailable)
		return
	}
	proxy.ServeHTTP(w, r)
}

func isLocalServiceRequest(path string) bool {
	return path == "/api/pos-session" || strings.HasPrefix(path, "/api/local/")
}

func isPosAPIRequest(path string) bool {
	return path == "/api/pos" || strings.HasPrefix(path, "/api/pos/")
}

func localDataDirectory() (string, error) {
	if configured := strings.TrimSpace(os.Getenv(dataDirectoryEnvironmentVariable)); configured != "" {
		if err := os.MkdirAll(configured, 0o700); err != nil {
			return "", fmt.Errorf("create configured local data directory: %w", err)
		}
		return configured, nil
	}
	root, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("find the local application data directory: %w", err)
	}
	directory := filepath.Join(root, "ZiadaPOS")
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return "", fmt.Errorf("create local ZiadaPOS data directory: %w", err)
	}
	return directory, nil
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
