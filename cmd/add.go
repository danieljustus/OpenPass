// Package cmd implements the OpenPass CLI commands using Cobra.
package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/danieljustus/OpenPass/internal/crypto"
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
)

var addCmd = &cobra.Command{
	Use:   "add <name>",
	Short: "Add a new password entry",
	Long: `Creates a new password entry in the vault.

The entry name can use slash notation for organization (e.g., work/aws).
Interactive mode prompts for username, password, and URL.

Examples:
  openpass add github
  openpass add work/aws
  openpass add personal/bank
  openpass add github-token --value "my-secret-token"
  openpass add secure-pass --generate --length 20`,
	Args: cobra.ExactArgs(1),
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

		name := args[0]

		if _, err := vaultpkg.ReadEntry(v.Dir, name, v.Identity); err == nil {
			return fmt.Errorf("entry already exists: %s (use 'set' to update or 'edit' to modify)", name)
		}

		data := map[string]any{}
		var reader *bufio.Reader

		// Non-interactive mode: use flags if provided
		if addUsername != "" {
			data["username"] = addUsername
		}

		if addValue != "" {
			data["password"] = addValue
		} else if addGenerate {
			password, err := generatePassword(addLength, true)
			if err != nil {
				return fmt.Errorf("generate password: %w", err)
			}
			data["password"] = password
		} else {
			// Interactive mode
			reader = bufio.NewReader(os.Stdin)

			fmt.Fprint(os.Stderr, "Username (optional): ")
			username, err := reader.ReadString('\n')
			if err != nil && username == "" {
				return fmt.Errorf("read username: %w", err)
			}
			username = strings.TrimSpace(username)
			if username != "" {
				data["username"] = username
			}

			password, err := readHiddenInput("Password: ", reader)
			if err != nil && password == "" {
				return fmt.Errorf("read password: %w", err)
			}
			if password != "" {
				data["password"] = password
			}
		}

		if addURL != "" {
			data["url"] = addURL
		} else if !addGenerate && addValue == "" {
			// Only prompt for URL in interactive mode when not using --value or --generate
			fmt.Fprint(os.Stderr, "URL (optional): ")
			url, err := reader.ReadString('\n')
			if err != nil && url == "" {
				return fmt.Errorf("read url: %w", err)
			}
			url = strings.TrimSpace(url)
			if url != "" {
				data["url"] = url
			}
		}

		if addNotes != "" {
			data["notes"] = addNotes
		} else if !addGenerate && addValue == "" {
			// Only prompt for notes in interactive mode when not using --value or --generate
			fmt.Fprint(os.Stderr, "Notes (optional, end with empty line):\n")
			var notes []string
			for {
				line, err := reader.ReadString('\n')
				if err != nil {
					break
				}
				line = strings.TrimSpace(line)
				if line == "" {
					break
				}
				notes = append(notes, line)
			}
			if len(notes) > 0 {
				data["notes"] = strings.Join(notes, "\n")
			}
		}

		// Handle TOTP
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
		} else if !addGenerate && addValue == "" {
			// Only prompt for TOTP in interactive mode when not using --value or --generate
			fmt.Fprint(os.Stderr, "TOTP Secret (optional): ")
			totpSecret, err := reader.ReadString('\n')
			if err != nil {
				return fmt.Errorf("read TOTP secret: %w", err)
			}
			totpSecret = strings.TrimSpace(totpSecret)
			if totpSecret != "" {
				totpData := map[string]any{
					"secret": totpSecret,
				}
				fmt.Fprint(os.Stderr, "TOTP Issuer (optional): ")
				totpIssuer, err := reader.ReadString('\n')
				if err != nil {
					return fmt.Errorf("read TOTP issuer: %w", err)
				}
				totpIssuer = strings.TrimSpace(totpIssuer)
				if totpIssuer != "" {
					totpData["issuer"] = totpIssuer
				}
				fmt.Fprint(os.Stderr, "TOTP Account (optional): ")
				totpAccount, err := reader.ReadString('\n')
				if err != nil {
					return fmt.Errorf("read TOTP account: %w", err)
				}
				totpAccount = strings.TrimSpace(totpAccount)
				if totpAccount != "" {
					totpData["account_name"] = totpAccount
				}
				data["totp"] = totpData
			}
		}

		if totpData, ok := data["totp"].(map[string]any); ok {
			if secret, ok := totpData["secret"].(string); ok && secret != "" {
				if err := crypto.ValidateTOTPSecret(secret); err != nil {
					return err
				}
			}
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
			fmt.Fprintf(os.Stderr, "Warning: auto-commit failed: %v\n", err)
		}
		fmt.Printf("Entry created: %s\n", name)
		return nil
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
	rootCmd.AddCommand(addCmd)
}
