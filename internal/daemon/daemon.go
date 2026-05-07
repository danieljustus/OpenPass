// Package daemon provides cross-platform service management for
// installing, uninstalling, and checking the status of the OpenPass
// MCP server as a background service (launchd on macOS, systemd on Linux).
package daemon

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"text/template"

	"github.com/danieljustus/OpenPass/internal/config"
	errorspkg "github.com/danieljustus/OpenPass/internal/errors"
)

const (
	// macOS launchd paths
	launchAgentDir      = "LaunchAgents"
	launchAgentLabel    = "com.openpass.mcp"
	launchAgentFileName = "com.openpass.mcp.plist"
	logDir              = "Logs"
	logFileName         = "openpass-mcp.log"
	errorLogFileName    = "openpass-mcp.error.log"

	// Linux systemd paths
	systemdUserDir  = ".config/systemd/user"
	systemdUnitName = "openpass-mcp.service"
)

// macOS launchd plist template
const plistTmpl = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.openpass.mcp</string>
    <key>ProgramArguments</key>
    <array>
        <string>{{.BinaryPath}}</string>
        <string>serve</string>
        <string>--port</string>
        <string>{{.PortStr}}</string>
        <string>--bind</string>
        <string>{{.Bind}}</string>
    </array>
    <key>EnvironmentVariables</key>
    <dict>
        <key>OPENPASS_VAULT</key>
        <string>{{.VaultDir}}</string>
    </dict>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>{{.LogPath}}</string>
    <key>StandardErrorPath</key>
    <string>{{.ErrorLogPath}}</string>
</dict>
</plist>`

// Linux systemd service template
const systemdTmpl = `[Unit]
Description=OpenPass MCP Server

[Service]
Type=simple
ExecStart={{.BinaryPath}} serve --port {{.PortStr}} --bind {{.Bind}}
Environment=OPENPASS_VAULT={{.VaultDir}}
Restart=on-failure

