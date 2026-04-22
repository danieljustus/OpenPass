//go:build smoke

package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestBetaSmokeFlow(t *testing.T) {
	binDir := t.TempDir()
	binPath := filepath.Join(binDir, "openpass")

	build := exec.Command("go", "build", "-o", binPath, ".")
	build.Dir = repoRoot(t)
	build.Env = append(os.Environ(), "GOWORK=off")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build openpass: %v\n%s", err, output)
	}

	vaultDir := filepath.Join(t.TempDir(), "vault")
	passphrase := "correct horse battery staple"

	initCmd := exec.Command(binPath, "init", vaultDir)
	initCmd.Dir = repoRoot(t)
	initCmd.Stdin = strings.NewReader(passphrase + "\n")
	if output, err := initCmd.CombinedOutput(); err != nil {
		t.Fatalf("init vault: %v\n%s", err, output)
	}

	run := func(args ...string) string {
		t.Helper()

		cmd := exec.Command(binPath, args...)
		cmd.Dir = repoRoot(t)
		cmd.Env = append(os.Environ(),
			"GOWORK=off",
			"OPENPASS_PASSPHRASE="+passphrase,
		)
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("%s: %v\n%s", strings.Join(args, " "), err, output)
		}
		return string(output)
	}

	run("--vault", vaultDir, "set", "demo.password", "--value", "hunter2")

	if output := strings.TrimSpace(run("--vault", vaultDir, "get", "demo.password")); output != "hunter2" {
		t.Fatalf("get demo.password = %q, want hunter2", output)
	}

	listOutput := run("--vault", vaultDir, "list")
	if !strings.Contains(listOutput, "demo") {
		t.Fatalf("list output missing entry: %s", listOutput)
	}

	findOutput := run("--vault", vaultDir, "find", "hunter2")
	if !strings.Contains(findOutput, "demo") {
		t.Fatalf("find output missing match: %s", findOutput)
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()

	root, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	return root
}
