package git

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ── Concurrent Access Protection ─────────────────────────────────────────────

// repoMu guards all git write operations to prevent race conditions when
// multiple gitBook instances operate on the same repository simultaneously.
// Read-only commands (status, log, diff, branch -l, etc.) bypass the lock
// for better responsiveness.
var repoMu sync.Mutex

// writeCommands is the set of git sub-commands that mutate repository state.
var writeCommands = map[string]bool{
	"init": true, "commit": true, "merge": true, "rebase": true,
	"reset": true, "revert": true, "cherry-pick": true, "stash": true,
	"tag": true, "push": true, "pull": true, "fetch": true, "clone": true,
	"checkout": true, "switch": true, "restore": true, "rm": true,
	"mv": true, "add": true, "apply": true, "am": true,
}

func isWriteCommand(args []string) bool {
	if len(args) == 0 {
		return false
	}
	return writeCommands[args[0]]
}

// ── Security Utilities ───────────────────────────────────────────────────────────

// sanitizeGitArg removes shell metacharacters that could lead to command injection
func sanitizeGitArg(arg string) string {
	// Remove shell metacharacters that could be dangerous
	dangerous := []string{";", "&", "|", "`", "$", "(", ")", "<", ">", "\"", "'", "\\", "\n", "\r", "\t"}
	result := arg
	for _, d := range dangerous {
		result = strings.ReplaceAll(result, d, "")
	}
	return strings.TrimSpace(result)
}

// sanitizeGitOutput removes sensitive information like tokens from URLs and other credentials
func sanitizeGitOutput(output string) string {
	if output == "" {
		return output
	}

	// Redact tokens in URLs (https://token@github.com/user/repo)
	tokenRegex := regexp.MustCompile(`(https?://)[^@/\s]+@`)
	output = tokenRegex.ReplaceAllString(output, "${1}[REDACTED]@")

	// Redact API keys and tokens in command output
	apiKeyRegex := regexp.MustCompile(`(?i)(api[_-]?key|token|secret|password|auth)[\s:=]+['"]?[a-zA-Z0-9+/=_-]{8,}['"]?`)
	output = apiKeyRegex.ReplaceAllString(output, "${1}[REDACTED]")

	// Redact potential bearer tokens
	bearerRegex := regexp.MustCompile(`(?i)bearer\s+[a-zA-Z0-9+/=_-]{20,}`)
	output = bearerRegex.ReplaceAllString(output, "bearer [REDACTED]")

	// Redact SSH keys (simplified pattern)
	sshKeyRegex := regexp.MustCompile(`ssh-(rsa|dss|ed25519|ecdsa)\s+[a-zA-Z0-9+/]{20,}`)
	output = sshKeyRegex.ReplaceAllString(output, "ssh-${1} [REDACTED]")

	return output
}

// validateInput checks for potentially dangerous input patterns
func validateInput(input string) error {
	if input == "" {
		return fmt.Errorf("empty input not allowed")
	}
	if strings.Contains(input, "..") {
		return fmt.Errorf("path traversal sequences not allowed")
	}
	if len(input) > 1000 {
		return fmt.Errorf("input too long (max 1000 characters)")
	}
	// Check for null bytes and other control characters
	if strings.ContainsAny(input, "\x00\x01\x02\x03\x04\x05\x06\x07\x08\x09\x0a\x0b\x0c\x0d\x0e\x0f") {
		return fmt.Errorf("invalid control characters in input")
	}
	return nil
}

// validateGitCommand validates git command arguments for safety
func validateGitCommand(args ...string) error {
	if len(args) == 0 {
		return fmt.Errorf("no git command specified")
	}

	// Validate each argument
	for i, arg := range args {
		if err := validateInput(arg); err != nil {
			return fmt.Errorf("invalid argument %d: %v", i, err)
		}
	}

	// Check for potentially dangerous command combinations
	dangerousCommands := []string{
		"config --global", "config --system",
		"clean -fd", "reset --hard",
		"update-server-info", "gc --aggressive",
	}

	fullCmd := strings.Join(args, " ")
	for _, dangerous := range dangerousCommands {
		if strings.Contains(fullCmd, dangerous) {
			return fmt.Errorf("potentially dangerous command blocked: %s", dangerous)
		}
	}

	return nil
}

