package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/danieljustus/OpenPass/internal/config"
	"github.com/danieljustus/OpenPass/internal/git"
	vaultpkg "github.com/danieljustus/OpenPass/internal/vault"
)

var initCmd = &cobra.Command{
	Use:   "init [vault-dir]",
	Short: "Initialize a new password vault",
	Long:  "Creates a new vault directory with identity and configuration.",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var (
			vaultDir string
			err      error
		)
		if len(args) > 0 {
			vaultDir, err = expandVaultDir(args[0])
		} else {
			vaultDir, err = vaultPath()
		}
		if err != nil {
			return err
		}

		if _, statErr := os.Stat(filepath.Join(vaultDir, "config.yaml")); statErr == nil {
			return fmt.Errorf("vault already initialized at %s", vaultDir)
		} else if !os.IsNotExist(statErr) {
			return fmt.Errorf("cannot check vault directory: %w", statErr)
		}

		if mkdirErr := os.MkdirAll(vaultDir, 0o700); mkdirErr != nil {
			return fmt.Errorf("cannot create vault directory: %w", mkdirErr)
		}

		passphrase, err := readHiddenInput("Enter passphrase for vault identity: ", nil)
		if err != nil {
			return fmt.Errorf("cannot read passphrase: %w", err)
		}
		if len(passphrase) < 12 {
			return fmt.Errorf("passphrase must be at least 12 characters")
		}

		cfg := config.Default()
		cfg.VaultDir = vaultDir
		cfg.DefaultAgent = "cli"
		cfg.Agents = map[string]config.AgentProfile{
			"cli": {
				Name:            "cli",
				AllowedPaths:    []string{"*"},
				CanWrite:        true,
				RequireApproval: false,
			},
		}

		// Initialize Git config with defaults (auto-push enabled)
		cfg.Git = &config.GitConfig{
			AutoPush:       true,
			CommitTemplate: "Update from OpenPass",
		}

		identity, err := vaultpkg.InitWithPassphrase(vaultDir, passphrase, cfg)
		if err != nil {
			return fmt.Errorf("cannot initialize vault: %w", err)
		}

		if err := git.Init(vaultDir); err != nil {
			return fmt.Errorf("cannot initialize git: %w", err)
		}

		if err := git.CreateGitignore(vaultDir); err != nil {
			return fmt.Errorf("cannot create .gitignore: %w", err)
		}

		fmt.Printf("Vault initialized at %s\n", vaultDir)
		fmt.Printf("Public key: %s\n", identity.Recipient().String())
		return nil
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}

func expandVaultDir(vaultDir string) (string, error) {
	if vaultDir == "~" {
		return os.UserHomeDir()
	}
	if strings.HasPrefix(vaultDir, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("cannot determine home directory: %w", err)
		}
		return filepath.Join(home, vaultDir[2:]), nil
	}
	return filepath.Clean(vaultDir), nil
}
