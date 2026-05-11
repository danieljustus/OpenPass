package cmd

import (
	"strings"

	"filippo.io/age"
	"github.com/spf13/cobra"

	vaultpkg "github.com/danieljustus/OpenPass/internal/vault"
)

// entryCompletionFunc returns a cobra completion function that suggests vault
// entry paths for the first positional argument. It only works when the vault
// is unlocked (via the cached identity in the OS keyring) — otherwise it
// silently returns no suggestions so the shell falls back to default
// completion. We never prompt for a passphrase from completion code.
func entryCompletionFunc(_ *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	vaultDir, err := vaultPath()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	if !vaultpkg.IsInitialized(vaultDir) {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	cachedIdentity, err := sessionLoadIdentity(vaultDir)
	if err != nil || cachedIdentity == "" {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	identity, err := age.ParseX25519Identity(cachedIdentity)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	v, err := vaultpkg.OpenWithCachedIdentity(vaultDir, identity)
	if err != nil || v == nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	prefix := strings.TrimSpace(toComplete)
	paths, err := vaultpkg.List(vaultDir, "")
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	matches := make([]string, 0, len(paths))
	for _, p := range paths {
		if prefix == "" || strings.HasPrefix(p, prefix) {
			matches = append(matches, p)
		}
	}
	return matches, cobra.ShellCompDirectiveNoFileComp
}