// ── GitResult ─────────────────────────────────────────────────────────────────

// GitResult holds the output and status of a git command.
type GitResult struct {
	Output string
	Err    string
	OK     bool
}

// runGit executes a git command and returns its result.
// Write operations are serialised via repoMu; reads run concurrently.
func runGit(args ...string) GitResult {
	// Enhanced command validation
	if err := validateGitCommand(args...); err != nil {
		return GitResult{Err: fmt.Sprintf("validation failed: %v", err), OK: false}
	}

	// Sanitize arguments (except for safe commands)
	sanitizedArgs := make([]string, len(args))
	for i, arg := range args {
		if i == 0 || !isSafeGitCommand(args[0]) {
			sanitizedArgs[i] = sanitizeGitArg(arg)
		} else {
			sanitizedArgs[i] = arg
		}
	}

	if len(sanitizedArgs) == 0 {
		return GitResult{Err: "no valid arguments after sanitization", OK: false}
	}

	// Serialise write operations to prevent concurrent-access corruption
	if isWriteCommand(sanitizedArgs) {
		repoMu.Lock()
		defer repoMu.Unlock()
	}

	return execGitWithRetry(sanitizedArgs, "", 2)
}

// execGitWithRetry runs the git command up to (1 + retries) times,
// retrying only on transient lock-file errors.
func execGitWithRetry(args []string, dir string, retries int) GitResult {
	var last GitResult
	for attempt := 0; attempt <= retries; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Duration(attempt*150) * time.Millisecond)
		}
		last = execGitOnce(args, dir)
		if last.OK {
			return last
		}
		// Only retry on lock-file contention
		if !strings.Contains(strings.ToLower(last.Err), "lock") &&
			!strings.Contains(strings.ToLower(last.Err), "index.lock") {
			break
		}
	}
	return last
}

// execGitOnce performs a single git invocation with a 30-second timeout.
func execGitOnce(args []string, dir string) GitResult {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", args...)
	if dir != "" {
		cmd.Dir = dir
	}

	stdout, err := cmd.Output()
	if err != nil {
		// Context deadline — surface as a recognisable "timed out" message
		if ctx.Err() != nil {
			auditLog("ERROR", strings.Join(args, " "), "timed out after 30s")
			return GitResult{Err: "git timed out after 30s", OK: false}
		}
		if ee, ok := err.(*exec.ExitError); ok {
			errMsg := sanitizeGitOutput(strings.TrimSpace(string(ee.Stderr)))
			auditLog("ERROR", strings.Join(args, " "), errMsg)
			return GitResult{
				Output: sanitizeGitOutput(strings.TrimSpace(string(stdout))),
				Err:    errMsg,
				OK:     false,
			}
		}
		errMsg := sanitizeGitOutput(err.Error())
		auditLog("ERROR", strings.Join(args, " "), errMsg)
		return GitResult{Err: errMsg, OK: false}
	}

	auditLog("OK", strings.Join(args, " "), "")
	return GitResult{Output: sanitizeGitOutput(strings.TrimSpace(string(stdout))), OK: true}
}


// auditLog writes a lightweight audit entry to a daily log file.
// It intentionally avoids importing the ui package to prevent circular imports.
func auditLog(status, command, errMsg string) {
	logDir := fmt.Sprintf("%s/gitbook/logs", os.TempDir())
	_ = os.MkdirAll(logDir, 0755)
	logPath := fmt.Sprintf("%s/gitbook-%s.log", logDir, time.Now().Format("2006-01-02"))
	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	entry := fmt.Sprintf("%s [%s] git %s", time.Now().Format(time.RFC3339), status, command)
	if errMsg != "" {
		entry += " | " + errMsg
	}
	f.WriteString(entry + "\n")
}

