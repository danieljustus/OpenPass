package config

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/danieljustus/OpenPass/internal/pathutil"
)

const (
	defaultConfigDir      = ".openpass"
	defaultConfigFile     = "config.yaml"
	defaultAgentName      = "default"
	defaultSessionTimeout = 15 * time.Minute
)

type Config struct {
	Agents         map[string]AgentProfile `yaml:"agents"`
	Vault          *VaultConfig            `yaml:"vault,omitempty"`
	Git            *GitConfig              `yaml:"git,omitempty"`
	MCP            *MCPConfig              `yaml:"mcp,omitempty"`
	Clipboard      *ClipboardConfig        `yaml:"clipboard,omitempty"`
	VaultDir       string                  `yaml:"vaultDir"`
	DefaultAgent   string                  `yaml:"defaultAgent"`
	SessionTimeout time.Duration           `yaml:"sessionTimeout"`
	UseTouchID     bool                    `yaml:"useTouchID"`
}

type AgentProfile struct {
	Name            string        `yaml:"-"`
	ApprovalMode    string        `yaml:"approvalMode"`
	AllowedPaths    []string      `yaml:"allowedPaths"`
	RedactFields    []string      `yaml:"redactFields,omitempty"`
	CanWrite        bool          `yaml:"canWrite"`
	RequireApproval bool          `yaml:"requireApproval"`
	ApprovalTimeout time.Duration `yaml:"approvalTimeout,omitempty"`
}

type fileConfig struct {
	Agents         map[string]fileAgentProfile `yaml:"agents,omitempty"`
	Vault          *fileVaultConfig            `yaml:"vault,omitempty"`
	Git            *fileGitConfig              `yaml:"git,omitempty"`
	MCP            *fileMCPConfig              `yaml:"mcp,omitempty"`
	Clipboard      *fileClipboardConfig        `yaml:"clipboard,omitempty"`
	VaultDir       string                      `yaml:"vaultDir,omitempty"`
	DefaultAgent   string                      `yaml:"defaultAgent,omitempty"`
	SessionTimeout time.Duration               `yaml:"sessionTimeout,omitempty"`
	UseTouchID     bool                        `yaml:"useTouchID,omitempty"`
}

type fileAgentProfile struct {
	ApprovalTimeout *time.Duration `yaml:"approvalTimeout,omitempty"`
	CanWrite        *bool          `yaml:"canWrite,omitempty"`
	RequireApproval *bool          `yaml:"requireApproval,omitempty"`
	ApprovalMode    *string        `yaml:"approvalMode,omitempty"`
	AllowedPaths    []string       `yaml:"allowedPaths,omitempty"`
	RedactFields    []string       `yaml:"redactFields,omitempty"`
}

func Default() *Config {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		home = "~"
	}

	return &Config{
		VaultDir:       filepath.Join(home, defaultConfigDir),
		DefaultAgent:   defaultAgentName,
		SessionTimeout: defaultSessionTimeout,
		Agents:         builtinAgentProfiles(),
	}
}

func validateConfigPath(path string) error {
	if pathutil.HasTraversal(path) {
		return errors.New("config file path escapes expected directory")
	}
	return nil
}

func Load(path string) (*Config, error) {
	if err := validateConfigPath(path); err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path) //nosec:G304
	if err != nil {
		return nil, err
	}

	cfg := Default()
	if len(bytes.TrimSpace(data)) == 0 {
		return cfg, nil
	}

	var raw fileConfig
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	if raw.VaultDir != "" {
		cfg.VaultDir = raw.VaultDir
	}
	if raw.DefaultAgent != "" {
		cfg.DefaultAgent = raw.DefaultAgent
	}
	if raw.SessionTimeout > 0 {
		cfg.SessionTimeout = raw.SessionTimeout
	}
	cfg.UseTouchID = raw.UseTouchID

	if raw.Agents != nil {
		for name, profile := range raw.Agents {
			current, ok := cfg.Agents[name]
			if !ok {
				current = newDefaultAgentProfile(name)
			}
			current.Name = name
			if profile.AllowedPaths != nil {
				current.AllowedPaths = append([]string(nil), profile.AllowedPaths...)
			} else if current.AllowedPaths == nil {
				current.AllowedPaths = []string{}
			}
			if profile.CanWrite != nil {
				current.CanWrite = *profile.CanWrite
			}
			if profile.RequireApproval != nil {
				current.RequireApproval = *profile.RequireApproval
			}
			if profile.ApprovalTimeout != nil {
				current.ApprovalTimeout = *profile.ApprovalTimeout
			}
			if profile.ApprovalMode != nil {
				current.ApprovalMode = *profile.ApprovalMode
			} else if profile.RequireApproval != nil {
				if *profile.RequireApproval {
					current.ApprovalMode = "prompt"
				} else {
					current.ApprovalMode = "none"
				}
			}
			if profile.RedactFields != nil {
				current.RedactFields = append([]string(nil), profile.RedactFields...)
			}
			cfg.Agents[name] = current
		}
	}

	// Validate ApprovalMode values
	for name, profile := range cfg.Agents {
		switch profile.ApprovalMode {
		case "", "none", "deny", "prompt":
			// valid
		default:
			return nil, fmt.Errorf("agent %q: invalid approvalMode %q (valid: none, deny, prompt)", name, profile.ApprovalMode)
		}
	}

	if cfg.Agents == nil {
		cfg.Agents = map[string]AgentProfile{}
	}
	for name, profile := range cfg.Agents {
		profile.Name = name
		cfg.Agents[name] = profile
	}
	if _, ok := cfg.Agents[cfg.DefaultAgent]; !ok {
		cfg.Agents[cfg.DefaultAgent] = newDefaultAgentProfile(cfg.DefaultAgent)
	}

	if raw.Vault != nil {
		defaults := defaultVaultConfig()
		merged := MergeFileVaultConfig(raw.Vault, defaults)
		cfg.Vault = &merged
	}
	if raw.Git != nil {
		defaults := defaultGitConfig()
		merged := MergeFileGitConfig(raw.Git, defaults)
		cfg.Git = &merged
	}
	if raw.MCP != nil {
		defaults := defaultMCPConfig()
		merged := MergeFileMCPConfig(raw.MCP, defaults)
		cfg.MCP = &merged
	}
	if raw.Clipboard != nil {
		defaults := defaultClipboardConfig()
		merged := MergeFileClipboardConfig(raw.Clipboard, defaults)
		cfg.Clipboard = &merged
	}

	if cfg.MCP != nil && cfg.MCP.Bind == "" {
		return nil, fmt.Errorf("mcp.bind must not be empty")
	}

	return cfg, nil
}

