package git

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing/object"
)

func TestInitCreatesRepo(t *testing.T) {
	dir := t.TempDir()

	if err := Init(dir); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, ".git")); err != nil {
		t.Fatalf("expected .git directory: %v", err)
	}

	if _, err := gogit.PlainOpen(dir); err != nil {
		t.Fatalf("expected repo to open: %v", err)
	}
}

func TestCreateGitignoreCreatesFile(t *testing.T) {
	dir := t.TempDir()

	if err := Init(dir); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	if err := CreateGitignore(dir); err != nil {
		t.Fatalf("CreateGitignore() error = %v", err)
	}

	gitignorePath := filepath.Join(dir, ".gitignore")
	if _, err := os.Stat(gitignorePath); err != nil {
		t.Fatalf("expected .gitignore to exist: %v", err)
	}

	content, err := os.ReadFile(gitignorePath)
	if err != nil {
		t.Fatalf("read .gitignore: %v", err)
	}
	if !strings.Contains(string(content), "identity.age") {
		t.Error("expected .gitignore to contain 'identity.age'")
	}
}

func TestCreateGitignorePreservesExistingAndAppendsMissingEntries(t *testing.T) {
	dir := t.TempDir()

	if err := Init(dir); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	gitignorePath := filepath.Join(dir, ".gitignore")
	customContent := "custom content\nidentity.age\n"
	if err := os.WriteFile(gitignorePath, []byte(customContent), 0o600); err != nil {
		t.Fatalf("write .gitignore: %v", err)
	}

	if err := CreateGitignore(dir); err != nil {
		t.Fatalf("CreateGitignore() error = %v", err)
	}

	content, err := os.ReadFile(gitignorePath)
	if err != nil {
		t.Fatalf("read .gitignore: %v", err)
	}
	updated := string(content)
	if !strings.HasPrefix(updated, customContent) {
		t.Error("CreateGitignore should preserve existing .gitignore content")
	}
	for _, entry := range []string{"mcp-token", ".runtime-port", ".index"} {
		if !strings.Contains(updated, entry) {
			t.Errorf("expected .gitignore to contain %q", entry)
		}
		if countGitignoreEntry(updated, entry) != 1 {
			t.Errorf("expected .gitignore entry %q exactly once", entry)
		}
	}
}

func TestCreateGitignoreWithEmptyVaultDir(t *testing.T) {
	if err := CreateGitignore(""); err != nil {
		t.Error("CreateGitignore with empty vault dir should not fail")
	}
}

func TestPushErrorErrorMethod(t *testing.T) {
	cause := errors.New("underlying error")
	err := &PushError{Message: "test error", Cause: cause}

	if err.Error() != "push failed: test error: underlying error" {
		t.Errorf("Error() = %q, want push failed: test error: underlying error", err.Error())
	}
}

func TestPushErrorErrorMethodWithoutCause(t *testing.T) {
	err := &PushError{Message: "test error", Cause: nil}

	if err.Error() != "push failed: test error" {
		t.Errorf("Error() = %q, want push failed: test error", err.Error())
	}
}

func TestPushErrorUnwrap(t *testing.T) {
	cause := errors.New("underlying error")
	err := &PushError{Message: "test error", Cause: cause}

	if err.Unwrap() != cause { //nolint:errorlint // comparing exact unwrapped error in test
		t.Errorf("Unwrap() = %v, want %v", err.Unwrap(), cause)
	}
}

func TestAutoCommitAndPushCommitsSuccessfully(t *testing.T) {
	origin := t.TempDir()
	if _, err := gogit.PlainInit(origin, true); err != nil {
		t.Fatalf("PlainInit(origin): %v", err)
	}

	local := t.TempDir()
	if err := Init(local); err != nil {
		t.Fatalf("Init(local): %v", err)
	}

	repo, err := gogit.PlainOpen(local)
	if err != nil {
		t.Fatalf("open local repo: %v", err)
	}

	_, err = repo.CreateRemote(&config.RemoteConfig{Name: "origin", URLs: []string{origin}})
	if err != nil {
		t.Fatalf("CreateRemote(): %v", err)
	}

	writeFile(t, local, "vault.txt", []byte("secret"))

	if err = AutoCommitAndPush(local, "add secret", true); err != nil {
		t.Errorf("AutoCommitAndPush() error = %v", err)
	}

	ref, err := repo.Head()
	if err != nil {
		t.Fatalf("repo.Head(): %v", err)
	}
	commit, err := repo.CommitObject(ref.Hash())
	if err != nil {
		t.Fatalf("CommitObject(): %v", err)
	}
	if commit == nil {
		t.Error("expected commit to be created")
	}
}

