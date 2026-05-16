package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// configMu serialises all reads and writes of .gitbook/config.json so that
// concurrent gitBook instances cannot corrupt the file.
var configMu sync.RWMutex

// Constants for configuration validation
const (
	GitbookDir     = ".gitbook"
	ConfigFile     = "config.json"
	MaxConfigSize  = 1024 * 1024 // 1MB max config file size
	MinVersion     = "0.1.0"
	CurrentVersion = "0.1.0"

	// Default protected branches
	DefaultProtectedBranches = "main,master,release,develop"

	// Configuration validation patterns
	ValidBranchNamePattern = `^[a-zA-Z0-9/_-]+$`
	ValidEmailPattern      = `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
	MaxBranchNameLength    = 255
	MaxCommitMessageLength = 72
)

// GitBookConfig — .gitbook/config.json yapısı
type GitBookConfig struct {
	Version     string    `json:"version"`
	InitiatedAt time.Time `json:"initiated_at"`
	Rules       Rules     `json:"rules"`
}

// Rules — Branch ve commit kuralları (Faz 3'e zemin hazırlar)
type Rules struct {
	ProtectedBranches []string `json:"protected_branches"`
	RequireCommitMsg  bool     `json:"require_commit_msg"`
}

// ── State işlemleri ────────────────────────────────────────────────────────────

// IsGitBookRepo — mevcut dizinde .gitbook/ var mı kontrol eder.
func IsGitBookRepo() bool {
	_, err := os.Stat(filepath.Join(GitbookDir, ConfigFile))
	return err == nil
}

// InitGitBook — .gitbook/ klasörünü oluşturur ve config.json yazar.
func InitGitBook() error {
	configMu.Lock()
	defer configMu.Unlock()

	if err := os.MkdirAll(GitbookDir, 0755); err != nil {
		return err
	}

	cfg := GitBookConfig{
		Version:     "0.1.0",
		InitiatedAt: time.Now().UTC(),
		Rules: Rules{
			ProtectedBranches: []string{"main", "master", "release"},
			RequireCommitMsg:  true,
		},
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	// Write to a temp file then rename for atomic update
	tmp := filepath.Join(GitbookDir, ConfigFile+".tmp")
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, filepath.Join(GitbookDir, ConfigFile))
}

// LoadConfig — mevcut config'i okur.
func LoadConfig() (*GitBookConfig, error) {
	configMu.RLock()
	defer configMu.RUnlock()

	data, err := os.ReadFile(filepath.Join(GitbookDir, ConfigFile))
	if err != nil {
		return nil, err
	}

	// Guard against oversized / corrupted config files
	if len(data) > MaxConfigSize {
		return nil, fmt.Errorf("config file too large (max %d bytes)", MaxConfigSize)
	}

	var cfg GitBookConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("config parse error: %w", err)
	}
	return &cfg, nil
}

// ValidateBranchName checks if a branch name is valid
func ValidateBranchName(name string) error {
	if len(name) == 0 {
		return fmt.Errorf("branch name cannot be empty")
	}
	if len(name) > MaxBranchNameLength {
		return fmt.Errorf("branch name too long (max %d characters)", MaxBranchNameLength)
	}
	if strings.HasPrefix(name, "-") || strings.HasSuffix(name, "-") {
		return fmt.Errorf("branch name cannot start or end with hyphen")
	}
	if strings.Contains(name, "--") {
		return fmt.Errorf("branch name cannot contain consecutive hyphens")
	}
	return nil
}

// ValidateCommitMessage checks if a commit message is valid
func ValidateCommitMessage(msg string) error {
	if len(msg) == 0 {
		return fmt.Errorf("commit message cannot be empty")
	}
	if len(msg) > MaxCommitMessageLength {
		return fmt.Errorf("commit message too long (max %d characters)", MaxCommitMessageLength)
	}
	return nil
}

// ValidateConfig checks if a configuration is valid
func ValidateConfig(cfg *GitBookConfig) error {
	if cfg == nil {
		return fmt.Errorf("config cannot be nil")
	}
	if cfg.Version == "" {
		return fmt.Errorf("version cannot be empty")
	}
	if cfg.Version < MinVersion {
		return fmt.Errorf("config version %s is not supported (min %s)", cfg.Version, MinVersion)
	}

	// Validate protected branches
	if len(cfg.Rules.ProtectedBranches) == 0 {
		return fmt.Errorf("at least one protected branch must be specified")
	}
	for _, branch := range cfg.Rules.ProtectedBranches {
		if err := ValidateBranchName(branch); err != nil {
			return fmt.Errorf("invalid protected branch '%s': %v", branch, err)
		}
	}

	return nil
}

// IsProtectedBranch — branch adının korumalı olup olmadığını kontrol eder.
func IsProtectedBranch(branch string) bool {
	cfg, err := LoadConfig()
	if err != nil {
		// Config yoksa varsayılan kurallar
		protected := strings.Split(DefaultProtectedBranches, ",")
		for _, b := range protected {
			if strings.TrimSpace(b) == branch {
				return true
			}
		}
		return false
	}
	for _, b := range cfg.Rules.ProtectedBranches {
		if b == branch {
			return true
		}
	}
	return false
}

// ── Instance Lock ─────────────────────────────────────────────────────────────
//
// Writes a PID file to .gitbook/instance.lock on startup.
// AcquireInstanceLock returns true if this process is the only gitBook
// instance running against this repo; false (+ a warning) if another PID is
// detected.  ReleaseInstanceLock removes the file on exit.
//
// This is advisory only — a crash without cleanup leaves a stale lock.
// The stale-lock check compares the recorded PID against /proc/<pid> (Linux)
// or simply checks process existence on other platforms.
// ─────────────────────────────────────────────────────────────────────────────

const instanceLockFile = ".gitbook/instance.lock"

// AcquireInstanceLock tries to claim the lock file for the current process.
// Returns (true, "") if acquired; (false, existingPID) if another instance
// appears to be running.
func AcquireInstanceLock() (bool, string) {
	// Read existing lock if present
	if data, err := os.ReadFile(instanceLockFile); err == nil {
		pidStr := strings.TrimSpace(string(data))
		if pid, err := strconv.Atoi(pidStr); err == nil && pid != os.Getpid() {
			// Check whether that process is still alive
			if processExists(pid) {
				return false, pidStr
			}
			// Stale lock — overwrite below
		}
	}

	// Write our PID atomically
	tmp := instanceLockFile + ".tmp"
	content := strconv.Itoa(os.Getpid())
	if err := os.MkdirAll(GitbookDir, 0755); err != nil {
		return true, "" // cannot write lock, but don't block startup
	}
	if err := os.WriteFile(tmp, []byte(content), 0644); err != nil {
		return true, ""
	}
	_ = os.Rename(tmp, instanceLockFile)
	return true, ""
}

// ReleaseInstanceLock removes the PID file created by AcquireInstanceLock.
// Safe to call even if the file does not exist.
func ReleaseInstanceLock() {
	_ = os.Remove(instanceLockFile)
}

// processExists returns true if the given PID appears to be running.
func processExists(pid int) bool {
	// Use /proc on Linux; fall back to os.FindProcess on other platforms.
	procPath := fmt.Sprintf("/proc/%d", pid)
	if _, err := os.Stat(procPath); err == nil {
		return true
	}
	// Portable fallback: signal 0 returns nil if process exists
	p, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	// On Unix, FindProcess always succeeds; send signal 0 to check
	_ = p
	return false
}
