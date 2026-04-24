package cmd

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/danieljustus/OpenPass/internal/config"
	vaultpkg "github.com/danieljustus/OpenPass/internal/vault"
)

func TestCreateBackup(t *testing.T) {
	vaultDir := t.TempDir()
	passphrase := "test-passphrase-123"
	cfg := config.Default()
	if _, err := vaultpkg.InitWithPassphrase(vaultDir, passphrase, cfg); err != nil {
		t.Fatalf("init vault: %v", err)
	}

	archivePath := filepath.Join(t.TempDir(), "backup.tar.gz")
	if err := createBackup(vaultDir, archivePath, false); err != nil {
		t.Fatalf("createBackup() error = %v", err)
	}

	if _, err := os.Stat(archivePath); err != nil {
		t.Fatalf("archive not created: %v", err)
	}
}

func TestCreateBackup_ExcludeGit(t *testing.T) {
	vaultDir := t.TempDir()
	passphrase := "test-passphrase-123"
	cfg := config.Default()
	if _, err := vaultpkg.InitWithPassphrase(vaultDir, passphrase, cfg); err != nil {
		t.Fatalf("init vault: %v", err)
	}

	gitDir := filepath.Join(vaultDir, ".git")
	if err := os.MkdirAll(gitDir, 0o700); err != nil {
		t.Fatalf("mkdir .git: %v", err)
	}
	if err := os.WriteFile(filepath.Join(gitDir, "config"), []byte("test"), 0o600); err != nil {
		t.Fatalf("write git config: %v", err)
	}

	archivePath := filepath.Join(t.TempDir(), "backup.tar.gz")
	if err := createBackup(vaultDir, archivePath, true); err != nil {
		t.Fatalf("createBackup() error = %v", err)
	}

	f, err := os.Open(archivePath)
	if err != nil {
		t.Fatalf("open archive: %v", err)
	}
	defer f.Close()

	gr, err := gzip.NewReader(f)
	if err != nil {
		t.Fatalf("gzip reader: %v", err)
	}
	defer gr.Close()

	tr := tar.NewReader(gr)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("read tar: %v", err)
		}
		if strings.HasPrefix(header.Name, ".git") {
			t.Fatalf("archive contains .git entry: %s", header.Name)
		}
	}
}

func TestRestoreBackup(t *testing.T) {
	vaultDir := t.TempDir()
	passphrase := "test-passphrase-123"
	cfg := config.Default()
	if _, err := vaultpkg.InitWithPassphrase(vaultDir, passphrase, cfg); err != nil {
		t.Fatalf("init vault: %v", err)
	}

	archivePath := filepath.Join(t.TempDir(), "backup.tar.gz")
	if err := createBackup(vaultDir, archivePath, false); err != nil {
		t.Fatalf("createBackup() error = %v", err)
	}

	restoreDir := t.TempDir()
	if err := restoreBackup(archivePath, restoreDir); err != nil {
		t.Fatalf("restoreBackup() error = %v", err)
	}

	if _, err := os.Stat(filepath.Join(restoreDir, "identity.age")); err != nil {
		t.Fatalf("identity.age not restored: %v", err)
	}
	if _, err := os.Stat(filepath.Join(restoreDir, "config.yaml")); err != nil {
		t.Fatalf("config.yaml not restored: %v", err)
	}
}

func TestRestoreBackup_PathTraversal(t *testing.T) {
	archivePath := filepath.Join(t.TempDir(), "evil.tar.gz")
	f, err := os.Create(archivePath)
	if err != nil {
		t.Fatalf("create archive: %v", err)
	}

	gw := gzip.NewWriter(f)
	tw := tar.NewWriter(gw)

	header := &tar.Header{
		Name: "../evil.txt",
		Mode: 0o600,
		Size: int64(len("evil")),
	}
	if err := tw.WriteHeader(header); err != nil {
		t.Fatalf("write header: %v", err)
	}
	if _, err := tw.Write([]byte("evil")); err != nil {
		t.Fatalf("write data: %v", err)
	}
	tw.Close()
	gw.Close()
	f.Close()

	restoreDir := t.TempDir()
	if err := restoreBackup(archivePath, restoreDir); err == nil {
		t.Fatal("expected error for path traversal in archive")
	}
}

