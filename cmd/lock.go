package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/danieljustus/OpenPass/internal/session"
	vaultpkg "github.com/danieljustus/OpenPass/internal/vault"
)

var lockCmd = &cobra.Command{
	Use:   "lock",
	Short: "Lock the vault (clear session)",
	RunE: func(cmd *cobra.Command, args []string) error {
		vaultDir, err := vaultPath()
		if err != nil {
			return err
		}

		if !vaultpkg.IsInitialized(vaultDir) {
			return fmt.Errorf("vault not initialized. Run 'openpass init' first")
		}

		if err := session.ClearSession(vaultDir); err != nil {
			return fmt.Errorf("cannot clear session: %w", err)
		}

		fmt.Fprintln(os.Stderr, "Vault locked")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(lockCmd)
}
