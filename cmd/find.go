package cmd

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/spf13/cobra"

	vaultpkg "github.com/danieljustus/OpenPass/internal/vault"
	vaultsvc "github.com/danieljustus/OpenPass/internal/vaultsvc"
)

var (
	findJSON bool
)

var findCmd = &cobra.Command{
	Use:     "find <query>",
	Aliases: []string{"search"},
	Short:   "Search for entries",
	Long:    `Searches entry paths and contents for the given query.`,
	Example: `  # Search for entries containing "bank"
  openpass find bank

  # JSON output
  openpass find bank --output json`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return withVault(func(svc *vaultsvc.Service) error {
			workers := runtime.GOMAXPROCS(0)
			if workers > 4 {
				workers = 4
			}

			matches, err := svc.Find(args[0], vaultpkg.FindOptions{MaxWorkers: workers})
			if err != nil {
				return mapVaultSvcError(err, "search failed")
			}

			if len(matches) == 0 {
				fmt.Fprintln(os.Stderr, "No matches found")
				return nil
			}

			if outputFormat != "text" {
				type matchEntry struct {
					Path   string   `json:"path"`
					Fields []string `json:"fields,omitempty"`
				}
				out := make([]matchEntry, 0, len(matches))
				for _, m := range matches {
					out = append(out, matchEntry{Path: m.Path, Fields: m.Fields})
				}
				if err := PrintResult(map[string]interface{}{"matches": out}); err != nil {
					return err
				}
				return nil
			}

			for _, m := range matches {
				printQuietAware("%s", m.Path)
				if len(m.Fields) > 0 {
					printQuietAware(" (matches: %s)", strings.Join(m.Fields, ", "))
				}
				printlnQuietAware()
			}

			return nil
		})
	},
}

func init() {
	rootCmd.AddCommand(findCmd)
}