[Install]
WantedBy=default.target
`

// tmplData holds the values substituted into service file templates.
type tmplData struct {
	BinaryPath   string
	VaultDir     string
	PortStr      string
	Bind         string
	LogPath      string
	ErrorLogPath string
}

// Installer manages the OpenPass MCP background service on the current platform.
type Installer struct {
	binaryPath string
	vaultDir   string
	port       int
	bind       string
	logPath    string
	errLogPath string
}

// NewInstaller creates an Installer for the given configuration and vault directory.
// If cfg or cfg.MCP is nil, default port (8080) and bind (127.0.0.1) are used.
func NewInstaller(cfg *config.Config, vaultDir string) (*Installer, error) {
	binaryPath, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("get executable path: %w", err)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("get home directory: %w", err)
	}

	port := 8080
	bind := "127.0.0.1"
	if cfg != nil && cfg.MCP != nil {
		if cfg.MCP.Port > 0 {
			port = cfg.MCP.Port
		}
		if cfg.MCP.Bind != "" {
			bind = cfg.MCP.Bind
		}
	}

	return &Installer{
		binaryPath: binaryPath,
		vaultDir:   vaultDir,
		port:       port,
		bind:       bind,
		logPath:    filepath.Join(home, logDir, logFileName),
		errLogPath: filepath.Join(home, logDir, errorLogFileName),
	}, nil
}

// Install installs the MCP server as a background service on the current platform.
func (i *Installer) Install() error {
	switch runtime.GOOS {
	case "darwin":
		return i.installDarwin()
	case "linux":
		return i.installLinux()
	default:
		return errorspkg.NewCLIError(errorspkg.ExitGeneralError,
			fmt.Sprintf("unsupported platform: %s; service templates are available for macOS (launchd) and Linux (systemd)", runtime.GOOS),
			nil)
	}
}

// Uninstall removes the MCP server background service from the current platform.
func (i *Installer) Uninstall() error {
	switch runtime.GOOS {
	case "darwin":
		return i.uninstallDarwin()
	case "linux":
		return i.uninstallLinux()
	default:
		return errorspkg.NewCLIError(errorspkg.ExitGeneralError,
			fmt.Sprintf("unsupported platform: %s", runtime.GOOS),
			nil)
	}
}

// Status returns the service status: "running", "stopped", or "not installed".
func (i *Installer) Status() (string, error) {
	switch runtime.GOOS {
	case "darwin":
		return i.statusDarwin()
	case "linux":
		return i.statusLinux()
	default:
		return "", errorspkg.NewCLIError(errorspkg.ExitGeneralError,
			fmt.Sprintf("unsupported platform: %s", runtime.GOOS),
			nil)
	}
}

// VaultDir returns the vault directory configured for the service.
func (i *Installer) VaultDir() string {
	return i.vaultDir
}

// Port returns the port configured for the service.
func (i *Installer) Port() int {
	return i.port
}

// Bind returns the bind address configured for the service.
func (i *Installer) Bind() string {
	return i.bind
}

// BinaryPath returns the path to the openpass binary.
func (i *Installer) BinaryPath() string {
	return i.binaryPath
}

// ServiceFilePath returns the path to the service definition file for the current platform.
func (i *Installer) ServiceFilePath() (string, error) {
	switch runtime.GOOS {
	case "darwin":
		return i.darwinPlistPath()
	case "linux":
		return i.linuxServicePath()
	default:
		return "", fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

// ---------------------------------------------------------------------------
// Darwin (macOS launchd)
// ---------------------------------------------------------------------------

func (i *Installer) darwinPlistPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get home directory: %w", err)
	}
	return filepath.Join(home, launchAgentDir, launchAgentFileName), nil
}

func (i *Installer) installDarwin() error {
	plistPath, err := i.darwinPlistPath()
	if err != nil {
		return err
	}

	if err := i.writePlist(plistPath); err != nil {
		return errorspkg.NewCLIError(errorspkg.ExitPermissionDenied,
			"failed to write launchd plist", err)
	}

	// Unload any existing instance first (ignore errors)
	// #nosec G204 -- plistPath is generated internally by darwinPlistPath()
	_ = exec.Command("launchctl", "unload", plistPath).Run()

	// #nosec G204 -- plistPath is generated internally by darwinPlistPath()
	if out, err := exec.Command("launchctl", "load", plistPath).CombinedOutput(); err != nil {
		return errorspkg.NewCLIError(errorspkg.ExitGeneralError,
			fmt.Sprintf("failed to load launchd service: %s", strings.TrimSpace(string(out))),
			err)
	}

	return nil
}

func (i *Installer) uninstallDarwin() error {
	plistPath, err := i.darwinPlistPath()
	if err != nil {
		return err
	}

	// Try to unload (best-effort)
	// #nosec G204 -- plistPath is generated internally by darwinPlistPath()
	_ = exec.Command("launchctl", "unload", plistPath).Run()

	if err := os.Remove(plistPath); err != nil && !os.IsNotExist(err) {
		return errorspkg.NewCLIError(errorspkg.ExitPermissionDenied,
			"failed to remove launchd plist", err)
	}

	return nil
}

func (i *Installer) statusDarwin() (string, error) {
	plistPath, err := i.darwinPlistPath()
	if err != nil {
		return "", err
	}

	if _, statErr := os.Stat(plistPath); os.IsNotExist(statErr) {
		return "not installed", nil
	}

	out, err := exec.Command("launchctl", "list", launchAgentLabel).CombinedOutput()
	if err != nil {
		return "stopped", nil
	}

	// launchctl list outputs status, PID, name on lines
	output := strings.TrimSpace(string(out))
	if strings.Contains(output, launchAgentLabel) {
		fields := strings.Fields(output)
		if len(fields) >= 2 && fields[0] != "-" {
			return "running", nil
		}
	}

	return "stopped", nil
}

func (i *Installer) writePlist(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(i.logPath), 0o700); err != nil {
		return fmt.Errorf("create log directory: %w", err)
	}

	data := tmplData{
		BinaryPath:   i.binaryPath,
		VaultDir:     i.vaultDir,
		PortStr:      strconv.Itoa(i.port),
		Bind:         i.bind,
		LogPath:      i.logPath,
		ErrorLogPath: i.errLogPath,
	}

	tmpl, err := template.New("plist").Parse(plistTmpl)
	if err != nil {
		return fmt.Errorf("parse plist template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("render plist template: %w", err)
	}

	if err := os.WriteFile(path, buf.Bytes(), 0o600); err != nil {
		return fmt.Errorf("write plist: %w", err)
	}

	return nil
}

// ---------------------------------------------------------------------------
// Linux (systemd user)
// ---------------------------------------------------------------------------

func (i *Installer) linuxServicePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get home directory: %w", err)
	}
	return filepath.Join(home, systemdUserDir, systemdUnitName), nil
}

func (i *Installer) installLinux() error {
	svcPath, err := i.linuxServicePath()
	if err != nil {
		return err
	}

	if err := i.writeService(svcPath); err != nil {
		return errorspkg.NewCLIError(errorspkg.ExitPermissionDenied,
			"failed to write systemd service file", err)
	}

	if out, err := exec.Command("systemctl", "--user", "daemon-reload").CombinedOutput(); err != nil {
		return errorspkg.NewCLIError(errorspkg.ExitGeneralError,
			fmt.Sprintf("systemctl daemon-reload failed: %s", strings.TrimSpace(string(out))),
			err)
	}

	if out, err := exec.Command("systemctl", "--user", "enable", "openpass-mcp").CombinedOutput(); err != nil {
		return errorspkg.NewCLIError(errorspkg.ExitGeneralError,
			fmt.Sprintf("systemctl enable failed: %s", strings.TrimSpace(string(out))),
			err)
	}

	if out, err := exec.Command("systemctl", "--user", "start", "openpass-mcp").CombinedOutput(); err != nil {
		return errorspkg.NewCLIError(errorspkg.ExitGeneralError,
			fmt.Sprintf("systemctl start failed: %s", strings.TrimSpace(string(out))),
			err)
	}

	return nil
}

func (i *Installer) uninstallLinux() error {
	// Best-effort stop and disable
	_ = exec.Command("systemctl", "--user", "stop", "openpass-mcp").Run()
	_ = exec.Command("systemctl", "--user", "disable", "openpass-mcp").Run()

	svcPath, err := i.linuxServicePath()
	if err != nil {
		return err
	}

	if err := os.Remove(svcPath); err != nil && !os.IsNotExist(err) {
		return errorspkg.NewCLIError(errorspkg.ExitPermissionDenied,
			"failed to remove systemd service file", err)
	}

	// Reload to pick up the removal
	_ = exec.Command("systemctl", "--user", "daemon-reload").Run()

	return nil
}

func (i *Installer) statusLinux() (string, error) {
	svcPath, err := i.linuxServicePath()
	if err != nil {
		return "", err
	}

	if _, statErr := os.Stat(svcPath); os.IsNotExist(statErr) {
		return "not installed", nil
	}

	out, err := exec.Command("systemctl", "--user", "is-active", "openpass-mcp").CombinedOutput()
	if err != nil {
		output := strings.TrimSpace(string(out))
		if output == "inactive" || output == "failed" {
			return "stopped", nil
		}
		return "stopped", nil
	}

	output := strings.TrimSpace(string(out))
	if output == "active" {
		return "running", nil
	}

	return "stopped", nil
}

func (i *Installer) writeService(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	data := tmplData{
		BinaryPath: i.binaryPath,
		VaultDir:   i.vaultDir,
		PortStr:    strconv.Itoa(i.port),
		Bind:       i.bind,
	}

	tmpl, err := template.New("service").Parse(systemdTmpl)
	if err != nil {
		return fmt.Errorf("parse service template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("render service template: %w", err)
	}

	if err := os.WriteFile(path, buf.Bytes(), 0o600); err != nil {
		return fmt.Errorf("write service file: %w", err)
	}

	return nil
}