func TestVerifyBackup_MissingIdentity(t *testing.T) {
	vaultDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(vaultDir, "config.yaml"), []byte("test"), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(vaultDir, "entries"), 0o700); err != nil {
		t.Fatalf("mkdir entries: %v", err)
	}

	if err := verifyBackup(vaultDir); err == nil {
		t.Fatal("expected error for missing identity.age")
	}
}

func TestVerifyBackup_MissingConfig(t *testing.T) {
	vaultDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(vaultDir, "identity.age"), []byte("test"), 0o600); err != nil {
		t.Fatalf("write identity: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(vaultDir, "entries"), 0o700); err != nil {
		t.Fatalf("mkdir entries: %v", err)
	}

	if err := verifyBackup(vaultDir); err == nil {
		t.Fatal("expected error for missing config.yaml")
	}
}

func TestVerifyBackup_MissingEntries(t *testing.T) {
	vaultDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(vaultDir, "identity.age"), []byte("test"), 0o600); err != nil {
		t.Fatalf("write identity: %v", err)
	}
	if err := os.WriteFile(filepath.Join(vaultDir, "config.yaml"), []byte("test"), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	if err := verifyBackup(vaultDir); err == nil {
		t.Fatal("expected error for missing entries directory")
	}
}

func TestVerifyBackup_Valid(t *testing.T) {
	vaultDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(vaultDir, "identity.age"), []byte("test"), 0o600); err != nil {
		t.Fatalf("write identity: %v", err)
	}
	if err := os.WriteFile(filepath.Join(vaultDir, "config.yaml"), []byte("test"), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(vaultDir, "entries"), 0o700); err != nil {
		t.Fatalf("mkdir entries: %v", err)
	}

	if err := verifyBackup(vaultDir); err != nil {
		t.Fatalf("verifyBackup() error = %v", err)
	}
}

func TestComputeSHA256(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "test.txt")
	content := []byte("hello world")
	if err := os.WriteFile(tmpFile, content, 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}

	hash, err := computeSHA256(tmpFile)
	if err != nil {
		t.Fatalf("computeSHA256() error = %v", err)
	}
	if len(hash) != 64 {
		t.Fatalf("hash length = %d, want 64", len(hash))
	}
}

