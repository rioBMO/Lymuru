package backend

// ──────────────────────────────────────────────────────────────────────────────
// SpotiFLAC compatibility shims.
//
// These functions provide the interface expected by code ported from
// SpotiFLAC/backend/ that references SpotiFLAC's JSON-file-based config
// system. Lymuru uses SQLite-based settings instead (config.go), so these
// shims return sensible defaults or delegate to Lymuru's config where
// possible.
//
// As the migration progresses, these should be wired into Lymuru's
// Settings struct properly.
// ──────────────────────────────────────────────────────────────────────────────

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// GetDefaultMusicPath returns the default music directory.
func GetDefaultMusicPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "C:\\Users\\Public\\Music"
	}
	return filepath.Join(homeDir, "Music")
}

// GetConfigPath returns the path to the Lymuru config.json file
// (used by SpotiFLAC-ported code for JSON-based settings).
func GetConfigPath() (string, error) {
	dir, err := EnsureAppDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

// LoadConfigSettings loads the JSON config file used by SpotiFLAC-ported code.
// Returns nil, nil if the file doesn't exist yet.
func LoadConfigSettings() (map[string]interface{}, error) {
	configPath, err := GetConfigPath()
	if err != nil {
		return nil, err
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var settings map[string]interface{}
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil, err
	}

	return settings, nil
}

// GetRedownloadWithSuffixSetting returns whether to add a suffix to
// redownloaded files instead of skipping them. Defaults to true.
func GetRedownloadWithSuffixSetting() bool {
	settings, err := LoadConfigSettings()
	if err != nil || settings == nil {
		return true
	}

	enabled, ok := settings["redownloadWithSuffix"].(bool)
	if !ok {
		return true
	}
	return enabled
}

// GetCustomTidalAPISetting returns a custom Tidal API URL if configured.
func GetCustomTidalAPISetting() string {
	settings, err := LoadConfigSettings()
	if err != nil || settings == nil {
		return ""
	}

	return normalizeCustomTidalAPIValue(settings["customTidalApi"])
}

func normalizeCustomTidalAPIValue(value interface{}) string {
	customAPI, _ := value.(string)
	customAPI = strings.TrimRight(strings.TrimSpace(customAPI), "/")
	if strings.HasPrefix(customAPI, "https://") {
		return customAPI
	}
	return ""
}

// GetCustomQobuzAPISetting returns a custom Qobuz API URL if configured.
func GetCustomQobuzAPISetting() string {
	settings, err := LoadConfigSettings()
	if err != nil || settings == nil {
		return ""
	}

	return normalizeCustomQobuzAPIValue(settings["customQobuzApi"])
}

func normalizeCustomQobuzAPIValue(value interface{}) string {
	customAPI, _ := value.(string)
	customAPI = strings.TrimRight(strings.TrimSpace(customAPI), "/")
	if strings.HasPrefix(customAPI, "https://") {
		return customAPI
	}
	return ""
}

// GetLinkResolverSetting returns the configured link resolver provider.
func GetLinkResolverSetting() string {
	settings, err := LoadConfigSettings()
	if err != nil || settings == nil {
		return linkResolverProviderDeezerSongLink
	}

	resolver, _ := settings["linkResolver"].(string)
	switch strings.TrimSpace(strings.ToLower(resolver)) {
	case "songlink", "deezer-songlink":
		return linkResolverProviderDeezerSongLink
	case "songstats":
		return linkResolverProviderSongstats
	case "":
		return linkResolverProviderDeezerSongLink
	default:
		return linkResolverProviderDeezerSongLink
	}
}

// GetLinkResolverAllowFallback returns whether fallback link resolution is allowed.
func GetLinkResolverAllowFallback() bool {
	settings, err := LoadConfigSettings()
	if err != nil || settings == nil {
		return true
	}

	allowFallback, ok := settings["allowResolverFallback"].(bool)
	if !ok {
		return true
	}

	return allowFallback
}
