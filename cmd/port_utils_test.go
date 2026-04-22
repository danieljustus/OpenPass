package cmd

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"testing"
)

func TestFindAvailablePort(t *testing.T) {
	preferredPort := getFreePort(t)

	port, isPreferred, err := findAvailablePort("127.0.0.1", preferredPort)
	if err != nil {
		t.Fatalf("findAvailablePort failed: %v", err)
	}
	if !isPreferred {
		t.Errorf("expected preferred port, got alternative")
	}
	if port != preferredPort {
		t.Errorf("expected port %d, got %d", preferredPort, port)
	}
}

func TestFindAvailablePort_WhenInUse(t *testing.T) {
	preferredPort := getFreePort(t)

	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", preferredPort))
	if err != nil {
		t.Fatalf("failed to occupy port: %v", err)
	}
	defer func() { _ = listener.Close() }()

	port, isPreferred, err := findAvailablePort("127.0.0.1", preferredPort)
	if err != nil {
		t.Fatalf("findAvailablePort failed: %v", err)
	}
	if isPreferred {
		t.Errorf("expected alternative port when preferred is in use")
	}
	if port == preferredPort {
		t.Errorf("expected different port when preferred is in use, got %d", port)
	}
	if port <= 0 {
		t.Errorf("expected valid port, got %d", port)
	}
}

func TestRuntimePortPersistence(t *testing.T) {
	tmpDir := t.TempDir()

	if _, ok := loadRuntimePort(tmpDir); ok {
		t.Error("expected no runtime port for empty directory")
	}

	testPort := 9999
	if err := saveRuntimePort(tmpDir, testPort); err != nil {
		t.Fatalf("saveRuntimePort failed: %v", err)
	}

	port, ok := loadRuntimePort(tmpDir)
	if !ok {
		t.Error("expected to load saved runtime port")
	}
	if port != testPort {
		t.Errorf("expected port %d, got %d", testPort, port)
	}

	if err := clearRuntimePort(tmpDir); err != nil {
		t.Fatalf("clearRuntimePort failed: %v", err)
	}

	if _, ok := loadRuntimePort(tmpDir); ok {
		t.Error("expected runtime port to be cleared")
	}
}

func TestResolvePort(t *testing.T) {
	tmpDir := t.TempDir()

	port := resolvePort(tmpDir, 0)
	if port != 8080 {
		t.Errorf("expected default port 8080, got %d", port)
	}

	port = resolvePort(tmpDir, 9090)
	if port != 9090 {
		t.Errorf("expected configured port 9090, got %d", port)
	}

	if err := saveRuntimePort(tmpDir, 7777); err != nil {
		t.Fatalf("saveRuntimePort failed: %v", err)
	}

	port = resolvePort(tmpDir, 9090)
	if port != 7777 {
		t.Errorf("expected runtime port 7777 to override configured port, got %d", port)
	}
}

func TestClearRuntimePort_Idempotent(t *testing.T) {
	tmpDir := t.TempDir()

	if err := clearRuntimePort(tmpDir); err != nil {
		t.Errorf("clearRuntimePort on non-existent file should not error, got: %v", err)
	}
}

func TestSaveRuntimePort_InvalidDirectory(t *testing.T) {
	nonExistentDir := filepath.Join(t.TempDir(), "does-not-exist")
	err := saveRuntimePort(nonExistentDir, 8080)
	if err == nil {
		t.Error("expected error when saving to non-existent directory")
	}
}

func TestLoadRuntimePort_InvalidContent(t *testing.T) {
	tmpDir := t.TempDir()
	portFile := filepath.Join(tmpDir, runtimePortFileName)

	if err := os.WriteFile(portFile, []byte("not-a-number"), 0600); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	if _, ok := loadRuntimePort(tmpDir); ok {
		t.Error("expected load to fail with invalid content")
	}
}

func getFreePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("find free port: %v", err)
	}
	tcpAddr, ok := l.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatalf("failed to get TCP address from listener")
	}
	port := tcpAddr.Port
	if err := l.Close(); err != nil {
		t.Fatalf("close probe listener: %v", err)
	}
	return port
}
