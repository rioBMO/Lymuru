package backend

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// ResolveOpenTarget returns the directory to show for path: the path itself
// if it's a directory, or its parent if it's a file. It returns an error if
// path is empty, not absolute, or doesn't exist.
func ResolveOpenTarget(path string) (string, error) {
	if path == "" {
		return "", errors.New("path is required")
	}
	if !filepath.IsAbs(path) {
		return "", fmt.Errorf("path is not absolute: %s", path)
	}
	info, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("folder does not exist: %s", path)
		}
		return "", err
	}
	if info.IsDir() {
		return filepath.Clean(path), nil
	}
	return filepath.Dir(path), nil
}

func OpenFolderInExplorer(path string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("explorer", path)
	case "darwin":
		cmd = exec.Command("open", path)
	case "linux":
		cmd = exec.Command("xdg-open", path)
	default:
		cmd = exec.Command("xdg-open", path)
	}

	return cmd.Start()
}

func SelectFolderDialog(ctx context.Context, defaultPath string) (string, error) {

	if defaultPath == "" {
		defaultPath = GetDefaultMusicPath()
	}

	options := wailsRuntime.OpenDialogOptions{
		Title:            "Select Download Folder",
		DefaultDirectory: defaultPath,
	}

	selectedPath, err := wailsRuntime.OpenDirectoryDialog(ctx, options)
	if err != nil {
		return "", err
	}

	if selectedPath == "" {
		return "", nil
	}

	return selectedPath, nil
}

func SelectMultipleFiles(ctx context.Context) ([]string, error) {
	files, err := wailsRuntime.OpenMultipleFilesDialog(ctx, wailsRuntime.OpenDialogOptions{
		Title: "Select Audio Files",
		Filters: []wailsRuntime.FileFilter{
			{
				DisplayName: "Audio Files (*.mp3, *.m4a, *.flac, *.aac)",
				Pattern:     "*.mp3;*.m4a;*.flac;*.aac",
			},
			{
				DisplayName: "MP3 Files (*.mp3)",
				Pattern:     "*.mp3",
			},
			{
				DisplayName: "M4A Files (*.m4a)",
				Pattern:     "*.m4a",
			},
			{
				DisplayName: "FLAC Files (*.flac)",
				Pattern:     "*.flac",
			},
			{
				DisplayName: "AAC Files (*.aac)",
				Pattern:     "*.aac",
			},
			{
				DisplayName: "All Files (*.*)",
				Pattern:     "*.*",
			},
		},
	})
	if err != nil {
		return nil, err
	}
	return files, nil
}
