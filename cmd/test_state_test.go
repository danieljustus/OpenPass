package cmd

import (
	"context"
	"os"
	"os/signal"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	clipboardapp "github.com/danieljustus/OpenPass/internal/clipboard"
	vaultcrypto "github.com/danieljustus/OpenPass/internal/crypto"
	"github.com/danieljustus/OpenPass/internal/mcp"
	"github.com/danieljustus/OpenPass/internal/mcp/serverbootstrap"
	vaultpkg "github.com/danieljustus/OpenPass/internal/vault"
)

func TestMain(m *testing.M) {
	restoreScryptWorkFactor := vaultcrypto.SetScryptWorkFactorForTests(12)
	code := m.Run()
	restoreScryptWorkFactor()
	os.Exit(code)
}

func resetCommandTestState() {
	resetCommandFlagGlobals()
	resetCobraCommand(rootCmd)
	resetCommandEnv()
	osExit = os.Exit
	serveSignalNotify = signal.Notify
	runStdioServerFunc = func(ctx context.Context, vault *vaultpkg.Vault, agentName string) error {
		return serverbootstrap.RunStdioServer(ctx, vault, agentName, mcp.New)
	}
	runHTTPServerFunc = func(ctx context.Context, bind string, port int, vault *vaultpkg.Vault) error {
		vaultDir, _ := vaultPath()
		return serverbootstrap.RunHTTPServer(ctx, bind, port, vault, vaultDir, Version, mcp.New)
	}
	serveUnlockVault = unlockVault
}

func resetCommandFlagGlobals() {
	vault = "~/.openpass"
	setValue = ""
	setTOTPSecret = ""
	setTOTPIssuer = ""
	setTOTPAccount = ""
	addValue = ""
	addGenerate = false
	addLength = 20
	addUsername = ""
	addURL = ""
	addNotes = ""
	addTOTPSecret = ""
	addTOTPIssuer = ""
	addTOTPAccount = ""
	editorFlag = ""
	confirmRemove = false
	getCopyToClipboard = false
	genLength = 20
	genSymbols = false
	genStore = ""
	getClipboard = clipboardapp.DefaultClipboard
}

func resetCobraCommand(cmd *cobra.Command) {
	if cmd == nil {
		return
	}

	cmd.SetArgs(nil)
	cmd.SetOut(nil)
	cmd.SetErr(nil)
	cmd.SetIn(nil)
	cmd.SilenceUsage = false
	cmd.SilenceErrors = false

	resetFlagSet(cmd.Flags())
	resetFlagSet(cmd.PersistentFlags())
	resetFlagSet(cmd.LocalFlags())
	resetFlagSet(cmd.InheritedFlags())

	for _, child := range cmd.Commands() {
		resetCobraCommand(child)
	}
}

func resetFlagSet(flags *pflag.FlagSet) {
	if flags == nil {
		return
	}

	flags.VisitAll(func(flag *pflag.Flag) {
		_ = flag.Value.Set(flag.DefValue)
		flag.Changed = false
	})
}

func resetCommandEnv() {
	_ = os.Unsetenv("OPENPASS_VAULT")
	_ = os.Unsetenv("OPENPASS_PASSPHRASE")
}