// isSafeGitCommand returns true if the command doesn't need argument sanitization
func isSafeGitCommand(cmd string) bool {
	safeCommands := []string{"add", "checkout", "branch", "tag", "remote", "clone", "init"}
	for _, safe := range safeCommands {
		if cmd == safe {
			return true
		}
	}
	return false
}

// runGitInDir executes git in a specific directory.
func runGitInDir(dir string, args ...string) GitResult {
	// Validate directory
	if err := validateInput(dir); err != nil {
		return GitResult{Err: fmt.Sprintf("invalid directory: %v", err), OK: false}
	}

	// Input validation for arguments
	if len(args) == 0 {
		return GitResult{Err: "no arguments provided", OK: false}
	}
	for _, arg := range args {
		if err := validateInput(arg); err != nil {
			return GitResult{Err: fmt.Sprintf("invalid argument: %v", err), OK: false}
		}
	}

	// Sanitize arguments
	sanitizedArgs := make([]string, len(args))
	for i, arg := range args {
		if i == 0 || !isSafeGitCommand(args[0]) {
			sanitizedArgs[i] = sanitizeGitArg(arg)
		} else {
			sanitizedArgs[i] = arg
		}
	}

	if isWriteCommand(sanitizedArgs) {
		repoMu.Lock()
		defer repoMu.Unlock()
	}

	return execGitWithRetry(sanitizedArgs, dir, 2)
}

// ── Repository State ──────────────────────────────────────────────────────────

func IsGitRepo() bool {
	r := runGit("rev-parse", "--is-inside-work-tree")
	return r.OK && strings.TrimSpace(r.Output) == "true"
}

func HasCommits() bool {
	r := runGit("rev-parse", "--verify", "HEAD")
	return r.OK
}

func IsGitIdentitySet() bool {
	name := runGit("config", "user.name")
	email := runGit("config", "user.email")
	return name.OK && strings.TrimSpace(name.Output) != "" &&
		email.OK && strings.TrimSpace(email.Output) != ""
}

func GitGetIdentity() (name, email string) {
	n := runGit("config", "user.name")
	e := runGit("config", "user.email")
	if n.OK {
		name = strings.TrimSpace(n.Output)
	}
	if e.OK {
		email = strings.TrimSpace(e.Output)
	}
	return
}

func GitConfig(key, value string) GitResult {
	return runGit("config", key, value)
}

func GitVersion() GitResult {
	return runGit("--version")
}

func GitRootDir() GitResult {
	return runGit("rev-parse", "--show-toplevel")
}

func GitRepoName() string {
	r := GitRootDir()
	if !r.OK {
		return ""
	}
	return filepath.Base(r.Output)
}

// ── Core Operations ───────────────────────────────────────────────────────────

func GitInit() GitResult                { return runGit("init") }
func GitInitInDir(dir string) GitResult { return runGitInDir(dir, "init") }
func GitStatus() GitResult              { return runGit("status", "--short") }
func GitStatusLong() GitResult          { return runGit("status") }
func GitBranch() GitResult              { return runGit("rev-parse", "--abbrev-ref", "HEAD") }
func GitBranchList() GitResult          { return runGit("branch", "--list", "--format=%(refname:short)") }

func GitBranchExists(name string) bool {
	r := runGit("rev-parse", "--verify", "refs/heads/"+name)
	return r.OK
}

func GitCreateBranch(name string) GitResult      { return runGit("branch", name) }
func GitCheckout(branch string) GitResult        { return runGit("checkout", branch) }
func GitCheckoutNewBranch(name string) GitResult { return runGit("checkout", "-b", name) }
func GitMerge(branch string) GitResult           { return runGit("merge", "--no-ff", branch) }
func GitMergeSquash(branch string) GitResult     { return runGit("merge", "--squash", branch) }
func GitDeleteBranch(name string) GitResult      { return runGit("branch", "-d", name) }
func GitDeleteBranchForce(name string) GitResult { return runGit("branch", "-D", name) }

