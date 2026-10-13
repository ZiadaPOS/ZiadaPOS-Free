package main

import (
	"embed"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"syscall"
)

func runBrowserMode(resources embed.FS) error {
	frontend, err := startFrontendServer(resources)
	if err != nil {
		return err
	}
	defer frontend.Close()

	if err := openSystemBrowser(frontend.url); err != nil {
		return fmt.Errorf("the local UI is running at %s but could not open the system browser: %w", frontend.url, err)
	}
	fmt.Printf("ZiadaPOS is available in your browser at %s. Press Ctrl+C to stop it.\n", frontend.url)

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(interrupt)
	<-interrupt
	return nil
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
