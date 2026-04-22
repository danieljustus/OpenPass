package mcp

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

func TestIsTTYPresent(t *testing.T) {
	// This test will return false in most CI/testing environments
	// since /dev/tty is not available
	result := IsTTYPresent()
	// Just verify it returns a boolean without panicking
	_ = result
}

func TestBuildPrompt(t *testing.T) {
	tests := []struct {
		name       string
		wantPrefix string
		req        ApprovalRequest
		wantOp     bool
	}{
		{
			name: "with operation and details",
			req: ApprovalRequest{
				Operation: "delete",
				Details:   "github/work",
				Timeout:   30 * time.Second,
			},
			wantPrefix: "\n",
			wantOp:     true,
		},
		{
			name: "with empty operation",
			req: ApprovalRequest{
				Operation: "",
				Details:   "some details",
				Timeout:   30 * time.Second,
			},
			wantPrefix: "\n",
			wantOp:     false,
		},
		{
			name: "with empty details",
			req: ApprovalRequest{
				Operation: "write",
				Details:   "",
				Timeout:   30 * time.Second,
			},
			wantPrefix: "\n",
			wantOp:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildPrompt(tt.req)
			if result == "" {
				t.Fatal("buildPrompt() returned empty string")
			}
			if result[:len(tt.wantPrefix)] != tt.wantPrefix {
				t.Errorf("buildPrompt() prefix = %q, want %q", result[:len(tt.wantPrefix)], tt.wantPrefix)
			}
			if tt.wantOp && tt.req.Operation != "" {
				if !containsString(result, tt.req.Operation) {
					t.Errorf("buildPrompt() = %q, want to contain operation %q", result, tt.req.Operation)
				}
			}
		})
	}
}

func TestBuildPrompt_Truncation(t *testing.T) {
	req := ApprovalRequest{
		Operation: "this is a very long operation name that should be truncated",
		Details:   "also some very long details that should be truncated to fit the box",
		Timeout:   30 * time.Second,
	}

	result := buildPrompt(req)
	if len(result) == 0 {
		t.Fatal("buildPrompt() returned empty string")
	}
}

func TestParseApprovalResponse(t *testing.T) {
	tests := []struct {
		name     string
		response string
		expected bool
	}{
		{"lowercase y", "y", true},
		{"uppercase Y", "Y", true},
		{"lowercase yes", "yes", true},
		{"uppercase YES", "YES", true},
		{"mixed case Yes", "Yes", true},
		{"no", "no", false},
		{"n", "n", false},
		{"NO", "NO", false},
		{"anything else", "maybe", false},
		{"empty string", "", false},
		{"with whitespace", "  y  ", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseApprovalResponse(tt.response)
			if result != tt.expected {
				t.Errorf("parseApprovalResponse(%q) = %v, want %v", tt.response, result, tt.expected)
			}
		})
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		want   string
		maxLen int
	}{
		{"short string", "hello", "hello", 10},
		{"exact length", "hello", "hello", 5},
		{"long string", "hello world", "he...", 5},
		{"maxLen 0", "hello", "", 0},
		{"maxLen 1", "hello", "h", 1},
		{"maxLen 2", "hello", "he", 2},
		{"maxLen 3", "hello", "hel", 3},
		{"maxLen 4", "hello", "h...", 4},
		{"empty string", "", "", 5},
		{"maxLen less than 3", "hello", "he", 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncate(tt.input, tt.maxLen)
			if result != tt.want {
				t.Errorf("truncate(%q, %d) = %q, want %q", tt.input, tt.maxLen, result, tt.want)
			}
		})
	}
}

func TestRequestApproval_NoTTY(t *testing.T) {
	original := openTTYDevice
	defer func() { openTTYDevice = original }()

	openTTYDevice = func() (ttyDevice, error) {
		return nil, errors.New("no tty available")
	}

	req := ApprovalRequest{
		Operation: "test",
		Details:   "test details",
		Timeout:   1 * time.Second,
	}

	result := RequestApproval(req)

	if result.Approved {
		t.Error("RequestApproval() expected not approved without TTY")
	}
	if result.Error == nil {
		t.Error("RequestApproval() expected error without TTY")
	}
	if !strings.Contains(result.Error.Error(), "no TTY available") {
		t.Errorf("RequestApproval() error = %v, want no TTY available error", result.Error)
	}
}