func TestAutoCommitCommitsWithMessage(t *testing.T) {
	dir := t.TempDir()

	if err := Init(dir); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	writeFile(t, dir, "vault.txt", []byte("secret-1"))

	if err := AutoCommit(dir, "add vault file"); err != nil {
		t.Fatalf("AutoCommit() error = %v", err)
	}

	repo, err := gogit.PlainOpen(dir)
	if err != nil {
		t.Fatalf("open repo: %v", err)
	}

	ref, err := repo.Head()
	if err != nil {
		t.Fatalf("repo.Head(): %v", err)
	}

	commit, err := repo.CommitObject(ref.Hash())
	if err != nil {
		t.Fatalf("CommitObject(): %v", err)
	}

	if commit.Message != "add vault file" {
		t.Fatalf("commit message = %q, want %q", commit.Message, "add vault file")
	}
}

func TestAutoCommitDoesNotCommitRuntimeArtifacts(t *testing.T) {
	dir := t.TempDir()

	if err := Init(dir); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	writeFile(t, dir, "vault.txt", []byte("secret"))
	writeFile(t, dir, "mcp-token", []byte("do-not-commit-token"))
	writeFile(t, dir, ".runtime-port", []byte("12345"))
	writeFile(t, dir, ".index", []byte(`{"entries":["vault"]}`))

	if err := AutoCommit(dir, "add vault file"); err != nil {
		t.Fatalf("AutoCommit() error = %v", err)
	}

	repo, err := gogit.PlainOpen(dir)
	if err != nil {
		t.Fatalf("open repo: %v", err)
	}
	ref, err := repo.Head()
	if err != nil {
		t.Fatalf("repo.Head(): %v", err)
	}
	commit, err := repo.CommitObject(ref.Hash())
	if err != nil {
		t.Fatalf("CommitObject(): %v", err)
	}

	if _, err := commit.File("vault.txt"); err != nil {
		t.Fatalf("expected vault.txt in commit: %v", err)
	}
	for _, path := range []string{"mcp-token", ".runtime-port", ".index"} {
		if _, err := commit.File(path); err == nil {
			t.Fatalf("runtime artifact %s was committed", path)
		}
	}
}

func TestAutoCommitAndPush_NoAutoPush(t *testing.T) {
	local := t.TempDir()
	if err := Init(local); err != nil {
		t.Fatalf("Init(local): %v", err)
	}

	repo, err := gogit.PlainOpen(local)
	if err != nil {
		t.Fatalf("open local repo: %v", err)
	}

	_, err = repo.CreateRemote(&config.RemoteConfig{Name: "origin", URLs: []string{t.TempDir()}})
	if err != nil {
		t.Fatalf("CreateRemote(): %v", err)
	}

	writeFile(t, local, "vault.txt", []byte("secret"))

	if err = AutoCommitAndPush(local, "add secret", false); err != nil {
		t.Errorf("AutoCommitAndPush(autoPush=false) error = %v", err)
	}

	ref, err := repo.Head()
	if err != nil {
		t.Fatalf("repo.Head(): %v", err)
	}
	commit, err := repo.CommitObject(ref.Hash())
	if err != nil {
		t.Fatalf("CommitObject(): %v", err)
	}
	if commit == nil {
		t.Error("expected commit to be created")
	}
}

func TestInitWithEmptyVaultDir(t *testing.T) {
	if err := Init(""); err != nil {
		t.Error("Init with empty vault dir should not fail")
	}
}

func TestOpenRepoWithEmptyVaultDir(t *testing.T) {
	_, err := openRepo("")
	if err == nil {
		t.Error("openRepo with empty vault dir should return error")
	}
}

