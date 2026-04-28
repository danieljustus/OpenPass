package cmd

import (
	"fmt"
	"os"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/danieljustus/OpenPass/internal/config"
	"github.com/danieljustus/OpenPass/internal/mcp"
	vaultpkg "github.com/danieljustus/OpenPass/internal/vault"
)

func resetVaultState(t *testing.T) {
	t.Helper()
	resetCommandTestState()
	t.Cleanup(resetCommandTestState)
}

func TestAdd_ErrorPaths(t *testing.T) {
	resetVaultState(t)
	tests := []struct {
		setupFunc  func()
		name       string
		errContain string
		args       []string
		wantErr    bool
	}{
		{
			name: "uninitialized vault",
			setupFunc: func() {
				tmpDir := t.TempDir()
				_ = os.Setenv("OPENPASS_VAULT", tmpDir)
			},
			args:       []string{"--vault", os.TempDir() + "/nonexistent", "add", "test"},
			wantErr:    true,
			errContain: "not initialized",
		},
		{
			name: "entry already exists",
			setupFunc: func() {
				tmpDir := t.TempDir()
				_ = os.Setenv("OPENPASS_VAULT", tmpDir)
				cfg := config.Default()
				identity, _ := vaultpkg.InitWithPassphrase(tmpDir, "testpass", cfg)
				_ = os.Setenv("OPENPASS_PASSPHRASE", "testpass")
				_ = vaultpkg.WriteEntry(tmpDir, "existing", &vaultpkg.Entry{Data: map[string]any{"password": "secret"}}, identity)
			},
			args:       []string{"add", "existing", "--value", "new"},
			wantErr:    true,
			errContain: "already exists",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_ = os.Unsetenv("OPENPASS_VAULT")
			_ = os.Unsetenv("OPENPASS_PASSPHRASE")

			if tt.setupFunc != nil {
				tt.setupFunc()
			}

			vault = ""
			if vaultFlag != nil {
				_ = vaultFlag.Value.Set("")
				vaultFlag.Changed = false
			}

			rootCmd.SetArgs(tt.args)
			defer rootCmd.SetArgs(nil)

			err := rootCmd.Execute()
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				} else if tt.errContain != "" && !strings.Contains(err.Error(), tt.errContain) {
					t.Errorf("error %q does not contain %q", err.Error(), tt.errContain)
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestDelete_ErrorPaths(t *testing.T) {
	resetVaultState(t)
	t.Run("uninitialized vault", func(t *testing.T) {
		tmpDir := t.TempDir()
		_ = os.Setenv("OPENPASS_VAULT", tmpDir)
		defer func() { _ = os.Unsetenv("OPENPASS_VAULT") }()

		rootCmd.SetArgs([]string{"--vault", tmpDir, "delete", "test"})
		defer rootCmd.SetArgs(nil)

		err := rootCmd.Execute()
		if err == nil || !strings.Contains(err.Error(), "not initialized") {
			t.Errorf("expected 'not initialized' error, got: %v", err)
		}
	})

	t.Run("delete canceled", func(t *testing.T) {
		tmpDir := t.TempDir()
		_ = os.Setenv("OPENPASS_VAULT", tmpDir)
		defer func() { _ = os.Unsetenv("OPENPASS_VAULT") }()

		cfg := config.Default()
		_, _ = vaultpkg.InitWithPassphrase(tmpDir, "test", cfg)
		_ = os.Setenv("OPENPASS_PASSPHRASE", "test")
		defer func() { _ = os.Unsetenv("OPENPASS_PASSPHRASE") }()

		oldStdin := os.Stdin
		r, w, _ := os.Pipe()
		os.Stdin = r
		_, _ = w.WriteString("n\n")
		_ = w.Close()

		rootCmd.SetArgs([]string{"--vault", tmpDir, "delete", "nonexistent"})
		defer rootCmd.SetArgs(nil)

		_ = rootCmd.Execute()
		os.Stdin = oldStdin
		_ = r.Close()
	})
}

func TestGet_ErrorPaths(t *testing.T) {
	resetVaultState(t)
	tests := []struct {
		setupFunc  func() string
		name       string
		errContain string
		args       []string
	}{
		{
			name: "uninitialized vault",
			setupFunc: func() string {
				tmpDir := t.TempDir()
				_ = os.Setenv("OPENPASS_VAULT", tmpDir)
				return tmpDir
			},
			args:       []string{"get", "test"},
			errContain: "not initialized",
		},
		{
			name: "entry not found",
			setupFunc: func() string {
				tmpDir := t.TempDir()
				_ = os.Setenv("OPENPASS_VAULT", tmpDir)
				cfg := config.Default()
				_, _ = vaultpkg.InitWithPassphrase(tmpDir, "test", cfg)
				_ = os.Setenv("OPENPASS_PASSPHRASE", "test")
				return tmpDir
			},
			args:       []string{"get", "nonexistent"},
			errContain: "not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_ = os.Unsetenv("OPENPASS_VAULT")
			_ = os.Unsetenv("OPENPASS_PASSPHRASE")
			vault = ""
			if vaultFlag != nil {
				_ = vaultFlag.Value.Set("")
				vaultFlag.Changed = false
			}

			vaultDir := tt.setupFunc()

			rootCmd.SetArgs(append([]string{"--vault", vaultDir}, tt.args...))
			defer rootCmd.SetArgs(nil)

			err := rootCmd.Execute()
			if err == nil {
				t.Errorf("expected error, got nil")
			} else if !strings.Contains(err.Error(), tt.errContain) {
				t.Errorf("error %q does not contain %q", err.Error(), tt.errContain)
			}
		})
	}
}

func TestFind_ErrorPaths(t *testing.T) {
	resetVaultState(t)
	t.Run("uninitialized vault", func(t *testing.T) {
		tmpDir := t.TempDir()
		_ = os.Setenv("OPENPASS_VAULT", tmpDir)
		defer func() { _ = os.Unsetenv("OPENPASS_VAULT") }()

		rootCmd.SetArgs([]string{"--vault", tmpDir, "find", "test"})
		defer rootCmd.SetArgs(nil)

		err := rootCmd.Execute()
		if err == nil || !strings.Contains(err.Error(), "not initialized") {
			t.Errorf("expected 'not initialized' error, got: %v", err)
		}
	})
}

func TestList_ErrorPaths(t *testing.T) {
	resetVaultState(t)
	t.Run("uninitialized vault", func(t *testing.T) {
		tmpDir := t.TempDir()
		_ = os.Setenv("OPENPASS_VAULT", tmpDir)
		defer func() { _ = os.Unsetenv("OPENPASS_VAULT") }()

		rootCmd.SetArgs([]string{"--vault", tmpDir, "list"})
		defer rootCmd.SetArgs(nil)

		err := rootCmd.Execute()
		if err == nil || !strings.Contains(err.Error(), "not initialized") {
			t.Errorf("expected 'not initialized' error, got: %v", err)
		}
	})
}

func TestEdit_ErrorPaths(t *testing.T) {
	resetVaultState(t)
	tests := []struct {
		name       string
		setupFunc  func() string
		errContain string
	}{
		{
			name: "uninitialized vault",
			setupFunc: func() string {
				tmpDir := t.TempDir()
				_ = os.Setenv("OPENPASS_VAULT", tmpDir)
				return tmpDir
			},
			errContain: "not initialized",
		},
		{
			name: "entry not found",
			setupFunc: func() string {
				tmpDir := t.TempDir()
				_ = os.Setenv("OPENPASS_VAULT", tmpDir)
				cfg := config.Default()
				_, _ = vaultpkg.InitWithPassphrase(tmpDir, "test", cfg)
				_ = os.Setenv("OPENPASS_PASSPHRASE", "test")
				return tmpDir
			},
			errContain: "not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_ = os.Unsetenv("OPENPASS_VAULT")
			_ = os.Unsetenv("OPENPASS_PASSPHRASE")
			vault = ""
			if vaultFlag != nil {
				_ = vaultFlag.Value.Set("")
				vaultFlag.Changed = false
			}

			vaultDir := tt.setupFunc()

			rootCmd.SetArgs([]string{"--vault", vaultDir, "edit", "nonexistent"})
			defer rootCmd.SetArgs(nil)

			err := rootCmd.Execute()
			if err == nil {
				t.Errorf("expected error, got nil")
			} else if !strings.Contains(err.Error(), tt.errContain) {
				t.Errorf("error %q does not contain %q", err.Error(), tt.errContain)
			}
		})
	}
}

func TestGenerate_ErrorPaths(t *testing.T) {
	resetVaultState(t)
	t.Run("invalid length", func(t *testing.T) {
		tmpDir := t.TempDir()
		_ = os.Setenv("OPENPASS_VAULT", tmpDir)
		_ = os.Setenv("OPENPASS_PASSPHRASE", "test")
		defer func() {
			_ = os.Unsetenv("OPENPASS_VAULT")
			_ = os.Unsetenv("OPENPASS_PASSPHRASE")
		}()

		cfg := config.Default()
		_, _ = vaultpkg.InitWithPassphrase(tmpDir, "test", cfg)

		rootCmd.SetArgs([]string{"--vault", tmpDir, "generate", "--length", "0"})
		defer rootCmd.SetArgs(nil)

		err := rootCmd.Execute()
		if err == nil {
			t.Error("expected error for zero length")
		}
	})
}

func TestLock_ErrorPaths(t *testing.T) {
	resetVaultState(t)
	t.Run("uninitialized vault", func(t *testing.T) {
		tmpDir := t.TempDir()
		_ = os.Setenv("OPENPASS_VAULT", tmpDir)
		defer func() { _ = os.Unsetenv("OPENPASS_VAULT") }()

		rootCmd.SetArgs([]string{"--vault", tmpDir, "lock"})
		defer rootCmd.SetArgs(nil)

		err := rootCmd.Execute()
		if err == nil || !strings.Contains(err.Error(), "not initialized") {
			t.Errorf("expected 'not initialized' error, got: %v", err)
		}
	})
}

func TestRecipients_ErrorPaths(t *testing.T) {
	resetVaultState(t)
	t.Run("list - uninitialized vault", func(t *testing.T) {
		tmpDir := t.TempDir()
		_ = os.Setenv("OPENPASS_VAULT", tmpDir)
		defer func() { _ = os.Unsetenv("OPENPASS_VAULT") }()

		rootCmd.SetArgs([]string{"--vault", tmpDir, "recipients", "list"})
		defer rootCmd.SetArgs(nil)

		err := rootCmd.Execute()
		if err == nil || !strings.Contains(err.Error(), "not initialized") {
			t.Errorf("expected 'not initialized' error, got: %v", err)
		}
	})

	t.Run("add - invalid recipient", func(t *testing.T) {
		tmpDir := t.TempDir()
		_ = os.Setenv("OPENPASS_VAULT", tmpDir)
		_ = os.Setenv("OPENPASS_PASSPHRASE", "test")
		defer func() {
			_ = os.Unsetenv("OPENPASS_VAULT")
			_ = os.Unsetenv("OPENPASS_PASSPHRASE")
		}()

		cfg := config.Default()
		_, _ = vaultpkg.InitWithPassphrase(tmpDir, "test", cfg)

		rootCmd.SetArgs([]string{"--vault", tmpDir, "recipients", "add", "invalid-key"})
		defer rootCmd.SetArgs(nil)

		err := rootCmd.Execute()
		if err == nil || !strings.Contains(err.Error(), "invalid") {
			t.Errorf("expected 'invalid' error, got: %v", err)
		}
	})

	t.Run("remove - invalid recipient format", func(t *testing.T) {
		tmpDir := t.TempDir()
		_ = os.Setenv("OPENPASS_VAULT", tmpDir)
		_ = os.Setenv("OPENPASS_PASSPHRASE", "test")
		defer func() {
			_ = os.Unsetenv("OPENPASS_VAULT")
			_ = os.Unsetenv("OPENPASS_PASSPHRASE")
		}()

		cfg := config.Default()
		_, _ = vaultpkg.InitWithPassphrase(tmpDir, "test", cfg)

		rootCmd.SetArgs([]string{"--vault", tmpDir, "recipients", "remove", "not-age1-key", "-y"})
		defer rootCmd.SetArgs(nil)

		err := rootCmd.Execute()
		if err == nil || !strings.Contains(err.Error(), "invalid") {
			t.Errorf("expected 'invalid' error, got: %v", err)
		}
	})

	t.Run("remove - recipient not in list", func(t *testing.T) {
		tmpDir := t.TempDir()
		_ = os.Setenv("OPENPASS_VAULT", tmpDir)
		_ = os.Setenv("OPENPASS_PASSPHRASE", "test")
		defer func() {
			_ = os.Unsetenv("OPENPASS_VAULT")
			_ = os.Unsetenv("OPENPASS_PASSPHRASE")
		}()

		cfg := config.Default()
		_, _ = vaultpkg.InitWithPassphrase(tmpDir, "test", cfg)

		_ = os.Setenv("OPENPASS_PASSPHRASE", "test")
		identity2, _ := vaultpkg.InitWithPassphrase(tmpDir+"_second", "test2", cfg)

		rootCmd.SetArgs([]string{"--vault", tmpDir, "recipients", "remove", identity2.Recipient().String(), "-y"})
		defer rootCmd.SetArgs(nil)

		err := rootCmd.Execute()
		if err == nil || !strings.Contains(err.Error(), "not found") {
			t.Errorf("expected 'not found' error, got: %v", err)
		}
	})
}

func TestInit_ErrorPaths(t *testing.T) {
	resetVaultState(t)
	t.Run("already initialized", func(t *testing.T) {
		tmpDir := t.TempDir()
		cfg := config.Default()
		_, _ = vaultpkg.InitWithPassphrase(tmpDir, "test", cfg)

		_ = os.Setenv("OPENPASS_VAULT", tmpDir)
		defer func() { _ = os.Unsetenv("OPENPASS_VAULT") }()

		rootCmd.SetArgs([]string{"--vault", tmpDir, "init"})
		defer rootCmd.SetArgs(nil)

		err := rootCmd.Execute()
		if err == nil || !strings.Contains(err.Error(), "already initialized") {
			t.Errorf("expected 'already initialized' error, got: %v", err)
		}
	})
}

func TestSet_ErrorPaths(t *testing.T) {
	resetVaultState(t)
	t.Run("uninitialized vault", func(t *testing.T) {
		tmpDir := t.TempDir()
		_ = os.Setenv("OPENPASS_VAULT", tmpDir)
		defer func() { _ = os.Unsetenv("OPENPASS_VAULT") }()

		rootCmd.SetArgs([]string{"--vault", tmpDir, "set", "test.key", "--value", "val"})
		defer rootCmd.SetArgs(nil)

		err := rootCmd.Execute()
		if err == nil || !strings.Contains(err.Error(), "not initialized") {
			t.Errorf("expected 'not initialized' error, got: %v", err)
		}
	})
}

func TestUnlock_ErrorPaths(t *testing.T) {
	resetVaultState(t)
	t.Run("uninitialized vault", func(t *testing.T) {
		tmpDir := t.TempDir()
		_ = os.Setenv("OPENPASS_VAULT", tmpDir)
		defer func() { _ = os.Unsetenv("OPENPASS_VAULT") }()

		rootCmd.SetArgs([]string{"--vault", tmpDir, "unlock"})
		defer rootCmd.SetArgs(nil)

		err := rootCmd.Execute()
		if err == nil || !strings.Contains(err.Error(), "not initialized") {
			t.Errorf("expected 'not initialized' error, got: %v", err)
		}
	})

	t.Run("wrong passphrase", func(t *testing.T) {
		tmpDir := t.TempDir()
		cfg := config.Default()
		_, _ = vaultpkg.InitWithPassphrase(tmpDir, "test", cfg)

		_ = os.Setenv("OPENPASS_VAULT", tmpDir)
		_ = os.Setenv("OPENPASS_PASSPHRASE", "wrong-password")
		defer func() {
			_ = os.Unsetenv("OPENPASS_VAULT")
			_ = os.Unsetenv("OPENPASS_PASSPHRASE")
		}()

		rootCmd.SetArgs([]string{"--vault", tmpDir, "unlock"})
		defer rootCmd.SetArgs(nil)

		err := rootCmd.Execute()
		if err == nil {
			t.Error("expected error for wrong passphrase")
		}
	})
}

func TestServe_ErrorPaths(t *testing.T) {
	resetVaultState(t)
	t.Run("uninitialized vault", func(t *testing.T) {
		tmpDir := t.TempDir()
		_ = os.Setenv("OPENPASS_VAULT", tmpDir)
		defer func() { _ = os.Unsetenv("OPENPASS_VAULT") }()

		rootCmd.SetArgs([]string{"--vault", tmpDir, "serve", "--port", "0"})
		defer rootCmd.SetArgs(nil)

		err := rootCmd.Execute()
		if err == nil || !strings.Contains(err.Error(), "not initialized") {
			t.Errorf("expected 'not initialized' error, got: %v", err)
		}
	})

	t.Run("stdio without agent", func(t *testing.T) {
		tmpDir := t.TempDir()
		cfg := config.Default()
		_, _ = vaultpkg.InitWithPassphrase(tmpDir, "test", cfg)

		_ = os.Setenv("OPENPASS_VAULT", tmpDir)
		_ = os.Setenv("OPENPASS_PASSPHRASE", "test")
		defer func() {
			_ = os.Unsetenv("OPENPASS_VAULT")
			_ = os.Unsetenv("OPENPASS_PASSPHRASE")
		}()

		rootCmd.SetArgs([]string{"--vault", tmpDir, "serve", "--stdio"})
		defer rootCmd.SetArgs(nil)

		err := rootCmd.Execute()
		if err == nil || !strings.Contains(err.Error(), "--agent is required") {
			t.Errorf("expected '--agent is required' error, got: %v", err)
		}
	})

	t.Run("empty bind address", func(t *testing.T) {
		tmpDir := t.TempDir()
		cfg := config.Default()
		_, _ = vaultpkg.InitWithPassphrase(tmpDir, "test", cfg)

		_ = os.Setenv("OPENPASS_VAULT", tmpDir)
		_ = os.Setenv("OPENPASS_PASSPHRASE", "test")
		defer func() {
			_ = os.Unsetenv("OPENPASS_VAULT")
			_ = os.Unsetenv("OPENPASS_PASSPHRASE")
		}()

		rootCmd.SetArgs([]string{"--vault", tmpDir, "serve", "--bind", ""})
		defer rootCmd.SetArgs(nil)

		err := rootCmd.Execute()
		if err == nil || !strings.Contains(err.Error(), "bind") {
			t.Errorf("expected bind error, got: %v", err)
		}
	})
}

func TestAdd_InteractiveMode(t *testing.T) {
	resetVaultState(t)

	tmpDir := t.TempDir()
	_ = os.Setenv("OPENPASS_VAULT", tmpDir)
	_ = os.Setenv("OPENPASS_PASSPHRASE", "test")
	defer func() {
		_ = os.Unsetenv("OPENPASS_VAULT")
		_ = os.Unsetenv("OPENPASS_PASSPHRASE")
	}()

	cfg := config.Default()
	_, _ = vaultpkg.InitWithPassphrase(tmpDir, "test", cfg)

	// Reset all add flags to ensure we're in interactive mode
	addValue = ""
	addGenerate = false
	addUsername = ""
	addURL = ""
	addNotes = ""
	addTOTPSecret = ""
	addTOTPIssuer = ""
	addTOTPAccount = ""

	// Create pipe for stdin with interactive input
	oldStdin := os.Stdin
	r, w, _ := os.Pipe()
	os.Stdin = r

	// Provide interactive input:
	// 1. Username
	// 2. Password
	// 3. URL
	// 4. Notes (line1, line2, empty line to end)
	// 5. TOTP Secret
	// 6. TOTP Issuer
	// 7. TOTP Account
	go func() {
		_, _ = w.WriteString("myuser\n")
		_, _ = w.WriteString("StrongP@ssw0rd123\n")
		_, _ = w.WriteString("https://example.com\n")
		_, _ = w.WriteString("note1\n")
		_, _ = w.WriteString("note2\n")
		_, _ = w.WriteString("\n")
		_, _ = w.WriteString("GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ\n")
		_, _ = w.WriteString("Example\n")
		_, _ = w.WriteString("myaccount\n")
		_ = w.Close()
	}()

	rootCmd.SetArgs([]string{"--vault", tmpDir, "add", "interactive-test"})
	defer rootCmd.SetArgs(nil)

	output := captureStdout(func() {
		_ = rootCmd.Execute()
	})

	os.Stdin = oldStdin
	_ = r.Close()

	if !strings.Contains(output, "Entry created") {
		t.Errorf("expected 'Entry created', got: %s", output)
	}
}

func TestSet_InteractiveMode(t *testing.T) {
	resetVaultState(t)

	tmpDir := t.TempDir()
	_ = os.Setenv("OPENPASS_VAULT", tmpDir)
	_ = os.Setenv("OPENPASS_PASSPHRASE", "test")
	defer func() {
		_ = os.Unsetenv("OPENPASS_VAULT")
		_ = os.Unsetenv("OPENPASS_PASSPHRASE")
	}()

	cfg := config.Default()
	_, _ = vaultpkg.InitWithPassphrase(tmpDir, "test", cfg)

	setValue = ""
	setTOTPSecret = ""
	setTOTPIssuer = ""
	setTOTPAccount = ""

	oldStdin := os.Stdin
	r, w, _ := os.Pipe()
	os.Stdin = r

	go func() {
		_, _ = w.WriteString("myuser\n")
		_, _ = w.WriteString("StrongP@ssw0rd123\n")
		_, _ = w.WriteString("https://example.com\n")
		_, _ = w.WriteString("GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ\n")
		_ = w.Close()
	}()

	rootCmd.SetArgs([]string{"--vault", tmpDir, "set", "interactive-set"})
	defer rootCmd.SetArgs(nil)

	output := captureStdout(func() {
		_ = rootCmd.Execute()
	})

	os.Stdin = oldStdin
	_ = r.Close()

	if !strings.Contains(output, "Entry saved") {
		t.Errorf("expected 'Entry saved', got: %s", output)
	}
}

func TestSet_InteractiveMode_Field(t *testing.T) {
	resetVaultState(t)

	tmpDir := t.TempDir()
	_ = os.Setenv("OPENPASS_VAULT", tmpDir)
	_ = os.Setenv("OPENPASS_PASSPHRASE", "test")
	defer func() {
		_ = os.Unsetenv("OPENPASS_VAULT")
		_ = os.Unsetenv("OPENPASS_PASSPHRASE")
	}()

	cfg := config.Default()
	identity, _ := vaultpkg.InitWithPassphrase(tmpDir, "test", cfg)
	_ = vaultpkg.WriteEntry(tmpDir, "test", &vaultpkg.Entry{Data: map[string]any{"password": "secret"}}, identity)

	setValue = ""

	oldStdin := os.Stdin
	r, w, _ := os.Pipe()
	os.Stdin = r

	go func() {
		_, _ = w.WriteString("fieldvalue\n")
		_ = w.Close()
	}()

	rootCmd.SetArgs([]string{"--vault", tmpDir, "set", "test.customfield"})
	defer rootCmd.SetArgs(nil)

	output := captureStdout(func() {
		_ = rootCmd.Execute()
	})

	os.Stdin = oldStdin
	_ = r.Close()

	if !strings.Contains(output, "Entry saved") {
		t.Errorf("expected 'Entry saved', got: %s", output)
	}
}

func TestSet_InteractiveReadErrors(t *testing.T) {
	resetVaultState(t)

	tests := []struct {
		name string
		path string
		err  string
	}{
		{"field value EOF", "test.field", "read value"},
		{"username EOF", "test", "read username"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			_ = os.Setenv("OPENPASS_VAULT", tmpDir)
			_ = os.Setenv("OPENPASS_PASSPHRASE", "test")
			defer func() {
				_ = os.Unsetenv("OPENPASS_VAULT")
				_ = os.Unsetenv("OPENPASS_PASSPHRASE")
			}()

			cfg := config.Default()
			_, _ = vaultpkg.InitWithPassphrase(tmpDir, "test", cfg)

			setValue = ""
			setTOTPSecret = ""

			oldStdin := os.Stdin
			r, w, _ := os.Pipe()
			os.Stdin = r
			_ = w.Close()

			rootCmd.SetArgs([]string{"--vault", tmpDir, "set", tt.path})
			defer rootCmd.SetArgs(nil)

			err := rootCmd.Execute()
			os.Stdin = oldStdin
			_ = r.Close()

			if err == nil || !strings.Contains(err.Error(), tt.err) {
				t.Errorf("expected %q error, got: %v", tt.err, err)
			}
		})
	}
}