func TestRequestApproval_DefaultTimeout(t *testing.T) {
	original := openTTYDevice
	defer func() { openTTYDevice = original }()

	openTTYDevice = func() (ttyDevice, error) {
		return &mockTTYDevice{
			readString: func() (string, error) { return "", os.ErrDeadlineExceeded },
			output:     newMockOutputFile(t),
			raw:        func() (func(), error) { return func() {}, nil },
		}, nil
	}

	req := ApprovalRequest{
		Operation: "test",
		Timeout:   0,
	}

	result := RequestApproval(req)
	if result.Approved {
		t.Error("RequestApproval() expected not approved on timeout")
	}
	if result.Error == nil {
		t.Error("RequestApproval() expected timeout error")
	}
	if !strings.Contains(result.Error.Error(), "timed out after 30s") {
		t.Errorf("RequestApproval() error = %v, want default timeout error", result.Error)
	}
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

type mockTTYDevice struct {
	readString func() (string, error)
	input      *os.File
	output     *os.File
	raw        func() (func(), error)
	closeFunc  func() error
}

func (m *mockTTYDevice) ReadString() (string, error) {
	if m.readString != nil {
		return m.readString()
	}
	return "", nil
}

func (m *mockTTYDevice) Input() *os.File {
	return m.input
}

func (m *mockTTYDevice) Output() *os.File {
	return m.output
}

func (m *mockTTYDevice) Raw() (func(), error) {
	if m.raw != nil {
		return m.raw()
	}
	return func() {}, nil
}

func (m *mockTTYDevice) Close() error {
	if m.closeFunc != nil {
		return m.closeFunc()
	}
	return nil
}

func newMockOutputFile(t *testing.T) *os.File {
	t.Helper()
	readEnd, writeEnd, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe failed: %v", err)
	}
	t.Cleanup(func() {
		_ = writeEnd.Close()
		_ = readEnd.Close()
	})
	return writeEnd
}

func TestIsTTYPresent_Error(t *testing.T) {
	original := openTTYDevice
	defer func() { openTTYDevice = original }()

	openTTYDevice = func() (ttyDevice, error) {
		return nil, errors.New("no tty available")
	}

	if IsTTYPresent() {
		t.Error("IsTTYPresent() = true, want false when openTTYDevice returns error")
	}
}

func TestRequestApproval_Approved(t *testing.T) {
	original := openTTYDevice
	defer func() { openTTYDevice = original }()

	tests := []struct {
		name     string
		response string
	}{
		{"yes lowercase", "yes"},
		{"y lowercase", "y"},
		{"YES uppercase", "YES"},
		{"Y uppercase", "Y"},
		{"Yes mixed", "Yes"},
		{"y with whitespace", "  y  "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			openTTYDevice = func() (ttyDevice, error) {
				return &mockTTYDevice{
					readString: func() (string, error) { return tt.response, nil },
					output:     newMockOutputFile(t),
					raw:        func() (func(), error) { return func() {}, nil },
				}, nil
			}

			req := ApprovalRequest{
				Operation: "test",
				Details:   "test details",
				Timeout:   1 * time.Second,
			}
			result := RequestApproval(req)
			if !result.Approved {
				t.Errorf("RequestApproval() approved = %v, want true", result.Approved)
			}
			if result.Error != nil {
				t.Errorf("RequestApproval() error = %v, want nil", result.Error)
			}
		})
	}
}

func TestRequestApproval_Denied(t *testing.T) {
	original := openTTYDevice
	defer func() { openTTYDevice = original }()

	tests := []struct {
		name     string
		response string
	}{
		{"no lowercase", "no"},
		{"n lowercase", "n"},
		{"NO uppercase", "NO"},
		{"N uppercase", "N"},
		{"maybe", "maybe"},
		{"empty", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			openTTYDevice = func() (ttyDevice, error) {
				return &mockTTYDevice{
					readString: func() (string, error) { return tt.response, nil },
					output:     newMockOutputFile(t),
					raw:        func() (func(), error) { return func() {}, nil },
				}, nil
			}

			req := ApprovalRequest{
				Operation: "test",
				Details:   "test details",
				Timeout:   1 * time.Second,
			}
			result := RequestApproval(req)
			if result.Approved {
				t.Errorf("RequestApproval() approved = %v, want false", result.Approved)
			}
			if result.Error != nil {
				t.Errorf("RequestApproval() error = %v, want nil", result.Error)
			}
		})
	}
}

func TestRequestApproval_Timeout(t *testing.T) {
	original := openTTYDevice
	defer func() { openTTYDevice = original }()

	openTTYDevice = func() (ttyDevice, error) {
		return &mockTTYDevice{
			readString: func() (string, error) { return "", os.ErrDeadlineExceeded },
			output:     newMockOutputFile(t),
			raw:        func() (func(), error) { return func() {}, nil },
		}, nil
	}

	req := ApprovalRequest{
		Operation: "test",
		Details:   "test details",
		Timeout:   1 * time.Millisecond,
	}
	result := RequestApproval(req)
	if result.Approved {
		t.Error("RequestApproval() approved = true, want false on timeout")
	}
	if result.Error == nil {
		t.Error("RequestApproval() error = nil, want timeout error")
	}
	if !strings.Contains(result.Error.Error(), "timed out") {
		t.Errorf("RequestApproval() error = %v, want timeout error", result.Error)
	}
}