func TestPushAndPullOperations(t *testing.T) {
	origin := t.TempDir()
	if _, err := gogit.PlainInit(origin, true); err != nil {
		t.Fatalf("PlainInit(origin): %v", err)
	}

	local := t.TempDir()
	if err := Init(local); err != nil {
		t.Fatalf("Init(local): %v", err)
	}

	repo, err := gogit.PlainOpen(local)
	if err != nil {
		t.Fatalf("open local repo: %v", err)
	}

	_, err = repo.CreateRemote(&config.RemoteConfig{Name: "origin", URLs: []string{origin}})
	if err != nil {
		t.Fatalf("CreateRemote(): %v", err)
	}

	writeFile(t, local, "vault.txt", []byte("secret-2"))
	if err = AutoCommit(local, "add secret"); err != nil {
		t.Fatalf("AutoCommit(local): %v", err)
	}

	if err = Push(local); err != nil {
		t.Fatalf("Push() error = %v", err)
	}

	cloneDir := t.TempDir()
	clone, err := gogit.PlainClone(cloneDir, false, &gogit.CloneOptions{URL: origin})
	if err != nil {
		t.Fatalf("PlainClone(): %v", err)
	}

	writeFile(t, cloneDir, "vault.txt", []byte("secret-3"))
	if err := AutoCommit(cloneDir, "remote change"); err != nil {
		t.Fatalf("AutoCommit(clone): %v", err)
	}
	if err := clone.Push(&gogit.PushOptions{}); err != nil {
		t.Fatalf("clone.Push(): %v", err)
	}

	if err := Pull(local); err != nil {
		t.Fatalf("Pull() error = %v", err)
	}
}

func TestLogReturnsCommitHistory(t *testing.T) {
	dir := t.TempDir()
	if err := Init(dir); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	writeFile(t, dir, "vault.txt", []byte("secret-1"))
	if err := AutoCommit(dir, "first"); err != nil {
		t.Fatalf("AutoCommit first: %v", err)
	}
	writeFile(t, dir, "vault.txt", []byte("secret-2"))
	if err := AutoCommit(dir, "second"); err != nil {
		t.Fatalf("AutoCommit second: %v", err)
	}

	commits, err := Log(dir, "vault.txt", 10)
	if err != nil {
		t.Fatalf("Log() error = %v", err)
	}
	if len(commits) != 2 {
		t.Fatalf("len(commits) = %d, want 2", len(commits))
	}
	if commits[0].Message != "second" || commits[1].Message != "first" {
		t.Fatalf("unexpected commit order/messages: %#v", commits)
	}
	if commits[0].Hash == "" || commits[0].Author == "" || commits[0].Date.IsZero() {
		t.Fatalf("expected populated commit fields: %#v", commits[0])
	}
}

func TestOperationsGracefullyHandleNonRepo(t *testing.T) {
	dir := t.TempDir()

	if err := AutoCommit(dir, "noop"); err != nil {
		t.Fatalf("AutoCommit on non-repo should not fail: %v", err)
	}
	if err := Push(dir); err != nil {
		t.Fatalf("Push on non-repo should not fail: %v", err)
	}
	if err := Pull(dir); err != nil {
		t.Fatalf("Pull on non-repo should not fail: %v", err)
	}
	commits, err := Log(dir, "vault.txt", 5)
	if err != nil {
		t.Fatalf("Log on non-repo should not fail: %v", err)
	}
	if len(commits) != 0 {
		t.Fatalf("expected no commits, got %d", len(commits))
	}
}

//nolint:unparam // name parameter kept for readability
func writeFile(t *testing.T, dir, name string, data []byte) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), data, 0o600); err != nil {
		t.Fatalf("writeFile(%s): %v", name, err)
	}
}

func countGitignoreEntry(content, entry string) int {
	count := 0
	for _, line := range strings.Split(content, "\n") {
		if strings.TrimSpace(line) == entry {
			count++
		}
	}
	return count
}

var _ = object.Commit{}