// ── Staging & Committing ──────────────────────────────────────────────────────

func GitAdd(path string) GitResult            { return runGit("add", path) }
func GitAddAll() GitResult                    { return runGit("add", "--all") }
func GitUnstage(path string) GitResult        { return runGit("restore", "--staged", path) }
func GitCommit(message string) GitResult      { return runGit("commit", "-m", message) }
func GitCommitAmend(message string) GitResult { return runGit("commit", "--amend", "-m", message) }
func GitCommitEmpty(message string) GitResult {
	return runGit("commit", "--allow-empty", "-m", message)
}

// ── Diff & Log ────────────────────────────────────────────────────────────────

func GitDiffStat() GitResult                { return runGit("diff", "--stat") }
func GitDiffStaged() GitResult              { return runGit("diff", "--cached", "--stat") }
func GitDiffFull() GitResult                { return runGit("diff") }
func GitDiffStagedFull() GitResult          { return runGit("diff", "--cached") }
func GitDiffBranch(branch string) GitResult { return runGit("diff", branch+"...HEAD", "--stat") }

func GitLog(n int) GitResult {
	return runGit("log", "--oneline", "--graph", "--decorate",
		fmt.Sprintf("--max-count=%d", n))
}

func GitLogFull(n int) GitResult {
	return runGit("log", "--pretty=format:%C(yellow)%h%Creset %C(blue)%an%Creset %C(green)%ar%Creset%n%s%n",
		fmt.Sprintf("--max-count=%d", n))
}

func GitBlame(path string) GitResult { return runGit("blame", "--date=short", path) }

// ── Remote Operations ─────────────────────────────────────────────────────────

func GitRemotes() GitResult                      { return runGit("remote", "-v") }
func GitRemoteList() GitResult                   { return runGit("remote") }
func GitRemoteAdd(name, url string) GitResult    { return runGit("remote", "add", name, url) }
func GitRemoteRemove(name string) GitResult      { return runGit("remote", "remove", name) }
func GitRemoteSetURL(name, url string) GitResult { return runGit("remote", "set-url", name, url) }
func GitFetch() GitResult                        { return runGit("fetch", "--all", "--prune") }
func GitFetchRemote(remote string) GitResult     { return runGit("fetch", remote, "--prune") }
func GitPull() GitResult                         { return runGit("pull", "--rebase=false") }
func GitPullRebase() GitResult                   { return runGit("pull", "--rebase") }
func GitPush(remote, branch string) GitResult    { return runGit("push", remote, branch) }
func GitPushSetUpstream(remote, branch string) GitResult {
	return runGit("push", "--set-upstream", remote, branch)
}
func GitPushForce(remote, branch string) GitResult {
	return runGit("push", "--force-with-lease", remote, branch)
}
func GitPushTags(remote string) GitResult { return runGit("push", remote, "--tags") }

func GitClone(url, dir string) GitResult {
	if dir == "" {
		return runGit("clone", url)
	}
	return runGit("clone", url, dir)
}

// ── Stash ─────────────────────────────────────────────────────────────────────

func GitStash(message string) GitResult {
	if message == "" {
		return runGit("stash", "push")
	}
	return runGit("stash", "push", "-m", message)
}
func GitStashList() GitResult { return runGit("stash", "list") }
func GitStashPop() GitResult  { return runGit("stash", "pop") }
func GitStashApply(ref string) GitResult {
	if ref == "" {
		return runGit("stash", "apply")
	}
	return runGit("stash", "apply", ref)
}
func GitStashDrop(ref string) GitResult {
	if ref == "" {
		return runGit("stash", "drop")
	}
	return runGit("stash", "drop", ref)
}

// ── Helper Functions ────────────────────────────────────────────────────────

