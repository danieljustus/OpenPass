// Package git provides Git integration for OpenPass vaults.
package git

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing/object"
)

var errStopIter = errors.New("stop iteration")

// PushError represents an error that occurred during push
type PushError struct {
	Cause   error
	Message string
}

func (e *PushError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("push failed: %s: %v", e.Message, e.Cause)
	}
	return fmt.Sprintf("push failed: %s", e.Message)
}

func (e *PushError) Unwrap() error {
	return e.Cause
}

// PushResult represents the result of a push operation
type PushResult struct {
	Error     error
	RemoteURL string
	Success   bool
	Skipped   bool
	HasRemote bool
}

// CommitOptions holds options for committing
type CommitOptions struct {
	Message  string
	Template string
	Author   string
	Email    string
}

// DefaultCommitTemplate is the default commit message template
const DefaultCommitTemplate = "Update from OpenPass"

// DefaultGitignoreContent is the default .gitignore content for OpenPass vaults
const DefaultGitignoreContent = `# OpenPass vault - ignore sensitive files
identity.age
*.key
*.pem
# Ignore OpenPass runtime artifacts
mcp-token
.runtime-port
# Ignore OS files
.DS_Store
Thumbs.db
# Ignore IDE files
.idea/
.vscode/
*.swp
*.swo
*~
`

var protectedRuntimePaths = []string{
	"mcp-token",
	".runtime-port",
}

type Commit struct {
	Hash    string
	Author  string
	Date    time.Time
	Message string
}

func Init(vaultDir string) error {
	if vaultDir == "" {
		return nil
	}

	if _, err := openRepo(vaultDir); err == nil {
		return nil
	}

	if err := os.MkdirAll(vaultDir, 0o700); err != nil {
		return err
	}
	_, err := gogit.PlainInit(vaultDir, false)
	return err
}

// CreateGitignore creates a .gitignore file in the vault directory
func CreateGitignore(vaultDir string) error {
	if vaultDir == "" {
		return nil
	}

	cleanVaultDir := filepath.Clean(vaultDir)
	gitignorePath := filepath.Join(cleanVaultDir, ".gitignore")
	cleanGitignorePath := filepath.Clean(gitignorePath)
	if !strings.HasPrefix(cleanGitignorePath, cleanVaultDir+string(filepath.Separator)) {
		return fmt.Errorf("invalid gitignore path: outside vault directory")
	}

	content, err := os.ReadFile(cleanGitignorePath) //nolint:gosec // path is validated above
	if err != nil {
		if os.IsNotExist(err) {
			return os.WriteFile(cleanGitignorePath, []byte(DefaultGitignoreContent), 0o600)
		}
		return err
	}

	updated := appendMissingGitignoreEntries(string(content), DefaultGitignoreContent)
	if updated == string(content) {
		return nil
	}
	return os.WriteFile(cleanGitignorePath, []byte(updated), 0o600) //#nosec G703 -- path validated above: cleaned and checked to be within vault dir
}

// AutoCommitWithOptions performs an auto-commit with the given options
func AutoCommitWithOptions(vaultDir string, opts CommitOptions) error {
	repo, err := openRepo(vaultDir)
	if err != nil {
		return nil
	}

	w, err := repo.Worktree()
	if err != nil {
		return nil
	}

	if gitignoreErr := CreateGitignore(vaultDir); gitignoreErr != nil {
		return gitignoreErr
	}

	if addErr := w.AddWithOptions(&gogit.AddOptions{All: true}); addErr != nil {
		return addErr
	}

	status, statusErr := w.Status()
	if statusErr != nil {
		return nil
	}
	if unstageErr := unstageProtectedRuntimeArtifacts(repo, w, status); unstageErr != nil {
		return unstageErr
	}

	status, statusErr = w.Status()
	if statusErr != nil {
		return nil
	}
	if !hasStagedChanges(status) {
		return nil
	}

	// Determine commit message
	message := opts.Message
	if message == "" {
		message = opts.Template
	}
	if message == "" {
		message = DefaultCommitTemplate
	}

	// Determine author
	authorName := opts.Author
	if authorName == "" {
		authorName = "OpenPass"
	}
	authorEmail := opts.Email
	if authorEmail == "" {
		authorEmail = "openpass@example.com"
	}

	_, err = w.Commit(message, &gogit.CommitOptions{
		Author: &object.Signature{
			Name:  authorName,
			Email: authorEmail,
			When:  time.Now(),
		},
	})
	if err != nil {
		return err
	}

	return nil
}

