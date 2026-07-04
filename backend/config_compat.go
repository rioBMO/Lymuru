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

// GetExistingFileCheckModeSetting returns how to check for existing files.
func GetExistingFileCheckModeSetting() string {
	settings, err := LoadConfigSettings()
	if err != nil || settings == nil {
		return "filename"
	}

	rawMode, _ := settings["existingFileCheckMode"].(string)
	return normalizeExistingFileCheckMode(rawMode)
}

func normalizeExistingFileCheckMode(value string) string {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case "isrc", "upc":
		return "isrc"
	default:
		return "filename"
	}
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

// SanitizeSettingsMap normalizes settings values.
func SanitizeSettingsMap(settings map[string]interface{}) map[string]interface{} {
	if settings == nil {
		return nil
	}

	sanitized := make(map[string]interface{}, len(settings))
	for key, value := range settings {
		sanitized[key] = value
	}

	customAPI := normalizeCustomTidalAPIValue(sanitized["customTidalApi"])
	sanitized["customTidalApi"] = customAPI
	sanitized["downloader"] = sanitizeDownloaderValue(sanitized["downloader"])
	sanitized["autoOrder"] = sanitizeAutoOrderValue(sanitized["autoOrder"])

	return sanitized
}

func sanitizeDownloaderValue(value interface{}) string {
	downloader, _ := value.(string)
	switch strings.TrimSpace(strings.ToLower(downloader)) {
	case "tidal":
		return "tidal"
	case "qobuz":
		return "qobuz"
	case "amazon":
		return "amazon"
	default:
		return "auto"
	}
}

func sanitizeAutoOrderValue(value interface{}) string {
	autoOrder, _ := value.(string)
	allowed := map[string]struct{}{
		"tidal":  {},
		"qobuz":  {},
		"amazon": {},
	}
	fallback := "tidal-qobuz-amazon"

	seen := make(map[string]struct{})
	parts := make([]string, 0, 3)
	for _, rawPart := range strings.Split(strings.TrimSpace(strings.ToLower(autoOrder)), "-") {
		part := strings.TrimSpace(rawPart)
		if part == "" {
			continue
		}
		if _, ok := allowed[part]; !ok {
			continue
		}
		if _, ok := seen[part]; ok {
			continue
		}
		seen[part] = struct{}{}
		parts = append(parts, part)
	}

	if len(parts) < 2 {
		return fallback
	}

	return strings.Join(parts, "-")
}

// SetMacOSFileIconFromImage is a no-op on non-macOS platforms.
// On macOS, SpotiFLAC sets custom file icons from cover art images.
func SetMacOSFileIconFromImage(filePath string, tmpPath string, iconSize int) error {
	// No-op — macOS file icon support is not ported.
	return nil
}
