package backend

import (
	"archive/tar"
	"archive/zip"

	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/ulikunitz/xz"
	"golang.org/x/text/unicode/norm"
)

// ──────────────────────────────────────────────────────────────────────────────
// FFmpeg discovery, validation, auto-download, and extraction.
//
// Ported from SpotiFLAC/backend/ffmpeg.go with the following changes:
//   - App directory: ~/.spotiflac → ~/.lymuru (%APPDATA%/Lymuru on Windows).
//   - Removed progress calls (SetDownloadProgress, SetDownloadSpeed,
//     SetDownloading) — replaced with fmt.Printf logging.
//   - Removed ConvertAudio() and related types — reserved for
//     backend/ffmpeg_convert.go in Phase 2.
//   - Kept public download URLs from spotbye/Dependencies.
// ──────────────────────────────────────────────────────────────────────────────

// norm import is used by filename normalisation helpers; it is also needed by
// Phase 2's ConvertAudio but imported here so the dependency is resolved early.
var _ = norm.NFC

type executableCandidate struct {
	path   string
	source string
}

func ValidateExecutable(path string) error {
	cleanedPath := filepath.Clean(path)
	if cleanedPath == "" {
		return fmt.Errorf("empty path")
	}

	if !filepath.IsAbs(cleanedPath) {
		return fmt.Errorf("path must be absolute: %s", path)
	}

	info, err := os.Stat(cleanedPath)
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}

	if info.IsDir() {
		return fmt.Errorf("path is a directory: %s", path)
	}

	if runtime.GOOS != "windows" {
		if info.Mode()&0111 == 0 {
			return fmt.Errorf("file is not executable: %s", path)
		}
	}

	base := filepath.Base(cleanedPath)
	validNames := map[string]bool{
		"ffmpeg":      true,
		"ffmpeg.exe":  true,
		"ffprobe":     true,
		"ffprobe.exe": true,
	}
	if !validNames[base] {
		return fmt.Errorf("invalid executable name: %s", base)
	}

	return nil
}

// GetAppDir returns the Lymuru application directory.
// On Windows this is %APPDATA%/Lymuru (via os.UserHomeDir → ~/.lymuru fallback).
// On other platforms it is ~/.lymuru.
func GetAppDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(homeDir, ".lymuru"), nil
}

func EnsureAppDir() (string, error) {
	appDir, err := GetAppDir()
	if err != nil {
		return "", err
	}

	if err := os.MkdirAll(appDir, 0o755); err != nil {
		return "", fmt.Errorf("failed to create app directory: %w", err)
	}

	return appDir, nil
}

func GetFFmpegDir() (string, error) {
	return EnsureAppDir()
}

func copyExecutable(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}

	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o755)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err = io.Copy(out, in); err != nil {
		return err
	}

	if err := out.Sync(); err != nil {
		return err
	}

	return prepareExecutableForUse(dst)
}

func appendExecutableCandidate(candidates []executableCandidate, seen map[string]struct{}, path, source string) []executableCandidate {
	cleanedPath := filepath.Clean(strings.TrimSpace(path))
	if cleanedPath == "" {
		return candidates
	}
	if _, exists := seen[cleanedPath]; exists {
		return candidates
	}

	seen[cleanedPath] = struct{}{}
	return append(candidates, executableCandidate{
		path:   cleanedPath,
		source: source,
	})
}

func resolveSystemExecutable(executableName string) string {
	if runtime.GOOS == "darwin" {
		candidates := []string{
			"/opt/homebrew/bin/" + executableName,
			"/usr/local/bin/" + executableName,
		}
		for _, candidate := range candidates {
			if _, err := os.Stat(candidate); err == nil {
				return candidate
			}
		}
	}

	if runtime.GOOS != "windows" {
		path, err := exec.Command("which", executableName).Output()
		if err == nil {
			trimmed := strings.TrimSpace(string(path))
			if trimmed != "" {
				return trimmed
			}
		}
	}

	path, err := exec.LookPath(executableName)
	if err == nil {
		return path
	}

	return ""
}

func runExecutableVersionCheck(path string) error {
	cmd := exec.Command(path, "-version")
	setHideWindow(cmd)
	return cmd.Run()
}

