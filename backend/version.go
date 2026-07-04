package backend

import "sync"

var (
	versionMu sync.RWMutex
	appVer    = "0.0.0"
)

// SetAppVersion sets the application version (seeded from wails.json).
func SetAppVersion(v string) {
	versionMu.Lock()
	defer versionMu.Unlock()
	if v != "" {
		appVer = v
	}
}

// AppVersion returns the application version.
func AppVersion() string {
	versionMu.RLock()
	defer versionMu.RUnlock()
	return appVer
}
