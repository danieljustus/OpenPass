package cmd

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/spf13/cobra"

	clipboardpkg "github.com/danieljustus/OpenPass/internal/clipboard"
	configpkg "github.com/danieljustus/OpenPass/internal/config"
	vaultcrypto "github.com/danieljustus/OpenPass/internal/crypto"
	vaultpkg "github.com/danieljustus/OpenPass/internal/vault"
)

var (
	getCopyToClipboard   bool
	getClipboardWriteAll = clipboard.WriteAll
	getJSON              bool
)

type totpOutput struct {
	Code      string `json:"code"`
	Period    int64  `json:"period"`
	Remaining int    `json:"remaining"`
}

type getEntryOutput struct {
	Fields   map[string]any
	TOTP     *totpOutput
	Path     string
	Modified string
}

var getCmd = &cobra.Command{
	Use:   "get <path[.field]>",
	Short: "Get a password entry",
	Long:  "Retrieves and displays a password entry. Use path.field syntax to get specific fields.",
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
			candidatePath := query[:idx]
			candidateField := query[idx+1:]

			if entry, readErr := vaultpkg.ReadEntry(v.Dir, candidatePath, v.Identity); readErr == nil {
				if _, ok := entry.Data[candidateField]; ok {
					path = candidatePath
					field = candidateField
				}
			}
		}

		entry, err := vaultpkg.ReadEntry(v.Dir, path, v.Identity)
		if err != nil {
			matches, findErr := vaultpkg.Find(v.Dir, path)
			if findErr != nil {
				return fmt.Errorf("search entry: %w", findErr)
			}

			switch len(matches) {
			case 0:
				return fmt.Errorf("entry not found: %s", path)
			case 1:
				path = matches[0].Path
				entry, err = vaultpkg.ReadEntry(v.Dir, path, v.Identity)
				if err != nil {
					return fmt.Errorf("cannot read entry: %w", err)
				}
			default:
				fmt.Fprintln(os.Stderr, "Multiple matches:")
				for _, m := range matches {
					fmt.Fprintf(os.Stderr, "  %s\n", m.Path)
				}
				return fmt.Errorf("ambiguous path: %s", path)
			}
		}

		if field != "" {
			value, ok := entry.Data[field]
			if !ok {
				return fmt.Errorf("field not found: %s", field)
			}

			strValue := fmt.Sprintf("%v", value)
			if getCopyToClipboard {
				if err := getClipboardWriteAll(strValue); err != nil {
					return fmt.Errorf("copy to clipboard: %w", err)
				}
				fmt.Fprintln(os.Stderr, "[copied to clipboard]")

				autoClearDuration := getAutoClearDuration()
				if autoClearDuration > 0 {
					cancelCh := make(chan struct{})
					go clipboardpkg.Countdown(autoClearDuration, func(remaining int) {
						fmt.Fprintf(os.Stderr, "\r[clearing clipboard in %ds] ", remaining)
					}, cancelCh)
					go clipboardpkg.StartAutoClear(autoClearDuration, func() {
						close(cancelCh)
						if err := getClipboardWriteAll(""); err != nil {
							fmt.Fprintf(os.Stderr, "Warning: failed to clear clipboard: %v\n", err)
						}
						fmt.Fprintln(os.Stderr, "\r[clipboard cleared]        ")
					}, cancelCh)
				}
				return nil
			}

			if getJSON {
				PrintJSON(strValue)
				return nil
			}
			fmt.Println(strValue)
			return nil
		}

		if getJSON {
			output := getEntryOutput{
				Path:     path,
				Modified: entry.Metadata.Updated.Format("2006-01-02 15:04"),
				Fields:   entry.Data,
			}
			entryV2 := vaultpkg.EntryV2FromLegacy(entry)
			if entryV2.TOTP != nil && entryV2.TOTP.Secret != "" {
				totpCode, err := vaultcrypto.GenerateTOTP(
					entryV2.TOTP.Secret,
					entryV2.TOTP.Algorithm,
					entryV2.TOTP.Digits,
					entryV2.TOTP.Period,
				)
				if err == nil {
					period := int64(totpCode.Period)
					if period == 0 {
						period = 30
					}
					now := time.Now().UTC()
					remaining := period - (now.Unix() % period)
					output.TOTP = &totpOutput{
						Code:      totpCode.Code,
						Period:    period,
						Remaining: int(remaining),
					}
				}
			}
			PrintJSON(output)
			return nil
		}

		fmt.Printf("Path: %s\n", path)
		fmt.Printf("Modified: %s\n", entry.Metadata.Updated.Format("2006-01-02 15:04"))
		fmt.Println()

		keys := make([]string, 0, len(entry.Data))
		for k := range entry.Data {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, k := range keys {
			fmt.Printf("%s: %v\n", k, entry.Data[k])
		}

		entryV2 := vaultpkg.EntryV2FromLegacy(entry)
		if entryV2.TOTP != nil && entryV2.TOTP.Secret != "" {
			totpCode, err := vaultcrypto.GenerateTOTP(
				entryV2.TOTP.Secret,
				entryV2.TOTP.Algorithm,
				entryV2.TOTP.Digits,
				entryV2.TOTP.Period,
			)
			if err != nil {
				fmt.Fprintf(os.Stderr, "\n[Warning: could not generate TOTP code: %v]\n", err)
			} else {
				period := int64(totpCode.Period)
				if period == 0 {
					period = 30
				}
				now := time.Now().UTC()
				remaining := period - (now.Unix() % period)

				fmt.Println()
				fmt.Printf("TOTP Code: %s (expires in %ds)\n", totpCode.Code, remaining)
			}
		}

		return nil
	},
}

func init() {
	getCmd.Flags().BoolVarP(&getCopyToClipboard, "clip", "c", false, "Copy value to clipboard")
	getCmd.Flags().BoolVarP(&getJSON, "json", "j", false, "Output as JSON")
	rootCmd.AddCommand(getCmd)
}

func getAutoClearDuration() int {
	vaultDir, err := vaultPath()
	if err != nil {
		return 30
	}
	cfg, err := configpkg.Load(vaultDir + "/config.yaml")
	if err != nil {
		return 30
	}
	if cfg.Clipboard == nil {
		return 30
	}
	return cfg.Clipboard.AutoClearDuration
}