func appendMissingGitignoreEntries(current string, defaults string) string {
	seen := make(map[string]bool)
	for _, line := range strings.Split(current, "\n") {
		seen[strings.TrimSpace(line)] = true
	}

	var missing []string
	for _, line := range strings.Split(defaults, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") || seen[trimmed] {
			continue
		}
		missing = append(missing, trimmed)
		seen[trimmed] = true
	}
	if len(missing) == 0 {
		return current
	}

	var b strings.Builder
	b.WriteString(current)
	if current != "" && !strings.HasSuffix(current, "\n") {
		b.WriteString("\n")
	}
	b.WriteString("# OpenPass runtime artifacts\n")
	for _, line := range missing {
		b.WriteString(line)
		b.WriteString("\n")
	}
	return b.String()
}

func hasStagedChanges(status gogit.Status) bool {
	for _, fileStatus := range status {
		if fileStatus.Staging != gogit.Unmodified && fileStatus.Staging != gogit.Untracked {
			return true
		}
	}
	return false
}

func unstageProtectedRuntimeArtifacts(repo *gogit.Repository, w *gogit.Worktree, status gogit.Status) error {
	var staged []string
	for path, fileStatus := range status {
		if !isProtectedRuntimePath(path) {
			continue
		}
		if fileStatus.Staging != gogit.Unmodified && fileStatus.Staging != gogit.Untracked {
			staged = append(staged, filepath.ToSlash(path))
		}
	}
	if len(staged) == 0 {
		return nil
	}

	if _, err := repo.Head(); err == nil {
		return w.Reset(&gogit.ResetOptions{Mode: gogit.MixedReset, Files: staged})
	}

	idx, err := repo.Storer.Index()
	if err != nil {
		return err
	}
	for _, path := range staged {
		_, _ = idx.Remove(path)
	}
	return repo.Storer.SetIndex(idx)
}

func isProtectedRuntimePath(path string) bool {
	normalized := filepath.ToSlash(filepath.Clean(path))
	for _, protected := range protectedRuntimePaths {
		if normalized == protected {
			return true
		}
		if strings.HasPrefix(normalized, protected+".") {
			return true
		}
	}
	return false
}

// AutoCommit performs a simple auto-commit with the given message
func AutoCommit(vaultDir string, message string) error {
	return AutoCommitWithOptions(vaultDir, CommitOptions{Message: message})
}

// PushWithResult pushes to origin and returns detailed result
func PushWithResult(vaultDir string) PushResult {
	result := PushResult{Success: false, Skipped: false}

	repo, err := openRepo(vaultDir)
	if err != nil {
		result.Skipped = true
		return result
	}

	// Check if remote exists
	remotes, err := repo.Remotes()
	if err != nil {
		result.Error = &PushError{Message: "failed to list remotes", Cause: err}
		return result
	}

	var originRemote *gogit.Remote
	for _, r := range remotes {
		if r.Config().Name == "origin" {
			originRemote = r
			result.HasRemote = true
			if len(r.Config().URLs) > 0 {
				result.RemoteURL = r.Config().URLs[0]
			}
			break
		}
	}

	if originRemote == nil {
		result.Skipped = true
		result.Error = &PushError{Message: "no 'origin' remote configured"}
		return result
	}

	err = repo.Push(&gogit.PushOptions{RemoteName: "origin"})
	if err == nil {
		result.Success = true
		return result
	}

	if errors.Is(err, gogit.NoErrAlreadyUpToDate) {
		result.Success = true
		result.Skipped = true
		return result
	}

	if errors.Is(err, gogit.ErrRemoteNotFound) || errors.Is(err, gogit.ErrRepositoryNotExists) {
		result.Skipped = true
		result.Error = &PushError{Message: "remote not found", Cause: err}
		return result
	}

	// Handle authentication errors gracefully
	errStr := err.Error()
	if strings.Contains(errStr, "authentication") || strings.Contains(errStr, "credentials") ||
		strings.Contains(errStr, "401") || strings.Contains(errStr, "403") {
		result.Error = &PushError{
			Message: "authentication failed - please check your credentials",
			Cause:   err,
		}
	} else if strings.Contains(errStr, "connection") || strings.Contains(errStr, "timeout") ||
		strings.Contains(errStr, "refused") {
		result.Error = &PushError{
			Message: "network error - please check your connection",
			Cause:   err,
		}
	} else {
		result.Error = &PushError{
			Message: "push failed",
			Cause:   err,
		}
	}

	return result
}

