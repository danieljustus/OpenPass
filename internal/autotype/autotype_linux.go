//go:build linux

// Package autotype provides cross-platform automated typing functionality
// for filling credentials into other applications.
package autotype

import (
	"fmt"
	"os"
	"os/exec"
)

func init() {
	defaultAutotypeFactory = NewLinuxAutotype
}

var (
	lookPath    = exec.LookPath
	execCommand = exec.Command
)

type linuxAutotype struct{}

func (a *linuxAutotype) Type(text string) error {
	if os.Getenv("WAYLAND_DISPLAY") != "" {
		return a.typeWayland(text)
	}
	return a.typeX11(text)
}

func (a *linuxAutotype) typeX11(text string) error {
	if _, err := lookPath("xdotool"); err != nil {
		return fmt.Errorf("autotype failed: xdotool not installed")
	}
	cmd := execCommand("xdotool", "type", "--delay", "0", text)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("autotype failed (xdotool may not be installed or display not available): %w", err)
	}
	return nil
}

func (a *linuxAutotype) typeWayland(text string) error {
	if _, err := lookPath("wtype"); err == nil {
		cmd := execCommand("wtype", text)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("autotype failed (wtype): %w", err)
		}
		return nil
	}

	if _, err := lookPath("ydotool"); err == nil {
		cmd := execCommand("ydotool", "type", text)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("autotype failed (ydotool): %w", err)
		}
		return nil
	}

	return fmt.Errorf("autotype failed: no Wayland typing tool found (install wtype or ydotool)")
}

func NewLinuxAutotype() Autotype {
	return &linuxAutotype{}
}
