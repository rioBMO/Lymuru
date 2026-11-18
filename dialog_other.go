//go:build !windows

package main

// showFatalDialog is a no-op off Windows; the error still goes to stderr and the log.
func showFatalDialog(title, msg string) {}
