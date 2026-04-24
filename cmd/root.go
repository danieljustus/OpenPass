package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"golang.org/x/term"

	configpkg "github.com/danieljustus/OpenPass/internal/config"
	errorspkg "github.com/danieljustus/OpenPass/internal/errors"
	"github.com/danieljustus/OpenPass/internal/session"
	vaultpkg "github.com/danieljustus/OpenPass/internal/vault"
)

var osExit = os.Exit

const requiresVaultAnnotation = "openpass/requires-vault"

var readPasswordFunc func(int) ([]byte, error) = term.ReadPassword

func readHiddenInput(prompt string, reader *bufio.Reader) (string, error) {
	fmt.Fprint(os.Stderr, prompt)
	fdRaw := os.Stdin.Fd()
	// Bounds check: file descriptors are small non-negative integers; ensure they fit in int
	if fdRaw > uintptr(^uint(0)>>1) {
		return "", fmt.Errorf("file descriptor %d exceeds int range", fdRaw)
	}
	fd := int(fdRaw)
	if term.IsTerminal(fd) {
		passphrase, err := readPasswordFunc(fd)
		fmt.Fprintln(os.Stderr)
		if err != nil {
			return "", fmt.Errorf("read %s: %w", strings.TrimSuffix(strings.TrimSuffix(prompt, ": "), ":"), err)
		}
		return strings.TrimSpace(string(passphrase)), nil
	}
	if reader != nil {
		line, err := reader.ReadString('\n')
		if err != nil && line == "" {
			return "", fmt.Errorf("read %s: %w", strings.TrimSuffix(strings.TrimSuffix(prompt, ": "), ":"), err)
		}
		return strings.TrimSpace(line), nil
	}
	line, err := readLineFromStdin()
	if err != nil && line == "" {
		return "", fmt.Errorf("read %s: %w", strings.TrimSuffix(strings.TrimSuffix(prompt, ": "), ":"), err)
	}
	return strings.TrimSpace(line), nil
}

func readLineFromStdin() (string, error) {
	var result []byte
	var buf [1]byte
	for {
		n, err := os.Stdin.Read(buf[:])
		if n > 0 {
			if buf[0] == '\n' {
				return string(result), nil
			}
			result = append(result, buf[0])
		}
		if err != nil {
			if len(result) == 0 {
				return "", err
			}
			return string(result), nil
		}
	}
}

var (
	sessionLoadPassphrase = session.LoadPassphrase
	sessionSavePassphrase = session.SavePassphrase
	sessionIsExpired      = session.IsSessionExpired
)

var vault string
var vaultFlag *pflag.Flag

var rootCmd = &cobra.Command{
	Use:           "openpass",
	Short:         "OpenPass is a Go CLI password manager",
	SilenceUsage:  true,
	SilenceErrors: true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if !commandRequiresVault(cmd) {
			return nil
		}
		_, err := vaultPath()
		return err
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		osExit(int(errorspkg.ExitCodeFromError(err)))
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&vault, "vault", "~/.openpass", "path to the password vault")
	vaultFlag = rootCmd.PersistentFlags().Lookup("vault")
}

func vaultPath() (string, error) {
	if vaultFlag != nil && vaultFlag.Changed {
		p, err := expandVaultDir(vault)
		if err != nil {
			return "", errorspkg.NewCLIError(errorspkg.ExitGeneralError, "expand vault path", err)
		}
		return p, nil
	}

	if envVault := strings.TrimSpace(os.Getenv("OPENPASS_VAULT")); envVault != "" {
		p, err := expandVaultDir(envVault)
		if err != nil {
			return "", errorspkg.NewCLIError(errorspkg.ExitGeneralError, "expand vault path", err)
		}
		return p, nil
	}

	p, err := expandVaultDir(vault)
	if err != nil {
		return "", errorspkg.NewCLIError(errorspkg.ExitGeneralError, "expand vault path", err)
	}
	return p, nil
}

// unlockVault attempts to decrypt the vault identity.
// When interactive is false, only the keyring and OPENPASS_PASSPHRASE env var are tried.
func unlockVault(vaultDir string, interactive bool) (*vaultpkg.Vault, error) {
	v, _, err := unlockVaultWithTTL(vaultDir, interactive, 0, false)
	return v, err
}

func unlockVaultWithTTL(vaultDir string, interactive bool, ttlOverride time.Duration, cacheEnvPassphrase bool) (*vaultpkg.Vault, time.Duration, error) {
	passphrase, err := sessionLoadPassphrase(vaultDir)
	passphraseFromEnv := false
	if err != nil || passphrase == "" {
		passphrase = os.Getenv("OPENPASS_PASSPHRASE")
		passphraseFromEnv = passphrase != ""
		// Clear the env var immediately to prevent exposure via /proc/<pid>/environ
		_ = os.Unsetenv("OPENPASS_PASSPHRASE")
	}
	if passphrase == "" {
		if !interactive {
			return nil, 0, errorspkg.NewCLIError(errorspkg.ExitVaultLocked, "vault locked: run 'openpass unlock' first, or set OPENPASS_PASSPHRASE", nil)
		}
		var readErr error
		passphrase, readErr = readHiddenInput("Passphrase: ", nil)
		if readErr != nil {
			return nil, 0, errorspkg.NewCLIError(errorspkg.ExitVaultLocked, "read passphrase", readErr)
		}
	}

	v, err := vaultpkg.OpenWithPassphrase(vaultDir, passphrase)
	if err != nil {
		return nil, 0, errorspkg.NewCLIError(errorspkg.ExitGeneralError, "open vault", err)
	}

	ttl := configuredSessionTTL(v, ttlOverride)

	// Normal commands avoid persisting env-provided secrets; the unlock command opts in.
	if !passphraseFromEnv || cacheEnvPassphrase {
		if err := sessionSavePassphrase(vaultDir, passphrase, ttl); err != nil {
			return nil, 0, errorspkg.NewCLIError(errorspkg.ExitGeneralError, "save session", err)
		}
	}

	return v, ttl, nil
}

func defaultSessionTTL() time.Duration {
	return configpkg.Default().SessionTimeout
}

func configuredSessionTTL(v *vaultpkg.Vault, override time.Duration) time.Duration {
	if override > 0 {
		return override
	}
	if v != nil && v.Config != nil && v.Config.SessionTimeout > 0 {
		return v.Config.SessionTimeout
	}
	return defaultSessionTTL()
}

func commandRequiresVault(cmd *cobra.Command) bool {
	for current := cmd; current != nil; current = current.Parent() {
		if current.Annotations == nil {
			continue
		}
		if value, ok := current.Annotations[requiresVaultAnnotation]; ok {
			return value != "false"
		}
	}
	return true
}