func SanitizeBranchName(name string) string {
	// Replace spaces and invalid characters with hyphens
	sanitized := strings.ReplaceAll(name, " ", "-")
	sanitized = strings.ReplaceAll(sanitized, "..", "-")
	sanitized = strings.ReplaceAll(sanitized, "~", "-")
	sanitized = strings.ReplaceAll(sanitized, "^", "-")
	sanitized = strings.ReplaceAll(sanitized, ":", "-")
	sanitized = strings.ReplaceAll(sanitized, "?", "-")
	sanitized = strings.ReplaceAll(sanitized, "*", "-")
	sanitized = strings.ReplaceAll(sanitized, "[", "-")
	sanitized = strings.ReplaceAll(sanitized, "]", "-")
	// Remove consecutive hyphens
	for strings.Contains(sanitized, "--") {
		sanitized = strings.ReplaceAll(sanitized, "--", "-")
	}
	return sanitized
}

func ResolveBaseBranch() string {
	// Try to determine the base branch (main/master)
	if main := runGit("rev-parse", "--verify", "refs/heads/main"); main.OK {
		return "main"
	}
	if master := runGit("rev-parse", "--verify", "refs/heads/master"); master.OK {
		return "master"
	}
	return "main" // fallback
}

// ── Tags ──────────────────────────────────────────────────────────────────────

func GitTagList() GitResult                          { return runGit("tag", "--list", "--sort=-creatordate") }
func GitTagCreate(name string) GitResult             { return runGit("tag", name) }
func GitTagAnnotated(name, message string) GitResult { return runGit("tag", "-a", name, "-m", message) }
func GitTagDelete(name string) GitResult             { return runGit("tag", "-d", name) }

// ── Rebase & Reset ────────────────────────────────────────────────────────────

func GitRebase(branch string) GitResult    { return runGit("rebase", branch) }
func GitRebaseInteractive(n int) GitResult { return runGit("rebase", "-i", fmt.Sprintf("HEAD~%d", n)) }
func GitRebaseAbort() GitResult            { return runGit("rebase", "--abort") }
func GitRebaseContinue() GitResult         { return runGit("rebase", "--continue") }
func GitResetSoft(n int) GitResult         { return runGit("reset", "--soft", fmt.Sprintf("HEAD~%d", n)) }
func GitResetMixed(n int) GitResult        { return runGit("reset", "--mixed", fmt.Sprintf("HEAD~%d", n)) }
func GitResetHard(n int) GitResult         { return runGit("reset", "--hard", fmt.Sprintf("HEAD~%d", n)) }
func GitResetFile(path string) GitResult   { return runGit("restore", path) }

// ── Cherry-pick & Revert ──────────────────────────────────────────────────────

func GitCherryPick(hash string) GitResult { return runGit("cherry-pick", hash) }
func GitRevert(hash string) GitResult     { return runGit("revert", "--no-edit", hash) }

// ── Config ────────────────────────────────────────────────────────────────────

func GitConfigSetName(name string) GitResult { return runGit("config", "--global", "user.name", name) }
func GitConfigSetEmail(email string) GitResult {
	return runGit("config", "--global", "user.email", email)
}
func GitConfigSetLocal(key, val string) GitResult { return runGit("config", "--local", key, val) }
func GitConfigGet(key string) GitResult           { return runGit("config", key) }
func GitSetDefaultBranch(name string) GitResult {
	return runGit("config", "--global", "init.defaultBranch", name)
}

// ── Smart Helpers ─────────────────────────────────────────────────────────────

func GitLastMergedBranch() string {
	r := runGit("log", "--merges", "--oneline", "--max-count=1", "--pretty=format:%s")
	if !r.OK || r.Output == "" {
		return ""
	}
	s := r.Output
	if i := strings.Index(s, "'"); i >= 0 {
		s = s[i+1:]
		if j := strings.Index(s, "'"); j >= 0 {
			return s[:j]
		}
	}
	return ""
}