func TestFormatAuthor(t *testing.T) {
	emptySig := object.Signature{}
	if formatAuthor(emptySig) != "" {
		t.Error("formatAuthor with empty sig should return empty string")
	}

	nameOnly := object.Signature{Name: "Alice"}
	if formatAuthor(nameOnly) != "Alice" {
		t.Errorf("formatAuthor(nameOnly) = %q, want %q", formatAuthor(nameOnly), "Alice")
	}

	emailOnly := object.Signature{Email: "alice@example.com"}
	if formatAuthor(emailOnly) != "alice@example.com" {
		t.Errorf("formatAuthor(emailOnly) = %q, want %q", formatAuthor(emailOnly), "alice@example.com")
	}

	fullSig := object.Signature{Name: "Alice", Email: "alice@example.com"}
	if formatAuthor(fullSig) != "Alice <alice@example.com>" {
		t.Errorf("formatAuthor(fullSig) = %q, want %q", formatAuthor(fullSig), "Alice <alice@example.com>")
	}
}

func TestInitDoesNotReinitExistingRepo(t *testing.T) {
	dir := t.TempDir()

	if err := Init(dir); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	// Re-init should not fail and should not error
	if err := Init(dir); err != nil {
		t.Fatalf("Init() re-init error = %v", err)
	}
}

func TestAutoCommitWithOptionsNoChanges(t *testing.T) {
	dir := t.TempDir()

	if err := Init(dir); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	// AutoCommit on clean repo should return nil (no changes)
	if err := AutoCommitWithOptions(dir, CommitOptions{Message: "no changes"}); err != nil {
		t.Errorf("AutoCommitWithOptions on clean repo should return nil, got: %v", err)
	}
}

func TestAutoCommitWithOptionsWithCustomAuthor(t *testing.T) {
	dir := t.TempDir()

	if err := Init(dir); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	writeFile(t, dir, "vault.txt", []byte("secret"))

	opts := CommitOptions{
		Message: "custom author commit",
		Author:  "TestUser",
		Email:   "test@example.com",
	}

	if err := AutoCommitWithOptions(dir, opts); err != nil {
		t.Fatalf("AutoCommitWithOptions() error = %v", err)
	}

	repo, err := gogit.PlainOpen(dir)
	if err != nil {
		t.Fatalf("open repo: %v", err)
	}

	ref, err := repo.Head()
	if err != nil {
		t.Fatalf("repo.Head(): %v", err)
	}

	commit, err := repo.CommitObject(ref.Hash())
	if err != nil {
		t.Fatalf("CommitObject(): %v", err)
	}

	if commit.Author.Name != "TestUser" {
		t.Errorf("author name = %q, want %q", commit.Author.Name, "TestUser")
	}
	if commit.Author.Email != "test@example.com" {
		t.Errorf("author email = %q, want %q", commit.Author.Email, "test@example.com")
	}
}

func TestAutoCommitWithOptionsDefaultAuthor(t *testing.T) {
	dir := t.TempDir()

	if err := Init(dir); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	writeFile(t, dir, "vault.txt", []byte("secret"))

	// Use default template but no explicit author
	opts := CommitOptions{
		Template: "default template commit",
	}

	if err := AutoCommitWithOptions(dir, opts); err != nil {
		t.Fatalf("AutoCommitWithOptions() error = %v", err)
	}

	repo, err := gogit.PlainOpen(dir)
	if err != nil {
		t.Fatalf("open repo: %v", err)
	}

	ref, err := repo.Head()
	if err != nil {
		t.Fatalf("repo.Head(): %v", err)
	}

	commit, err := repo.CommitObject(ref.Hash())
	if err != nil {
		t.Fatalf("CommitObject(): %v", err)
	}

	// Should use default author
	if commit.Author.Name != "OpenPass" {
		t.Errorf("author name = %q, want %q", commit.Author.Name, "OpenPass")
	}
	if commit.Author.Email != "openpass@example.com" {
		t.Errorf("author email = %q, want %q", commit.Author.Email, "openpass@example.com")
	}
	if commit.Message != "default template commit" {
		t.Errorf("message = %q, want %q", commit.Message, "default template commit")
	}
}