func removeMacOSQuarantineAttribute(path string) error {
	cmd := exec.Command("xattr", "-d", "com.apple.quarantine", path)
	setHideWindow(cmd)
	output, err := cmd.CombinedOutput()
	if err == nil {
		return nil
	}

	trimmedOutput := strings.TrimSpace(string(output))
	lowerOutput := strings.ToLower(trimmedOutput)
	if strings.Contains(lowerOutput, "no such xattr") || strings.Contains(lowerOutput, "attribute not found") {
		return nil
	}

	if trimmedOutput != "" {
		return fmt.Errorf("%w: %s", err, trimmedOutput)
	}

	return err
}

func prepareExecutableForUse(path string) error {
	cleanedPath := filepath.Clean(strings.TrimSpace(path))
	if cleanedPath == "" {
		return fmt.Errorf("empty path")
	}

	if runtime.GOOS == "windows" {
		return nil
	}

	if err := os.Chmod(cleanedPath, 0755); err != nil {
		return fmt.Errorf("failed to mark executable: %w", err)
	}

	if runtime.GOOS == "darwin" {
		if err := removeMacOSQuarantineAttribute(cleanedPath); err != nil {
			fmt.Printf("[FFmpeg] Warning: failed to remove macOS quarantine from %s: %v\n", cleanedPath, err)
		}
	}

	return nil
}

func resolveExecutablePath(executableName string) (string, string, error) {
	ffmpegDir, err := GetFFmpegDir()
	if err != nil {
		return "", "", err
	}

	localPath := filepath.Join(ffmpegDir, executableName)
	// NOTE: SpotiFLAC uses ".spotiflac-next" for migration; Lymuru uses ".lymuru-next".
	nextDir := filepath.Join(filepath.Dir(ffmpegDir), ".lymuru-next")
	nextPath := filepath.Join(nextDir, executableName)
	localExists := false
	candidates := make([]executableCandidate, 0, 3)
	seen := make(map[string]struct{}, 3)

	if systemPath := resolveSystemExecutable(executableName); systemPath != "" {
		candidates = appendExecutableCandidate(candidates, seen, systemPath, "system")
	}

	if _, err := os.Stat(localPath); err == nil {
		localExists = true
		candidates = appendExecutableCandidate(candidates, seen, localPath, "local")
	}

	if !localExists {
		if _, err := os.Stat(nextPath); err == nil {
			if copyErr := copyExecutable(nextPath, localPath); copyErr == nil {
				fmt.Printf("[FFmpeg] Copied %s from Lymuru-Next folder\n", executableName)
				candidates = appendExecutableCandidate(candidates, seen, localPath, "migrated")
			}
		}
	}

	var lastErr error
	for _, candidate := range candidates {
		if candidate.source != "system" {
			if err := prepareExecutableForUse(candidate.path); err != nil {
				lastErr = err
				fmt.Printf("[FFmpeg] Skipping %s %s: %v\n", candidate.source, candidate.path, err)
				continue
			}
		}

		if err := ValidateExecutable(candidate.path); err != nil {
			lastErr = err
			fmt.Printf("[FFmpeg] Skipping %s %s: %v\n", candidate.source, candidate.path, err)
			continue
		}

		if err := runExecutableVersionCheck(candidate.path); err != nil {
			lastErr = err
			fmt.Printf("[FFmpeg] Skipping %s %s: %v\n", candidate.source, candidate.path, err)
			continue
		}

		return candidate.path, localPath, nil
	}

	if len(candidates) > 0 {
		if lastErr != nil {
			return "", localPath, fmt.Errorf("no working %s executable found: %w", executableName, lastErr)
		}
		return "", localPath, fmt.Errorf("no working %s executable found", executableName)
	}

	return "", localPath, fmt.Errorf("%s not found in app directory or system path", executableName)
}

// GetFFmpegPath locates an ffmpeg executable. It checks the app directory,
// system PATH, and migration paths. Returns the resolved path or an error.
func GetFFmpegPath() (string, error) {
	ffmpegName := "ffmpeg"
	if runtime.GOOS == "windows" {
		ffmpegName = "ffmpeg.exe"
	}

	path, localPath, err := resolveExecutablePath(ffmpegName)
	if err != nil {
		if localPath != "" {
			return localPath, err
		}
		return "", err
	}

	return path, nil
}

// GetFFprobePath locates an ffprobe executable using the same resolution
// strategy as GetFFmpegPath.
func GetFFprobePath() (string, error) {
	ffprobeName := "ffprobe"
	if runtime.GOOS == "windows" {
		ffprobeName = "ffprobe.exe"
	}

	path, localPath, err := resolveExecutablePath(ffprobeName)
	if err != nil {
		if localPath != "" {
			return localPath, err
		}
		return "", err
	}

	return path, nil
}