func TestAdd_InteractiveReadErrors(t *testing.T) {
	resetVaultState(t)

	tmpDir := t.TempDir()
	_ = os.Setenv("OPENPASS_VAULT", tmpDir)
	_ = os.Setenv("OPENPASS_PASSPHRASE", "test")
	defer func() {
		_ = os.Unsetenv("OPENPASS_VAULT")
		_ = os.Unsetenv("OPENPASS_PASSPHRASE")
	}()

	cfg := config.Default()
	_, _ = vaultpkg.InitWithPassphrase(tmpDir, "test", cfg)

	addValue = ""
	addGenerate = false

	oldStdin := os.Stdin
	r, w, _ := os.Pipe()
	os.Stdin = r
	_ = w.Close()

	rootCmd.SetArgs([]string{"--vault", tmpDir, "add", "test"})
	defer rootCmd.SetArgs(nil)

	err := rootCmd.Execute()
	os.Stdin = oldStdin
	_ = r.Close()

	if err == nil || !strings.Contains(err.Error(), "read username") {
		t.Errorf("expected 'read username' error, got: %v", err)
	}
}

func TestRecipients_ListEmpty(t *testing.T) {
	resetVaultState(t)

	tmpDir := t.TempDir()
	_ = os.Setenv("OPENPASS_VAULT", tmpDir)
	_ = os.Setenv("OPENPASS_PASSPHRASE", "test")
	defer func() {
		_ = os.Unsetenv("OPENPASS_VAULT")
		_ = os.Unsetenv("OPENPASS_PASSPHRASE")
	}()

	cfg := config.Default()
	_, _ = vaultpkg.InitWithPassphrase(tmpDir, "test", cfg)

	rootCmd.SetArgs([]string{"--vault", tmpDir, "recipients", "list"})
	defer rootCmd.SetArgs(nil)

	output := captureStdout(func() {
		_ = rootCmd.Execute()
	})

	if !strings.Contains(output, "No recipients configured") {
		t.Errorf("expected 'No recipients configured', got: %s", output)
	}
}