func GitHasUncommittedChanges() bool {
	r := runGit("status", "--porcelain")
	return r.OK && strings.TrimSpace(r.Output) != ""
}

func GitCountAhead() int {
	r := runGit("rev-list", "--count", "HEAD@{u}..HEAD")
	if !r.OK {
		return 0
	}
	n, _ := strconv.Atoi(strings.TrimSpace(r.Output))
	return n
}

func GitCountBehind() int {
	r := runGit("rev-list", "--count", "HEAD..HEAD@{u}")
	if !r.OK {
		return 0
	}
	n, _ := strconv.Atoi(strings.TrimSpace(r.Output))
	return n
}

func GitCurrentHash() string {
	r := runGit("rev-parse", "--short", "HEAD")
	if r.OK {
		return r.Output
	}
	return ""
}

func GitUpstreamURL() string {
	r := runGit("remote", "get-url", "origin")
	if r.OK {
		return strings.TrimSpace(r.Output)
	}
	return ""
}

// ── GitHub Setup ──────────────────────────────────────────────────────────────

// GitHubLink connects the current repo to a GitHub remote and pushes.
func GitHubLink(url string) GitResult {
	if r := runGit("remote"); r.OK {
		for _, rem := range strings.Fields(r.Output) {
			if rem == "origin" {
				runGit("remote", "remove", "origin")
				break
			}
		}
	}
	if r := GitRemoteAdd("origin", url); !r.OK {
		return r
	}
	branch := "main"
	if b := GitBranch(); b.OK {
		branch = b.Output
	}
	return GitPushSetUpstream("origin", branch)
}

// GitInitWithRemote initialises a repo, makes an initial commit, and pushes.
func GitInitWithRemote(dir, remoteURL, initialMsg string) GitResult {
	if r := runGitInDir(dir, "init"); !r.OK {
		return r
	}
	if r := runGitInDir(dir, "add", "--all"); !r.OK {
		return r
	}
	if r := runGitInDir(dir, "commit", "-m", initialMsg); !r.OK {
		return r
	}
	runGitInDir(dir, "remote", "add", "origin", remoteURL)
	return runGitInDir(dir, "push", "--set-upstream", "origin", "main")
}

// ── Path Utilities ────────────────────────────────────────────────────────────

// ResolveLocalPath expands ~ and cleans a file path with security validation.
func ResolveLocalPath(p string) (string, error) {
	// Input validation
	if err := validateInput(p); err != nil {
		return "", fmt.Errorf("invalid path: %v", err)
	}

	if strings.HasPrefix(p, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			p = filepath.Join(home, p[2:])
		}
	}

	cleaned := filepath.Clean(p)

	// Convert to absolute path for security checks
	absPath, err := filepath.Abs(cleaned)
	if err != nil {
		return "", fmt.Errorf("cannot resolve absolute path: %v", err)
	}

	// Check if we're in a git repo and enforce staying within repo bounds
	if IsGitRepo() {
		if rootResult := GitRootDir(); rootResult.OK {
			repoRoot, err := filepath.Abs(rootResult.Output)
			if err == nil && !strings.HasPrefix(absPath, repoRoot) {
				return "", fmt.Errorf("path outside repository bounds not allowed")
			}
		}
	}

	return cleaned, nil
}

// ── Internal Helpers ──────────────────────────────────────────────────────────

func resolveBaseBranch() string {
	for _, b := range []string{"main", "master"} {
		if GitBranchExists(b) {
			return b
		}
	}
	return "main"
}

func itoa(n int) string { return strconv.Itoa(n) }

// sanitizeBranchName converts text into a valid git branch name.
func sanitizeBranchName(name string) string {
	var result strings.Builder
	for _, r := range strings.ToLower(name) {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-':
			result.WriteRune(r)
		case r == ' ' || r == '_':
			result.WriteRune('-')
		}
	}
	s := strings.Trim(result.String(), "-")
	for strings.Contains(s, "--") {
		s = strings.ReplaceAll(s, "--", "-")
	}
	return s
}