func TestRequestApproval_ReadError(t *testing.T) {
	original := openTTYDevice
	defer func() { openTTYDevice = original }()

	openTTYDevice = func() (ttyDevice, error) {
		return &mockTTYDevice{
			readString: func() (string, error) { return "", errors.New("read failed") },
			output:     newMockOutputFile(t),
			raw:        func() (func(), error) { return func() {}, nil },
		}, nil
	}

	req := ApprovalRequest{
		Operation: "test",
		Details:   "test details",
		Timeout:   1 * time.Second,
	}
	result := RequestApproval(req)
	if result.Approved {
		t.Error("RequestApproval() approved = true, want false on read error")
	}
	if result.Error == nil {
		t.Error("RequestApproval() error = nil, want read error")
	}
	if !strings.Contains(result.Error.Error(), "failed to read from terminal") {
		t.Errorf("RequestApproval() error = %v, want read error", result.Error)
	}
}

func TestRequestApproval_RawError(t *testing.T) {
	original := openTTYDevice
	defer func() { openTTYDevice = original }()

	openTTYDevice = func() (ttyDevice, error) {
		return &mockTTYDevice{
			output: newMockOutputFile(t),
			raw:    func() (func(), error) { return nil, errors.New("raw failed") },
		}, nil
	}

	req := ApprovalRequest{
		Operation: "test",
		Details:   "test details",
		Timeout:   1 * time.Second,
	}
	result := RequestApproval(req)
	if result.Approved {
		t.Error("RequestApproval() approved = true, want false on raw error")
	}
	if result.Error == nil {
		t.Error("RequestApproval() error = nil, want raw error")
	}
	if !strings.Contains(result.Error.Error(), "failed to set terminal raw mode") {
		t.Errorf("RequestApproval() error = %v, want raw mode error", result.Error)
	}
}

func TestRequestApproval_WriteError(t *testing.T) {
	original := openTTYDevice
	defer func() { openTTYDevice = original }()

	tmpfile, err := os.CreateTemp("", "mocktty")
	if err != nil {
		t.Fatalf("CreateTemp failed: %v", err)
	}
	//nolint:errcheck // best-effort close in test
	tmpfile.Close()
	//nolint:errcheck // best-effort remove in test
	defer os.Remove(tmpfile.Name())

	readOnlyFile, err := os.Open(tmpfile.Name())
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	//nolint:errcheck // best-effort close in test
	defer readOnlyFile.Close()

	openTTYDevice = func() (ttyDevice, error) {
		return &mockTTYDevice{
			output: readOnlyFile,
			raw:    func() (func(), error) { return func() {}, nil },
		}, nil
	}

	req := ApprovalRequest{
		Operation: "test",
		Details:   "test details",
		Timeout:   1 * time.Second,
	}
	result := RequestApproval(req)
	if result.Approved {
		t.Error("RequestApproval() approved = true, want false on write error")
	}
	if result.Error == nil {
		t.Error("RequestApproval() error = nil, want write error")
	}
	if !strings.Contains(result.Error.Error(), "failed to write to terminal") {
		t.Errorf("RequestApproval() error = %v, want write error", result.Error)
	}
}

func TestRequestApproval_DefaultTimeoutWithTTY(t *testing.T) {
	original := openTTYDevice
	defer func() { openTTYDevice = original }()

	openTTYDevice = func() (ttyDevice, error) {
		return &mockTTYDevice{
			readString: func() (string, error) { return "y", nil },
			output:     newMockOutputFile(t),
			raw:        func() (func(), error) { return func() {}, nil },
		}, nil
	}

	req := ApprovalRequest{
		Operation: "test",
		Details:   "test details",
		Timeout:   0,
	}
	result := RequestApproval(req)
	if !result.Approved {
		t.Error("RequestApproval() approved = false, want true with default timeout")
	}
	if result.Error != nil {
		t.Errorf("RequestApproval() error = %v, want nil", result.Error)
	}
}

func TestIsTTYPresent_Success(t *testing.T) {
	original := openTTYDevice
	defer func() { openTTYDevice = original }()

	openTTYDevice = func() (ttyDevice, error) {
		return &mockTTYDevice{}, nil
	}

	if !IsTTYPresent() {
		t.Error("IsTTYPresent() = false, want true when TTY is available")
	}
}

