package cmd

import (
	"strings"
	"testing"
)

func TestPrintJSON_MarshalError(t *testing.T) {
	stderr := captureStderr(func() {
		PrintJSON(make(chan int))
	})
	if !strings.Contains(stderr, "JSON encoding error") {
		t.Errorf("expected JSON encoding error in stderr, got: %s", stderr)
	}
}
