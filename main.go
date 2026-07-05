package main

import (
	"embed"
	"encoding/json"
	"log"

	"github.com/lymuru/lymuru/backend"
	"github.com/lymuru/lymuru/backend/storage"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed wails.json
var wailsJSON []byte

func main() {
	// Read the product version from wails.json and seed the backend.
	backend.SetAppVersion(readProductVersion())

	app := NewApp()

	// Try to open the SQLite database early so we fail fast on misconfig.
	db, err := storage.Open("data/lymuru.db")
	if err != nil {
		log.Fatalf("storage open: %v", err)
	}
	app.SetStorage(db)

	// Initialize structured logger (writes to data/logs/lymuru.log).
	if err := backend.InitLogger("data"); err != nil {
		log.Printf("logger init: %v (continuing without file log)", err)
	}

	err = wails.Run(&options.App{
		Title:     "Lymuru",
		Width:     1280,
		Height:    800,
		MinWidth:  1024,
		MinHeight: 640,
		Frameless: true,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 0, G: 0, B: 0, A: 255},
		OnStartup:        app.startup,
		OnShutdown:       app.shutdown,
		DragAndDrop: &options.DragAndDrop{
			EnableFileDrop:     true,
			DisableWebViewDrop: false,
			CSSDropProperty:    "--wails-drop-target",
			CSSDropValue:       "drop",
		},
		Bind: []interface{}{
			app,
		},
		Windows: &windows.Options{
			WebviewIsTransparent:              false,
			WindowIsTranslucent:               false,
			DisableWindowIcon:                 false,
			DisableFramelessWindowDecorations: false,
		},
	})
	if err != nil {
		log.Fatal("Error:", err.Error())
	}
}

func readProductVersion() string {
	type wailsInfo struct {
		Info struct {
			ProductVersion string `json:"productVersion"`
		} `json:"info"`
	}
	var v wailsInfo
	if err := json.Unmarshal(wailsJSON, &v); err == nil {
		return v.Info.ProductVersion
	}
	return "0.0.0"
}
