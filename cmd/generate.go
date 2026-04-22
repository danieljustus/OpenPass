package cmd

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/danieljustus/OpenPass/internal/crypto"
	vaultpkg "github.com/danieljustus/OpenPass/internal/vault"
)

var (
	genLength  int
	genSymbols bool
	genStore   string
	genJSON    bool
	genReveal  bool
	genQuiet   bool
)

var generateCmd = &cobra.Command{
	Use:     "generate",
	Aliases: []string{"gen"},
	Short:   "Generate a secure password",
	RunE: func(cmd *cobra.Command, args []string) error {
		password, err := generatePassword(genLength, genSymbols)
		if err != nil {
			return err
		}

		if genStore != "" {
			vaultDir, err := vaultPath()
			if err != nil {
				return err
			}

			v, err := unlockVault(vaultDir, true)
			if err != nil {
				return err
			}

			entryPath := filepath.Join(v.Dir, genStore+".age")
			if _, err := vaultpkg.ReadEntry(v.Dir, genStore, v.Identity); err == nil {
				if _, err := vaultpkg.MergeEntryWithRecipients(v.Dir, genStore, map[string]any{"password": password}, v.Identity); err != nil {
					return fmt.Errorf("cannot store password: %w", err)
				}
			} else {
				entry := &vaultpkg.Entry{Data: map[string]any{"password": password}, Metadata: vaultpkg.EntryMetadata{Created: time.Now().UTC(), Updated: time.Now().UTC(), Version: 0}}
				if err := vaultpkg.WriteEntryWithRecipients(v.Dir, genStore, entry, v.Identity); err != nil {
					return fmt.Errorf("cannot store password: %w", err)
				}
			}

			if err := v.AutoCommit(fmt.Sprintf("Generate password for %s", genStore)); err != nil {
				return fmt.Errorf("auto-commit failed: %w", err)
			}

			if genJSON {
				result := map[string]any{
					"stored": true,
					"path":   genStore,
					"file":   entryPath,
				}
				if genReveal {
					result["password"] = password
				}
				PrintJSON(result)
				return nil
			}
			if genQuiet {
				return nil
			}
			fmt.Printf("Password stored at: %s\n", entryPath)
		}

		if genJSON {
			PrintJSON(map[string]interface{}{"password": password})
			return nil
		}

		fmt.Println(password)
		return nil
	},
}

func generatePassword(length int, useSymbols bool) (string, error) {
	if length <= 0 {
		return "", fmt.Errorf("length must be greater than zero")
	}
	if length > crypto.MaxPasswordLength {
		return "", fmt.Errorf("length must be at most %d", crypto.MaxPasswordLength)
	}
	return crypto.GeneratePassword(length, useSymbols)
}

func init() {
	generateCmd.Flags().IntVarP(&genLength, "length", "l", 20, "Password length")
	generateCmd.Flags().BoolVarP(&genSymbols, "symbols", "s", false, "Include symbols")
	generateCmd.Flags().StringVar(&genStore, "store", "", "Store at path (optional)")
	generateCmd.Flags().BoolVarP(&genJSON, "json", "j", false, "Output as JSON")
	generateCmd.Flags().BoolVar(&genReveal, "reveal", false, "Include generated password in output when using --store")
	generateCmd.Flags().BoolVar(&genQuiet, "quiet", false, "Suppress success output when using --store")
	generateCmd.AddCommand(manpagesCmd)
	rootCmd.AddCommand(generateCmd)
}