func TestRecipients_ListInvalidRecipient(t *testing.T) {
	resetVaultState(t)

	tmpDir := t.TempDir()
	_ = os.Setenv("OPENPASS_VAULT", tmpDir)
	_ = os.Setenv("OPENPASS_PASSPHRASE", "test")
	defer func() {
		_ = os.Unsetenv("OPENPASS_VAULT")
		_ = os.Unsetenv("OPENPASS_PASSPHRASE")
	}()

	cfg := config.Default()
	_, _ = vaultpkg.InitWithPassphrase(tmpDir, "test", cfg)

	// Write an invalid recipient directly to recipients.txt
	_ = os.WriteFile(tmpDir+"/recipients.txt", []byte("invalid-key\n"), 0o600)

	rootCmd.SetArgs([]string{"--vault", tmpDir, "recipients", "list"})
	defer rootCmd.SetArgs(nil)

	output := captureStdout(func() {
		_ = rootCmd.Execute()
	})

	if !strings.Contains(output, "invalid key format") {
		t.Errorf("expected output to contain invalid recipient error, got: %s", output)
	}
}

func TestRecipients_AddAlreadyExists(t *testing.T) {
	resetVaultState(t)

	tmpDir := t.TempDir()
	_ = os.Setenv("OPENPASS_VAULT", tmpDir)
	_ = os.Setenv("OPENPASS_PASSPHRASE", "test")
	defer func() {
		_ = os.Unsetenv("OPENPASS_VAULT")
		_ = os.Unsetenv("OPENPASS_PASSPHRASE")
	}()

	cfg := config.Default()
	identity, _ := vaultpkg.InitWithPassphrase(tmpDir, "test", cfg)
	recipient := identity.Recipient().String()

	// Add once
	_ = vaultpkg.NewRecipientsManager(tmpDir).AddRecipient(recipient)

	// Try to add again via CLI
	rootCmd.SetArgs([]string{"--vault", tmpDir, "recipients", "add", recipient})
	defer rootCmd.SetArgs(nil)

	err := rootCmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Errorf("expected 'already exists' error, got: %v", err)
	}
}

