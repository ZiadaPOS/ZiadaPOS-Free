package main

import (
	"embed"
	"log"
	"os"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/linux"
)

// The platform-specific build job places the Next.js standalone server and a
// matching Node.js runtime under resources before Wails compiles this binary.
//
//go:embed all:resources
var appResources embed.FS

// The shared brand source is also used by Wails to generate the native
// Windows executable icon and macOS app-bundle icon during packaging.
//
//go:embed build/appicon.png
var appIcon []byte

func main() {
	if hasArgument("--browser") {
		if err := runBrowserMode(appResources); err != nil {
			log.Fatal(err)
		}
		return
	}

	app := newDesktopApp(appResources)
	err := wails.Run(&options.App{
		Title:     "ZiadaPOS",
		Width:     1440,
		Height:    960,
		MinWidth:  1024,
		MinHeight: 680,
		AssetServer: &assetserver.Options{
			Handler: app,
		},
		Linux: &linux.Options{
			Icon:        appIcon,
			ProgramName: "ZiadaPOS",
		},
		OnStartup:  app.startup,
		OnShutdown: app.shutdown,
	})
	if err != nil {
		log.Fatal(err)
	}
}

func hasArgument(want string) bool {
	for _, arg := range os.Args[1:] {
		if arg == want {
			return true
		}
	}
	return false
}
