package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/danieljustus/OpenPass/internal/git"
	vaultpkg "github.com/danieljustus/OpenPass/internal/vault"
)

var gitCmd = &cobra.Command{
	Use:   "git <push|pull|log> [path]",
	Short: "Git operations on vault",
	Args:  cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		vaultDir, err := vaultPath()
		if err != nil {
			return err
		}

		action := args[0]

		var v *vaultpkg.Vault
		if action == "log" {
			var err error
			v, err = unlockVault(vaultDir, true)
			if err != nil {
				return err
			}
		}

		switch action {
		case "push":
			if err := git.Push(vaultDir); err != nil {
				return fmt.Errorf("push failed: %w", err)
			}
			fmt.Println("Pushed to remote")

		case "pull":
			if err := git.Pull(vaultDir); err != nil {
				return fmt.Errorf("pull failed: %w", err)
			}
			fmt.Println("Pulled from remote")

		case "log":
			path := ""
			if len(args) > 1 {
				path = args[1]
			}

			history, err := git.Log(vaultDir, path, 0)
			if err != nil {
				return fmt.Errorf("cannot get log: %w", err)
			}

			_ = v
			for _, h := range history {
				fmt.Printf("%s  %s  %s\n", h.Hash[:7], h.Date.Format("2006-01-02"), h.Message)
				fmt.Printf("  Author: %s\n", h.Author)
			}

		default:
			return fmt.Errorf("unknown action: %s (use push, pull, or log)", action)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(gitCmd)
}
