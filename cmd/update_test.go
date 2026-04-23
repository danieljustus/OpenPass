package cmd

import (
	"bytes"
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	updatepkg "github.com/danieljustus/OpenPass/internal/update"
)

type stubUpdateChecker struct {
	currentVersion string
	err            error
	result         *updatepkg.Result
}

func (s *stubUpdateChecker) Check(_ context.Context, currentVersion string) (*updatepkg.Result, error) {
	s.currentVersion = currentVersion
	return s.result, s.err
}

func prepareRootCommandOutput(t *testing.T) *bytes.Buffer {
	t.Helper()

	resetCommandTestState()
	t.Cleanup(resetCommandTestState)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	return buf
}

func setUpdateCheckerForTest(t *testing.T, checker updateChecker) {
	t.Helper()

	original := updateCheckerFactory
	updateCheckerFactory = func() updateChecker { return checker }
	t.Cleanup(func() { updateCheckerFactory = original })
}

func setVersionInfoForTest(t *testing.T, version string) {
	t.Helper()

	originalVersion := appVersion
	originalCommit := appCommit
	originalDate := appDate
	SetVersionInfo(version, "test-commit", "test-date")
	t.Cleanup(func() {
		SetVersionInfo(originalVersion, originalCommit, originalDate)
	})
}

func TestUpdateCheckCommandReportsAvailableUpdate(t *testing.T) {
	buf := prepareRootCommandOutput(t)
	setVersionInfoForTest(t, "1.0.0")

	checker := &stubUpdateChecker{
		result: &updatepkg.Result{
			CurrentVersion:  "1.0.0",
			LatestVersion:   "1.1.0",
			ReleaseURL:      "https://github.com/danieljustus/OpenPass/releases/tag/v1.1.0",
			Checkable:       true,
			UpdateAvailable: true,
		},
	}
	setUpdateCheckerForTest(t, checker)

	rootCmd.SetArgs([]string{"update", "check"})
	t.Cleanup(func() { rootCmd.SetArgs(nil) })

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	got := buf.String()
	for _, want := range []string{
		"Update available: 1.0.0 -> 1.1.0",
		"https://github.com/danieljustus/OpenPass/releases/tag/v1.1.0",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("output missing %q: %q", want, got)
		}
	}
	if checker.currentVersion != "1.0.0" {
		t.Fatalf("checker received version %q, want %q", checker.currentVersion, "1.0.0")
	}
}

func TestUpdateCheckCommandReportsUpToDate(t *testing.T) {
	buf := prepareRootCommandOutput(t)
	setVersionInfoForTest(t, "1.0.0")

	setUpdateCheckerForTest(t, &stubUpdateChecker{
		result: &updatepkg.Result{
			CurrentVersion: "1.0.0",
			LatestVersion:  "1.0.0",
			Checkable:      true,
		},
	})

	rootCmd.SetArgs([]string{"update", "check"})
	t.Cleanup(func() { rootCmd.SetArgs(nil) })

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	got := buf.String()
	if !strings.Contains(got, "OpenPass is up to date (1.0.0).") {
		t.Fatalf("unexpected output: %q", got)
	}
}

func TestUpdateCheckCommandHandlesNonReleaseBuild(t *testing.T) {
	buf := prepareRootCommandOutput(t)
	setVersionInfoForTest(t, "dev")

	checker := &stubUpdateChecker{
		result: &updatepkg.Result{
			CurrentVersion: "dev",
			Checkable:      false,
		},
	}
	setUpdateCheckerForTest(t, checker)

	rootCmd.SetArgs([]string{"update", "check"})
	t.Cleanup(func() { rootCmd.SetArgs(nil) })

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	got := buf.String()
	if !strings.Contains(got, "Update checks are only available for stable release builds. Current version: dev") {
		t.Fatalf("unexpected output: %q", got)
	}
	if checker.currentVersion != "dev" {
		t.Fatalf("checker received version %q, want %q", checker.currentVersion, "dev")
	}
}

func TestUpdateCheckCommandReturnsCheckerError(t *testing.T) {
	prepareRootCommandOutput(t)
	setVersionInfoForTest(t, "1.0.0")
	setUpdateCheckerForTest(t, &stubUpdateChecker{err: errors.New("boom")})

	rootCmd.SetArgs([]string{"update", "check"})
	t.Cleanup(func() { rootCmd.SetArgs(nil) })

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected Execute() to return an error")
	}
	if !strings.Contains(err.Error(), "check for updates: boom") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUpdateCheckCommandDoesNotRequireVault(t *testing.T) {
	buf := prepareRootCommandOutput(t)
	setVersionInfoForTest(t, "dev")
	setUpdateCheckerForTest(t, &stubUpdateChecker{
		result: &updatepkg.Result{
			CurrentVersion: "dev",
			Checkable:      false,
		},
	})

	originalHome := os.Getenv("HOME")
	originalVaultEnv := os.Getenv("OPENPASS_VAULT")
	originalVault := vault
	originalChanged := vaultFlag.Changed
	_ = os.Unsetenv("HOME")
	_ = os.Unsetenv("OPENPASS_VAULT")
	vault = "~/.openpass"
	if vaultFlag != nil {
		_ = vaultFlag.Value.Set(vault)
		vaultFlag.Changed = false
	}
	t.Cleanup(func() {
		_ = os.Setenv("HOME", originalHome)
		_ = os.Setenv("OPENPASS_VAULT", originalVaultEnv)
		vault = originalVault
		if vaultFlag != nil {
			_ = vaultFlag.Value.Set(originalVault)
			vaultFlag.Changed = originalChanged
		}
	})

	rootCmd.SetArgs([]string{"update", "check"})
	t.Cleanup(func() { rootCmd.SetArgs(nil) })

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !strings.Contains(buf.String(), "stable release builds") {
		t.Fatalf("unexpected output: %q", buf.String())
	}
}
