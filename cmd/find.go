package cmd

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/spf13/cobra"

	vaultpkg "github.com/danieljustus/OpenPass/internal/vault"
)

var findJSON bool

var findCmd = &cobra.Command{
	Use:     "find <query>",
	Aliases: []string{"search"},
	Short:   "Search for entries",
	Long:    `Searches entry paths and contents for the given query.`,
	Args:    cobra.ExactArgs(1),
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

		workers := runtime.GOMAXPROCS(0)
		if workers > 4 {
			workers = 4
		}
		matches, err := vaultpkg.FindConcurrent(v.Dir, args[0], workers)
		if err != nil {
			return fmt.Errorf("search failed: %w", err)
		}

		if len(matches) == 0 {
			fmt.Fprintln(os.Stderr, "No matches found")
			return nil
		}

		if findJSON {
			type matchEntry struct {
				Path   string   `json:"path"`
				Fields []string `json:"fields,omitempty"`
			}
			out := make([]matchEntry, 0, len(matches))
			for _, m := range matches {
				out = append(out, matchEntry{Path: m.Path, Fields: m.Fields})
			}
			PrintJSON(map[string]interface{}{"matches": out})
			return nil
		}

		for _, m := range matches {
			fmt.Printf("%s", m.Path)
			if len(m.Fields) > 0 {
				fmt.Printf(" (matches: %s)", strings.Join(m.Fields, ", "))
			}
			fmt.Println()
		}

		return nil
	},
}

func init() {
	findCmd.Flags().BoolVarP(&findJSON, "json", "j", false, "Output as JSON")
	rootCmd.AddCommand(findCmd)
}
