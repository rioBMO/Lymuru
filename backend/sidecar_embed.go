package backend

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
)

const sidecarScriptName = "deezload.py"

var (
	embeddedSidecarMu sync.RWMutex
	embeddedSidecar   fs.FS
)

// SetEmbeddedSidecar registers the sidecar files bundled into the binary
// (a filesystem whose root holds deezload.py and requirements.txt). Installed
// and portable builds have no sidecar folder on disk, so these are extracted
// into the data directory when needed.
func SetEmbeddedSidecar(fsys fs.FS) {
	embeddedSidecarMu.Lock()
	defer embeddedSidecarMu.Unlock()
	embeddedSidecar = fsys
}

// extractEmbeddedSidecar writes the bundled sidecar files to <dataDir>/sidecar
// and returns the script path. A file is rewritten only when its content
// differs, so an unchanged script is not touched on every start.
func extractEmbeddedSidecar(dataDir string) (string, error) {
	embeddedSidecarMu.RLock()
	fsys := embeddedSidecar
	embeddedSidecarMu.RUnlock()
	if fsys == nil {
		return "", errors.New("no sidecar script is bundled in this build")
	}

	entries, err := fs.ReadDir(fsys, ".")
	if err != nil {
		return "", fmt.Errorf("read bundled sidecar: %w", err)
	}
	outDir := filepath.Join(dataDir, "sidecar")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return "", err
	}

	found := false
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		data, err := fs.ReadFile(fsys, e.Name())
		if err != nil {
			return "", err
		}
		if err := writeFileIfChanged(filepath.Join(outDir, e.Name()), data); err != nil {
			return "", fmt.Errorf("extract %s: %w", e.Name(), err)
		}
		if e.Name() == sidecarScriptName {
			found = true
		}
	}
	if !found {
		return "", fmt.Errorf("bundled sidecar has no %s", sidecarScriptName)
	}
	return filepath.Join(outDir, sidecarScriptName), nil
}

func writeFileIfChanged(path string, data []byte) error {
	if cur, err := os.ReadFile(path); err == nil && bytes.Equal(cur, data) {
		return nil
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return err
	}
	return nil
}
