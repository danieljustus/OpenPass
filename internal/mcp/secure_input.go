package mcp

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/mattn/go-tty"
)

// secureInputDevice abstracts TTY for secure input (testability)
type secureInputDevice interface {
	ReadString() (string, error)
	Input() *os.File
	Output() *os.File
	Close() error
}

type secureTTYWrapper struct {
	*tty.TTY
}

func (w *secureTTYWrapper) Input() *os.File {
	return w.TTY.Input()
}

func (w *secureTTYWrapper) Output() *os.File {
	return w.TTY.Output()
}

var openSecureTTY = func() (secureInputDevice, error) {
	dev, err := tty.Open()
	if err != nil {
		return nil, err
	}
	return &secureTTYWrapper{TTY: dev}, nil
}

func secureInputToolAvailable(s *Server) bool {
	if s == nil || s.transport != "stdio" {
		return false
	}
	dev, err := openSecureTTY()
	if err != nil {
		return false
	}
	_ = dev.Close()
	return true
}

// SecureInputPrompt reads sensitive data from the user via TTY without echoing input.
// It displays a prompt on the terminal and reads the response with character hiding.
// The value is never exposed to the agent or logged.
// If TTY is not available, returns an error.
func SecureInputPrompt(prompt string, timeout time.Duration) (string, error) {
	if timeout <= 0 {
		timeout = 60 * time.Second
	}

	tt, err := openSecureTTY()
	if err != nil {
		return "", fmt.Errorf("secure input requires an interactive terminal (TTY not available)")
	}
	defer func() { _ = tt.Close() }()

	output := tt.Output()
	if output != nil {
		if _, writeErr := output.WriteString(prompt); writeErr != nil {
			return "", fmt.Errorf("failed to write prompt to terminal: %w", writeErr)
		}
	}

	value, readErr := readSecureInput(tt, timeout)
	if readErr != nil {
		if isTimeoutError(readErr) {
			return "", fmt.Errorf("secure input timed out after %v", timeout)
		}
		return "", fmt.Errorf("failed to read secure input: %w", readErr)
	}

	if output != nil {
		_, _ = fmt.Fprintln(output)
	}

	return strings.TrimSpace(value), nil
}

// readSecureInput reads input from TTY without echoing characters.
// It uses the tty package's ReadString which handles raw mode internally.
func readSecureInput(tt secureInputDevice, timeout time.Duration) (string, error) {
	input := tt.Input()
	if input != nil {
		if err := input.SetReadDeadline(time.Now().Add(timeout)); err != nil {
			return "", err
		}
		defer func() {
			_ = input.SetReadDeadline(time.Time{})
		}()
	}

	return tt.ReadString()
}

// buildSecureInputPrompt creates a secure input prompt string
func buildSecureInputPrompt(path, field, description string) string {
	var sb strings.Builder

	sb.WriteString("\n")
	sb.WriteString("╔══════════════════════════════════════════════════════════════╗\n")
	sb.WriteString("║           SECURE INPUT REQUIRED - AGENT CANNOT SEE          ║\n")
	sb.WriteString("╠══════════════════════════════════════════════════════════════╣\n")

	if path != "" {
		fmt.Fprintf(&sb, "║ Entry:    %-50s ║\n", truncate(path, 50))
	}
	if field != "" {
		fmt.Fprintf(&sb, "║ Field:    %-50s ║\n", truncate(field, 50))
	}
	if description != "" {
		fmt.Fprintf(&sb, "║ Details:  %-50s ║\n", truncate(description, 50))
	}

	sb.WriteString("╚══════════════════════════════════════════════════════════════╝\n")
	sb.WriteString("Enter value (input hidden): ")

	return sb.String()
}
