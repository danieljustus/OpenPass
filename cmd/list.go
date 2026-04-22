package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	vaultpkg "github.com/danieljustus/OpenPass/internal/vault"
)

var listJSON bool
var listNoIndex bool

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

		var entries []string
		if listNoIndex {
			entries, err = vaultpkg.List(v.Dir, prefix)
		} else {
			entries, err = listWithIndex(v.Dir, prefix)
		}
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

func listWithIndex(vaultDir, prefix string) ([]string, error) {
	idx, err := vaultpkg.LoadIndex(vaultDir)
	if err != nil {
		// Index corrupted or unreadable - fall back to full scan
		return vaultpkg.List(vaultDir, prefix)
	}
	if idx == nil || idx.IsStale(vaultDir, 5*time.Minute) {
		// No index or stale - build new one
		idx, err = vaultpkg.BuildIndex(vaultDir)
		if err != nil {
			return vaultpkg.List(vaultDir, prefix)
		}
		// Save index for subsequent fast lookups
		_ = idx.Save(vaultDir)
	}
	return idx.Filter(prefix), nil
}

func init() {
	listCmd.Flags().BoolVarP(&listJSON, "json", "j", false, "Output as JSON")
	listCmd.Flags().BoolVar(&listNoIndex, "no-index", false, "Bypass index cache and scan vault directly")
	rootCmd.AddCommand(listCmd)
}
