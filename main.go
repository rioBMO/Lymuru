package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"log"
	"os"

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

	// Per-user data dir (%APPDATA%\Lymuru on Windows). Never relative to the
	// working directory: an installed copy lives in read-only Program Files.
	dataDir, err := backend.ResolveDataDir()
	if err != nil {
		fatalStartup("Could not locate the Lymuru data folder", err)
	}
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		fatalStartup("Could not create the Lymuru data folder "+dataDir, err)
	}

	// Initialize structured logger (writes to <dataDir>/logs/lymuru.log).
	if err := backend.InitLogger(dataDir); err != nil {
		log.Printf("logger init: %v (continuing without file log)", err)
	}

	// Carry over the database from builds that stored it in ./data.
	if src, err := backend.MigrateLegacyDataDir(dataDir, backend.LegacyDataDirCandidates()); err != nil {
		backend.LogWarn("[Startup] legacy data migration failed: %v", err)
	} else if src != "" {
		backend.LogInfo("[Startup] migrated data from %s to %s", src, dataDir)
	}

	app := NewApp(dataDir)

	// Open the SQLite database early so we fail fast on misconfig.
	db, err := storage.Open(backend.DatabasePath(dataDir))
	if err != nil {
		fatalStartup("Could not open the Lymuru database in "+dataDir, err)
	}
	app.SetStorage(db)

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
		fatalStartup("Lymuru failed to start", err)
	}
}

// fatalStartup records a startup failure and exits. The release binary has no
// console, so the error is also shown in a native dialog where supported.
func fatalStartup(msg string, err error) {
	full := fmt.Sprintf("%s:\n\n%v", msg, err)
	backend.LogError("[Startup] %s", full)
	showFatalDialog("Lymuru", full)
	log.Fatal(full)
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
