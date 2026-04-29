package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	configpkg "github.com/danieljustus/OpenPass/internal/config"
	cryptopkg "github.com/danieljustus/OpenPass/internal/crypto"
	"github.com/danieljustus/OpenPass/internal/session"
	vaultpkg "github.com/danieljustus/OpenPass/internal/vault"
)

var authStatusJSON bool

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage vault unlock authentication",
}

var authStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show vault unlock authentication status",
	RunE: func(cmd *cobra.Command, args []string) error {
		_, _ = cmd, args
		vaultDir, cfg, err := loadAuthConfig()
		if err != nil {
			return err
		}
		method := cfg.EffectiveAuthMethod()
		cache := session.GetCacheStatus()
		payload := map[string]any{
			"vault":            vaultDir,
			"method":           method,
			"touchIDAvailable": session.BiometricAvailable(),
			"cache":            cache,
		}
		if authStatusJSON {
			PrintJSON(payload)
			return nil
		}
		printlnQuietAware("Vault: " + vaultDir)
		printlnQuietAware("Auth method: " + method)
		printlnQuietAware(fmt.Sprintf("Touch ID available: %t", payload["touchIDAvailable"]))
		printlnQuietAware(fmt.Sprintf("Session cache: %s (persistent: %t)", cache.Backend, cache.Persistent))
		return nil
	},
}

var authSetCmd = &cobra.Command{
	Use:   "set passphrase|touchid",
	Short: "Set the vault unlock authentication method",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		_, _ = cmd, args
		method, err := configpkg.NormalizeAuthMethod(args[0])
		if err != nil {
			return err
		}
		vaultDir, cfg, err := loadAuthConfig()
		if err != nil {
			return err
		}

		switch method {
		case configpkg.AuthMethodPassphrase:
			if err := cfg.SetAuthMethod(configpkg.AuthMethodPassphrase); err != nil {
				return err
			}
			if err := cfg.SaveTo(filepath.Join(vaultDir, "config.yaml")); err != nil {
				return fmt.Errorf("save config: %w", err)
			}
			if err := session.ClearBiometricPassphrase(vaultDir); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: could not remove Touch ID unlock item: %v\n", err)
			}
			printlnQuietAware("Auth method set to passphrase")
			return nil
		case configpkg.AuthMethodTouchID:
			if !session.BiometricAvailable() {
				return fmt.Errorf("touch ID is not available in this OpenPass build or on this Mac")
			}
			passphrase, err := passphraseForBiometricSetup(vaultDir)
			if err != nil {
				return err
			}
			defer cryptopkg.Wipe([]byte(passphrase))
			if err := session.SaveBiometricPassphrase(context.Background(), vaultDir, passphrase); err != nil {
				return fmt.Errorf("save Touch ID unlock item: %w", err)
			}
			if err := cfg.SetAuthMethod(configpkg.AuthMethodTouchID); err != nil {
				return err
			}
			if err := cfg.SaveTo(filepath.Join(vaultDir, "config.yaml")); err != nil {
				return fmt.Errorf("save config: %w", err)
			}
			printlnQuietAware("Auth method set to touchid")
			return nil
		default:
			return fmt.Errorf("unsupported auth method %q", method)
		}
	},
}

func init() {
	authStatusCmd.Flags().BoolVar(&authStatusJSON, "json", false, "output auth status as JSON")
	authCmd.AddCommand(authStatusCmd)
	authCmd.AddCommand(authSetCmd)
	rootCmd.AddCommand(authCmd)
}

func loadAuthConfig() (string, *configpkg.Config, error) {
	vaultDir, err := vaultPath()
	if err != nil {
		return "", nil, err
	}
	if !vaultpkg.IsInitialized(vaultDir) {
		return "", nil, fmt.Errorf("vault not initialized. Run 'openpass init' first")
	}
	cfg, err := configpkg.Load(filepath.Join(vaultDir, "config.yaml"))
	if err != nil {
		return "", nil, fmt.Errorf("load config: %w", err)
	}
	return vaultDir, cfg, nil
}

func passphraseForBiometricSetup(vaultDir string) (string, error) {
	if passphrase, err := sessionLoadPassphrase(vaultDir); err == nil && passphrase != "" {
		return passphrase, nil
	}

	passphrase := os.Getenv("OPENPASS_PASSPHRASE")
	if passphrase != "" {
		_ = os.Unsetenv("OPENPASS_PASSPHRASE")
		if _, err := vaultpkg.OpenWithPassphrase(vaultDir, passphrase); err != nil {
			return "", fmt.Errorf("open vault: %w", err)
		}
		return passphrase, nil
	}

	passphrase, err := readHiddenInput("Passphrase: ", nil)
	if err != nil {
		return "", err
	}
	if _, err := vaultpkg.OpenWithPassphrase(vaultDir, passphrase); err != nil {
		return "", fmt.Errorf("open vault: %w", err)
	}
	return passphrase, nil
}
