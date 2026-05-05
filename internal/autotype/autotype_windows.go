//go:build windows

package autotype

import (
	"fmt"
	"os/exec"
	"strings"
)

func init() {
	defaultAutotypeFactory = NewWindowsAutotype
}

type windowsAutotype struct{}

func (a *windowsAutotype) Type(text string) error {
	escaped := strings.ReplaceAll(text, "'", "''")

	script := fmt.Sprintf("$wshell = New-Object -ComObject WScript.Shell; $wshell.SendKeys('%s')", escaped)
	cmd := exec.Command("powershell.exe", "-Command", script)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("autotype failed: %w", err)
	}
	return nil
}

func NewWindowsAutotype() Autotype {
	return &windowsAutotype{}
}
