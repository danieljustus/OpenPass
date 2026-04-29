package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	errorspkg "github.com/danieljustus/OpenPass/internal/errors"
	vaultpkg "github.com/danieljustus/OpenPass/internal/vault"
	vaultsvc "github.com/danieljustus/OpenPass/internal/vaultsvc"
)

var (
	deleteYes  bool
	deleteJSON bool
)

var deleteCmd = &cobra.Command{
	Use:   "delete <path>",
	Short: "Delete a password entry",
	Args:  cobra.ExactArgs(1),
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

		path := args[0]
		if !deleteYes {
			fmt.Fprintf(os.Stderr, "Delete %s? (y/N): ", path)
			answer, err := bufio.NewReader(os.Stdin).ReadString('\n')
			if err != nil && answer == "" {
				return fmt.Errorf("read confirmation: %w", err)
			}
			if strings.ToLower(strings.TrimSpace(answer)) != "y" {
				if deleteJSON {
					PrintJSON(map[string]any{"deleted": false, "path": path, "canceled": true})
				} else {
					fmt.Fprintln(os.Stderr, "Canceled")
				}
				return nil
			}
		}

		svc := vaultsvc.New(v)
		if err := svc.Delete(path); err != nil {
			return mapVaultSvcError(err, "cannot delete entry")
		}
		if deleteJSON {
			PrintJSON(map[string]any{"deleted": true, "path": path})
			return nil
		}
		printQuietAware("Deleted: %s\n", path)
		return nil
	},
}

func init() {
	deleteCmd.Flags().BoolVarP(&deleteYes, "yes", "y", false, "Skip confirmation prompt")
	deleteCmd.Flags().BoolVarP(&deleteJSON, "json", "j", false, "Output as JSON")
	rootCmd.AddCommand(deleteCmd)
}
