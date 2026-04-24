//go:build windows

package main

import (
	"os"
	"time"
)

// peekByte attempts to read one byte from stdin within a short deadline.
// On Windows, term.MakeRaw enables ENABLE_VIRTUAL_TERMINAL_INPUT so arrow keys
// arrive as complete VT escape sequences. A 50 ms window is enough to separate
// a lone ESC from the sequence bytes that follow immediately.
//
// Note: if the deadline fires (lone ESC case) the background goroutine remains
// alive until the user presses the next key, at which point that byte is
// silently dropped. This is an acceptable trade-off for interactive menus where
// ESC means "cancel" and no further input is expected in that session.
func peekByte() (byte, bool) {
	ch := make(chan byte, 1)
	go func() {
		b := make([]byte, 1)
		if _, err := os.Stdin.Read(b); err == nil {
			ch <- b[0]
		}
	}()
	select {
	case b := <-ch:
		return b, true
	case <-time.After(50 * time.Millisecond):
		return 0, false
	}
}