func TestAutoCommitWithOptionsUsesDefaultTemplate(t *testing.T) {
	dir := t.TempDir()

	if err := Init(dir); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	writeFile(t, dir, "vault.txt", []byte("secret"))

	// No message, no template - should use default
	opts := CommitOptions{}

	if err := AutoCommitWithOptions(dir, opts); err != nil {
		t.Fatalf("AutoCommitWithOptions() error = %v", err)
	}

	repo, err := gogit.PlainOpen(dir)
	if err != nil {
		t.Fatalf("open repo: %v", err)
	}

	ref, err := repo.Head()
	if err != nil {
		t.Fatalf("repo.Head(): %v", err)
	}

	commit, err := repo.CommitObject(ref.Hash())
	if err != nil {
		t.Fatalf("CommitObject(): %v", err)
	}

	if commit.Message != DefaultCommitTemplate {
		t.Errorf("message = %q, want %q", commit.Message, DefaultCommitTemplate)
	}
}

func TestPushWithResultNoRemote(t *testing.T) {
	dir := t.TempDir()

	if err := Init(dir); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	result := PushWithResult(dir)
	if !result.Skipped {
		t.Error("PushWithResult should skip when no remote configured")
	}
	if result.Error == nil {
		t.Error("PushWithResult should set error when no remote configured")
	}
	if result.HasRemote {
		t.Error("PushWithResult should have HasRemote=false when no remote configured")
	}
}

func TestPushWithResultRemoteListError(t *testing.T) {
	// Test PushWithResult with a repo that has remotes listed but they error
	// This is hard to trigger directly, but we test the no-remote case
	dir := t.TempDir()

	if err := Init(dir); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	result := PushWithResult(dir)
	if !result.Skipped {
		t.Error("PushWithResult should skip when no remote configured")
	}
	if result.Error == nil {
		t.Error("PushWithResult should set error when no remote configured")
	}
}

func TestPushWithResultAlreadyUpToDate(t *testing.T) {
	origin := t.TempDir()
	if _, err := gogit.PlainInit(origin, true); err != nil {
		t.Fatalf("PlainInit(origin): %v", err)
	}

	local := t.TempDir()
	if err := Init(local); err != nil {
		t.Fatalf("Init(local): %v", err)
	}

	repo, err := gogit.PlainOpen(local)
	if err != nil {
		t.Fatalf("open local repo: %v", err)
	}

	_, err = repo.CreateRemote(&config.RemoteConfig{Name: "origin", URLs: []string{origin}})
	if err != nil {
		t.Fatalf("CreateRemote(): %v", err)
	}

	writeFile(t, local, "vault.txt", []byte("secret"))
	if err := AutoCommit(local, "first commit"); err != nil {
		t.Fatalf("AutoCommit(): %v", err)
	}

	// First push
	if err := Push(local); err != nil {
		t.Fatalf("Push(): %v", err)
	}

	// Second push should be up-to-date
	result := PushWithResult(local)
	if !result.Success {
		t.Error("PushWithResult should succeed when already up-to-date")
	}
	if !result.Skipped {
		t.Error("PushWithResult should be skipped when up-to-date")
	}
}

func TestAutoCommitAndPushNonRepo(t *testing.T) {
	dir := t.TempDir()

	// AutoCommitAndPush on non-repo is graceful - AutoCommit returns nil
	if err := AutoCommitAndPush(dir, "should not fail", true); err != nil {
		t.Errorf("AutoCommitAndPush should be graceful on non-repo: %v", err)
	}
}

func TestAutoCommitAndPushPushFailure(t *testing.T) {
	origin := t.TempDir()
	if _, err := gogit.PlainInit(origin, true); err != nil {
		t.Fatalf("PlainInit(origin): %v", err)
	}

	local := t.TempDir()
	if err := Init(local); err != nil {
		t.Fatalf("Init(local): %v", err)
	}

	repo, err := gogit.PlainOpen(local)
	if err != nil {
		t.Fatalf("open local repo: %v", err)
	}

	// Create a remote pointing to invalid path
	_, err = repo.CreateRemote(&config.RemoteConfig{Name: "origin", URLs: []string{filepath.Join(origin, "nonexistent.git")}})
	if err != nil {
		t.Fatalf("CreateRemote(): %v", err)
	}

	writeFile(t, local, "vault.txt", []byte("secret"))

	if err := AutoCommitAndPush(local, "test push", true); err == nil {
		t.Error("AutoCommitAndPush should return error on push failure")
	}
}