// IsFFprobeInstalled returns true if ffprobe can be found and executed.
func IsFFprobeInstalled() (bool, error) {
	_, err := GetFFprobePath()
	return err == nil, nil
}

// IsFFmpegInstalled returns true if both ffmpeg and ffprobe can be found.
func IsFFmpegInstalled() (bool, error) {
	if _, err := GetFFmpegPath(); err != nil {
		return false, nil
	}

	return IsFFprobeInstalled()
}

// GetBrewPath returns the path to Homebrew's brew executable, or "" if not found.
func GetBrewPath() string {
	brewPaths := []string{
		"/opt/homebrew/bin/brew",
		"/usr/local/bin/brew",
	}

	for _, path := range brewPaths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	return ""
}

// IsBrewFFmpegInstalled checks whether FFmpeg is installed via Homebrew.
func IsBrewFFmpegInstalled() (bool, error) {
	brewPath := GetBrewPath()
	if brewPath == "" {
		return false, nil
	}

	cmd := exec.Command(brewPath, "list", "ffmpeg")
	setHideWindow(cmd)
	err := cmd.Run()
	return err == nil, nil
}

// InstallFFmpegWithBrew installs FFmpeg using Homebrew.
func InstallFFmpegWithBrew(progressCallback func(int, string)) error {
	brewPath := GetBrewPath()
	if brewPath == "" {
		return fmt.Errorf("brew not found")
	}

	progressCallback(10, "Installing FFmpeg via Homebrew...")

	cmd := exec.Command(brewPath, "install", "ffmpeg")
	setHideWindow(cmd)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to install ffmpeg: %w - %s", err, string(output))
	}

	progressCallback(100, "done")

	return nil
}

// ──────────────────────────────────────────────────────────────────────────────
// FFmpeg auto-download from GitHub releases.
// ──────────────────────────────────────────────────────────────────────────────

const ffmpegReleaseBaseURL = "https://github.com/spotbye/Dependencies/releases/download/FFmpeg-8.1"

func buildFFmpegReleaseURL(assetName string) string {
	return ffmpegReleaseBaseURL + "/" + assetName
}

