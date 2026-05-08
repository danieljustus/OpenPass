//go:build darwin

// Package autotype provides cross-platform automated typing functionality
// for filling credentials into other applications.
package autotype

import (
	"fmt"
	"os/exec"
)

func init() {
	defaultAutotypeFactory = NewDarwinAutotype
}

type darwinAutotype struct{}

func (a *darwinAutotype) Type(text string) error {
	escaped := escapeAppleScriptString(text)

	script := fmt.Sprintf(`tell application "System Events" to keystroke "%s"`, escaped)
	// #nosec G204 -- script is constructed internally for AppleScript keystroke
	cmd := exec.Command("osascript", "-e", script)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("autotype failed: %w", err)
	}
	return nil
}

func NewDarwinAutotype() Autotype {
	return &darwinAutotype{}
}
