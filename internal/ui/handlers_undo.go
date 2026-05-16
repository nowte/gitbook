package ui

import (
	"fmt"
	"strings"

	"github.com/nowte/gitbook/internal/git"
	"github.com/nowte/gitbook/internal/lang"
)

// ── /undo handler ─────────────────────────────────────────────────────────────
//
// Restores the repository to the state captured in the most recent snapshot.
// Shows the snapshot details and asks for confirmation before proceeding.
// ─────────────────────────────────────────────────────────────────────────────

func (m *Model) handleUndo(args []string) {
	if !git.IsGitRepo() {
		m.bad(lang.T("msg_not_git_repo"))
		return
	}

	snap, err := git.LatestSnapshot()
	if err != nil {
		m.bad(lang.Tf("msg_undo_load_error", err.Error()))
		return
	}
	if snap == nil {
		m.warn(lang.T("msg_undo_no_snapshots"))
		return
	}

	// Show what we are about to restore
	m.warn(lang.T("msg_undo_about_to_restore"))
	m.gray(fmt.Sprintf("  %s  %s → %s  (%s)",
		snap.Timestamp.Format("2006-01-02 15:04:05"),
		snap.Operation,
		snap.CommitHash[:min(8, len(snap.CommitHash))],
		snap.Branch,
	))
	if len(snap.StagedFiles) > 0 {
		m.gray(lang.Tf("msg_undo_staged_files", strings.Join(snap.StagedFiles, ", ")))
	}

	snapCopy := snap
	m.beginWizard(&command{
		name: "/__confirm_undo",
		fields: []wizardField{
			{label: lang.T("msg_please_confirm_cancel"), placeholder: "confirm / cancel", required: true},
		},
		handler: func(m *Model, a []string) {
			if len(a) == 0 || a[0] != "confirm" {
				m.info(lang.T("msg_undo_cancelled"))
				return
			}

			r := git.RestoreSnapshot(snapCopy)
			if !r.OK {
				m.bad(lang.Tf("msg_undo_failed", r.Err))
				return
			}

			// Remove the consumed snapshot
			_ = git.DeleteSnapshot(snapCopy.Timestamp)
			m.ok(lang.Tf("msg_undo_done", snapCopy.CommitHash[:min(8, len(snapCopy.CommitHash))]))
		},
	})
}

// ── /snapshots handler ───────────────────────────────────────────────────────

func (m *Model) handleSnapshots(args []string) {
	if !git.IsGitRepo() {
		m.bad(lang.T("msg_not_git_repo"))
		return
	}

	snaps, err := git.ListSnapshots()
	if err != nil {
		m.bad(lang.Tf("msg_undo_load_error", err.Error()))
		return
	}
	if len(snaps) == 0 {
		m.info(lang.T("msg_undo_no_snapshots"))
		return
	}

	var sb strings.Builder
	sb.WriteString(lang.T("msg_snapshots_title") + "\n")
	for i, s := range snaps {
		marker := "  "
		if i == len(snaps)-1 {
			marker = "* " // most recent
		}
		sb.WriteString(fmt.Sprintf("%s[%d] %s  %-14s  %s  (%s)\n",
			marker,
			i+1,
			s.Timestamp.Format("01-02 15:04:05"),
			s.Operation,
			s.CommitHash[:min(8, len(s.CommitHash))],
			s.Branch,
		))
	}
	sb.WriteString("\n" + lang.T("msg_snapshots_hint"))
	m.info(sb.String())
}

// ── min helper (Go < 1.21 compatibility) ────────────────────────────────────

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
