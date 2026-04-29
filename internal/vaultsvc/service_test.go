package vaultsvc

import (
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/danieljustus/OpenPass/internal/config"
	gitpkg "github.com/danieljustus/OpenPass/internal/git"
	vaultpkg "github.com/danieljustus/OpenPass/internal/vault"
)

const testPassphrase = "test-passphrase"

func newTestService(t *testing.T, withGit bool) *Service {
	t.Helper()

	vaultDir := t.TempDir()
	cfg := config.Default()
	cfg.Git = &config.GitConfig{AutoPush: false, CommitTemplate: "Update from OpenPass"}

	if _, err := vaultpkg.InitWithPassphrase(vaultDir, testPassphrase, cfg); err != nil {
		t.Fatalf("init vault: %v", err)
	}
	if withGit {
		if err := gitpkg.Init(vaultDir); err != nil {
			t.Fatalf("init git: %v", err)
		}
	}

	v, err := vaultpkg.OpenWithPassphrase(vaultDir, testPassphrase)
	if err != nil {
		t.Fatalf("open vault: %v", err)
	}
	return New(v)
}

func writeTestEntry(t *testing.T, svc *Service, path string, data map[string]any) {
	t.Helper()
	if err := svc.WriteEntry(path, &vaultpkg.Entry{Data: data}); err != nil {
		t.Fatalf("write entry %q: %v", path, err)
	}
}

func latestCommitMessage(t *testing.T, svc *Service) string {
	t.Helper()
	commits, err := gitpkg.Log(svc.GetDir(), "", 1)
	if err != nil {
		t.Fatalf("read git log: %v", err)
	}
	if len(commits) == 0 {
		t.Fatal("expected at least one git commit")
	}
	return strings.TrimSpace(commits[0].Message)
}

func TestNewAndVault(t *testing.T) {
	svc := newTestService(t, false)
	if svc == nil {
		t.Fatal("New returned nil")
	}
	if svc.Vault() == nil {
		t.Fatal("Vault returned nil")
	}
	if svc.Vault().Dir != svc.GetDir() {
		t.Fatalf("Vault().Dir = %q, GetDir() = %q", svc.Vault().Dir, svc.GetDir())
	}
}

