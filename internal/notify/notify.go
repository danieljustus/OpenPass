// Package notify provides cross-platform desktop notifications for security events.
//
// Supported platforms:
//   - macOS: osascript (display notification)
//   - Linux: notify-send (libnotify)
//   - Other: no-op (log-only)
//
// Notifications are best-effort and never block the caller.
package notify

import (
	"fmt"
	"log/slog"
	"os/exec"
	"runtime"
)

// Notify displays a desktop notification with the given title and message.
// Returns nil on success (or when the platform doesn't support notifications).
// Errors are logged but never returned to avoid blocking the caller.
func Notify(title, message string) {
	cmd := buildCommand(title, message)
	if cmd == nil {
		// Unsupported platform — just log.
		slog.Debug("desktop notification not supported on " + runtime.GOOS)
		return
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		slog.Warn("desktop notification failed", "error", err, "output", string(out))
	}
}

// AlertNotify sends a high-urgency notification for security events.
// On supported platforms, this uses the "alert" or "critical" urgency level.
func AlertNotify(title, message string) {
	NotifyCritical(title, message)
}

// NotifyCritical sends a critical-urgency notification (persistent on macOS,
// critical on Linux). This should only be used for security events that
// require immediate user attention.
func NotifyCritical(title, message string) {
	cmd := buildCommand(title, message)
	if cmd == nil {
		slog.Debug("desktop notification not supported on " + runtime.GOOS)
		return
	}

	// macOS: use with sound for critical alerts
	if runtime.GOOS == "darwin" {
		cmd = exec.Command("osascript", "-e",
			fmt.Sprintf(`display notification %q with title %q subtitle %q sound name "default"`,
				message, "SECURITY ALERT", title))
	}

	// Linux: use critical urgency
	if runtime.GOOS == "linux" {
		cmd = exec.Command("notify-send", "--urgency=critical", title, message)
	}

	if cmd == nil {
		return
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		slog.Warn("critical notification failed", "error", err, "output", string(out))
	}
}

func buildCommand(title, message string) *exec.Cmd {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("osascript", "-e",
			fmt.Sprintf(`display notification %q with title %q`, message, title))
	case "linux":
		return exec.Command("notify-send", title, message)
	default:
		return nil
	}
}
