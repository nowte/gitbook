package git

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// ── Snapshot System ───────────────────────────────────────────────────────────
//
// Before every destructive operation (reset --hard, rebase, merge, clean) a
// lightweight JSON snapshot is written to .gitbook/snapshots/.  The /undo
// command restores the most recent snapshot.
//
// Storage layout:
//
//	.gitbook/snapshots/<unix-nano>.json   (max 10 files; oldest auto-pruned)
//
// ─────────────────────────────────────────────────────────────────────────────

const (
	snapshotDir    = ".gitbook/snapshots"
	maxSnapshots   = 10
)

// Snapshot captures the repository state before a destructive operation.
type Snapshot struct {
	Timestamp   time.Time `json:"timestamp"`
	CommitHash  string    `json:"commit_hash"`
	Branch      string    `json:"branch"`
	Operation   string    `json:"operation"`   // human-readable trigger ("reset-hard", "rebase", …)
	StagedFiles []string  `json:"staged_files"`
}

// TakeSnapshot records the current HEAD state and returns the snapshot.
// Returns an error only if the write fails; a missing HEAD (empty repo) is
// silently skipped and returns (nil, nil).
func TakeSnapshot(operation string) (*Snapshot, error) {
	// Require a git repo
	if !IsGitRepo() {
		return nil, nil
	}

	// Collect state — best-effort; partial data is still useful
	hash := ""
	if r := runGit("rev-parse", "HEAD"); r.OK {
		hash = strings.TrimSpace(r.Output)
	}
	if hash == "" {
		// Empty repo — nothing to snapshot
		return nil, nil
	}

	branch := ""
	if r := GitBranch(); r.OK {
		branch = r.Output
	}

	staged := []string{}
	if r := runGit("diff", "--cached", "--name-only"); r.OK && r.Output != "" {
		for _, f := range strings.Split(r.Output, "\n") {
			if f = strings.TrimSpace(f); f != "" {
				staged = append(staged, f)
			}
		}
	}

	snap := &Snapshot{
		Timestamp:   time.Now().UTC(),
		CommitHash:  hash,
		Branch:      branch,
		Operation:   operation,
		StagedFiles: staged,
	}

	if err := writeSnapshot(snap); err != nil {
		return snap, err
	}
	return snap, nil
}

// LatestSnapshot returns the most recently written snapshot, or nil if none.
func LatestSnapshot() (*Snapshot, error) {
	files, err := listSnapshotFiles()
	if err != nil || len(files) == 0 {
		return nil, err
	}
	return readSnapshot(files[len(files)-1])
}

// DeleteSnapshot removes the snapshot file that matches the given timestamp.
func DeleteSnapshot(ts time.Time) error {
	files, err := listSnapshotFiles()
	if err != nil {
		return err
	}
	target := fmt.Sprintf("%d.json", ts.UnixNano())
	for _, f := range files {
		if filepath.Base(f) == target {
			return os.Remove(f)
		}
	}
	return nil
}

// ListSnapshots returns all stored snapshots ordered oldest → newest.
func ListSnapshots() ([]*Snapshot, error) {
	files, err := listSnapshotFiles()
	if err != nil {
		return nil, err
	}
	snaps := make([]*Snapshot, 0, len(files))
	for _, f := range files {
		s, err := readSnapshot(f)
		if err != nil {
			continue // skip corrupt entries
		}
		snaps = append(snaps, s)
	}
	return snaps, nil
}

// ── Restore ───────────────────────────────────────────────────────────────────

// RestoreSnapshot hard-resets the working tree to the commit captured in snap.
// It also restores the branch if we are currently on a different one.
func RestoreSnapshot(snap *Snapshot) GitResult {
	// Switch branch first if needed
	if snap.Branch != "" {
		cur := ""
		if b := GitBranch(); b.OK {
			cur = b.Output
		}
		if cur != snap.Branch {
			if r := runGit("checkout", snap.Branch); !r.OK {
				return r
			}
		}
	}

	// Hard-reset to the snapshotted commit
	return runGit("reset", "--hard", snap.CommitHash)
}

// ── Internal helpers ──────────────────────────────────────────────────────────

func writeSnapshot(snap *Snapshot) error {
	if err := os.MkdirAll(snapshotDir, 0755); err != nil {
		return fmt.Errorf("cannot create snapshot dir: %w", err)
	}

	data, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return err
	}

	name := fmt.Sprintf("%d.json", snap.Timestamp.UnixNano())
	path := filepath.Join(snapshotDir, name)

	// Atomic write
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		return err
	}

	return pruneSnapshots()
}

// pruneSnapshots removes the oldest files when we exceed maxSnapshots.
func pruneSnapshots() error {
	files, err := listSnapshotFiles()
	if err != nil {
		return err
	}
	for len(files) > maxSnapshots {
		if err := os.Remove(files[0]); err != nil {
			return err
		}
		files = files[1:]
	}
	return nil
}

// listSnapshotFiles returns snapshot paths sorted oldest → newest.
func listSnapshotFiles() ([]string, error) {
	entries, err := os.ReadDir(snapshotDir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var paths []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".json") &&
			!strings.HasSuffix(e.Name(), ".tmp") {
			paths = append(paths, filepath.Join(snapshotDir, e.Name()))
		}
	}
	sort.Strings(paths) // unix-nano names sort chronologically
	return paths, nil
}

func readSnapshot(path string) (*Snapshot, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var s Snapshot
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	return &s, nil
}
