package main

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"syscall"
	"time"
)

func runBrowserMode(resources embed.FS) error {
	app := newDesktopApp(resources)
	app.startup(context.Background())
	app.mu.RLock()
	startupError := app.startupError
	app.mu.RUnlock()
	if startupError != nil {
		app.shutdown(context.Background())
		return startupError
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		app.shutdown(context.Background())
		return fmt.Errorf("start the local browser gateway: %w", err)
	}
	server := &http.Server{Handler: app, ReadHeaderTimeout: 10 * time.Second}
	serveResult := make(chan error, 1)
	go func() {
		err := server.Serve(listener)
		if !errors.Is(err, http.ErrServerClosed) {
			serveResult <- err
		}
	}()
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(ctx)
		app.shutdown(context.Background())
	}()

	target := "http://" + listener.Addr().String()
	if err := openSystemBrowser(target); err != nil {
		return fmt.Errorf("the local UI is running at %s but could not open the system browser: %w", target, err)
	}
	fmt.Printf("ZiadaPOS local mode is available in your browser at %s. Press Ctrl+C to stop it.\n", target)

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(interrupt)
	select {
	case <-interrupt:
		return nil
	case err := <-serveResult:
		return fmt.Errorf("local browser gateway stopped: %w", err)
	}
}

func openSystemBrowser(target string) error {
	var command *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		command = exec.Command("rundll32", "url.dll,FileProtocolHandler", target)
	case "darwin":
		command = exec.Command("open", target)
	default:
		command = exec.Command("xdg-open", target)
	}
	if err := command.Start(); err != nil {
		return err
	}
	go func() { _ = command.Wait() }()
	return nil
}