func getFFmpegDownloadURLs() ([]string, []string, error) {
	switch runtime.GOOS {
	case "windows":
		return []string{buildFFmpegReleaseURL("ffmpeg-windows.zip")}, []string{buildFFmpegReleaseURL("ffprobe-windows.zip")}, nil
	case "linux":
		switch runtime.GOARCH {
		case "amd64":
			return []string{buildFFmpegReleaseURL("ffmpeg-linux-amd64.zip")}, []string{buildFFmpegReleaseURL("ffprobe-linux-amd64.zip")}, nil
		case "arm64":
			return []string{buildFFmpegReleaseURL("ffmpeg-linux-arm64v8.zip")}, []string{buildFFmpegReleaseURL("ffprobe-linux-arm64v8.zip")}, nil
		default:
			return nil, nil, fmt.Errorf("unsupported Linux architecture: %s", runtime.GOARCH)
		}
	case "darwin":
		switch runtime.GOARCH {
		case "amd64":
			return []string{buildFFmpegReleaseURL("ffmpeg-macos-amd64.zip")}, []string{buildFFmpegReleaseURL("ffprobe-macos-amd64.zip")}, nil
		case "arm64":
			return []string{buildFFmpegReleaseURL("ffmpeg-macos-arm64.zip")}, []string{buildFFmpegReleaseURL("ffprobe-macos-arm64.zip")}, nil
		default:
			return nil, nil, fmt.Errorf("unsupported macOS architecture: %s", runtime.GOARCH)
		}
	default:
		return nil, nil, fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
}

// DownloadFFmpeg downloads ffmpeg and ffprobe binaries from GitHub releases.
// Progress is reported via the optional progressCallback (0–100).
//
// NOTE: SpotiFLAC's version calls SetDownloadProgress/SetDownloadSpeed/SetDownloading
// which are part of its progress.go. Those functions don't exist in Lymuru yet
// (they arrive in Phase 2), so this version uses fmt.Printf logging instead.
func DownloadFFmpeg(progressCallback func(int)) error {

	fmt.Printf("[FFmpeg] Starting FFmpeg download...\n")
	SetDownloading(true)
	defer SetDownloading(false)

	ffmpegDir, err := GetFFmpegDir()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(ffmpegDir, 0755); err != nil {
		return fmt.Errorf("failed to create ffmpeg directory: %w", err)
	}

	ffmpegInstalled, _ := IsFFmpegInstalled()
	ffprobeInstalled, _ := IsFFprobeInstalled()

	ffmpegURLs, ffprobeURLs, err := getFFmpegDownloadURLs()
	if err != nil {
		return err
	}

	if !ffmpegInstalled && !ffprobeInstalled {
		if err := downloadWithFallback(ffmpegURLs, ffmpegDir, progressCallback, 0, 50); err != nil {
			return err
		}
		if err := downloadWithFallback(ffprobeURLs, ffmpegDir, progressCallback, 50, 100); err != nil {
			return err
		}
		return nil
	}

	if !ffmpegInstalled {
		return downloadWithFallback(ffmpegURLs, ffmpegDir, progressCallback, 0, 100)
	}

	if !ffprobeInstalled {
		return downloadWithFallback(ffprobeURLs, ffmpegDir, progressCallback, 0, 100)
	}

	return nil
}

func downloadWithFallback(urls []string, destDir string, progressCallback func(int), start, end int) error {
	var lastErr error
	for _, url := range urls {
		fmt.Printf("[FFmpeg] Trying to download from: %s\n", url)
		err := downloadAndExtract(url, destDir, progressCallback, start, end)
		if err == nil {
			return nil
		}
		lastErr = err
		fmt.Printf("[FFmpeg] Attempt failed: %v\n", err)
	}
	return fmt.Errorf("all download attempts failed: %w", lastErr)
}

func downloadAndExtract(url, destDir string, progressCallback func(int), progressStart, progressEnd int) error {

	tmpFile, err := os.CreateTemp("", "ffmpeg-*")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/145.0.0.0 Safari/537.36")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download: HTTP %d", resp.StatusCode)
	}

	totalSize := resp.ContentLength
	var downloaded int64
	lastTime := time.Now()
	var lastBytes int64

	if totalSize > 0 {
		totalSizeMB := float64(totalSize) / (1024 * 1024)
		fmt.Printf("[FFmpeg] Total size: %.2f MB\n", totalSizeMB)
	} else {
		fmt.Printf("[FFmpeg] Downloading... (size unknown)\n")
	}

	buf := make([]byte, 32*1024)
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			_, writeErr := tmpFile.Write(buf[:n])
			if writeErr != nil {
				return fmt.Errorf("failed to write to temp file: %w", writeErr)
			}
			downloaded += int64(n)

			mbDownloaded := float64(downloaded) / (1024 * 1024)
			now := time.Now()
			timeDiff := now.Sub(lastTime).Seconds()
			var speedMBps float64

			if timeDiff > 0.1 {
				bytesDiff := float64(downloaded - lastBytes)
				speedMBps = (bytesDiff / (1024 * 1024)) / timeDiff
				lastTime = now
				lastBytes = downloaded
			}

			// Log progress to stdout (replaces SpotiFLAC's SetDownloadProgress/SetDownloadSpeed).
			if totalSize > 0 && progressCallback != nil {
				rawProgress := float64(downloaded) / float64(totalSize)
				scaledProgress := progressStart + int(rawProgress*float64(progressEnd-progressStart))
				progressCallback(scaledProgress)
				SetDownloadProgress(rawProgress)
			}
			if speedMBps > 0 {
				SetDownloadSpeed(speedMBps * 1024 * 1024)
			}

			if totalSize > 0 {
				percent := float64(downloaded) * 100 / float64(totalSize)
				if speedMBps > 0 {
					fmt.Printf("\r[FFmpeg] Downloading: %.2f MB / %.2f MB (%.1f%%) - %.2f MB/s",
						mbDownloaded, float64(totalSize)/(1024*1024), percent, speedMBps)
				} else {
					fmt.Printf("\r[FFmpeg] Downloading: %.2f MB / %.2f MB (%.1f%%)",
						mbDownloaded, float64(totalSize)/(1024*1024), percent)
				}
			} else {
				if speedMBps > 0 {
					fmt.Printf("\r[FFmpeg] Downloading: %.2f MB - %.2f MB/s", mbDownloaded, speedMBps)
				} else {
					fmt.Printf("\r[FFmpeg] Downloading: %.2f MB", mbDownloaded)
				}
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read response: %w", err)
		}
	}

	tmpFile.Close()

	if totalSize > 0 {
		fmt.Printf("\r[FFmpeg] Download complete: %.2f MB / %.2f MB (100%%)          \n",
			float64(downloaded)/(1024*1024), float64(totalSize)/(1024*1024))
	} else {
		fmt.Printf("\r[FFmpeg] Download complete: %.2f MB          \n", float64(downloaded)/(1024*1024))
	}
	fmt.Printf("[FFmpeg] Extracting...\n")

	if strings.HasSuffix(url, ".tar.xz") {
		return extractTarXz(tmpFile.Name(), destDir)
	}
	if strings.HasSuffix(url, ".zip") {
		return extractZip(tmpFile.Name(), destDir)
	}
	return fmt.Errorf("unsupported archive format for %s", url)
}

