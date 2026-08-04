//go:build !windows

package main

import (
	"syscall"
)

// setReusePort sets SO_REUSEADDR and SO_REUSEPORT socket options.
// This is used to optimize socket usage under heavy load.
func setReusePort(fd uintptr) error {
	err := syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1)
	if err != nil {
		return err
	}
	// 15 is SO_REUSEPORT on Linux/macOS
	return syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, 15, 1)
}
