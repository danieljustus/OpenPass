package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	vaultpkg "github.com/danieljustus/OpenPass/internal/vault"
)

var listJSON bool

var listCmd = &cobra.Command{
	Use:     "list [prefix]",
	Aliases: []string{"ls"},
	Short:   "List password entries",
	Args:    cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		vaultDir, err := vaultPath()
		if err != nil {
			return err
		}

		if !vaultpkg.IsInitialized(vaultDir) {
			return fmt.Errorf("vault not initialized. Run 'openpass init' first")
		}

		v, err := unlockVault(vaultDir, true)
		if err != nil {
			return err
		}

		prefix := ""
		if len(args) > 0 {
			prefix = args[0]
		}

		entries, err := vaultpkg.List(v.Dir, prefix)
		if err != nil {
			return fmt.Errorf("cannot list entries: %w", err)
		}

		if listJSON {
			PrintJSON(entries)
			return nil
		}

		for _, e := range entries {
			fmt.Println(e)
		}

		return nil
	},
}

func init() {
	listCmd.Flags().BoolVarP(&listJSON, "json", "j", false, "Output as JSON")
	rootCmd.AddCommand(listCmd)
}
