//go:build !windows
// +build !windows

package backend

import (
	"os/exec"
)

func setHideWindow(cmd *exec.Cmd) {
	// No-op on non-Windows platforms.
}
