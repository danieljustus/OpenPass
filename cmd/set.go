package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	cryptopkg "github.com/danieljustus/OpenPass/internal/crypto"
	vaultsvc "github.com/danieljustus/OpenPass/internal/vaultsvc"
)

var (
	setValue       string
	setTOTPSecret  string
	setTOTPIssuer  string
	setTOTPAccount string
	setForce       bool
)

var setCmd = &cobra.Command{
	Use:   "set <path[.field]> [--value value]",
	Short: "Set a password entry or field",
	Long:  "Creates or updates a password entry. Use --value or interactive mode.",
	Example: `  # Set a field non-interactively
  openpass set github.password --value "mysecret"

  # Set TOTP data
  openpass set github --totp-secret JBSWY3DPEHPK3PXP`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
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
			if !setForce && (field == "" || field == "password") {
				if err := cryptopkg.ValidatePasswordStrength(setValue); err != nil {
					return err
				}
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
				if err != nil && len(password) == 0 {
					return fmt.Errorf("read password: %w", err)
				}
				defer cryptopkg.Wipe(password)
				data["password"] = string(password)
				if !setForce {
					if validateErr := cryptopkg.ValidatePasswordStrength(string(password)); validateErr != nil {
						return validateErr
					}
				}

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

		if err := cryptopkg.ValidateTOTPData(data); err != nil {
			return err
		}

		return withVault(func(svc *vaultsvc.Service) error {
			if err := svc.SetFields(path, data); err != nil {
				return mapVaultSvcError(err, "cannot write entry")
			}
			printQuietAware("Entry saved: %s\n", path)
			return nil
		})
	},
}

func init() {
	setCmd.Flags().StringVar(&setValue, "value", "", "Value to set (skip interactive)")
	setCmd.Flags().StringVar(&setTOTPSecret, "totp-secret", "", "TOTP secret key (base32 encoded)")
	setCmd.Flags().StringVar(&setTOTPIssuer, "totp-issuer", "", "TOTP issuer/service name")
	setCmd.Flags().StringVar(&setTOTPAccount, "totp-account", "", "TOTP account name/username")
	setCmd.Flags().BoolVar(&setForce, "force", false, "Skip password strength validation")
	rootCmd.AddCommand(setCmd)
}
