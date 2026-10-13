package main

import (
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

type frontendServer struct {
	url        string
	command    *exec.Cmd
	done       chan struct{}
	processErr error
	root       string
}

func startFrontendServer(resources fs.FS) (*frontendServer, error) {
	resourceFS, err := fs.Sub(resources, "resources")
	if err != nil {
		return nil, missingFrontendError(err)
	}

	cacheRoot, err := os.UserCacheDir()
	if err != nil {
		cacheRoot = os.TempDir()
	}
	if err := os.MkdirAll(cacheRoot, 0o755); err != nil {
		return nil, fmt.Errorf("create runtime cache directory: %w", err)
	}
	root, err := os.MkdirTemp(cacheRoot, "ziadapos-desktop-")
	if err != nil {
		return nil, fmt.Errorf("create temporary frontend directory: %w", err)
	}
	cleanup := func() { _ = os.RemoveAll(root) }

	if err := copyEmbeddedTree(resourceFS, root); err != nil {
		cleanup()
		return nil, fmt.Errorf("extract packaged frontend: %w", err)
	}

	nodeName := "node"
	if runtime.GOOS == "windows" {
		nodeName += ".exe"
	}
	nodePath := filepath.Join(root, "runtime", nodeName)
	webRoot := filepath.Join(root, "web")
	serverScript := filepath.Join(webRoot, "server.js")
	for _, required := range []string{nodePath, serverScript} {
		if _, err := os.Stat(required); err != nil {
			cleanup()
			return nil, missingFrontendError(err)
		}
	}

	port, err := freeLoopbackPort()
	if err != nil {
		cleanup()
		return nil, fmt.Errorf("reserve a local frontend port: %w", err)
	}

	command := exec.Command(nodePath, serverScript)
	command.Dir = webRoot
	command.Env = mergeEnvironment(os.Environ(), map[string]string{
		"HOSTNAME":                    "127.0.0.1",
		"PORT":                        fmt.Sprint(port),
		"NODE_ENV":                    "production",
		"NEXT_PUBLIC_SENTRY_DISABLED": "1",
		"NEXT_TELEMETRY_DISABLED":     "1",
	})
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	if err := command.Start(); err != nil {
		cleanup()
		return nil, fmt.Errorf("start the bundled local web UI: %w", err)
	}

	server := &frontendServer{
		url:     fmt.Sprintf("http://127.0.0.1:%d", port),
		command: command,
		done:    make(chan struct{}),
		root:    root,
	}
	go func() {
		server.processErr = command.Wait()
		close(server.done)
	}()

	if err := server.waitUntilReady(30 * time.Second); err != nil {
		server.Close()
		return nil, err
	}
	return server, nil
}

func copyEmbeddedTree(source fs.FS, destination string) error {
	return fs.WalkDir(source, ".", func(name string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if name == "." {
			return nil
		}
		target := filepath.Join(destination, filepath.FromSlash(name))
		if entry.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := fs.ReadFile(source, name)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		mode := os.FileMode(0o644)
		if strings.HasPrefix(name, "runtime/node") {
			mode = 0o755
		}
		return os.WriteFile(target, data, mode)
	})
}

func (s *frontendServer) waitUntilReady(timeout time.Duration) error {
	client := &http.Client{
		Timeout: 1200 * time.Millisecond,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	deadline := time.NewTimer(timeout)
	defer deadline.Stop()
	ticker := time.NewTicker(150 * time.Millisecond)
	defer ticker.Stop()

	for {
		request, _ := http.NewRequest(http.MethodGet, s.url+"/", nil)
		response, err := client.Do(request)
		if err == nil {
			_ = response.Body.Close()
			if response.StatusCode < http.StatusInternalServerError {
				return nil
			}
		}

		select {
		case <-s.done:
			if s.processErr == nil {
				return fmt.Errorf("local web UI stopped before it became ready")
			}
			return fmt.Errorf("local web UI stopped before it became ready: %w", s.processErr)
		case <-deadline.C:
			return fmt.Errorf("local web UI did not become ready within %s", timeout)
		case <-ticker.C:
		}
	}
}

func (s *frontendServer) Close() {
	if s.command != nil && s.command.Process != nil {
		_ = s.command.Process.Kill()
		select {
		case <-s.done:
		case <-time.After(3 * time.Second):
		}
	}
	if s.root != "" {
		_ = os.RemoveAll(s.root)
	}
}

func freeLoopbackPort() (int, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer listener.Close()
	return listener.Addr().(*net.TCPAddr).Port, nil
}

func mergeEnvironment(base []string, overrides map[string]string) []string {
	values := make(map[string]string, len(base)+len(overrides))
	for _, item := range base {
		key, value, ok := strings.Cut(item, "=")
		if ok {
			values[key] = value
		}
	}
	for key, value := range overrides {
		values[key] = value
	}
	result := make([]string, 0, len(values))
	for key, value := range values {
		result = append(result, key+"="+value)
	}
	return result
}

func missingFrontendError(cause error) error {
	return fmt.Errorf("desktop web runtime is not bundled (%v); build from the repository root with `wails build` in apps/desktop so the Next.js standalone output and matching Node.js runtime are embedded", cause)
}