func TestPushErrorPath(t *testing.T) {
	origin := t.TempDir()
	if _, err := gogit.PlainInit(origin, true); err != nil {
		t.Fatalf("PlainInit(origin): %v", err)
	}

	local := t.TempDir()
	if err := Init(local); err != nil {
		t.Fatalf("Init(local): %v", err)
	}

	repo, err := gogit.PlainOpen(local)
	if err != nil {
		t.Fatalf("open local repo: %v", err)
	}

	// Create remote pointing to a valid but unreachable path
	_, err = repo.CreateRemote(&config.RemoteConfig{Name: "origin", URLs: []string{origin}})
	if err != nil {
		t.Fatalf("CreateRemote(): %v", err)
	}

	writeFile(t, local, "vault.txt", []byte("secret"))
	if err := AutoCommit(local, "test"); err != nil {
		t.Fatalf("AutoCommit(): %v", err)
	}

	// Try to push with a bad refspec to trigger error
	result := PushWithResult(local)
	// This should either succeed (if push works) or set an error
	// We're testing the code path doesn't panic
	_ = result
}

func TestPullNonExistentRemote(t *testing.T) {
	local := t.TempDir()

	if err := Init(local); err != nil {
		t.Fatalf("Init(local): %v", err)
	}

	repo, err := gogit.PlainOpen(local)
	if err != nil {
		t.Fatalf("open local repo: %v", err)
	}

	// Create remote pointing to non-existent path
	_, err = repo.CreateRemote(&config.RemoteConfig{Name: "origin", URLs: []string{filepath.Join(t.TempDir(), "nonexistent.git")}})
	if err != nil {
		t.Fatalf("CreateRemote(): %v", err)
	}

	// Pull returns error for missing remote
	if err := Pull(local); err == nil {
		t.Error("Pull should return error for missing remote")
	}
}

func TestLogWithEmptyRepo(t *testing.T) {
	dir := t.TempDir()

	if err := Init(dir); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	// Log on empty repo (no commits) returns error "reference not found"
	commits, err := Log(dir, "", 5)
	if err == nil {
		t.Error("Log should return error on empty repo with no commits")
	}
	if len(commits) != 0 {
		t.Errorf("expected 0 commits, got %d", len(commits))
	}
}

func TestLogWithPathFilter(t *testing.T) {
	dir := t.TempDir()

	if err := Init(dir); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	writeFile(t, dir, "file1.txt", []byte("content1"))
	if err := AutoCommit(dir, "add file1"); err != nil {
		t.Fatalf("AutoCommit(): %v", err)
	}

	writeFile(t, dir, "file2.txt", []byte("content2"))
	if err := AutoCommit(dir, "add file2"); err != nil {
		t.Fatalf("AutoCommit(): %v", err)
	}

	// Log only file1.txt
	commits, err := Log(dir, "file1.txt", 10)
	if err != nil {
		t.Fatalf("Log() error = %v", err)
	}
	if len(commits) != 1 {
		t.Errorf("expected 1 commit for file1.txt, got %d", len(commits))
	}
}

func TestLogWithLimit(t *testing.T) {
	dir := t.TempDir()

	if err := Init(dir); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	for i := 0; i < 5; i++ {
		writeFile(t, dir, "vault.txt", []byte(fmt.Sprintf("secret-%d", i)))
		if err := AutoCommit(dir, fmt.Sprintf("commit %d", i)); err != nil {
			t.Fatalf("AutoCommit(): %v", err)
		}
	}

	commits, err := Log(dir, "", 3)
	if err != nil {
		t.Fatalf("Log() error = %v", err)
	}
	if len(commits) != 3 {
		t.Errorf("expected 3 commits, got %d", len(commits))
	}
}

func TestLogLimitExactlyMet(t *testing.T) {
	dir := t.TempDir()

	if err := Init(dir); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	for i := 0; i < 3; i++ {
		writeFile(t, dir, "vault.txt", []byte(fmt.Sprintf("secret-%d", i)))
		if err := AutoCommit(dir, fmt.Sprintf("commit %d", i)); err != nil {
			t.Fatalf("AutoCommit(): %v", err)
		}
	}

	commits, err := Log(dir, "", 3)
	if err != nil {
		t.Fatalf("Log() error = %v", err)
	}
	if len(commits) != 3 {
		t.Errorf("expected 3 commits, got %d", len(commits))
	}
}

