package main

import (
	"fmt"
	"os"
	"strings"
	"syscall"
	"time"

	"golang.org/x/term"
)

// ANSI sequences for cursor / screen control
const (
	ansiUp        = "\033[A"
	ansiClearLine = "\033[2K"
	ansiHideCursor = "\033[?25l"
	ansiShowCursor = "\033[?25h"
	ansiCR        = "\r"
)

// stdinFd caches the stdin file descriptor once so we never call Fd() repeatedly
// (each call sets the fd to blocking mode in Go's runtime scheduler).
var stdinFd = int(os.Stdin.Fd())

// ─── Log helpers ───────────────────────────────────────────────────────────────

func logInfo(msg string) {
	fmt.Printf("  %s•%s %s\n", ansiDim, ansiReset, msg)
}

func logSuccess(msg string) {
	fmt.Printf("  %s✓%s %s\n", ansiText, ansiReset, msg)
}

func logWarn(msg string) {
	fmt.Printf("  %s!%s %s\n", ansiText, ansiReset, msg)
}

func logError(msg string) {
	fmt.Fprintf(os.Stderr, "  %s✗%s %s\n", ansiText, ansiReset, msg)
}

func intro(title string) {
	fmt.Printf("\n%s%s%s\n\n", ansiText, title, ansiReset)
}

func outro(message string) {
	fmt.Printf("\n%s%s%s\n\n", ansiDim, message, ansiReset)
}

// ─── Note / box ────────────────────────────────────────────────────────────────

