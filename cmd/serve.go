package cmd

import (
	"os/signal"

	"github.com/spf13/cobra"
)

var serveSignalNotify = signal.Notify
var Version = "dev"

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start MCP server for agent access",
	Long: `Start an MCP server that exposes vault operations to AI agents.

Each agent must be configured in ~/.openpass/config.yaml with specific
permissions and scope restrictions.

The server can run in HTTP mode or stdio mode.`,
	RunE: runServe,
}

func init() {
	rootCmd.AddCommand(serveCmd)
	serveCmd.Flags().String("agent", "", "Agent name (required for --stdio; HTTP mode resolves agents per-request via X-OpenPass-Agent header)")
	serveCmd.Flags().Int("port", 8080, "Server port")
	serveCmd.Flags().Bool("stdio", false, "Enable stdio transport (for MCP)")
	serveCmd.Flags().String("bind", "127.0.0.1", "Bind address for HTTP server")
}
