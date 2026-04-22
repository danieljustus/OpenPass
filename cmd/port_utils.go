package cmd

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const runtimePortFileName = ".runtime-port"

func findAvailablePort(bind string, preferredPort int) (port int, isPreferred bool, err error) {
	addr := fmt.Sprintf("%s:%d", bind, preferredPort)
	listener, err := net.Listen("tcp", addr)
	if err == nil {
		if closeErr := listener.Close(); closeErr != nil {
			return 0, false, fmt.Errorf("close preferred port probe: %w", closeErr)
		}
		return preferredPort, true, nil
	}

	listener, err = net.Listen("tcp", fmt.Sprintf("%s:0", bind))
	if err != nil {
		return 0, false, fmt.Errorf("no available port found in range %s:*: %w", bind, err)
	}
	defer func() { _ = listener.Close() }()

	actualPort, ok := listener.Addr().(*net.TCPAddr)
	if !ok {
		return 0, false, fmt.Errorf("failed to get TCP address from listener")
	}
	return actualPort.Port, false, nil
}

func saveRuntimePort(vaultDir string, port int) error {
	portFile := filepath.Join(vaultDir, runtimePortFileName)
	return os.WriteFile(portFile, []byte(strconv.Itoa(port)), 0600)
}

func loadRuntimePort(vaultDir string) (int, bool) {
	portFile := filepath.Join(vaultDir, runtimePortFileName)
	data, err := os.ReadFile(portFile)
	if err != nil {
		return 0, false
	}
	port, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0, false
	}
	return port, true
}

func clearRuntimePort(vaultDir string) error {
	portFile := filepath.Join(vaultDir, runtimePortFileName)
	if err := os.Remove(portFile); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func resolvePort(vaultDir string, configuredPort int) int {
	if port, ok := loadRuntimePort(vaultDir); ok {
		return port
	}
	if configuredPort > 0 {
		return configuredPort
	}
	return 8080
}