func TestRequestApproval_EmptyRequest(t *testing.T) {
	original := openTTYDevice
	defer func() { openTTYDevice = original }()

	openTTYDevice = func() (ttyDevice, error) {
		return &mockTTYDevice{
			readString: func() (string, error) { return "y", nil },
			output:     newMockOutputFile(t),
			raw:        func() (func(), error) { return func() {}, nil },
		}, nil
	}

	req := ApprovalRequest{
		Operation: "",
		Details:   "",
		Timeout:   1 * time.Second,
	}
	result := RequestApproval(req)
	if !result.Approved {
		t.Errorf("RequestApproval() approved = %v, want true for empty request", result.Approved)
	}
	if result.Error != nil {
		t.Errorf("RequestApproval() error = %v, want nil", result.Error)
	}
}

func TestRequestApproval_VeryLongTimeout(t *testing.T) {
	original := openTTYDevice
	defer func() { openTTYDevice = original }()

	openTTYDevice = func() (ttyDevice, error) {
		return &mockTTYDevice{
			readString: func() (string, error) { return "y", nil },
			output:     newMockOutputFile(t),
			raw:        func() (func(), error) { return func() {}, nil },
		}, nil
	}

	req := ApprovalRequest{
		Operation: "test",
		Details:   "test details",
		Timeout:   24 * time.Hour,
	}
	result := RequestApproval(req)
	if !result.Approved {
		t.Error("RequestApproval() approved = false, want true with very long timeout")
	}
	if result.Error != nil {
		t.Errorf("RequestApproval() error = %v, want nil", result.Error)
	}
}

func TestRequestApproval_ConcurrentRequests(t *testing.T) {
	original := openTTYDevice
	defer func() { openTTYDevice = original }()

	openTTYDevice = func() (ttyDevice, error) {
		return &mockTTYDevice{
			readString: func() (string, error) { return "y", nil },
			output:     newMockOutputFile(t),
			raw:        func() (func(), error) { return func() {}, nil },
		}, nil
	}

	const numRequests = 10
	results := make(chan ApprovalResult, numRequests)

	for i := 0; i < numRequests; i++ {
		go func(idx int) {
			req := ApprovalRequest{
				Operation: fmt.Sprintf("operation-%d", idx),
				Details:   fmt.Sprintf("details-%d", idx),
				Timeout:   1 * time.Second,
			}
			results <- RequestApproval(req)
		}(i)
	}

	for i := 0; i < numRequests; i++ {
		result := <-results
		if !result.Approved {
			t.Errorf("RequestApproval() approved = %v, want true in concurrent request %d", result.Approved, i)
		}
		if result.Error != nil {
			t.Errorf("RequestApproval() error = %v, want nil in concurrent request %d", result.Error, i)
		}
	}
}

func TestRequestApproval_CloseError(t *testing.T) {
	original := openTTYDevice
	defer func() { openTTYDevice = original }()

	closeCalled := false
	openTTYDevice = func() (ttyDevice, error) {
		return &mockTTYDevice{
			readString: func() (string, error) { return "y", nil },
			output:     newMockOutputFile(t),
			raw:        func() (func(), error) { return func() {}, nil },
			closeFunc: func() error {
				closeCalled = true
				return errors.New("close failed")
			},
		}, nil
	}

	req := ApprovalRequest{
		Operation: "test",
		Details:   "test details",
		Timeout:   1 * time.Second,
	}
	result := RequestApproval(req)
	if !result.Approved {
		t.Error("RequestApproval() approved = false, want true")
	}
	if result.Error != nil {
		t.Errorf("RequestApproval() error = %v, want nil", result.Error)
	}
	if !closeCalled {
		t.Error("RequestApproval() close was not called")
	}
}

func TestRequestApproval_InputNil(t *testing.T) {
	original := openTTYDevice
	defer func() { openTTYDevice = original }()

	openTTYDevice = func() (ttyDevice, error) {
		return &mockTTYDevice{
			readString: func() (string, error) { return "y", nil },
			output:     newMockOutputFile(t),
			raw:        func() (func(), error) { return func() {}, nil },
			input:      nil,
		}, nil
	}

	req := ApprovalRequest{
		Operation: "test",
		Details:   "test details",
		Timeout:   1 * time.Second,
	}
	result := RequestApproval(req)
	if !result.Approved {
		t.Error("RequestApproval() approved = false, want true with nil input")
	}
	if result.Error != nil {
		t.Errorf("RequestApproval() error = %v, want nil", result.Error)
	}
}

func TestIsTimeoutError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"os.ErrDeadlineExceeded", os.ErrDeadlineExceeded, true},
		{"timeout error", &timeoutError{}, true},
		{"non-timeout error", errors.New("some error"), false},
		{"nil error", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isTimeoutError(tt.err)
			if result != tt.expected {
				t.Errorf("isTimeoutError(%v) = %v, want %v", tt.err, result, tt.expected)
			}
		})
	}
}

type timeoutError struct{}

func (e *timeoutError) Error() string   { return "timeout" }
func (e *timeoutError) Timeout() bool   { return true }
func (e *timeoutError) Temporary() bool { return true }
