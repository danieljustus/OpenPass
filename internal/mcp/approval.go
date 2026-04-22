package mcp

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/mattn/go-tty"
)

// ttyDevice abstracts a TTY for testability.
type ttyDevice interface {
	ReadString() (string, error)
	Input() *os.File
	Output() *os.File
	Raw() (func(), error)
	Close() error
}

type ttyWrapper struct {
	*tty.TTY
}

func (w *ttyWrapper) Raw() (func(), error) {
	restore, err := w.TTY.Raw()
	if err != nil {
		return nil, err
	}
	return func() { _ = restore() }, nil
}

var openTTYDevice = func() (ttyDevice, error) {
	dev, err := tty.Open()
	if err != nil {
		return nil, err
	}
	return &ttyWrapper{TTY: dev}, nil
}

// ApprovalRequest represents a request for user approval of a sensitive operation
type ApprovalRequest struct {
	Operation string
	Details   string
	Timeout   time.Duration
}

// ApprovalResult represents the outcome of an approval request
type ApprovalResult struct {
	Error    error
	Approved bool
}

// defaultTimeout is the default timeout for approval requests (30 seconds)
const defaultTimeout = 30 * time.Second

// IsTTYPresent checks if a TTY is available for reading and writing.
// Uses go-tty for cross-platform support (works on Unix and Windows).
func IsTTYPresent() bool {
	tt, err := openTTYDevice()
	if err != nil {
		return false
	}
	_ = tt.Close()
	return true
}

// RequestApproval prompts the user via TTY for approval of a sensitive operation.
// It displays the operation details and waits for user input (y/yes to approve).
// The prompt times out after the specified duration (defaults to 30 seconds).
// If TTY is not available, the operation is denied with an error.
func RequestApproval(req ApprovalRequest) ApprovalResult {
	timeout := req.Timeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}

	tt, err := openTTYDevice()
	if err != nil {
		return ApprovalResult{
			Approved: false,
			Error:    fmt.Errorf("approval required but no TTY available (running non-interactively)"),
		}
	}
	defer func() { _ = tt.Close() }()

	restore, err := tt.Raw()
	if err != nil {
		return ApprovalResult{
			Approved: false,
			Error:    fmt.Errorf("failed to set terminal raw mode: %w", err),
		}
	}

	prompt := buildPrompt(req)
	if _, writeErr := tt.Output().WriteString(prompt); writeErr != nil {
		restore()
		return ApprovalResult{
			Approved: false,
			Error:    fmt.Errorf("failed to write to terminal: %w", writeErr),
		}
	}

	response, readErr := readTTYResponse(tt, timeout)
	if readErr != nil {
		restore()
		if isTimeoutError(readErr) {
			return ApprovalResult{
				Approved: false,
				Error:    fmt.Errorf("approval timed out after %v", timeout),
			}
		}
		return ApprovalResult{
			Approved: false,
			Error:    fmt.Errorf("failed to read from terminal: %w", readErr),
		}
	}
	restore()

	approved := parseApprovalResponse(response)

	if approved {
		_, _ = fmt.Fprintln(tt.Output(), "yes")
	} else {
		_, _ = fmt.Fprintln(tt.Output(), "no")
	}

	return ApprovalResult{
		Approved: approved,
		Error:    nil,
	}
}

// buildPrompt creates the approval prompt string
func buildPrompt(req ApprovalRequest) string {
	var sb strings.Builder

	sb.WriteString("\n")
	sb.WriteString("╔══════════════════════════════════════════════════════════════╗\n")
	sb.WriteString("║                MCP OPERATION APPROVAL REQUIRED               ║\n")
	sb.WriteString("╠══════════════════════════════════════════════════════════════╣\n")

	if req.Operation != "" {
		fmt.Fprintf(&sb, "║ Operation: %-50s ║\n", truncate(req.Operation, 50))
	}

	if req.Details != "" {
		fmt.Fprintf(&sb, "║ Details:   %-50s ║\n", truncate(req.Details, 50))
	}

	sb.WriteString("╚══════════════════════════════════════════════════════════════╝\n")
	sb.WriteString("\nApprove this operation? (y/n): ")

	return sb.String()
}

func readTTYResponse(tt ttyDevice, timeout time.Duration) (string, error) {
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

func isTimeoutError(err error) bool {
	if errors.Is(err, os.ErrDeadlineExceeded) {
		return true
	}
	var timeoutErr interface {
		Timeout() bool
	}
	return errors.As(err, &timeoutErr) && timeoutErr.Timeout()
}

// parseApprovalResponse determines if the user approved the operation
// Accepts "y", "yes" (case insensitive) as approval
// Everything else is considered a denial
func parseApprovalResponse(response string) bool {
	lowerResponse := strings.ToLower(strings.TrimSpace(response))
	return lowerResponse == "y" || lowerResponse == "yes"
}

// truncate truncates a string to the specified maximum length
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}
