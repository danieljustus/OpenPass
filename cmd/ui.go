package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/danieljustus/OpenPass/internal/ui"
	vaultsvc "github.com/danieljustus/OpenPass/internal/vaultsvc"
)

var uiCmd = &cobra.Command{
	Use:   "ui",
	Short: "Launch the OpenPass terminal UI",
	Long:  "Launches the interactive terminal UI for browsing and managing the vault.",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return withVault(func(svc vaultsvc.Service) error {
			if err := ui.Run(svc); err != nil {
				return fmt.Errorf("ui failed: %w", err)
			}

			return nil
		})
	},
}

func init() {
	rootCmd.AddCommand(uiCmd)
}