// AutoCommitAndPush performs auto-commit and optionally auto-push
func AutoCommitAndPush(vaultDir string, message string, autoPush bool) error {
	// Perform commit
	if err := AutoCommit(vaultDir, message); err != nil {
		return fmt.Errorf("commit failed: %w", err)
	}

	// Perform push if enabled
	if autoPush {
		result := PushWithResult(vaultDir)
		if result.Error != nil {
			if result.Skipped {
				return nil
			}
			return fmt.Errorf("push failed: %w", result.Error)
		}
	}

	return nil
}

func Push(vaultDir string) error {
	result := PushWithResult(vaultDir)
	if result.Error != nil && !result.Skipped {
		return result.Error
	}
	return nil
}

func Pull(vaultDir string) error {
	repo, err := openRepo(vaultDir)
	if err != nil {
		return nil
	}

	w, err := repo.Worktree()
	if err != nil {
		return nil
	}

	err = w.Pull(&gogit.PullOptions{RemoteName: "origin"})
	if err == nil || errors.Is(err, gogit.NoErrAlreadyUpToDate) {
		return nil
	}
	if errors.Is(err, gogit.ErrRemoteNotFound) || errors.Is(err, gogit.ErrRepositoryNotExists) {
		return nil
	}
	return err
}

func Log(vaultDir string, path string, limit int) ([]Commit, error) {
	repo, err := openRepo(vaultDir)
	if err != nil {
		return []Commit{}, nil
	}

	var opts gogit.LogOptions
	if path != "" {
		rel := filepath.ToSlash(path)
		opts.FileName = &rel
	}

	iter, err := repo.Log(&opts)
	if err != nil {
		if errors.Is(err, gogit.ErrRepositoryNotExists) {
			return []Commit{}, nil
		}
		return nil, err
	}
	defer iter.Close()

	commits := make([]Commit, 0)
	err = iter.ForEach(func(c *object.Commit) error {
		commits = append(commits, Commit{
			Hash:    c.Hash.String(),
			Author:  formatAuthor(c.Author),
			Date:    c.Author.When,
			Message: c.Message,
		})
		if limit > 0 && len(commits) >= limit {
			return errStopIter
		}
		return nil
	})
	if err != nil && !errors.Is(err, errStopIter) {
		return nil, err
	}
	if limit > 0 && len(commits) > limit {
		commits = commits[:limit]
	}
	return commits, nil
}

func openRepo(vaultDir string) (*gogit.Repository, error) {
	if vaultDir == "" {
		return nil, fmt.Errorf("empty vault dir")
	}
	repo, err := gogit.PlainOpen(vaultDir)
	if err != nil {
		return nil, err
	}
	return repo, nil
}

func formatAuthor(sig object.Signature) string {
	if sig.Email == "" {
		return sig.Name
	}
	if sig.Name == "" {
		return sig.Email
	}
	return fmt.Sprintf("%s <%s>", sig.Name, sig.Email)
}

var _ = config.RemoteConfig{}
