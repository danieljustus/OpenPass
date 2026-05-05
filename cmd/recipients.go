package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	errorspkg "github.com/danieljustus/OpenPass/internal/errors"
	vaultpkg "github.com/danieljustus/OpenPass/internal/vault"
)

var (
	confirmRemove      bool
	recipientsListJSON bool
)

var recipientsCmd = &cobra.Command{
	Use:   "recipients",
	Short: "Manage vault recipients for multi-user encryption",
	Long: `Manage recipients (public keys) that can decrypt vault entries.

Recipients are stored in recipients.txt in the vault directory.
Each line contains one public key in age format (starting with "age1").
Lines starting with # are treated as comments.`,
	Example: `  openpass recipients list              # List all recipients
  openpass recipients add age1...       # Add a new recipient
  openpass recipients remove age1...    # Remove a recipient`,
}

var recipientsListCmd = &cobra.Command{
	Use:     "list",
	Short:   "List all recipients",
	Long:    `List all recipients from the recipients.txt file.`,
	Example: `  openpass recipients list`,
	RunE: func(cmd *cobra.Command, args []string) error {
		vaultDir, err := vaultPath()
		if err != nil {
			return err
		}

		if !vaultpkg.IsInitialized(vaultDir) {
			return errorspkg.NewCLIError(errorspkg.ExitNotInitialized, "vault not initialized. Run 'openpass init' first", errorspkg.ErrVaultNotInitialized)
		}

		rm := vaultpkg.NewRecipientsManager(vaultDir)
		recipients, err := rm.ListRecipients()
		if err != nil {
			return fmt.Errorf("cannot list recipients: %w", err)
		}

		if len(recipients) == 0 {
			if outputFormat == "text" {
				printlnQuietAware("No recipients configured.")
				printlnQuietAware("Use 'openpass recipients add <public-key>' to add a recipient.")
			} else {
				if err := PrintResult(map[string]interface{}{"recipients": []string{}}); err != nil {
					return err
				}
			}
			return nil
		}

		if outputFormat == "text" {
			printQuietAware("Recipients (%d):\n\n", len(recipients))
			for _, r := range recipients {
				status := "✓"
				if !r.Valid {
					status = "✗"
				}
				printlnQuietAware("  " + status + " " + r.Normalized)
				if !r.Valid {
					printlnQuietAware("    Error: " + r.Error)
				}
			}
		} else {
			recipientStrings := make([]string, 0, len(recipients))
			for _, r := range recipients {
				recipientStrings = append(recipientStrings, r.Normalized)
			}
			if err := PrintResult(map[string]interface{}{"recipients": recipientStrings}); err != nil {
				return err
			}
		}

		return nil
	},
}

var recipientsAddCmd = &cobra.Command{
	Use:   "add <public-key>",
	Short: "Add a recipient",
	Long: `Add a new recipient (public key) to the vault.

The public key must be a valid age public key starting with "age1".
Once added, all new entries will be encrypted for this recipient.`,
	Example: `  openpass recipients add age1ql3z7hjy54pw3hyww5ayyfg7zqgvc7w3j2elw8zmrj2kg5sfn9aqmcac8p`,
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		vaultDir, err := vaultPath()
		if err != nil {
			return err
		}

		_, err = unlockVault(vaultDir, true)
		if err != nil {
			return err
		}

		recipient := args[0]

		if !vaultpkg.IsInitialized(vaultDir) {
			return errorspkg.NewCLIError(errorspkg.ExitNotInitialized, "vault not initialized. Run 'openpass init' first", errorspkg.ErrVaultNotInitialized)
		}

		rm := vaultpkg.NewRecipientsManager(vaultDir)
		if err := rm.AddRecipient(recipient); err != nil {
			if errors.Is(err, vaultpkg.ErrRecipientAlreadyExists) {
				return fmt.Errorf("recipient already exists")
			}
			if errors.Is(err, vaultpkg.ErrInvalidRecipient) {
				return fmt.Errorf("invalid recipient: must be a valid age public key starting with 'age1'")
			}
			return fmt.Errorf("cannot add recipient: %w", err)
		}

		printlnQuietAware("Recipient added successfully.")
		return nil
	},
}

var recipientsRemoveCmd = &cobra.Command{
	Use:     "remove <public-key>",
	Aliases: []string{"rm"},
	Short:   "Remove a recipient",
	Long: `Remove a recipient (public key) from the vault.

The public key must match exactly. Use 'openpass recipients list' to see current recipients.

Use --yes to skip confirmation (useful for scripts).`,
	Example: `  openpass recipients remove age1ql3z7hjy54pw3hyww5ayyfg7zqgvc7w3j2elw8zmrj2kg5sfn9aqmcac8p`,
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		vaultDir, err := vaultPath()
		if err != nil {
			return err
		}

		_, err = unlockVault(vaultDir, true)
		if err != nil {
			return err
		}

		recipient := args[0]

		if !vaultpkg.IsInitialized(vaultDir) {
			return errorspkg.NewCLIError(errorspkg.ExitNotInitialized, "vault not initialized. Run 'openpass init' first", errorspkg.ErrVaultNotInitialized)
		}

		// Confirmation prompt unless --yes is passed or config disables it
		if !confirmRemove {
			fmt.Fprintf(os.Stderr, "Remove recipient %s? (y/N): ", recipient)
			answer, err := bufio.NewReader(os.Stdin).ReadString('\n')
			if err != nil && answer == "" {
				return fmt.Errorf("read confirmation: %w", err)
			}
			if strings.ToLower(strings.TrimSpace(answer)) != "y" {
				fmt.Fprintln(os.Stderr, "Canceled")
				return nil
			}
		}

		rm := vaultpkg.NewRecipientsManager(vaultDir)
		if err := rm.RemoveRecipient(recipient); err != nil {
			if errors.Is(err, vaultpkg.ErrRecipientNotFound) {
				return fmt.Errorf("recipient not found")
			}
			if errors.Is(err, vaultpkg.ErrInvalidRecipient) {
				return fmt.Errorf("invalid recipient: must be a valid age public key starting with 'age1'")
			}
			return fmt.Errorf("cannot remove recipient: %w", err)
		}

		printlnQuietAware("Recipient removed successfully.")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(recipientsCmd)
	recipientsCmd.AddCommand(recipientsListCmd)
	recipientsCmd.AddCommand(recipientsAddCmd)
	recipientsCmd.AddCommand(recipientsRemoveCmd)
	recipientsRemoveCmd.Flags().BoolVarP(&confirmRemove, "yes", "y", false, "Skip confirmation prompt")
}
