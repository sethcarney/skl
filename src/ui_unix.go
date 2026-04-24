//go:build !windows

package main

import "golang.org/x/sys/unix"

// peekByte does a single non-blocking read from stdin.
// Returns the byte and true if one was immediately available, 0/false otherwise.
// Used to distinguish a lone ESC keypress from the start of an escape sequence.
func peekByte() (byte, bool) {
	if err := unix.SetNonblock(stdinFd, true); err != nil {
		return 0, false
	}
	defer unix.SetNonblock(stdinFd, false)
	b := make([]byte, 1)
	n, _ := unix.Read(stdinFd, b)
	if n <= 0 {
		return 0, false
	}
	return b[0], true
}
