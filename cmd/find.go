package cmd

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/spf13/cobra"

	errorspkg "github.com/danieljustus/OpenPass/internal/errors"
	vaultpkg "github.com/danieljustus/OpenPass/internal/vault"
	vaultsvc "github.com/danieljustus/OpenPass/internal/vaultsvc"
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
			return errorspkg.NewCLIError(errorspkg.ExitNotInitialized, "vault not initialized. Run 'openpass init' first", errorspkg.ErrVaultNotInitialized)
		}

		v, err := unlockVault(vaultDir, true)
		if err != nil {
			return err
		}

		workers := runtime.GOMAXPROCS(0)
		if workers > 4 {
			workers = 4
		}
		svc := vaultsvc.New(v)
		matches, err := svc.Find(args[0], vaultsvc.FindOptions{MaxWorkers: workers})
		if err != nil {
			return mapVaultSvcError(err, "search failed")
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
			printQuietAware("%s", m.Path)
			if len(m.Fields) > 0 {
				printQuietAware(" (matches: %s)", strings.Join(m.Fields, ", "))
			}
			printlnQuietAware()
		}

		return nil
	},
}

func init() {
	findCmd.Flags().BoolVarP(&findJSON, "json", "j", false, "Output as JSON")
	rootCmd.AddCommand(findCmd)
}
