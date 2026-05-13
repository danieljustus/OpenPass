package cliout

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w
	defer func() { os.Stderr = old }()

	fn()

	w.Close()
	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

func TestErrorf(t *testing.T) {
	SetQuiet(false)
	out := captureStderr(t, func() {
		Errorf("test error %d", 42)
	})
	if !strings.Contains(out, "test error 42") {
		t.Errorf("Errorf output = %q, want to contain 'test error 42'", out)
	}
}

func TestErrorfQuiet(t *testing.T) {
	SetQuiet(true)
	out := captureStderr(t, func() {
		Errorf("should not appear")
	})
	if out != "" {
		t.Errorf("Errorf in quiet mode output = %q, want empty", out)
	}
	SetQuiet(false)
}

func TestWarnf(t *testing.T) {
	SetQuiet(false)
	out := captureStderr(t, func() {
		Warnf("test warning")
	})
	if !strings.Contains(out, "test warning") {
		t.Errorf("Warnf output = %q, want to contain 'test warning'", out)
	}
}

func TestHintf(t *testing.T) {
	SetQuiet(false)
	out := captureStderr(t, func() {
		Hintf("test hint")
	})
	if !strings.Contains(out, "test hint") {
		t.Errorf("Hintf output = %q, want to contain 'test hint'", out)
	}
}

func TestNoColor(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	SetQuiet(false)
	out := captureStderr(t, func() {
		Errorf("no color")
	})
	if strings.Contains(out, "\033[") {
		t.Errorf("Errorf with NO_COLOR should not contain ANSI codes, got %q", out)
	}
	os.Unsetenv("NO_COLOR")
}

func TestColorWithCodes(t *testing.T) {
	os.Unsetenv("NO_COLOR")
	SetQuiet(false)
	out := captureStderr(t, func() {
		Errorf("with color")
	})
	// When stderr is a pipe (as in captureStderr), ANSI codes are suppressed.
	if strings.Contains(out, "\033[") {
		t.Errorf("Errorf with piped stderr should not contain ANSI codes, got %q", out)
	}
}

func TestColorizeFunctions(t *testing.T) {
	os.Unsetenv("NO_COLOR")
	// When stderr is piped (as in tests), color is suppressed.
	// Verify the functions return text correctly in either case.
	cases := []struct {
		name string
		fn   func(string) string
	}{
		{"ColorizeError", ColorizeError},
		{"ColorizeWarn", ColorizeWarn},
		{"ColorizeSuccess", ColorizeSuccess},
		{"ColorizeDim", ColorizeDim},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.fn("hello")
			if !strings.Contains(got, "hello") {
				t.Errorf("%s = %q, want to contain 'hello'", tc.name, got)
			}
		})
	}
}

func TestColorizeFunctionsNoColor(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	got := ColorizeError("secret")
	if strings.Contains(got, "\033[") {
		t.Errorf("ColorizeError with NO_COLOR should not contain ANSI, got %q", got)
	}
	got = ColorizeWarn("secret")
	if strings.Contains(got, "\033[") {
		t.Errorf("ColorizeWarn with NO_COLOR should not contain ANSI, got %q", got)
	}
	got = ColorizeSuccess("secret")
	if strings.Contains(got, "\033[") {
		t.Errorf("ColorizeSuccess with NO_COLOR should not contain ANSI, got %q", got)
	}
}
