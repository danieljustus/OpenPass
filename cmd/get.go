package cmd

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"

	clipboardapp "github.com/danieljustus/OpenPass/internal/clipboard"
	configpkg "github.com/danieljustus/OpenPass/internal/config"
	vaultcrypto "github.com/danieljustus/OpenPass/internal/crypto"
	errorspkg "github.com/danieljustus/OpenPass/internal/errors"
	vaultpkg "github.com/danieljustus/OpenPass/internal/vault"
	vaultsvc "github.com/danieljustus/OpenPass/internal/vaultsvc"
)

var (
	getCopyToClipboard bool
	getClipboard       = clipboardapp.DefaultClipboard
	getJSON            bool
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
			return errorspkg.NewCLIError(errorspkg.ExitNotInitialized, "vault not initialized. Run 'openpass init' first", errorspkg.ErrVaultNotInitialized)
		}

		v, err := unlockVault(vaultDir, true)
		if err != nil {
			return err
		}
		svc := vaultsvc.New(v)

		query := args[0]
		path := query
		field := ""

		if idx := strings.LastIndex(query, "."); idx > 0 {
			candidatePath := query[:idx]
			candidateField := query[idx+1:]

			if _, readErr := svc.GetField(candidatePath, candidateField); readErr == nil {
				path = candidatePath
				field = candidateField
			}
		}

		value, err := svc.GetField(path, field)
		if err != nil {
			var vaultErr *vaultsvc.Error
			if !errors.As(err, &vaultErr) || vaultErr.Kind != vaultsvc.ErrNotFound {
				if errors.As(err, &vaultErr) {
					switch vaultErr.Kind {
					case vaultsvc.ErrFieldNotFound:
						return errorspkg.NewCLIError(errorspkg.ExitNotFound, vaultErr.Message, errorspkg.ErrEntryNotFound)
					default:
					}
				}
				return fmt.Errorf("cannot read entry: %w", err)
			}

			matches, findErr := svc.Find(path, vaultsvc.FindOptions{MaxWorkers: 4})
			if findErr != nil {
				return fmt.Errorf("search entry: %w", findErr)
			}

			switch len(matches) {
			case 0:
				return errorspkg.NewCLIError(errorspkg.ExitNotFound, vaultErr.Message, errorspkg.ErrEntryNotFound)
			case 1:
				path = matches[0].Path
				value, err = svc.GetField(path, field)
				if err != nil {
					var vaultErr *vaultsvc.Error
					if errors.As(err, &vaultErr) {
						switch vaultErr.Kind {
						case vaultsvc.ErrNotFound, vaultsvc.ErrFieldNotFound:
							return errorspkg.NewCLIError(errorspkg.ExitNotFound, vaultErr.Message, errorspkg.ErrEntryNotFound)
						default:
						}
					}
					return fmt.Errorf("cannot read entry: %w", err)
				}
			default:
				fmt.Fprintln(os.Stderr, "Multiple matches:")
				for _, m := range matches {
					fmt.Fprintf(os.Stderr, "  %s\n", m.Path)
				}
				return errorspkg.NewCLIError(errorspkg.ExitNotFound, fmt.Sprintf("ambiguous path: %s", path), errorspkg.ErrEntryNotFound)
			}
		}

		if field != "" {
			strValue := fmt.Sprintf("%v", value)
			if getCopyToClipboard {
				if clipErr := getClipboard().Copy(strValue); clipErr != nil {
					return fmt.Errorf("copy to clipboard: %w", clipErr)
				}
				fmt.Fprintln(os.Stderr, "[copied to clipboard]")

				autoClearDuration := getAutoClearDuration()
				if autoClearDuration > 0 {
					cancelCh := make(chan struct{})
					go clipboardapp.Countdown(autoClearDuration, func(remaining int) {
						fmt.Fprintf(os.Stderr, "\r[clearing clipboard in %ds] ", remaining)
					}, cancelCh)
					go clipboardapp.StartAutoClear(autoClearDuration, func() {
						close(cancelCh)
						if clearErr := getClipboard().Copy(""); clearErr != nil {
							fmt.Fprintf(os.Stderr, "Warning: failed to clear clipboard: %v\n", clearErr)
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
			printlnQuietAware(strValue)
			return nil
		}

		entry, err := svc.GetEntry(path)
		if err != nil {
			var vaultErr *vaultsvc.Error
			if errors.As(err, &vaultErr) {
				switch vaultErr.Kind {
				case vaultsvc.ErrNotFound, vaultsvc.ErrFieldNotFound:
					return errorspkg.NewCLIError(errorspkg.ExitNotFound, vaultErr.Message, errorspkg.ErrEntryNotFound)
				default:
				}
			}
			return fmt.Errorf("cannot read entry: %w", err)
		}

		if getJSON {
			output := getEntryOutput{
				Path:     path,
				Modified: entry.Metadata.Updated.Format("2006-01-02 15:04"),
				Fields:   entry.Data,
			}
			if secret, algorithm, digits, period, hasTOTP := vaultpkg.ExtractTOTP(entry.Data); hasTOTP {
				totpCode, err := vaultcrypto.GenerateTOTP(secret, algorithm, digits, period)
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

		printQuietAware("Path: %s\n", path)
		printQuietAware("Modified: %s\n", entry.Metadata.Updated.Format("2006-01-02 15:04"))
		printlnQuietAware()

		keys := make([]string, 0, len(entry.Data))
		for k := range entry.Data {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, k := range keys {
			printQuietAware("%s: %v\n", k, entry.Data[k])
		}

		if secret, algorithm, digits, period, hasTOTP := vaultpkg.ExtractTOTP(entry.Data); hasTOTP {
			totpCode, err := vaultcrypto.GenerateTOTP(secret, algorithm, digits, period)
			if err != nil {
				fmt.Fprintf(os.Stderr, "\n[Warning: could not generate TOTP code: %v]\n", err)
			} else {
				period := int64(totpCode.Period)
				if period == 0 {
					period = 30
				}
				now := time.Now().UTC()
				remaining := period - (now.Unix() % period)

				printlnQuietAware()
				fmt.Fprintf(os.Stderr, "TOTP Code: %s (expires in %ds)\n", totpCode.Code, remaining)
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
