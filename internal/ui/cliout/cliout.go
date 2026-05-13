// Package cliout provides consistent colored CLI output that respects
// --quiet and NO_COLOR settings.
package cliout

import (
	"fmt"
	"os"
	"sync"

	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"

	"github.com/danieljustus/OpenPass/internal/ui/theme"
)

var (
	quiet bool
	mu    sync.RWMutex
)

// SetQuiet enables or disables quiet mode.
func SetQuiet(v bool) {
	mu.Lock()
	quiet = v
	mu.Unlock()
}

func isQuiet() bool {
	mu.RLock()
	defer mu.RUnlock()
	return quiet
}

func noColor() bool {
	if os.Getenv("NO_COLOR") != "" {
		return true
	}
	// Suppress ANSI when stderr is redirected so log files stay readable.
	return !term.IsTerminal(int(os.Stderr.Fd()))
}

func colorize(style lipgloss.Style, format string, args ...any) string {
	if noColor() {
		return fmt.Sprintf(format, args...)
	}
	return style.Render(fmt.Sprintf(format, args...))
}

// Errorf prints a red error message to stderr unless quiet mode is enabled.
func Errorf(format string, args ...any) {
	if isQuiet() {
		return
	}
	fmt.Fprintln(os.Stderr, colorize(theme.ErrorStyle, format, args...))
}

// Warnf prints a yellow warning message to stderr unless quiet mode is enabled.
func Warnf(format string, args ...any) {
	if isQuiet() {
		return
	}
	fmt.Fprintln(os.Stderr, colorize(theme.WarnStyle, format, args...))
}

// Hintf prints a green success hint message to stderr unless quiet mode is enabled.
func Hintf(format string, args ...any) {
	if isQuiet() {
		return
	}
	fmt.Fprintln(os.Stderr, colorize(theme.SuccessStyle, format, args...))
}

// ColorizeSuccess returns text styled with the success/green color.
// It respects NO_COLOR and terminal detection.
func ColorizeSuccess(text string) string {
	if noColor() {
		return text
	}
	return theme.SuccessStyle.Render(text)
}

// ColorizeWarn returns text styled with the warning/yellow color.
// It respects NO_COLOR and terminal detection.
func ColorizeWarn(text string) string {
	if noColor() {
		return text
	}
	return theme.WarnStyle.Render(text)
}

// ColorizeError returns text styled with the error/red color.
// It respects NO_COLOR and terminal detection.
func ColorizeError(text string) string {
	if noColor() {
		return text
	}
	return theme.ErrorStyle.Render(text)
}

// ColorizeDim returns text styled with muted/dim foreground color.
// It respects NO_COLOR and terminal detection.
func ColorizeDim(text string) string {
	if noColor() {
		return text
	}
	return theme.DimStyle.Render(text)
}
