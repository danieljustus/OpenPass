package importer

import (
	"archive/zip"
	"bytes"
	"io"
	"strings"
	"testing"
)

func TestOnePUXNormalImport(t *testing.T) {
	exportJSON := `{"accounts":[{"vaults":[{"items":[{"categoryUuid":"001","title":"example.com","trashed":false,"details":{"loginFields":[{"fieldType":"T","designation":"username","value":"user"},{"fieldType":"P","designation":"password","value":"pass"}],"sections":[],"notesPlain":""},"overview":{"urls":[{"url":"https://example.com"}],"tags":[]}}]}]}]}`

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create("export.json")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, err := w.Write([]byte(exportJSON)); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	imp := &onePUXImporter{}
	entries, err := imp.Parse(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Path != "example.com" {
		t.Errorf("Path = %q, want example.com", entries[0].Path)
	}
}

func TestOnePUXOversizedEntryRejected(t *testing.T) {
	oldLimit := maxZipEntrySize
	maxZipEntrySize = 1024
	defer func() { maxZipEntrySize = oldLimit }()

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create("export.json")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, err := io.WriteString(w, strings.Repeat("x", 1025)); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	imp := &onePUXImporter{}
	_, err = imp.Parse(bytes.NewReader(buf.Bytes()))
	if err == nil {
		t.Fatal("expected error for oversized zip entry, got nil")
	}
	if !strings.Contains(err.Error(), "exceeds maximum size") {
		t.Fatalf("expected 'exceeds maximum size' error, got: %v", err)
	}
}

func TestOnePUXLimitNotHitForSmallFile(t *testing.T) {
	oldLimit := maxZipEntrySize
	maxZipEntrySize = 1024
	defer func() { maxZipEntrySize = oldLimit }()

	exportJSON := strings.Repeat("x", 512)

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create("export.json")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, err := io.WriteString(w, exportJSON); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	imp := &onePUXImporter{}
	_, err = imp.Parse(bytes.NewReader(buf.Bytes()))
	if err != nil && strings.Contains(err.Error(), "exceeds maximum size") {
		t.Fatalf("unexpected size limit error for small entry: %v", err)
	}
}
