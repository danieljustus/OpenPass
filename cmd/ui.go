package cmd

import (
	"github.com/spf13/cobra"

	errorspkg "github.com/danieljustus/OpenPass/internal/errors"
	"github.com/danieljustus/OpenPass/internal/ui"
	vaultpkg "github.com/danieljustus/OpenPass/internal/vault"
	vaultsvc "github.com/danieljustus/OpenPass/internal/vaultsvc"
)

var uiCmd = &cobra.Command{
	Use:   "ui",
	Short: "Launch the OpenPass terminal UI",
	Long:  "Launches the interactive terminal UI for browsing and managing the vault.",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		vaultDir, err := vaultPath()
		if err != nil {
			return err
		}

		if !vaultpkg.IsInitialized(vaultDir) {
			return errorspkg.NewCLIError(errorspkg.ExitNotInitialized, "vault not initialized. Run 'openpass init' first", errorspkg.ErrVaultNotInitialized)
		}

		v, err := unlockVault(vaultDir, true)
		if err != nil {
			return err
		}

		svc := vaultsvc.New(v)
		if err := ui.Run(svc); err != nil {
			return mapVaultSvcError(err, "ui failed")
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(uiCmd)
}
