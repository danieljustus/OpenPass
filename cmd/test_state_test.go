package cmd

import (
	"os"
	"os/signal"
	"testing"

	"github.com/atotto/clipboard"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	vaultcrypto "github.com/danieljustus/OpenPass/internal/crypto"
	"github.com/danieljustus/OpenPass/internal/mcp"
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
	mcpFactory.New = mcp.New
	runStdioServerFunc = runStdioServer
	runHTTPServerFunc = runHTTPServer
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
	getClipboardWriteAll = clipboard.WriteAll
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
