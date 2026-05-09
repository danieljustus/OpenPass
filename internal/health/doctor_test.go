package health_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/danieljustus/OpenPass/internal/health"
)

func TestRunChecks_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	results := health.RunChecks(dir, health.Options{NoNetwork: true})
	if len(results) == 0 {
		t.Fatal("expected at least one result")
	}
	// vault.initialized should be fail for an empty dir.
	var found bool
	for _, r := range results {
		if r.ID == "vault.initialized" {
			found = true
			if r.Status != health.StatusFail {
				t.Errorf("expected vault.initialized=fail, got %s", r.Status)
			}
		}
	}
	if !found {
		t.Error("vault.initialized check not found")
	}
}

func TestRunChecks_InitializedVault(t *testing.T) {
	dir := t.TempDir()
	// Write minimal vault structure.
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte("vaultDir: "+dir+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	// Write a fake age-encrypted identity file.
	ageContent := "age-encryption.org/v1\n-> scrypt fakesalt\nfakebody\n"
	if err := os.WriteFile(filepath.Join(dir, "identity.age"), []byte(ageContent), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "entries"), 0o700); err != nil {
		t.Fatal(err)
	}

	results := health.RunChecks(dir, health.Options{NoNetwork: true})
	byID := map[string]health.Result{}
	for _, r := range results {
		byID[r.ID] = r
	}

	if r := byID["vault.initialized"]; r.Status != health.StatusOK {
		t.Errorf("vault.initialized: expected ok, got %s: %s", r.Status, r.Message)
	}
	if r := byID["vault.identity.encrypted"]; r.Status != health.StatusOK {
		t.Errorf("vault.identity.encrypted: expected ok, got %s: %s", r.Status, r.Message)
	}
}

func TestScore(t *testing.T) {
	results := []health.Result{
		{Status: health.StatusOK},
		{Status: health.StatusOK},
		{Status: health.StatusWarn},
		{Status: health.StatusFail},
	}
	ok, warn, fail := health.Score(results)
	if ok != 2 || warn != 1 || fail != 1 {
		t.Errorf("Score: got ok=%d warn=%d fail=%d", ok, warn, fail)
	}
}

func TestRunChecks_NoNetwork_SkipsGitRemoteReachable(t *testing.T) {
	dir := t.TempDir()
	results := health.RunChecks(dir, health.Options{NoNetwork: true})
	for _, r := range results {
		if r.ID == "git.remote.reachable" || r.ID == "update.available" {
			t.Errorf("expected check %s to be skipped with --no-network", r.ID)
		}
	}
}