func TestGetField(t *testing.T) {
	svc := newTestService(t, false)
	data := map[string]any{
		"username": "alice",
		"password": "secret",
		"profile":  map[string]any{"email": "alice@example.com"},
	}
	writeTestEntry(t, svc, "work/aws", data)

	tests := []struct {
		name     string
		path     string
		field    string
		want     any
		wantKind ErrorKind
	}{
		{name: "existing field", path: "work/aws", field: "password", want: "secret"},
		{name: "non-existent entry", path: "missing", field: "password", wantKind: ErrNotFound},
		{name: "non-existent field", path: "work/aws", field: "missing", wantKind: ErrFieldNotFound},
		{name: "empty field returns data", path: "work/aws", field: "", want: data},
		{name: "nested entry path", path: "work/aws", field: "username", want: "alice"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := svc.GetField(tt.path, tt.field)
			if tt.wantKind != 0 || tt.name == "non-existent entry" {
				assertServiceErrorKind(t, err, tt.wantKind)
				return
			}
			if err != nil {
				t.Fatalf("GetField returned error: %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("GetField() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestSetFieldAndSetFields(t *testing.T) {
	t.Run("create new entry", func(t *testing.T) {
		svc := newTestService(t, true)
		if err := svc.SetField("github", "password", "secret"); err != nil {
			t.Fatalf("SetField: %v", err)
		}
		got, err := svc.GetField("github", "password")
		if err != nil {
			t.Fatalf("GetField: %v", err)
		}
		if got != "secret" {
			t.Fatalf("password = %#v, want %q", got, "secret")
		}
		if msg := latestCommitMessage(t, svc); !strings.Contains(msg, "Update github") {
			t.Fatalf("latest commit message = %q, want Update github", msg)
		}
	})

	t.Run("update existing entry merges data", func(t *testing.T) {
		svc := newTestService(t, false)
		writeTestEntry(t, svc, "github", map[string]any{"username": "alice", "password": "old"})
		if err := svc.SetField("github", "password", "new"); err != nil {
			t.Fatalf("SetField: %v", err)
		}
		got, err := svc.GetField("github", "")
		if err != nil {
			t.Fatalf("GetField: %v", err)
		}
		want := map[string]any{"username": "alice", "password": "new"}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("merged data = %#v, want %#v", got, want)
		}
	})

	t.Run("set multiple fields", func(t *testing.T) {
		svc := newTestService(t, false)
		fields := map[string]any{"username": "alice", "password": "secret", "url": "https://example.com"}
		if err := svc.SetFields("example", fields); err != nil {
			t.Fatalf("SetFields: %v", err)
		}
		got, err := svc.GetField("example", "")
		if err != nil {
			t.Fatalf("GetField: %v", err)
		}
		if !reflect.DeepEqual(got, fields) {
			t.Fatalf("fields = %#v, want %#v", got, fields)
		}
	})
}

func TestDelete(t *testing.T) {
	t.Run("delete existing entry commits", func(t *testing.T) {
		svc := newTestService(t, true)
		writeTestEntry(t, svc, "github", map[string]any{"password": "secret"})
		if err := svc.Delete("github"); err != nil {
			t.Fatalf("Delete: %v", err)
		}
		_, err := svc.GetEntry("github")
		assertServiceErrorKind(t, err, ErrNotFound)
		if msg := latestCommitMessage(t, svc); !strings.Contains(msg, "Delete github") {
			t.Fatalf("latest commit message = %q, want Delete github", msg)
		}
	})

	t.Run("delete missing entry", func(t *testing.T) {
		svc := newTestService(t, false)
		err := svc.Delete("missing")
		assertServiceErrorKind(t, err, ErrNotFound)
	})
}

func TestList(t *testing.T) {
	t.Run("all entries and prefix", func(t *testing.T) {
		svc := newTestService(t, false)
		writeTestEntry(t, svc, "github", map[string]any{"password": "secret"})
		writeTestEntry(t, svc, "work/aws", map[string]any{"password": "aws"})
		writeTestEntry(t, svc, "work/gcp", map[string]any{"password": "gcp"})

		all, err := svc.List("")
		if err != nil {
			t.Fatalf("List all: %v", err)
		}
		wantAll := []string{"github", "work/aws", "work/gcp"}
		if !reflect.DeepEqual(all, wantAll) {
			t.Fatalf("List all = %#v, want %#v", all, wantAll)
		}

		work, err := svc.List("work/")
		if err != nil {
			t.Fatalf("List prefix: %v", err)
		}
		wantWork := []string{"work/aws", "work/gcp"}
		if !reflect.DeepEqual(work, wantWork) {
			t.Fatalf("List prefix = %#v, want %#v", work, wantWork)
		}
	})

	t.Run("empty vault", func(t *testing.T) {
		svc := newTestService(t, false)
		got, err := svc.List("")
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if len(got) != 0 {
			t.Fatalf("List empty = %#v, want empty slice", got)
		}
	})
}

func TestFind(t *testing.T) {
	svc := newTestService(t, false)
	writeTestEntry(t, svc, "github", map[string]any{"username": "alice", "password": "secret"})
	writeTestEntry(t, svc, "work/aws", map[string]any{"username": "bob", "password": "cloud-secret"})

	tests := []struct {
		name      string
		query     string
		opts      FindOptions
		wantPaths []string
	}{
		{name: "matching query", query: "cloud", wantPaths: []string{"work/aws"}},
		{name: "no results", query: "does-not-exist", wantPaths: []string{}},
		{name: "max workers option", query: "github", opts: FindOptions{MaxWorkers: 2}, wantPaths: []string{"github"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches, err := svc.Find(tt.query, tt.opts)
			if err != nil {
				t.Fatalf("Find: %v", err)
			}
			gotPaths := make([]string, 0, len(matches))
			for _, match := range matches {
				gotPaths = append(gotPaths, match.Path)
			}
			slices.Sort(gotPaths)
			if !reflect.DeepEqual(gotPaths, tt.wantPaths) {
				t.Fatalf("Find paths = %#v, want %#v", gotPaths, tt.wantPaths)
			}
		})
	}
}

func TestGetEntry(t *testing.T) {
	svc := newTestService(t, false)
	writeTestEntry(t, svc, "github", map[string]any{"username": "alice"})

	entry, err := svc.GetEntry("github")
	if err != nil {
		t.Fatalf("GetEntry: %v", err)
	}
	if entry.Data["username"] != "alice" {
		t.Fatalf("username = %#v, want %q", entry.Data["username"], "alice")
	}
	if entry.Metadata.Version != 1 {
		t.Fatalf("version = %d, want 1", entry.Metadata.Version)
	}

	_, err = svc.GetEntry("missing")
	assertServiceErrorKind(t, err, ErrNotFound)
}

func TestWriteEntry(t *testing.T) {
	svc := newTestService(t, false)
	created := time.Date(2026, 4, 29, 12, 0, 0, 0, time.UTC)

	if err := svc.WriteEntry("github", &vaultpkg.Entry{
		Data:     map[string]any{"username": "alice", "password": "old"},
		Metadata: vaultpkg.EntryMetadata{Created: created},
	}); err != nil {
		t.Fatalf("WriteEntry new: %v", err)
	}
	entry, err := svc.GetEntry("github")
	if err != nil {
		t.Fatalf("GetEntry: %v", err)
	}
	if entry.Data["password"] != "old" {
		t.Fatalf("password = %#v, want old", entry.Data["password"])
	}

	if err := svc.WriteEntry("github", &vaultpkg.Entry{Data: map[string]any{"password": "new"}}); err != nil {
		t.Fatalf("WriteEntry overwrite: %v", err)
	}
	entry, err = svc.GetEntry("github")
	if err != nil {
		t.Fatalf("GetEntry after overwrite: %v", err)
	}
	if _, ok := entry.Data["username"]; ok {
		t.Fatalf("overwrite preserved username unexpectedly: %#v", entry.Data)
	}
	if entry.Data["password"] != "new" {
		t.Fatalf("password = %#v, want new", entry.Data["password"])
	}
}

func TestErrorTypes(t *testing.T) {
	cause := errors.New("disk full")

	tests := []struct {
		name string
		err  *Error
		want string
	}{
		{name: "without cause", err: NewError(ErrNotFound, "missing entry", nil), want: "missing entry"},
		{name: "with cause", err: NewError(ErrWriteFailed, "write failed", cause), want: "write failed: disk full"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.want {
				t.Fatalf("Error() = %q, want %q", got, tt.want)
			}
		})
	}

	wrappedNotFound := fmt.Errorf("outer: %w", NewError(ErrNotFound, "missing", nil))
	wrappedFieldNotFound := fmt.Errorf("outer: %w", NewError(ErrFieldNotFound, "field missing", nil))
	wrappedWrite := fmt.Errorf("outer: %w", NewError(ErrWriteFailed, "write failed", cause))

	for _, err := range []error{wrappedNotFound, wrappedFieldNotFound} {
		if !IsNotFound(err) {
			t.Fatalf("IsNotFound(%v) = false, want true", err)
		}
	}
	for _, err := range []error{nil, cause, wrappedWrite, NewError(ErrReadFailed, "read", nil)} {
		if IsNotFound(err) {
			t.Fatalf("IsNotFound(%v) = true, want false", err)
		}
	}

	if !IsWriteError(wrappedWrite) {
		t.Fatal("IsWriteError(wrapped write) = false, want true")
	}
	for _, err := range []error{nil, cause, wrappedNotFound, NewError(ErrReadFailed, "read", nil)} {
		if IsWriteError(err) {
			t.Fatalf("IsWriteError(%v) = true, want false", err)
		}
	}

	var svcErr *Error
	if !errors.As(wrappedWrite, &svcErr) {
		t.Fatal("errors.As did not extract *Error")
	}
	if svcErr.Kind != ErrWriteFailed || !errors.Is(wrappedWrite, cause) {
		t.Fatalf("errors.As/Unwrap mismatch: svcErr=%#v", svcErr)
	}
}

func TestGetIdentityAndGetDir(t *testing.T) {
	svc := newTestService(t, false)
	if svc.GetIdentity() == nil {
		t.Fatal("GetIdentity returned nil")
	}
	if svc.GetIdentity() != svc.Vault().Identity {
		t.Fatal("GetIdentity did not return vault identity")
	}
	if svc.GetDir() == "" {
		t.Fatal("GetDir returned empty string")
	}
	if svc.GetDir() != svc.Vault().Dir {
		t.Fatalf("GetDir = %q, want %q", svc.GetDir(), svc.Vault().Dir)
	}
}

func TestServiceErrorPaths(t *testing.T) {
	svc := newTestService(t, false)

	_, err := svc.GetField("../bad", "password")
	assertServiceErrorKind(t, err, ErrReadFailed)

	err = svc.SetField("../bad", "password", "secret")
	assertServiceErrorKind(t, err, ErrReadFailed)

	err = svc.Delete("../bad")
	assertServiceErrorKind(t, err, ErrWriteFailed)

	_, err = svc.GetEntry("../bad")
	assertServiceErrorKind(t, err, ErrReadFailed)

	err = svc.WriteEntry("github", nil)
	assertServiceErrorKind(t, err, ErrWriteFailed)

	err = svc.WriteEntry("../bad", &vaultpkg.Entry{Data: map[string]any{"password": "secret"}})
	assertServiceErrorKind(t, err, ErrWriteFailed)

	missingVault := New(&vaultpkg.Vault{Dir: filepath.Join(t.TempDir(), "missing"), Identity: svc.GetIdentity()})
	_, err = missingVault.List("")
	assertServiceErrorKind(t, err, ErrReadFailed)

	_, err = missingVault.Find("anything", FindOptions{})
	assertServiceErrorKind(t, err, ErrReadFailed)
}

func assertServiceErrorKind(t *testing.T, err error, kind ErrorKind) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected service error kind %v, got nil", kind)
	}
	var svcErr *Error
	if !errors.As(err, &svcErr) {
		t.Fatalf("error %T is not *Error: %v", err, err)
	}
	if svcErr.Kind != kind {
		t.Fatalf("error kind = %v, want %v", svcErr.Kind, kind)
	}
}