func TestComputeSHA256_MissingFile(t *testing.T) {
	_, err := computeSHA256("/nonexistent/file.txt")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestBackupCommand(t *testing.T) {
	resetCommandTestState()
	t.Cleanup(resetCommandTestState)

	vaultDir := t.TempDir()
	passphrase := "test-passphrase-123"
	cfg := config.Default()
	if _, err := vaultpkg.InitWithPassphrase(vaultDir, passphrase, cfg); err != nil {
		t.Fatalf("init vault: %v", err)
	}

	archivePath := filepath.Join(t.TempDir(), "backup")

	prepareRootCommandOutput(t)
	rootCmd.SetArgs([]string{"--vault", vaultDir, "backup", archivePath})
	t.Cleanup(func() { rootCmd.SetArgs(nil) })

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if _, err := os.Stat(archivePath + ".tar.gz"); err != nil {
		t.Fatalf("archive not created: %v", err)
	}
}

func TestRestoreCommand(t *testing.T) {
	resetCommandTestState()
	t.Cleanup(resetCommandTestState)

	srcVault := t.TempDir()
	passphrase := "test-passphrase-123"
	cfg := config.Default()
	if _, err := vaultpkg.InitWithPassphrase(srcVault, passphrase, cfg); err != nil {
		t.Fatalf("init vault: %v", err)
	}

	archivePath := filepath.Join(t.TempDir(), "backup.tar.gz")
	if err := createBackup(srcVault, archivePath, false); err != nil {
		t.Fatalf("createBackup() error = %v", err)
	}

	restoreDir := t.TempDir()

	prepareRootCommandOutput(t)
	rootCmd.SetArgs([]string{"--vault", restoreDir, "restore", archivePath})
	t.Cleanup(func() { rootCmd.SetArgs(nil) })

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if _, err := os.Stat(filepath.Join(restoreDir, "identity.age")); err != nil {
		t.Fatalf("identity.age not restored: %v", err)
	}
}

func TestBackupCommand_UninitializedVault(t *testing.T) {
	resetCommandTestState()
	t.Cleanup(resetCommandTestState)

	vaultDir := t.TempDir()

	prepareRootCommandOutput(t)
	rootCmd.SetArgs([]string{"--vault", vaultDir, "backup", "/tmp/backup.tar.gz"})
	t.Cleanup(func() { rootCmd.SetArgs(nil) })

	if err := rootCmd.Execute(); err == nil {
		t.Fatal("expected error for uninitialized vault")
	}
}

func TestRestoreCommand_MissingArchive(t *testing.T) {
	resetCommandTestState()
	t.Cleanup(resetCommandTestState)

	vaultDir := t.TempDir()

	prepareRootCommandOutput(t)
	rootCmd.SetArgs([]string{"--vault", vaultDir, "restore", "/nonexistent/backup.tar.gz"})
	t.Cleanup(func() { rootCmd.SetArgs(nil) })

	if err := rootCmd.Execute(); err == nil {
		t.Fatal("expected error for missing archive")
	}
}

func TestRestoreCommand_CorruptArchive(t *testing.T) {
	resetCommandTestState()
	t.Cleanup(resetCommandTestState)

	archivePath := filepath.Join(t.TempDir(), "corrupt.tar.gz")
	if err := os.WriteFile(archivePath, []byte("not a valid archive"), 0o600); err != nil {
		t.Fatalf("write corrupt archive: %v", err)
	}

	vaultDir := t.TempDir()

	prepareRootCommandOutput(t)
	rootCmd.SetArgs([]string{"--vault", vaultDir, "restore", archivePath})
	t.Cleanup(func() { rootCmd.SetArgs(nil) })

	if err := rootCmd.Execute(); err == nil {
		t.Fatal("expected error for corrupt archive")
	}
}

func TestBackupCommand_ExcludeGit(t *testing.T) {
	resetCommandTestState()
	t.Cleanup(resetCommandTestState)

	vaultDir := t.TempDir()
	passphrase := "test-passphrase-123"
	cfg := config.Default()
	if _, err := vaultpkg.InitWithPassphrase(vaultDir, passphrase, cfg); err != nil {
		t.Fatalf("init vault: %v", err)
	}

	gitDir := filepath.Join(vaultDir, ".git")
	if err := os.MkdirAll(gitDir, 0o700); err != nil {
		t.Fatalf("mkdir .git: %v", err)
	}
	if err := os.WriteFile(filepath.Join(gitDir, "config"), []byte("test"), 0o600); err != nil {
		t.Fatalf("write git config: %v", err)
	}

	archivePath := filepath.Join(t.TempDir(), "backup")

	prepareRootCommandOutput(t)
	rootCmd.SetArgs([]string{"--vault", vaultDir, "backup", archivePath, "--exclude-git"})
	t.Cleanup(func() { rootCmd.SetArgs(nil) })

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if _, err := os.Stat(archivePath + ".tar.gz"); err != nil {
		t.Fatalf("archive not created: %v", err)
	}
}

func TestCreateBackup_UnreadableFile(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("running as root; chmod 0 has no effect")
	}

	vaultDir := t.TempDir()
	passphrase := "test-passphrase-123"
	cfg := config.Default()
	if _, err := vaultpkg.InitWithPassphrase(vaultDir, passphrase, cfg); err != nil {
		t.Fatalf("init vault: %v", err)
	}

	unreadableFile := filepath.Join(vaultDir, "unreadable.txt")
	if err := os.WriteFile(unreadableFile, []byte("test"), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}
	if err := os.Chmod(unreadableFile, 0o000); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	defer os.Chmod(unreadableFile, 0o600)

	archivePath := filepath.Join(t.TempDir(), "backup.tar.gz")
	if err := createBackup(vaultDir, archivePath, false); err == nil {
		t.Fatal("expected error for unreadable file")
	}
}