// ──────────────────────────────────────────────────────────────────────────────
// Archive extraction helpers.
// ──────────────────────────────────────────────────────────────────────────────

func extractZip(zipPath, destDir string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("failed to open zip: %w", err)
	}
	defer r.Close()

	ffmpegName := "ffmpeg"
	ffprobeName := "ffprobe"
	if runtime.GOOS == "windows" {
		ffmpegName = "ffmpeg.exe"
		ffprobeName = "ffprobe.exe"
	}

	foundFFmpeg := false
	foundFFprobe := false

	for _, f := range r.File {
		baseName := filepath.Base(f.Name)
		if f.FileInfo().IsDir() {
			continue
		}

		var destPath string
		if baseName == ffmpegName {
			destPath = filepath.Join(destDir, ffmpegName)
			foundFFmpeg = true
		} else if baseName == ffprobeName {
			destPath = filepath.Join(destDir, ffprobeName)
			foundFFprobe = true
		} else {

			continue
		}

		fmt.Printf("[FFmpeg] Found: %s\n", f.Name)

		rc, err := f.Open()
		if err != nil {
			return fmt.Errorf("failed to open file in zip: %w", err)
		}

		outFile, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
		if err != nil {
			rc.Close()
			return fmt.Errorf("failed to create output file: %w", err)
		}

		_, err = io.Copy(outFile, rc)
		rc.Close()
		outFile.Close()

		if err != nil {
			return fmt.Errorf("failed to extract file: %w", err)
		}

		if err := prepareExecutableForUse(destPath); err != nil {
			return fmt.Errorf("failed to prepare extracted executable: %w", err)
		}

		fmt.Printf("[FFmpeg] Extracted to: %s\n", destPath)
	}

	if !foundFFmpeg && !foundFFprobe {
		return fmt.Errorf("neither ffmpeg nor ffprobe found in archive")
	}

	if foundFFmpeg {
		fmt.Printf("[FFmpeg] ffmpeg extracted successfully\n")
	}
	if foundFFprobe {
		fmt.Printf("[FFmpeg] ffprobe extracted successfully\n")
	}

	return nil
}

func extractTarXz(tarXzPath, destDir string) error {
	file, err := os.Open(tarXzPath)
	if err != nil {
		return fmt.Errorf("failed to open tar.xz: %w", err)
	}
	defer file.Close()

	xzReader, err := xz.NewReader(file)
	if err != nil {
		return fmt.Errorf("failed to create xz reader: %w", err)
	}

	tarReader := tar.NewReader(xzReader)

	ffmpegName := "ffmpeg"
	ffprobeName := "ffprobe"
	foundFFmpeg := false
	foundFFprobe := false

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar: %w", err)
		}

		if header.Typeflag != tar.TypeReg {
			continue
		}

		baseName := filepath.Base(header.Name)
		var destPath string

		if baseName == ffmpegName {
			destPath = filepath.Join(destDir, ffmpegName)
			foundFFmpeg = true
		} else if baseName == ffprobeName {
			destPath = filepath.Join(destDir, ffprobeName)
			foundFFprobe = true
		} else {

			continue
		}

		fmt.Printf("[FFmpeg] Found: %s\n", header.Name)

		outFile, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}

		_, err = io.Copy(outFile, tarReader)
		outFile.Close()

		if err != nil {
			return fmt.Errorf("failed to extract file: %w", err)
		}

		if err := prepareExecutableForUse(destPath); err != nil {
			return fmt.Errorf("failed to prepare extracted executable: %w", err)
		}

		fmt.Printf("[FFmpeg] Extracted to: %s\n", destPath)
	}

	if !foundFFmpeg && !foundFFprobe {
		return fmt.Errorf("neither ffmpeg nor ffprobe found in archive")
	}

	if foundFFmpeg {
		fmt.Printf("[FFmpeg] ffmpeg extracted successfully\n")
	}
	if foundFFprobe {
		fmt.Printf("[FFmpeg] ffprobe extracted successfully\n")
	}

	return nil
}
