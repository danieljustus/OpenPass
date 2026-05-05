package cmd

import (
	"github.com/spf13/cobra"

	vaultsvc "github.com/danieljustus/OpenPass/internal/vaultsvc"
)

var listCmd = &cobra.Command{
	Use:     "list [prefix]",
	Aliases: []string{"ls"},
	Short:   "List password entries",
	Example: `  # List all entries
  openpass list

  # List entries under "work/" prefix
  openpass list work/

  # JSON output
  openpass list --output json`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return withVault(func(svc *vaultsvc.Service) error {
			prefix := ""
			if len(args) > 0 {
				prefix = args[0]
			}

			entries, err := svc.List(prefix)
			if err != nil {
				return mapVaultSvcError(err, "cannot list entries")
			}

			if outputFormat != "text" {
				if err := PrintResult(entries); err != nil {
					return err
				}
				return nil
			}

			for _, e := range entries {
				printlnQuietAware(e)
			}

			return nil
		})
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
