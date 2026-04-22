package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	vaultpkg "github.com/danieljustus/OpenPass/internal/vault"
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
			return fmt.Errorf("vault not initialized. Run 'openpass init' first")
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

		if err := vaultpkg.DeleteEntry(v.Dir, path); err != nil {
			return fmt.Errorf("cannot delete entry: %w", err)
		}

		if err := v.AutoCommit(fmt.Sprintf("Delete %s", path)); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: auto-commit failed: %v\n", err)
		}
		if deleteJSON {
			PrintJSON(map[string]any{"deleted": true, "path": path})
			return nil
		}
		fmt.Printf("Deleted: %s\n", path)
		return nil
	},
}

func init() {
	deleteCmd.Flags().BoolVarP(&deleteYes, "yes", "y", false, "Skip confirmation prompt")
	deleteCmd.Flags().BoolVarP(&deleteJSON, "json", "j", false, "Output as JSON")
	rootCmd.AddCommand(deleteCmd)
}
