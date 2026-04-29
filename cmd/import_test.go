package cmd

import (
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	vaultpkg "github.com/danieljustus/OpenPass/internal/vault"
	vaultsvc "github.com/danieljustus/OpenPass/internal/vaultsvc"
)

var expectedCSVImportPaths = []string{
	"Bank-Checking",
	"GitHub,-Personal",
	"Work-AWS",
}

func TestImportCommandDryRunDoesNotWriteEntries(t *testing.T) {
	vaultDir, passphrase := initVault(t)
	setPassEnv(t, passphrase)
	defer setupVaultFlag(t, vaultDir)()

	output := runImportCommand(t, passphrase, "--vault", vaultDir, "import", "csv", csvImportFixture(t), "--dry-run")

	svc := importTestVaultService(t, vaultDir, passphrase)
	entries, err := svc.List("")
	if err != nil {
		t.Fatalf("list entries: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("dry-run wrote entries: %#v", entries)
	}
	for _, path := range expectedCSVImportPaths {
		if !strings.Contains(output, "Would import: "+path) {
			t.Errorf("dry-run output missing %q: %s", path, output)
		}
	}
	if !strings.Contains(output, "Import summary: 3 imported, 0 skipped") {
		t.Errorf("dry-run output missing summary: %s", output)
	}
}

func TestImportCommandCSVWritesEntries(t *testing.T) {
	vaultDir, passphrase := initVault(t)
	setPassEnv(t, passphrase)
	defer setupVaultFlag(t, vaultDir)()

	output := runImportCommand(t, passphrase, "--vault", vaultDir, "import", "csv", csvImportFixture(t))

	if !strings.Contains(output, "Import summary: 3 imported, 0 skipped") {
		t.Errorf("import output missing summary: %s", output)
	}
	svc := importTestVaultService(t, vaultDir, passphrase)
	assertCSVImportedEntries(t, svc, "")
}

func TestImportCommandSkipExistingDoesNotChangeEntries(t *testing.T) {
	vaultDir, passphrase := initVault(t)
	defer setupVaultFlag(t, vaultDir)()
	source := csvImportFixture(t)

	runImportCommand(t, passphrase, "--vault", vaultDir, "import", "csv", source)
	svc := importTestVaultService(t, vaultDir, passphrase)
	before := snapshotImportEntries(t, svc, expectedCSVImportPaths)

	output := runImportCommand(t, passphrase, "--vault", vaultDir, "import", "csv", source, "--skip-existing")
	after := snapshotImportEntries(t, svc, expectedCSVImportPaths)

	if !reflect.DeepEqual(after, before) {
		t.Fatalf("entries changed with --skip-existing\nbefore: %#v\nafter:  %#v", before, after)
	}
	if !strings.Contains(output, "Import summary: 0 imported, 3 skipped") {
		t.Errorf("skip-existing output missing summary: %s", output)
	}
}

func TestImportCommandOverwriteUpdatesExistingEntries(t *testing.T) {
	vaultDir, passphrase := initVault(t)
	setPassEnv(t, passphrase)
	defer setupVaultFlag(t, vaultDir)()
	source := csvImportFixture(t)

	runImportCommand(t, passphrase, "--vault", vaultDir, "import", "csv", source)
	svc := importTestVaultService(t, vaultDir, passphrase)
	if err := svc.SetFields("GitHub,-Personal", map[string]any{
		"username": "changed@example.com",
		"extra":    "remove-me",
	}); err != nil {
		t.Fatalf("modify imported entry: %v", err)
	}

	output := runImportCommand(t, passphrase, "--vault", vaultDir, "import", "csv", source, "--overwrite")
	entry, err := svc.GetEntry("GitHub,-Personal")
	if err != nil {
		t.Fatalf("get overwritten entry: %v", err)
	}

	if entry.Data["username"] != "user@example.com" {
		t.Errorf("username was not overwritten, got %#v", entry.Data["username"])
	}
	if _, ok := entry.Data["extra"]; ok {
		t.Errorf("overwrite kept stale field: %#v", entry.Data)
	}
	if !strings.Contains(output, "Import summary: 3 imported, 0 skipped") {
		t.Errorf("overwrite output missing summary: %s", output)
	}
}

func TestImportCommandPrefixWritesEntriesUnderPrefix(t *testing.T) {
	vaultDir, passphrase := initVault(t)
	setPassEnv(t, passphrase)
	defer setupVaultFlag(t, vaultDir)()

	output := runImportCommand(t, passphrase, "--vault", vaultDir, "import", "csv", csvImportFixture(t), "--prefix", "imports/")

	if !strings.Contains(output, "Imported: imports/GitHub,-Personal") {
		t.Errorf("prefix output missing imported path: %s", output)
	}
	svc := importTestVaultService(t, vaultDir, passphrase)
	assertCSVImportedEntries(t, svc, "imports/")
}

func runImportCommand(t *testing.T, passphrase string, args ...string) string {
	t.Helper()
	if err := os.Setenv("OPENPASS_PASSPHRASE", passphrase); err != nil {
		t.Fatalf("set passphrase env: %v", err)
	}
	rootCmd.SetArgs(args)
	defer rootCmd.SetArgs(nil)

	var execErr error
	output := captureStdout(func() {
		execErr = rootCmd.Execute()
	})
	if execErr != nil {
		t.Fatalf("import command failed: %v", execErr)
	}
	return output
}

func csvImportFixture(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate import test file")
	}
	return filepath.Join(filepath.Dir(file), "..", "testdata", "importer", "csv", "sample.csv")
}

func importTestVaultService(t *testing.T, vaultDir, passphrase string) *vaultsvc.Service {
	t.Helper()
	v, err := vaultpkg.OpenWithPassphrase(vaultDir, passphrase)
	if err != nil {
		t.Fatalf("open vault: %v", err)
	}
	return vaultsvc.New(v)
}

func assertCSVImportedEntries(t *testing.T, svc *vaultsvc.Service, prefix string) {
	t.Helper()
	entryAssertions := map[string]map[string]any{
		"GitHub,-Personal": {
			"username": "user@example.com",
			"password": "mysecretpassword",
			"url":      "https://github.com/login",
			"notes":    "Primary account, includes comma in title",
		},
		"Bank-Checking": {
			"username": "bank.user@example.com",
			"password": "p@ss,with,commas",
			"url":      "https://bank.example.com/login",
			"notes":    "Security questions: mother's maiden name? Use generated answers.",
		},
		"Work-AWS": {
			"username": "admin@company.com",
			"password": "work-aws-secret",
			"url":      "https://aws.amazon.com",
			"notes":    "TOTP enabled; owner: Cloud Team",
		},
	}

	for path, want := range entryAssertions {
		entry, err := svc.GetEntry(prefix + path)
		if err != nil {
			t.Fatalf("get imported entry %q: %v", prefix+path, err)
		}
		for field, wantValue := range want {
			if entry.Data[field] != wantValue {
				t.Errorf("%s.%s = %#v, want %#v", prefix+path, field, entry.Data[field], wantValue)
			}
		}
	}
}

func snapshotImportEntries(t *testing.T, svc *vaultsvc.Service, paths []string) map[string]*vaultpkg.Entry {
	t.Helper()
	snapshot := make(map[string]*vaultpkg.Entry, len(paths))
	for _, path := range paths {
		entry, err := svc.GetEntry(path)
		if err != nil {
			t.Fatalf("get entry %q: %v", path, err)
		}
		snapshot[path] = entry
	}
	return snapshot
}
