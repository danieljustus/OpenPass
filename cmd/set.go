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
	setValue       string
	setTOTPSecret  string
	setTOTPIssuer  string
	setTOTPAccount string
)

var setCmd = &cobra.Command{
	Use:   "set <path[.field]> [--value value]",
	Short: "Set a password entry or field",
	Long:  "Creates or updates a password entry. Use --value or interactive mode.",
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

		query := args[0]
		path := query
		field := ""
		if idx := strings.LastIndex(query, "."); idx > 0 {
			path = query[:idx]
			field = query[idx+1:]
		}

		data := map[string]any{}
		if setValue != "" {
			if field != "" {
				data[field] = setValue
			} else {
				data["password"] = setValue
			}
		} else {
			reader := bufio.NewReader(os.Stdin)
			if field != "" {
				fmt.Fprintf(os.Stderr, "Enter value for %s: ", field)
				value, err := reader.ReadString('\n')
				if err != nil && value == "" {
					return fmt.Errorf("read value: %w", err)
				}
				data[field] = strings.TrimSpace(value)
			} else {
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
				data["password"] = password

				fmt.Fprint(os.Stderr, "URL (optional): ")
				url, err := reader.ReadString('\n')
				if err != nil && url == "" {
					return fmt.Errorf("read url: %w", err)
				}
				url = strings.TrimSpace(url)
				if url != "" {
					data["url"] = url
				}

				if setTOTPSecret == "" {
					fmt.Fprint(os.Stderr, "TOTP Secret (optional): ")
					totpSecret, err := reader.ReadString('\n')
					if err != nil {
						return fmt.Errorf("read TOTP secret: %w", err)
					}
					setTOTPSecret = strings.TrimSpace(totpSecret)
				}
			}
		}

		if setTOTPSecret != "" {
			totpData := map[string]any{
				"secret": setTOTPSecret,
			}
			if setTOTPIssuer != "" {
				totpData["issuer"] = setTOTPIssuer
			}
			if setTOTPAccount != "" {
				totpData["account_name"] = setTOTPAccount
			}
			data["totp"] = totpData
		}

		if totpData, ok := data["totp"].(map[string]any); ok {
			if secret, ok := totpData["secret"].(string); ok && secret != "" {
				if err := crypto.ValidateTOTPSecret(secret); err != nil {
					return err
				}
			}
		}

		existing, readErr := vaultpkg.ReadEntry(v.Dir, path, v.Identity)
		entryPath := path
		if readErr == nil && existing != nil {
			if _, err := vaultpkg.MergeEntryWithRecipients(v.Dir, entryPath, data, v.Identity); err != nil {
				return fmt.Errorf("cannot write entry: %w", err)
			}
		} else {
			entry := &vaultpkg.Entry{Data: data, Metadata: vaultpkg.EntryMetadata{Created: time.Now().UTC(), Updated: time.Now().UTC(), Version: 0}}
			if err := vaultpkg.WriteEntryWithRecipients(v.Dir, entryPath, entry, v.Identity); err != nil {
				return fmt.Errorf("cannot write entry: %w", err)
			}
		}

		if err := v.AutoCommit(fmt.Sprintf("Update %s", path)); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: auto-commit failed: %v\n", err)
		}
		fmt.Printf("Entry saved: %s\n", path)
		return nil
	},
}

func init() {
	setCmd.Flags().StringVar(&setValue, "value", "", "Value to set (skip interactive)")
	setCmd.Flags().StringVar(&setTOTPSecret, "totp-secret", "", "TOTP secret key (base32 encoded)")
	setCmd.Flags().StringVar(&setTOTPIssuer, "totp-issuer", "", "TOTP issuer/service name")
	setCmd.Flags().StringVar(&setTOTPAccount, "totp-account", "", "TOTP account name/username")
	rootCmd.AddCommand(setCmd)
}