func note(content, title string) {
	lines := strings.Split(content, "\n")
	width := len(title) + 4
	for _, l := range lines {
		if len(l)+4 > width {
			width = len(l) + 4
		}
	}

	border := strings.Repeat("─", width-2)
	if title != "" {
		fmt.Printf("%s┌─ %s%s%s %s─┐%s\n", ansiDim, ansiReset, title, ansiDim, strings.Repeat("─", max(0, width-len(title)-5)), ansiReset)
	} else {
		fmt.Printf("%s┌%s┐%s\n", ansiDim, border, ansiReset)
	}
	for _, l := range lines {
		pad := width - len(l) - 3
		if pad < 0 {
			pad = 0
		}
		fmt.Printf("%s│%s %s%s%s\n", ansiDim, ansiReset, l, strings.Repeat(" ", pad), ansiDim+"│"+ansiReset)
	}
	fmt.Printf("%s└%s┘%s\n", ansiDim, border, ansiReset)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// ─── Spinner ───────────────────────────────────────────────────────────────────

type Spinner struct {
	msg  string
	done chan struct{}
}

func NewSpinner(msg string) *Spinner {
	s := &Spinner{msg: msg, done: make(chan struct{})}
	go s.run()
	return s
}

func (s *Spinner) run() {
	frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	i := 0
	fmt.Print(ansiHideCursor)
	for {
		select {
		case <-s.done:
			return
		default:
			fmt.Printf("%s%s%s %s%s", ansiCR, ansiDim, frames[i%len(frames)], s.msg, ansiReset)
			i++
			// ~80ms per frame
			select {
			case <-s.done:
				return
			case <-after(80):
			}
		}
	}
}

func (s *Spinner) Stop(msg string) {
	close(s.done)
	fmt.Print(ansiShowCursor)
	fmt.Printf("%s%s%s %s\n", ansiCR, ansiClearLine, ansiDim, ansiReset)
	if msg != "" {
		fmt.Printf("  %s%s%s\n", ansiDim, msg, ansiReset)
	}
}

// after returns a channel that closes after ms milliseconds
func after(ms int) <-chan struct{} {
	ch := make(chan struct{})
	go func() {
		time.Sleep(time.Duration(ms) * time.Millisecond)
		close(ch)
	}()
	return ch
}

// ─── Raw terminal helpers ───────────────────────────────────────────────────────

// readKey reads a single keypress from stdin.
//
// The terminal MUST already be in raw mode when this is called (raw mode is
// managed by the calling menu function, not here).  Reading in raw mode the
// whole time prevents the TTY line-discipline from echoing buffered escape
// bytes to stdout while clear/redraw output is in flight, which is what caused
// the "entire menu reprints / scrolls to bottom" symptom.
//
// We read one byte at a time and use a non-blocking peek for the bytes that
// follow an ESC, letting us distinguish a lone ESC keypress from an arrow-key
// escape sequence without any fixed-size buffer fragility.
//
// Returns (key string, ok bool). key is one of: "up", "down", "left", "right",
// "enter", "space", "backspace", "esc", printable char, or "".
func readKey() (string, bool) {
	b := make([]byte, 1)
	if _, err := os.Stdin.Read(b); err != nil {
		return "", false
	}

	switch b[0] {
	case 13: // Enter
		return "enter", true
	case 32: // Space
		return "space", true
	case 127, 8: // Backspace / Delete
		return "backspace", true
	case 3: // Ctrl-C
		return "esc", true
	case 27: // ESC — peek non-blocking to detect escape sequences
		// Switch to non-blocking so we return immediately if no bytes follow.
		syscall.SetNonblock(stdinFd, true)
		peek := make([]byte, 1)
		n, _ := syscall.Read(stdinFd, peek)
		syscall.SetNonblock(stdinFd, false)

		if n <= 0 || (peek[0] != '[' && peek[0] != 'O') {
			// Either a lone ESC key, or an unrecognised sequence prefix.
			return "esc", true
		}

		// Read the direction / command byte (blocking — it must follow promptly).
		cmd := make([]byte, 1)
		if _, err := os.Stdin.Read(cmd); err != nil {
			return "esc", true
		}
		switch cmd[0] {
		case 'A':
			return "up", true
		case 'B':
			return "down", true
		case 'C':
			return "right", true
		case 'D':
			return "left", true
		}
		// Extended sequence (Home, End, PgUp, F1-F12, …): drain silently so
		// the stray bytes don't confuse the next readKey call.
		syscall.SetNonblock(stdinFd, true)
		drain := make([]byte, 16)
		syscall.Read(stdinFd, drain) //nolint:errcheck
		syscall.SetNonblock(stdinFd, false)
		return "", true
	}

	if b[0] >= 32 && b[0] < 127 {
		return string(b[:1]), true
	}
	return "", true
}

// ─── UIOption ──────────────────────────────────────────────────────────────────

type UIOption struct {
	Label string
	Value string
	Hint  string // shown dim after label
}

// ─── Select (single choice) ────────────────────────────────────────────────────

func uiSelect(message string, options []UIOption) (int, bool) {
	if len(options) == 0 {
		return -1, false
	}

	// Keep the terminal in raw mode for the entire interaction.  Toggling raw
	// mode around each individual Read was the root cause of the arrow-key bug:
	// in the cooked-mode windows between reads the TTY echoed buffered escape
	// bytes to stdout, corrupting cursor position tracking.
	oldState, err := term.MakeRaw(stdinFd)
	if err != nil {
		return -1, false
	}
	defer term.Restore(stdinFd, oldState)

	cursor := 0

	// term.MakeRaw disables OPOST, so \n no longer implies a carriage return.
	// All print helpers inside this function use \r\n explicitly.
	printSelect := func() {
		fmt.Printf("%s%s%s\r\n", ansiText, message, ansiReset)
		for i, opt := range options {
			if i == cursor {
				fmt.Printf("  %s❯%s %s", ansiText, ansiReset, opt.Label)
			} else {
				fmt.Printf("    %s%s%s", ansiDim, opt.Label, ansiReset)
			}
			if opt.Hint != "" {
				fmt.Printf("  %s%s%s", ansiDim, opt.Hint, ansiReset)
			}
			fmt.Print("\r\n")
		}
	}

	clearSelect := func(n int) {
		for i := 0; i < n+1; i++ {
			fmt.Print(ansiUp + ansiCR + ansiClearLine)
		}
	}

	fmt.Print(ansiHideCursor)
	defer fmt.Print(ansiShowCursor)

	printSelect()

	for {
		key, ok := readKey()
		if !ok {
			return -1, false
		}

		clearSelect(len(options))

		switch key {
		case "up":
			if cursor > 0 {
				cursor--
			}
		case "down":
			if cursor < len(options)-1 {
				cursor++
			}
		case "enter":
			fmt.Printf("%s%s%s  %s%s%s\r\n", ansiText, message, ansiReset, ansiDim, options[cursor].Label, ansiReset)
			return cursor, true
		case "esc":
			fmt.Printf("%s%s%s\r\n", ansiDim, message, ansiReset)
			return -1, false
		}

		printSelect()
	}
}

// ─── Confirm ───────────────────────────────────────────────────────────────────

func uiConfirm(message string) (bool, bool) {
	idx, ok := uiSelect(message, []UIOption{
		{Label: "Yes"},
		{Label: "No"},
	})
	if !ok {
		return false, false
	}
	return idx == 0, true
}

// ─── Multiselect ───────────────────────────────────────────────────────────────

// uiMultiselect shows a checkbox list. Returns selected indices.
// required: at least one must be chosen.
// locked: indices that are pre-checked and cannot be unchecked.
func uiMultiselect(message string, options []UIOption, required bool, initialSelected []int, locked []int) ([]int, bool) {
	if len(options) == 0 {
		return nil, false
	}

	oldState, err := term.MakeRaw(stdinFd)
	if err != nil {
		return nil, false
	}
	defer term.Restore(stdinFd, oldState)

	selected := make([]bool, len(options))
	for _, i := range initialSelected {
		if i >= 0 && i < len(options) {
			selected[i] = true
		}
	}
	lockedSet := make(map[int]bool)
	for _, i := range locked {
		lockedSet[i] = true
		if i >= 0 && i < len(options) {
			selected[i] = true
		}
	}

	cursor := 0

	printMulti := func() {
		fmt.Printf("%s%s%s\r\n", ansiText, message, ansiReset)
		for i, opt := range options {
			isLocked := lockedSet[i]
			checkbox := "○"
			if selected[i] {
				checkbox = "●"
			}
			if isLocked {
				checkbox = "◉"
			}
			if i == cursor {
				fmt.Printf("  %s❯%s %s%s%s %s", ansiText, ansiReset, ansiText, checkbox, ansiReset, opt.Label)
			} else {
				fmt.Printf("    %s%s%s %s%s%s", ansiDim, checkbox, ansiReset, ansiDim, opt.Label, ansiReset)
			}
			if opt.Hint != "" {
				fmt.Printf("  %s%s%s", ansiDim, opt.Hint, ansiReset)
			}
			fmt.Print("\r\n")
		}
		fmt.Printf("%sspace to toggle, enter to confirm%s\r\n", ansiDim, ansiReset)
	}

	clearMulti := func(n int) {
		for i := 0; i < n+2; i++ {
			fmt.Print(ansiUp + ansiCR + ansiClearLine)
		}
	}

	fmt.Print(ansiHideCursor)
	defer fmt.Print(ansiShowCursor)

	printMulti()

	for {
		key, ok := readKey()
		if !ok {
			return nil, false
		}

		clearMulti(len(options))

		switch key {
		case "up":
			if cursor > 0 {
				cursor--
			}
		case "down":
			if cursor < len(options)-1 {
				cursor++
			}
		case "space":
			if !lockedSet[cursor] {
				selected[cursor] = !selected[cursor]
			}
		case "enter":
			var result []int
			for i, s := range selected {
				if s {
					result = append(result, i)
				}
			}
			if required && len(result) == 0 {
				// show again
			} else {
				var labels []string
				for _, i := range result {
					labels = append(labels, options[i].Label)
				}
				fmt.Printf("%s%s%s  %s%s%s\r\n", ansiText, message, ansiReset, ansiDim, strings.Join(labels, ", "), ansiReset)
				return result, true
			}
		case "esc":
			fmt.Printf("%s%s%s\r\n", ansiDim, message, ansiReset)
			return nil, false
		}

		printMulti()
	}
}

// ─── SearchMultiselect ─────────────────────────────────────────────────────────

// uiSearchMultiselect is a searchable multiselect. locked options are always
// shown at the top and cannot be deselected (they represent universal agents).
func uiSearchMultiselect(message string, options []UIOption, locked []UIOption, initialSelected []int) ([]int, bool) {
	oldState, err := term.MakeRaw(stdinFd)
	if err != nil {
		return nil, false
	}
	defer term.Restore(stdinFd, oldState)

	query := ""
	selected := make(map[int]bool) // index into options (non-locked)
	for _, i := range initialSelected {
		if i >= 0 && i < len(options) {
			selected[i] = true
		}
	}
	cursor := 0
	showLocked := len(locked) > 0

	getFiltered := func() []int {
		var indices []int
		q := strings.ToLower(query)
		for i, o := range options {
			if q == "" || strings.Contains(strings.ToLower(o.Label), q) || strings.Contains(strings.ToLower(o.Hint), q) {
				indices = append(indices, i)
			}
		}
		return indices
	}

	printSearch := func(filtered []int) {
		fmt.Printf("%s%s%s\r\n", ansiText, message, ansiReset)
		// Search box
		fmt.Printf("  %s[%s%s%s]%s\r\n", ansiDim, ansiReset, query, ansiDim, ansiReset)

		// Locked section
		if showLocked {
			fmt.Printf("  %s── always included ──%s\r\n", ansiDim, ansiReset)
			for _, lo := range locked {
				fmt.Printf("    %s◉%s %s%s%s", ansiDim, ansiReset, ansiDim, lo.Label, ansiReset)
				if lo.Hint != "" {
					fmt.Printf("  %s%s%s", ansiDim, lo.Hint, ansiReset)
				}
				fmt.Print("\r\n")
			}
		}

		// Filtered options
		if len(filtered) == 0 {
			fmt.Printf("  %sno matches%s\r\n", ansiDim, ansiReset)
		} else {
			for idx, fi := range filtered {
				opt := options[fi]
				checkbox := "○"
				if selected[fi] {
					checkbox = "●"
				}
				if idx == cursor {
					fmt.Printf("  %s❯%s %s%s%s %s", ansiText, ansiReset, ansiText, checkbox, ansiReset, opt.Label)
				} else {
					fmt.Printf("    %s%s%s %s%s%s", ansiDim, checkbox, ansiReset, ansiDim, opt.Label, ansiReset)
				}
				if opt.Hint != "" {
					fmt.Printf("  %s%s%s", ansiDim, opt.Hint, ansiReset)
				}
				fmt.Print("\r\n")
			}
		}
		hintLine := "type to filter, space to toggle, enter to confirm"
		fmt.Printf("%s%s%s\r\n", ansiDim, hintLine, ansiReset)
	}

	countLines := func(filtered []int) int {
		n := 3 // message + search box + hint
		if showLocked {
			n += len(locked) + 1
		}
		if len(filtered) == 0 {
			n++
		} else {
			n += len(filtered)
		}
		return n
	}

	clearSearch := func(filtered []int) {
		for i := 0; i < countLines(filtered); i++ {
			fmt.Print(ansiUp + ansiCR + ansiClearLine)
		}
	}

	fmt.Print(ansiHideCursor)
	defer fmt.Print(ansiShowCursor)

	filtered := getFiltered()
	printSearch(filtered)

	for {
		key, ok := readKey()
		if !ok {
			return nil, false
		}

		prevFiltered := filtered
		clearSearch(prevFiltered)

		switch key {
		case "up":
			if cursor > 0 {
				cursor--
			}
		case "down":
			if len(filtered) > 0 && cursor < len(filtered)-1 {
				cursor++
			}
		case "space":
			if len(filtered) > 0 && cursor < len(filtered) {
				fi := filtered[cursor]
				selected[fi] = !selected[fi]
			}
		case "backspace":
			if len(query) > 0 {
				query = query[:len(query)-1]
				filtered = getFiltered()
				if cursor >= len(filtered) {
					cursor = max(0, len(filtered)-1)
				}
			}
		case "enter":
			var result []int
			for i, s := range selected {
				if s {
					result = append(result, i)
				}
			}
			var labels []string
			for _, i := range result {
				labels = append(labels, options[i].Label)
			}
			for _, lo := range locked {
				labels = append([]string{lo.Label}, labels...)
			}
			fmt.Printf("%s%s%s  %s%s%s\r\n", ansiText, message, ansiReset, ansiDim, strings.Join(labels, ", "), ansiReset)
			return result, true
		case "esc":
			fmt.Printf("%s%s%s\r\n", ansiDim, message, ansiReset)
			return nil, false
		default:
			if len(key) == 1 {
				query += key
				filtered = getFiltered()
				if cursor >= len(filtered) {
					cursor = max(0, len(filtered)-1)
				}
			}
		}

		filtered = getFiltered()
		printSearch(filtered)
	}
}