func TestRecipients_RemoveCancelled(t *testing.T) {
	resetVaultState(t)

	tmpDir := t.TempDir()
	_ = os.Setenv("OPENPASS_VAULT", tmpDir)
	_ = os.Setenv("OPENPASS_PASSPHRASE", "test")
	defer func() {
		_ = os.Unsetenv("OPENPASS_VAULT")
		_ = os.Unsetenv("OPENPASS_PASSPHRASE")
	}()

	cfg := config.Default()
	identity, _ := vaultpkg.InitWithPassphrase(tmpDir, "test", cfg)
	recipient := identity.Recipient().String()
	_ = vaultpkg.NewRecipientsManager(tmpDir).AddRecipient(recipient)

	oldStdin := os.Stdin
	r, w, _ := os.Pipe()
	os.Stdin = r
	_, _ = w.WriteString("n\n")
	_ = w.Close()

	rootCmd.SetArgs([]string{"--vault", tmpDir, "recipients", "remove", recipient})
	defer rootCmd.SetArgs(nil)

	_ = rootCmd.Execute()
	os.Stdin = oldStdin
	_ = r.Close()
}

func TestServe_HTTPSignalShutdown(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping slow integration server test in short mode")
	}
	resetVaultState(t)

	tmpDir := t.TempDir()
	_ = os.Setenv("OPENPASS_VAULT", tmpDir)
	_ = os.Setenv("OPENPASS_PASSPHRASE", "test")
	defer func() {
		_ = os.Unsetenv("OPENPASS_VAULT")
		_ = os.Unsetenv("OPENPASS_PASSPHRASE")
	}()

	cfg := config.Default()
	_, _ = vaultpkg.InitWithPassphrase(tmpDir, "test", cfg)

	_ = serveCmd.Flags().Set("bind", "127.0.0.1")
	_ = serveCmd.Flags().Set("stdio", "false")

	port := findFreePort(t)

	origNotify := serveSignalNotify
	t.Cleanup(func() { serveSignalNotify = origNotify })
	serveSignalNotify = func(c chan<- os.Signal, sigs ...os.Signal) {
		go func() {
			time.Sleep(50 * time.Millisecond)
			c <- syscall.SIGTERM
		}()
	}

	rootCmd.SetArgs([]string{"--vault", tmpDir, "serve", "--port", fmt.Sprintf("%d", port)})
	defer rootCmd.SetArgs(nil)

	done := make(chan struct{})
	go func() {
		_ = rootCmd.Execute()
		close(done)
	}()

	// Wait for server to start
	time.Sleep(200 * time.Millisecond)

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("serve did not exit after signal")
	}
}

