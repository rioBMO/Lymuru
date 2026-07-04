package backend

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/lymuru/lymuru/backend/sidecarscript"
)

// ExtractEmbeddedSidecar writes the Python sidecar script that was
// embedded at compile time (see ./sidecarscript) to a stable, writable
// per-user location and returns the absolute path of the extracted file.
// The directory is created if it does not exist.
//
// On Windows, the location is %APPDATA%/Lymuru/sidecar/. On Linux/macOS,
// it is $HOME/.Lymuru/sidecar/. The user is expected to drop their
// Telegram `.env` file in that same directory so deezload.py can load it.
//
// Python and the sidecar's third-party dependencies (telethon, mutagen,
// requests, python-dotenv) must be installed separately.
func ExtractEmbeddedSidecar() (string, error) {
	target, err := sidecarExtractDir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(target, 0o755); err != nil {
		return "", fmt.Errorf("sidecar: create extract dir: %w", err)
	}
	dst := filepath.Join(target, "deezload.py")
	// Only write if missing or out of date so user edits to the
	// extracted copy (e.g. debugging tweaks) are preserved across runs.
	existing, err := os.ReadFile(dst)
	if err != nil || string(existing) != sidecarscript.Source {
		if err := os.WriteFile(dst, []byte(sidecarscript.Source), 0o644); err != nil {
			return "", fmt.Errorf("sidecar: write script: %w", err)
		}
	}
	return dst, nil
}

// sidecarExtractDir returns the per-user directory where the embedded
// Python script and its sibling .env file are stored.
func sidecarExtractDir() (string, error) {
	if runtime.GOOS == "windows" {
		if appdata := os.Getenv("APPDATA"); appdata != "" {
			return filepath.Join(appdata, "Lymuru", "sidecar"), nil
		}
	}
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		return filepath.Join(home, ".Lymuru", "sidecar"), nil
	}
	return "", fmt.Errorf("sidecar: cannot determine user directory")
}
