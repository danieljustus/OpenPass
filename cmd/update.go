package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	updatepkg "github.com/danieljustus/OpenPass/internal/update"
)

type updateChecker interface {
	Check(ctx context.Context, currentVersion string) (*updatepkg.Result, error)
}

var updateCheckerFactory = func() updateChecker {
	return updatepkg.NewChecker(nil)
}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Check for OpenPass updates",
	Args:  cobra.NoArgs,
	Annotations: map[string]string{
		requiresVaultAnnotation: "false",
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

var updateCheckCmd = &cobra.Command{
	Use:   "check",
	Short: "Check GitHub for a newer OpenPass release",
	Args:  cobra.NoArgs,
	Annotations: map[string]string{
		requiresVaultAnnotation: "false",
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		checker := updateCheckerFactory()
		result, err := checker.Check(cmd.Context(), appVersion)
		if err != nil {
			return fmt.Errorf("check for updates: %w", err)
		}

		if !result.Checkable {
			cmd.Printf("Update checks are only available for stable release builds. Current version: %s\n", result.CurrentVersion)
			return nil
		}

		if result.UpdateAvailable {
			cmd.Printf("Update available: %s -> %s\n", result.CurrentVersion, result.LatestVersion)
			if result.ReleaseURL != "" {
				cmd.Printf("Download: %s\n", result.ReleaseURL)
			}
			return nil
		}

		if result.CurrentVersion == result.LatestVersion {
			cmd.Printf("OpenPass is up to date (%s).\n", result.CurrentVersion)
			return nil
		}

		cmd.Printf("No newer stable release found. Current version: %s. Latest published stable release: %s.\n", result.CurrentVersion, result.LatestVersion)
		return nil
	},
}

func init() {
	updateCmd.AddCommand(updateCheckCmd)
	rootCmd.AddCommand(updateCmd)
}