func TestServe_StdioError(t *testing.T) {
	resetVaultState(t)

	tmpDir := t.TempDir()
	_ = os.Setenv("OPENPASS_VAULT", tmpDir)
	_ = os.Setenv("OPENPASS_PASSPHRASE", "test")
	defer func() {
		_ = os.Unsetenv("OPENPASS_VAULT")
		_ = os.Unsetenv("OPENPASS_PASSPHRASE")
	}()

	cfg := config.Default()
	_, _ = vaultpkg.InitWithPassphrase(tmpDir, "test", cfg)

	_ = serveCmd.Flags().Set("bind", "127.0.0.1")
	_ = serveCmd.Flags().Set("stdio", "false")
	_ = serveCmd.Flags().Set("agent", "")

	port := findFreePort(t)

	origNew := mcpFactory.New
	mcpFactory.New = func(_ *vaultpkg.Vault, _ string, _ string) (*mcp.Server, error) {
		return nil, fmt.Errorf("mock stdio error")
	}
	defer func() { mcpFactory.New = origNew }()

	rootCmd.SetArgs([]string{"--vault", tmpDir, "serve", "--stdio", "--agent", "test-agent", "--port", fmt.Sprintf("%d", port)})
	defer rootCmd.SetArgs(nil)

	err := rootCmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "mock stdio error") {
		t.Errorf("expected mock stdio error, got: %v", err)
	}
}
