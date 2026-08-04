//go:build windows

package main

// setReusePort skips setting SO_REUSEPORT on Windows.
// Windows does not support SO_REUSEPORT, and SO_REUSEADDR requires syscall.Handle instead of int.
// For the purpose of this mock server, we safely skip setting these options on Windows.
func setReusePort(fd uintptr) error {
	return nil
}