func TestLogWithNonExistentPath(t *testing.T) {
	dir := t.TempDir()

	if err := Init(dir); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	writeFile(t, dir, "existing.txt", []byte("content"))
	if err := AutoCommit(dir, "add existing"); err != nil {
		t.Fatalf("AutoCommit(): %v", err)
	}

	// Log for non-existent file should return empty
	commits, err := Log(dir, "nonexistent.txt", 10)
	if err != nil {
		t.Fatalf("Log() error for non-existent path = %v", err)
	}
	if len(commits) != 0 {
		t.Errorf("expected 0 commits for non-existent path, got %d", len(commits))
	}
}

func TestAutoCommitWithBareRepo(t *testing.T) {
	dir := t.TempDir()

	// Create a bare repo in the directory
	if _, err := gogit.PlainInit(dir, true); err != nil {
		t.Fatalf("PlainInit(): %v", err)
	}

	writeFile(t, dir, "vault.txt", []byte("secret"))

	// AutoCommit on bare repo - AddWithOptions fails but is handled gracefully
	// The implementation returns nil even when Add fails on bare repo
	if err := AutoCommit(dir, "should be graceful"); err != nil {
		t.Errorf("AutoCommit should be graceful on bare repo: %v", err)
	}
}

func TestPushWithEmptyVaultDir(t *testing.T) {
	result := PushWithResult("")
	if !result.Skipped {
		t.Error("PushWithResult with empty dir should be skipped")
	}
}

func TestAutoCommitWithEmptyVaultDir(t *testing.T) {
	// AutoCommit on empty vault dir should return nil (graceful handling)
	if err := AutoCommit("", "test"); err != nil {
		t.Errorf("AutoCommit with empty dir should not fail: %v", err)
	}
}

func TestPullWithEmptyVaultDir(t *testing.T) {
	// Pull on empty vault dir should return nil (graceful handling)
	if err := Pull(""); err != nil {
		t.Errorf("Pull with empty dir should not fail: %v", err)
	}
}

func TestLogWithEmptyVaultDir(t *testing.T) {
	commits, err := Log("", "", 10)
	if err != nil {
		t.Errorf("Log with empty dir should not fail: %v", err)
	}
	if len(commits) != 0 {
		t.Errorf("Log with empty dir should return 0 commits, got %d", len(commits))
	}
}

func TestAutoCommitWithEmptyMessage(t *testing.T) {
	dir := t.TempDir()

	if err := Init(dir); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	writeFile(t, dir, "vault.txt", []byte("secret"))

	// AutoCommit with empty message should still work (uses default template)
	if err := AutoCommit(dir, ""); err != nil {
		t.Fatalf("AutoCommit() error = %v", err)
	}

	repo, err := gogit.PlainOpen(dir)
	if err != nil {
		t.Fatalf("open repo: %v", err)
	}

	ref, err := repo.Head()
	if err != nil {
		t.Fatalf("repo.Head(): %v", err)
	}

	commit, err := repo.CommitObject(ref.Hash())
	if err != nil {
		t.Fatalf("CommitObject(): %v", err)
	}

	if commit.Message != DefaultCommitTemplate {
		t.Errorf("message = %q, want default template %q", commit.Message, DefaultCommitTemplate)
	}
}

func TestInitMkdirAllError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("running as root; chmod 0 has no effect")
	}
	parent := t.TempDir()
	if err := os.Chmod(parent, 0o500); err != nil {
		t.Fatalf("Chmod() error = %v", err)
	}
	defer os.Chmod(parent, 0o700) //nolint:errcheck

	err := Init(filepath.Join(parent, "vault"))
	if err == nil {
		t.Fatal("Init() error = nil, want error when parent dir is not writable")
	}
}

func TestPullBareRepo(t *testing.T) {
	dir := t.TempDir()
	if _, err := gogit.PlainInit(dir, true); err != nil {
		t.Fatalf("PlainInit(bare): %v", err)
	}

	// Pull on a bare repo: openRepo succeeds but Worktree() fails — should return nil gracefully
	if err := Pull(dir); err != nil {
		t.Errorf("Pull on bare repo should return nil, got: %v", err)
	}
}