func (c *Config) Save() error {
	if c == nil {
		return errors.New("config is nil")
	}

	path, err := defaultConfigPath()
	if err != nil {
		return err
	}

	if mkdirErr := os.MkdirAll(filepath.Dir(path), 0o700); mkdirErr != nil {
		return mkdirErr
	}

	raw := fileConfig{
		VaultDir:       c.VaultDir,
		DefaultAgent:   c.DefaultAgent,
		SessionTimeout: c.SessionTimeout,
		UseTouchID:     c.UseTouchID,
		Agents:         make(map[string]fileAgentProfile, len(c.Agents)),
	}

	if c.Vault != nil {
		confirmRemove := c.Vault.ConfirmRemove
		useTouchID := c.Vault.UseTouchID
		raw.Vault = &fileVaultConfig{
			Path:              c.Vault.Path,
			DefaultRecipients: append([]string(nil), c.Vault.DefaultRecipients...),
			ConfirmRemove:     &confirmRemove,
			UseTouchID:        &useTouchID,
		}
	}

	if c.Git != nil {
		autoPush := c.Git.AutoPush
		commitTemplate := c.Git.CommitTemplate
		raw.Git = &fileGitConfig{
			AutoPush:       &autoPush,
			CommitTemplate: &commitTemplate,
		}
	}

	if c.MCP != nil {
		mcpPort := c.MCP.Port
		mcpBind := c.MCP.Bind
		mcpStdio := c.MCP.Stdio
		mcpTokenFile := c.MCP.HTTPTokenFile
		raw.MCP = &fileMCPConfig{
			Port:              &mcpPort,
			Bind:              &mcpBind,
			Stdio:             &mcpStdio,
			HTTPTokenFile:     &mcpTokenFile,
			ReadHeaderTimeout: &c.MCP.ReadHeaderTimeout,
			ReadTimeout:       &c.MCP.ReadTimeout,
			WriteTimeout:      &c.MCP.WriteTimeout,
			ShutdownTimeout:   &c.MCP.ShutdownTimeout,
			ApprovalTimeout:   &c.MCP.ApprovalTimeout,
		}
	}

	if c.Clipboard != nil {
		autoClear := c.Clipboard.AutoClearDuration
		raw.Clipboard = &fileClipboardConfig{
			AutoClearDuration: &autoClear,
		}
	}
	for name, profile := range c.Agents {
		allowed := append([]string(nil), profile.AllowedPaths...)
		canWrite := profile.CanWrite
		requireApproval := profile.RequireApproval
		fap := fileAgentProfile{
			AllowedPaths:    allowed,
			CanWrite:        &canWrite,
			RequireApproval: &requireApproval,
		}
		if profile.ApprovalMode != "" {
			am := profile.ApprovalMode
			fap.ApprovalMode = &am
		}
		if profile.ApprovalTimeout > 0 {
			t := profile.ApprovalTimeout
			fap.ApprovalTimeout = &t
		}
		if profile.RedactFields != nil {
			fap.RedactFields = append([]string(nil), profile.RedactFields...)
		}
		raw.Agents[name] = fap
	}

	data, err := yaml.Marshal(&raw)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

func defaultConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return "", errors.New("unable to determine home directory")
	}
	return filepath.Join(home, defaultConfigDir, defaultConfigFile), nil
}

func newDefaultAgentProfile(name string) AgentProfile {
	return AgentProfile{
		Name:         name,
		AllowedPaths: []string{},
		CanWrite:     false,
		ApprovalMode: "none",
	}
}

func builtinAgentProfiles() map[string]AgentProfile {
	return map[string]AgentProfile{
		"default":     {Name: "default", AllowedPaths: []string{"*"}, CanWrite: false, ApprovalMode: "none"},
		"claude-code": {Name: "claude-code", AllowedPaths: []string{"*"}, CanWrite: true, ApprovalMode: "none"},
		"codex":       {Name: "codex", AllowedPaths: []string{"*"}, CanWrite: false, ApprovalMode: "none"},
		"hermes":      {Name: "hermes", AllowedPaths: []string{"*"}, CanWrite: true, ApprovalMode: "none"},
		"openclaw":    {Name: "openclaw", AllowedPaths: []string{"*"}, CanWrite: true, ApprovalMode: "none"},
		"opencode":    {Name: "opencode", AllowedPaths: []string{"*"}, CanWrite: false, ApprovalMode: "none"},
	}
}
