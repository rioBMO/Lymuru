//go:build windows

package main

import (
	"syscall"
	"unsafe"
)

// showFatalDialog displays a blocking error message box (MB_OK | MB_ICONERROR).
func showFatalDialog(title, msg string) {
	t, err1 := syscall.UTF16PtrFromString(title)
	m, err2 := syscall.UTF16PtrFromString(msg)
	if err1 != nil || err2 != nil {
		return
	}
	const mbIconError = 0x00000010
	proc := syscall.NewLazyDLL("user32.dll").NewProc("MessageBoxW")
	_, _, _ = proc.Call(0, uintptr(unsafe.Pointer(m)), uintptr(unsafe.Pointer(t)), mbIconError)
}