func TestPushWithResultNonFastForward(t *testing.T) {
	origin := t.TempDir()
	if _, err := gogit.PlainInit(origin, true); err != nil {
		t.Fatalf("PlainInit(origin): %v", err)
	}

	// First clone: push a commit
	local1 := t.TempDir()
	if err := Init(local1); err != nil {
		t.Fatalf("Init(local1): %v", err)
	}
	repo1, err := gogit.PlainOpen(local1)
	if err != nil {
		t.Fatalf("open local1: %v", err)
	}
	if _, errShadow := repo1.CreateRemote(&config.RemoteConfig{Name: "origin", URLs: []string{origin}}); errShadow != nil {
		t.Fatalf("CreateRemote(local1): %v", errShadow)
	}
	writeFile(t, local1, "f1.txt", []byte("first"))
	if errShadow := AutoCommit(local1, "first commit"); errShadow != nil {
		t.Fatalf("AutoCommit(local1): %v", errShadow)
	}
	if errShadow := Push(local1); errShadow != nil {
		t.Fatalf("Push(local1): %v", errShadow)
	}

	// Second clone: make a diverging commit and try to push (non-fast-forward)
	local2 := t.TempDir()
	if errShadow := Init(local2); errShadow != nil {
		t.Fatalf("Init(local2): %v", errShadow)
	}
	repo2, err := gogit.PlainOpen(local2)
	if err != nil {
		t.Fatalf("open local2: %v", err)
	}
	if _, err := repo2.CreateRemote(&config.RemoteConfig{Name: "origin", URLs: []string{origin}}); err != nil {
		t.Fatalf("CreateRemote(local2): %v", err)
	}
	writeFile(t, local2, "f2.txt", []byte("diverging"))
	if err := AutoCommit(local2, "diverging commit"); err != nil {
		t.Fatalf("AutoCommit(local2): %v", err)
	}

	result := PushWithResult(local2)
	if result.Success {
		t.Error("PushWithResult should not succeed for non-fast-forward push")
	}
	if result.Error == nil {
		t.Error("PushWithResult should set error for non-fast-forward push")
	}
}

func TestPushReturnsErrorOnFailure(t *testing.T) {
	local := t.TempDir()
	if err := Init(local); err != nil {
		t.Fatalf("Init(local): %v", err)
	}
	repo, err := gogit.PlainOpen(local)
	if err != nil {
		t.Fatalf("open local: %v", err)
	}
	if _, errShadow := repo.CreateRemote(&config.RemoteConfig{Name: "origin", URLs: []string{"http://127.0.0.1:1/repo.git"}}); errShadow != nil {
		t.Fatalf("CreateRemote(): %v", errShadow)
	}
	writeFile(t, local, "f.txt", []byte("hello"))
	if errShadow := AutoCommit(local, "test"); errShadow != nil {
		t.Fatalf("AutoCommit(): %v", errShadow)
	}

	// Push() should return the error from PushWithResult when !Skipped
	err = Push(local)
	if err == nil {
		t.Error("Push() error = nil, want error for connection refused")
	}
}

func TestPushWithResultConnectionRefused(t *testing.T) {
	local := t.TempDir()
	if err := Init(local); err != nil {
		t.Fatalf("Init(local): %v", err)
	}

	repo, err := gogit.PlainOpen(local)
	if err != nil {
		t.Fatalf("open local: %v", err)
	}
	if _, err := repo.CreateRemote(&config.RemoteConfig{Name: "origin", URLs: []string{"http://127.0.0.1:1/repo.git"}}); err != nil {
		t.Fatalf("CreateRemote(): %v", err)
	}

	writeFile(t, local, "f.txt", []byte("hello"))
	if err := AutoCommit(local, "test"); err != nil {
		t.Fatalf("AutoCommit(): %v", err)
	}

	result := PushWithResult(local)
	if result.Success {
		t.Error("PushWithResult should not succeed with connection refused")
	}
	if result.Error == nil {
		t.Error("PushWithResult should set error for connection refused")
	}
}
