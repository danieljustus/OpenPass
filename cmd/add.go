// Package cmd implements the OpenPass CLI commands using Cobra.
package cmd

import (
	"bufio"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	cryptopkg "github.com/danieljustus/OpenPass/internal/crypto"
	"github.com/danieljustus/OpenPass/internal/ui/cliout"
	"github.com/danieljustus/OpenPass/internal/ui/forms"
	vaultpkg "github.com/danieljustus/OpenPass/internal/vault"
)

var (
	addValue       string
	addGenerate    bool
	addLength      int
	addUsername    string
	addURL         string
	addNotes       string
	addTOTPSecret  string
	addTOTPIssuer  string
	addTOTPAccount string
	addForce       bool
)

var addCmd = &cobra.Command{
	Use:   "add <name>",
	Short: "Add a new password entry",
	Long: `Creates a new password entry in the vault.

The entry name can use slash notation for organization (e.g., work/aws).
Interactive mode prompts for username, password, and URL.`,
	Example: `  openpass add github
  openpass add work/aws
  openpass add personal/bank
  openpass add github-token --value "my-secret-token"
  openpass add secure-pass --generate --length 20`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return withVaultRaw(func(v *vaultpkg.Vault) error {
			name := args[0]

			if _, err := vaultpkg.ReadEntry(v.Dir, name, v.Identity); err == nil {
				return fmt.Errorf("entry already exists: %s (use 'set' to update or 'edit' to modify)", name)
			}

			data := map[string]any{}
			var reader *bufio.Reader
			var readerUsed bool

			// Non-interactive mode: use flags if provided
			if addUsername != "" {
				data["username"] = addUsername
			}

			if addValue != "" {
				data["password"] = addValue
				if !addForce {
					if err := cryptopkg.ValidatePasswordStrength(addValue); err != nil {
						return err
					}
				}
			} else if addGenerate {
				password, err := generatePassword(addLength, true)
				if err != nil {
					return fmt.Errorf("generate password: %w", err)
				}
				data["password"] = password
			} else {
				// Interactive mode
				fdRaw := os.Stdin.Fd()
				if fdRaw > uintptr(^uint(0)>>1) {
					return fmt.Errorf("file descriptor %d exceeds int range", fdRaw)
				}
				fd := int(fdRaw)

				if isTerminalFunc(fd) {
					defaults := map[string]any{}
					if addUsername != "" {
						defaults["username"] = addUsername
					}
					if addURL != "" {
						defaults["url"] = addURL
					}
					if addNotes != "" {
						defaults["notes"] = addNotes
					}
					if addTOTPSecret != "" {
						totpDefaults := map[string]any{
							"secret": addTOTPSecret,
						}
						if addTOTPIssuer != "" {
							totpDefaults["issuer"] = addTOTPIssuer
						}
						if addTOTPAccount != "" {
							totpDefaults["account_name"] = addTOTPAccount
						}
						defaults["totp"] = totpDefaults
					}

					formData, err := forms.RunAddEntryForm(addForce, defaults)
					if err != nil {
						return err
					}
					for k, v := range formData {
						data[k] = v
					}
				} else {
					reader = bufio.NewReader(os.Stdin)
					collected, err := collectEntryData(reader, entryFlags{
						username:    addUsername,
						url:         addURL,
						notes:       addNotes,
						totpSecret:  addTOTPSecret,
						totpIssuer:  addTOTPIssuer,
						totpAccount: addTOTPAccount,
						force:       addForce,
					})
					if err != nil {
						return err
					}
					for k, v := range collected {
						data[k] = v
					}
					readerUsed = true
				}
			}

			if !readerUsed {
				if addURL != "" {
					data["url"] = addURL
				}
			}

			if !readerUsed {
				if addNotes != "" {
					data["notes"] = addNotes
				}
			}

			if !readerUsed {
				if addTOTPSecret != "" {
					totpData := map[string]any{
						"secret": addTOTPSecret,
					}
					if addTOTPIssuer != "" {
						totpData["issuer"] = addTOTPIssuer
					}
					if addTOTPAccount != "" {
						totpData["account_name"] = addTOTPAccount
					}
					data["totp"] = totpData
				}
			}

			if err := cryptopkg.ValidateTOTPData(data); err != nil {
				return err
			}

			entry := &vaultpkg.Entry{
				Data: data,
				Metadata: vaultpkg.EntryMetadata{
					Created: time.Now().UTC(),
					Updated: time.Now().UTC(),
					Version: 1,
				},
			}

			if err := vaultpkg.WriteEntryWithRecipients(v.Dir, name, entry, v.Identity); err != nil {
				return fmt.Errorf("cannot create entry: %w", err)
			}

			if err := v.AutoCommit(fmt.Sprintf("Add %s", name)); err != nil {
				cliout.Warnf("Warning: auto-commit failed: %v", err)
			}
			printQuietAware("Entry created: %s\n", name)
			return nil
		})
	},
}

func init() {
	addCmd.Flags().StringVar(&addValue, "value", "", "Password value (non-interactive)")
	addCmd.Flags().BoolVar(&addGenerate, "generate", false, "Generate a secure password (non-interactive)")
	addCmd.Flags().IntVar(&addLength, "length", 20, "Generated password length for --generate")
	addCmd.Flags().StringVar(&addUsername, "username", "", "Username (non-interactive)")
	addCmd.Flags().StringVar(&addURL, "url", "", "URL (non-interactive)")
	addCmd.Flags().StringVar(&addNotes, "notes", "", "Notes (non-interactive)")
	addCmd.Flags().StringVar(&addTOTPSecret, "totp-secret", "", "TOTP secret key (base32 encoded)")
	addCmd.Flags().StringVar(&addTOTPIssuer, "totp-issuer", "", "TOTP issuer/service name")
	addCmd.Flags().StringVar(&addTOTPAccount, "totp-account", "", "TOTP account name/username")
	addCmd.Flags().BoolVar(&addForce, "force", false, "Skip password strength validation")
	rootCmd.AddCommand(addCmd)
}
