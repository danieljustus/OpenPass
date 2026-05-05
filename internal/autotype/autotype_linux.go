//go:build linux

package autotype

import (
	"fmt"
	"os/exec"
)

func init() {
	defaultAutotypeFactory = NewLinuxAutotype
}

type linuxAutotype struct{}

func (a *linuxAutotype) Type(text string) error {
	cmd := exec.Command("xdotool", "type", "--delay", "0", text)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("autotype failed (xdotool may not be installed or display not available): %w", err)
	}
	return nil
}

func NewLinuxAutotype() Autotype {
	return &linuxAutotype{}
}
