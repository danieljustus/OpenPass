package cmd

import (
	"bytes"
	"testing"
)

func TestVersionOutputEndsWithNewline(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"version"})
	defer rootCmd.SetArgs(nil)

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	got := buf.String()
	if len(got) == 0 {
		t.Fatal("output is empty")
	}

	if got[len(got)-1] != '\n' {
		t.Errorf("output ends with %q, want newline", got[len(got)-1])
	}

	if bytes.Contains([]byte(got), []byte{'\\', 'n'}) {
		t.Error("output contains literal backslash-n instead of newline")
	}
}

func TestVersionOutputContainsExpectedFields(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"version"})
	defer rootCmd.SetArgs(nil)

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	got := buf.String()
	expectedPrefix := "openpass"
	if !bytes.HasPrefix([]byte(got), []byte(expectedPrefix)) {
		t.Errorf("output missing expected prefix %q, got %q", expectedPrefix, got)
	}

	if !bytes.Contains([]byte(got), []byte("(commit:")) {
		t.Errorf("output missing '(commit:' label, got %q", got)
	}

	if !bytes.Contains([]byte(got), []byte("built:")) {
		t.Errorf("output missing 'built:' label, got %q", got)
	}
}
