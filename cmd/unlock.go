package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	vaultpkg "github.com/danieljustus/OpenPass/internal/vault"
)

var unlockCmd = &cobra.Command{
	Use:   "unlock",
	Short: "Unlock the vault and cache the passphrase",
	Long: `Unlock the vault by validating the passphrase and caching it in the
OS keyring. This allows MCP servers to start without interactive prompts.

Use --check to verify if an active session exists without prompting.

Environment variable OPENPASS_PASSPHRASE can be used in CI/CD environments
but should NOT be used on shared machines (visible in process listings).`,
	RunE: func(cmd *cobra.Command, args []string) error {
		check, _ := cmd.Flags().GetBool("check")
		ttl, _ := cmd.Flags().GetDuration("ttl")
		ttlFlag := cmd.Flags().Lookup("ttl")
		ttlOverride := time.Duration(0)
		if ttlFlag != nil && ttlFlag.Changed {
			ttlOverride = ttl
		}

		vaultDir, err := vaultPath()
		if err != nil {
			return err
		}

		if !vaultpkg.IsInitialized(vaultDir) {
			return fmt.Errorf("vault not initialized. Run 'openpass init' first")
		}

		if check {
			if sessionIsExpired(vaultDir) {
				cmd.SilenceUsage = true
				return fmt.Errorf("no active session")
			}
			fmt.Fprintln(os.Stderr, "Session active")
			return nil
		}

		v, effectiveTTL, err := unlockVaultWithTTL(vaultDir, true, ttlOverride, true)
		if err != nil {
			return err
		}
		_ = v

		fmt.Fprintf(os.Stderr, "Vault unlocked (session TTL: %s)\n", effectiveTTL)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(unlockCmd)
	unlockCmd.Flags().Duration("ttl", defaultSessionTTL(), "Session duration (overrides config sessionTimeout)")
	unlockCmd.Flags().Bool("check", false, "Check if session is active (exit 0 = active, exit 1 = expired)")
}
