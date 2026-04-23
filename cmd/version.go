package cmd

import (
	"github.com/spf13/cobra"
)

var (
	appVersion = "dev"
	appCommit  = "none"
	appDate    = "unknown"
)

// SetVersionInfo is called from main to inject build-time values.
func SetVersionInfo(version, commit, date string) {
	appVersion = version
	appCommit = commit
	appDate = date
	rootCmd.Version = version
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version of OpenPass",
	Annotations: map[string]string{
		requiresVaultAnnotation: "false",
	},
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Printf("openpass %s (commit: %s, built: %s)\n", appVersion, appCommit, appDate)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
