package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/nowte/gitbook/internal/config"
	"github.com/nowte/gitbook/internal/git"
	"github.com/nowte/gitbook/internal/lang"
)

// ── Command Handlers ─────────────────────────────────────────────────────────────

func (m *Model) handleInit(args []string) {
	if git.IsGitRepo() {
		m.warn(lang.T("msg_already_git_repo"))
		return
	}
	r := git.GitInit()
	if !r.OK {
		m.bad(lang.Tf("git_init_failed", r.Err))
		return
	}
	if err := config.InitGitBook(); err != nil {
		m.warn(lang.Tf("gitbook_config_error", err.Error()))
		return
	}
	m.ok(lang.T("msg_repository_initialised"))
}

func (m *Model) handleStatus(args []string) {
	if !git.IsGitRepo() {
		m.bad(lang.T("msg_not_git_repo"))
		return
	}
	var sb strings.Builder
	if b := git.GitBranch(); b.OK {
		label := b.Output
		if config.IsProtectedBranch(label) {
			label = "[!] " + label + " " + lang.T("status_protected")
		}
		sb.WriteString(lang.T("status_branch_label") + " " + label + "\n")
	}
	if config.IsGitBookRepo() {
		sb.WriteString(lang.T("msg_gitbook_initialised") + "\n")
	}
	ahead, behind := git.GitCountAhead(), git.GitCountBehind()
	if ahead > 0 || behind > 0 {
		sb.WriteString(lang.Tf("msg_sync_status", ahead, behind) + "\n")
	}
	s := git.GitStatus()
	if s.OK && s.Output != "" {
		sb.WriteString("\n" + lang.T("status_changed_files") + "\n" + s.Output)
	} else if s.OK {
		sb.WriteString("\n" + lang.T("msg_working_tree_clean"))
	}
	hash := git.GitCurrentHash()
	if hash != "" {
		sb.WriteString("\n" + lang.T("status_head_label") + " " + hash)
	}
	m.info(sb.String())
}

func (m *Model) handleBranch(args []string) {
	if !git.IsGitRepo() {
		m.bad(lang.T("msg_not_git_repo"))
		return
	}
	r := git.GitBranchList()
	cur := ""
	if b := git.GitBranch(); b.OK {
		cur = b.Output
	}
	if !r.OK {
		m.bad(lang.Tf("msg_cannot_list_branches", r.Err))
		return
	}
	var sb strings.Builder
	sb.WriteString(lang.T("git_local_branches") + "\n")
	for _, b := range strings.Split(r.Output, "\n") {
		b = strings.TrimSpace(b)
		if b == "" {
			continue
		}
		marker := "  "
		if b == cur {
			marker = "* "
		}
		sb.WriteString(marker + b + "\n")
	}
	m.info(sb.String())
}

func (m *Model) handleConfig(args []string) {
	if !git.IsGitRepo() {
		m.bad(lang.T("msg_not_git_repo"))
		return
	}
	var sb strings.Builder
	if name, email := git.GitGetIdentity(); name != "" || email != "" {
		sb.WriteString(lang.Tf("info_user", name, email))
	} else {
		sb.WriteString(lang.T("info_identity_not_configured") + "\n")
	}
	if repoName := git.GitRepoName(); repoName != "" {
		sb.WriteString(lang.Tf("info_repository", repoName))
	}
	if upstream := git.GitUpstreamURL(); upstream != "" {
		sb.WriteString(lang.Tf("info_remote", upstream))
	}
	m.info(sb.String())
}

func (m *Model) handleLanguage(args []string) {
	if len(args) == 0 {
		currentLang := lang.GetGlobalLang().GetCurrentLanguage()
		availableLangs := lang.GetAvailableLanguages()
		var sb strings.Builder
		sb.WriteString(lang.Tf("msg_current_language", currentLang))
		sb.WriteString(lang.T("msg_available_languages") + "\n")
		for _, l := range availableLangs {
			if l == currentLang {
				sb.WriteString(fmt.Sprintf("  * %s %s\n", l, lang.T("msg_language_current")))
			} else {
				sb.WriteString(fmt.Sprintf("    %s\n", l))
			}
		}
		sb.WriteString("\n" + lang.T("usage_language") + "\n")
		sb.WriteString(lang.T("usage_language_examples"))
		m.info(sb.String())
		return
	}
	langCode := args[0]
	if err := lang.GetGlobalLang().SetLanguage(langCode); err != nil {
		m.bad(lang.Tf("msg_language_change_failed", err))
		return
	}
	newLang := lang.GetGlobalLang().GetCurrentLanguage()
	m.ok(lang.Tf("msg_language_changed", newLang))
}

func (m *Model) handleInfo(args []string) {
	var sb strings.Builder
	sb.WriteString("[GB] gitBook\n")
	sb.WriteString(lang.Tf("info_version", GitbookVersion))
	if m.latestVersion != "" && UpdateAvailable(m.latestVersion) {
		sb.WriteString(lang.Tf("info_update_available", GitbookVersion, m.latestVersion))
	} else {
		sb.WriteString(lang.T("info_up_to_date") + "\n")
	}
	sb.WriteString("\n[L] " + lang.T("info_links_header") + "\n")
	sb.WriteString("GitHub: https://github.com/nowte/gitbook\n")
	sb.WriteString(lang.T("info_developer") + "\n")
	if git.IsGitRepo() {
		sb.WriteString("\n[D] " + lang.T("info_repo_header") + "\n")
		if name, email := git.GitGetIdentity(); name != "" || email != "" {
			sb.WriteString(lang.Tf("info_user", name, email))
		}
		if repoName := git.GitRepoName(); repoName != "" {
			sb.WriteString(lang.Tf("info_repository", repoName))
		}
		if upstream := git.GitUpstreamURL(); upstream != "" {
			sb.WriteString(lang.Tf("info_remote", upstream))
		}
		if branch := git.GitBranch(); branch.OK {
			sb.WriteString(lang.Tf("info_branch", branch.Output))
		}
	}
	m.info(sb.String())
}

func (m *Model) handleLog(args []string) {
	if !git.IsGitRepo() {
		m.bad(lang.T("msg_not_git_repo"))
		return
	}
	n := 10
	if len(args) > 0 {
		if parsed, err := strconv.Atoi(args[0]); err == nil && parsed > 0 {
			n = parsed
		}
	}
	r := git.GitLog(n)
	m.gitResultLine(r, "")
}

func (m *Model) handleDiff(args []string) {
	if !git.IsGitRepo() {
		m.bad(lang.T("msg_not_git_repo"))
		return
	}
	if len(args) == 0 {
		r := git.GitDiffFull()
		m.gitResultLine(r, lang.T("msg_diff_unstaged"))
	} else {
		r := git.GitDiffBranch(args[0])
		m.gitResultLine(r, lang.Tf("msg_diff_vs_branch", args[0]))
	}
}

func (m *Model) handleBlame(args []string) {
	if !git.IsGitRepo() {
		m.bad(lang.T("msg_not_git_repo"))
		return
	}
	if len(args) == 0 {
		m.bad(lang.T("usage_blame"))
		return
	}
	resolved, err := git.ResolveLocalPath(args[0])
	if err != nil {
		m.bad(lang.Tf("msg_invalid_path", args[0], err))
		return
	}
	r := git.GitBlame(resolved)
	m.gitResultLine(r, lang.Tf("msg_blame_for", args[0]))
}

func (m *Model) handleRemote(args []string) {
	if !git.IsGitRepo() {
		m.bad(lang.T("msg_not_git_repo"))
		return
	}
	r := git.GitRemotes()
	m.gitResultLine(r, lang.T("git_configured_remotes"))
}

func (m *Model) handlePush(args []string) {
	if !git.IsGitRepo() {
		m.bad(lang.T("msg_not_git_repo"))
		return
	}
	remote := "origin"
	if len(args) > 0 {
		remote = args[0]
	}
	branch := ""
	if b := git.GitBranch(); b.OK {
		branch = b.Output
	}
	if branch == "" {
		m.bad(lang.T("msg_cannot_determine_branch"))
		return
	}
	r := git.GitPush(remote, branch)
	m.gitResultLine(r, lang.Tf("git_push_completed", remote, branch))
}

func (m *Model) handlePull(args []string) {
	if !git.IsGitRepo() {
		m.bad(lang.T("msg_not_git_repo"))
		return
	}
	r := git.GitPull()
	m.gitResultLine(r, lang.T("git_pull_completed"))
}

func (m *Model) handleFetch(args []string) {
	if !git.IsGitRepo() {
		m.bad(lang.T("msg_not_git_repo"))
		return
	}
	r := git.GitFetch()
	m.gitResultLine(r, lang.T("git_fetch_completed"))
}

func (m *Model) handleClone(args []string) {
	if len(args) == 0 {
		m.bad(lang.T("usage_clone"))
		return
	}
	url := args[0]
	dir := ""
	if len(args) > 1 {
		dir = args[1]
	}
	r := git.GitClone(url, dir)
	m.gitResultLine(r, lang.T("git_clone_completed"))
}

func (m *Model) handleSync(args []string) {
	if !git.IsGitRepo() {
		m.bad(lang.T("msg_not_git_repo"))
		return
	}
	if r := git.GitFetch(); !r.OK {
		m.bad(lang.Tf("msg_fetch_failed", r.Err))
		return
	}
	ahead, behind := git.GitCountAhead(), git.GitCountBehind()
	if ahead > 0 || behind > 0 {
		m.info(lang.Tf("msg_sync_status", ahead, behind))
	} else {
		m.info(lang.T("msg_up_to_date"))
	}
}

func (m *Model) handleGitHub(args []string) {
	if len(args) == 0 {
		m.bad(lang.T("usage_github"))
		return
	}
	if !git.IsGitRepo() {
		m.bad(lang.T("msg_not_git_repo"))
		return
	}
	r := git.GitHubLink(args[0])
	m.gitResultLine(r, lang.T("github_setup_completed"))
}

func (m *Model) handleStash(args []string) {
	if !git.IsGitRepo() {
		m.bad(lang.T("msg_not_git_repo"))
		return
	}
	message := ""
	if len(args) > 0 {
		message = strings.Join(args, " ")
	}
	r := git.GitStash(message)
	m.gitResultLine(r, lang.T("git_stashed_changes"))
}

func (m *Model) handleStashList(args []string) {
	if !git.IsGitRepo() {
		m.bad(lang.T("msg_not_git_repo"))
		return
	}
	r := git.GitStashList()
	m.gitResultLine(r, lang.T("git_stash_list"))
}

func (m *Model) handleStashPop(args []string) {
	if !git.IsGitRepo() {
		m.bad(lang.T("msg_not_git_repo"))
		return
	}
	r := git.GitStashPop()
	m.gitResultLine(r, lang.T("git_applied_stash"))
}

func (m *Model) handleTag(args []string) {
	if !git.IsGitRepo() {
		m.bad(lang.T("msg_not_git_repo"))
		return
	}
	if len(args) == 0 {
		r := git.GitTagList()
		m.gitResultLine(r, lang.T("git_tags"))
	} else {
		r := git.GitTagCreate(args[0])
		m.gitResultLine(r, lang.Tf("git_created_tag", args[0]))
	}
}

func (m *Model) handleTagPush(args []string) {
	if !git.IsGitRepo() {
		m.bad(lang.T("msg_not_git_repo"))
		return
	}
	remote := "origin"
	if len(args) > 0 {
		remote = args[0]
	}
	r := git.GitPushTags(remote)
	m.gitResultLine(r, lang.Tf("git_pushed_tags", remote))
}

func (m *Model) handleReset(args []string) {
	if !git.IsGitRepo() {
		m.bad(lang.T("msg_not_git_repo"))
		return
	}
	n := 1
	if len(args) > 0 {
		if parsed, err := strconv.Atoi(args[0]); err == nil && parsed > 0 {
			n = parsed
		}
	}
	r := git.GitResetSoft(n)
	m.gitResultLine(r, lang.Tf("msg_soft_reset_completed", n))
}

func (m *Model) handleResetHard(args []string) {
	if !git.IsGitRepo() {
		m.bad(lang.T("msg_not_git_repo"))
		return
	}
	n := "1"
	if len(args) > 0 {
		n = args[0]
	}
	m.setupContext = n
	m.warn(lang.Tf("msg_about_to_hard_reset", n))
	m.beginWizard(&command{
		name: "/__confirm_reset_hard",
		fields: []wizardField{
			{label: lang.T("msg_please_confirm_cancel"), placeholder: "confirm / cancel", required: true},
		},
		handler: func(m *Model, a []string) {
			if len(a) > 0 && a[0] == "confirm" {
				nInt := 1
				if parsed, err := strconv.Atoi(m.setupContext); err == nil && parsed > 0 {
					nInt = parsed
				}
				// Snapshot before destructive operation
				if _, err := git.TakeSnapshot("reset-hard"); err != nil {
					m.gray(lang.Tf("msg_snapshot_warn", err.Error()))
				} else {
					m.gray(lang.T("msg_snapshot_taken"))
				}
				r := git.GitResetHard(nInt)
				m.gitResultLine(r, lang.Tf("msg_hard_reset_completed", nInt))
			} else {
				m.info(lang.T("msg_hard_reset_cancelled"))
			}
			m.setupContext = ""
		},
	})
}

func (m *Model) handleRebase(args []string) {
	if !git.IsGitRepo() {
		m.bad(lang.T("msg_not_git_repo"))
		return
	}
	if len(args) == 0 {
		m.bad(lang.T("usage_rebase"))
		return
	}
	target := args[0]
	if git.GitHasUncommittedChanges() {
		m.warn(lang.T("msg_rebase_uncommitted"))
		return
	}
	cur := ""
	if b := git.GitBranch(); b.OK {
		cur = b.Output
	}
	m.gray(lang.Tf("msg_rebase_onto", cur, target))
	// Snapshot before rebase
	if _, err := git.TakeSnapshot("rebase"); err != nil {
		m.gray(lang.Tf("msg_snapshot_warn", err.Error()))
	} else {
		m.gray(lang.T("msg_snapshot_taken"))
	}
	r := git.GitRebase(target)
	m.gitResultLine(r, lang.Tf("msg_rebase_done", cur, target))
}

func (m *Model) handleRevert(args []string) {
	if !git.IsGitRepo() {
		m.bad(lang.T("msg_not_git_repo"))
		return
	}
	if len(args) == 0 {
		m.bad(lang.T("usage_revert"))
		return
	}
	r := git.GitRevert(args[0])
	m.gitResultLine(r, lang.Tf("msg_reverted_commit", args[0]))
}

func (m *Model) handleCherryPick(args []string) {
	if !git.IsGitRepo() {
		m.bad(lang.T("msg_not_git_repo"))
		return
	}
	if len(args) == 0 {
		m.bad(lang.T("usage_cherry_pick"))
		return
	}
	r := git.GitCherryPick(args[0])
	m.gitResultLine(r, lang.Tf("msg_cherry_picked_commit", args[0]))
}

// ── Additional Handlers ─────────────────────────────────────────────────────

func (m *Model) handleCd(args []string) {
	if len(args) == 0 {
		m.bad(lang.T("usage_cd"))
		return
	}
	resolved, err := git.ResolveLocalPath(args[0])
	if err != nil {
		m.bad(lang.Tf("msg_invalid_path", args[0], err))
		return
	}
	if err := os.Chdir(resolved); err != nil {
		m.bad(lang.Tf("msg_cannot_change_to", resolved, err))
		return
	}
	cwd, _ := os.Getwd()
	m.ok(lang.Tf("msg_working_directory", cwd))
}

func (m *Model) handlePath(args []string) {
	m.enterFileBrowserMode()
}

func (m *Model) handleStart(args []string) {
	if !git.IsGitRepo() {
		m.bad(lang.T("msg_not_git_repo"))
		return
	}
	if len(args) == 0 {
		m.bad(lang.T("usage_start"))
		return
	}
	if !git.IsGitIdentitySet() {
		m.bad(lang.T("msg_git_identity_not_set"))
		return
	}
	branchName := "feature/" + git.SanitizeBranchName(args[0])
	if git.GitBranchExists(branchName) {
		m.warn(lang.Tf("msg_branch_already_exists", branchName))
		m.gitResultLine(git.GitCheckout(branchName), lang.Tf("msg_switched_to", branchName))
		return
	}
	base := git.ResolveBaseBranch()
	if r := git.GitCheckout(base); !r.OK {
		m.bad(lang.Tf("msg_cannot_switch_to_base", base, r.Err))
		return
	}
	r := git.GitCheckoutNewBranch(branchName)
	m.gitResultLine(r, lang.Tf("msg_started_branch", branchName))
}

func (m *Model) handleFinish(args []string) {
	if !git.IsGitRepo() {
		m.bad(lang.T("msg_not_git_repo"))
		return
	}
	cur := ""
	if b := git.GitBranch(); b.OK {
		cur = b.Output
	}
	if !strings.HasPrefix(cur, "feature/") {
		m.bad(lang.T("msg_not_on_feature_branch"))
		return
	}
	if git.GitHasUncommittedChanges() {
		m.warn(lang.T("msg_has_uncommitted_changes"))
		return
	}
	base := git.ResolveBaseBranch()
	if r := git.GitCheckout(base); !r.OK {
		m.bad(lang.Tf("msg_cannot_switch_to_base", base, r.Err))
		return
	}
	// Snapshot before merge
	if _, err := git.TakeSnapshot("merge"); err != nil {
		m.gray(lang.Tf("msg_snapshot_warn", err.Error()))
	} else {
		m.gray(lang.T("msg_snapshot_taken"))
	}
	r := git.GitMerge(cur)
	m.gitResultLine(r, lang.Tf("msg_merged", cur, base))
}

func (m *Model) handleCleanup(args []string) {
	if !git.IsGitRepo() {
		m.bad(lang.T("msg_not_git_repo"))
		return
	}
	target := ""
	if len(args) > 0 {
		target = args[0]
	} else {
		target = git.GitLastMergedBranch()
	}
	if target == "" {
		m.bad(lang.T("msg_no_merged_branch"))
		return
	}
	if config.IsProtectedBranch(target) {
		m.bad(lang.Tf("msg_protected_branch", target))
		return
	}
	m.setupContext = target
	m.warn(lang.Tf("msg_about_to_delete_branch", target))
	m.beginWizard(&command{
		name: "/__confirm_delete_branch",
		fields: []wizardField{
			{label: lang.T("msg_please_confirm_cancel"), placeholder: "confirm / cancel", required: true},
		},
		handler: func(m *Model, a []string) {
			if len(a) > 0 && a[0] == "confirm" {
				r := git.GitDeleteBranch(m.setupContext)
				m.gitResultLine(r, lang.Tf("msg_branch_deleted", m.setupContext))
			} else {
				m.info(lang.T("msg_branch_deletion_cancelled"))
			}
			m.setupContext = ""
		},
	})
}

func (m *Model) handleStage(args []string) {
	if !git.IsGitRepo() {
		m.bad(lang.T("msg_not_git_repo"))
		return
	}
	path := "."
	if len(args) > 0 {
		resolved, err := git.ResolveLocalPath(args[0])
		if err != nil {
			m.bad(lang.Tf("msg_invalid_path", args[0], err))
			return
		}
		path = resolved
	}
	r := git.GitAdd(path)
	if !r.OK {
		m.bad(lang.Tf("msg_stage_failed", r.Err))
		return
	}
	staged := git.GitDiffStaged()
	if staged.OK && staged.Output != "" {
		m.ok(lang.Tf("msg_staged", staged.Output))
	} else {
		m.ok(lang.T("msg_staged_files"))
	}
}

func (m *Model) handleUnstage(args []string) {
	if !git.IsGitRepo() {
		m.bad(lang.T("msg_not_git_repo"))
		return
	}
	if len(args) == 0 {
		m.bad(lang.T("usage_unstage"))
		return
	}
	resolved, err := git.ResolveLocalPath(args[0])
	if err != nil {
		m.bad(lang.Tf("msg_invalid_path", args[0], err))
		return
	}
	m.gitResultLine(git.GitUnstage(resolved), lang.Tf("msg_unstaged", args[0]))
}

func (m *Model) handleCommit(args []string) {
	if !git.IsGitRepo() {
		m.bad(lang.T("msg_not_git_repo"))
		return
	}
	if len(args) == 0 {
		m.bad(lang.T("usage_commit"))
		return
	}
	r := git.GitCommit(args[0])
	m.gitResultLine(r, lang.Tf("msg_committed", args[0]))
}

func (m *Model) handleAmend(args []string) {
	if !git.IsGitRepo() {
		m.bad(lang.T("msg_not_git_repo"))
		return
	}
	if len(args) == 0 {
		m.bad(lang.T("usage_amend"))
		return
	}
	r := git.GitCommitAmend(args[0])
	m.gitResultLine(r, lang.T("msg_amended_commit"))
}

func (m *Model) handleReview(args []string) {
	if !git.IsGitRepo() {
		m.bad(lang.T("msg_not_git_repo"))
		return
	}
	var sb strings.Builder
	if staged := git.GitDiffStaged(); staged.OK && staged.Output != "" {
		sb.WriteString(lang.T("review_staged_changes") + "\n" + staged.Output + "\n")
	}
	if unstaged := git.GitDiffStat(); unstaged.OK && unstaged.Output != "" {
		sb.WriteString(lang.T("review_unstaged_changes") + "\n" + unstaged.Output)
	}
	if sb.Len() == 0 {
		sb.WriteString(lang.T("review_no_changes"))
	}
	m.info(sb.String())
}

func (m *Model) handleTutorial(args []string) {
	// Build pages as slices of outputLine so nothing is written to the
	// live viewport until showTutorialPage() renders the first page.
	type line = outputLine

	page := func(lines ...outputLine) []outputLine { return lines }
	ok   := func(s string) line { return line{kindGreen, s} }
	inf  := func(s string) line { return line{kindBlue, s} }
	warn := func(s string) line { return line{kindYellow, s} }
	gray := func(s string) line { return line{kindGray, s} }
	sep  := func() line         { return line{kindSep, ""} }

	m.tutorialPages = [][]outputLine{
		// ── Page 1 : What is Git? ──────────────────────────────────────────
		page(
			inf(lang.T("tutorial_title")),
			sep(),
			inf(lang.T("tutorial_what_is_git_title")),
			gray(lang.T("tutorial_what_is_git_line1")),
			gray(lang.T("tutorial_what_is_git_line2")),
			sep(),
			ok(lang.T("tutorial_concepts_title")),
			warn(lang.T("tutorial_concept_repo")),
			warn(lang.T("tutorial_concept_commit")),
			warn(lang.T("tutorial_concept_branch")),
			warn(lang.T("tutorial_concept_merge")),
		),

		// ── Page 2 : Getting Started ───────────────────────────────────────
		page(
			inf(lang.T("tutorial_getting_started_title")),
			sep(),
			warn(lang.T("tutorial_step1_title")),
			gray(lang.T("tutorial_step1_cmd")),
			warn(lang.T("tutorial_step2_title")),
			gray(lang.T("tutorial_step2_cmd")),
			warn(lang.T("tutorial_step3_title")),
			gray(lang.T("tutorial_step3_cmd")),
		),

		// ── Page 3 : Daily Workflow ────────────────────────────────────────
		page(
			ok(lang.T("tutorial_daily_workflow_title")),
			sep(),
			warn(lang.T("tutorial_daily_step1")),
			warn(lang.T("tutorial_daily_step2")),
			gray(lang.T("tutorial_daily_stage_cmd")),
			gray(lang.T("tutorial_daily_commit_cmd")),
			warn(lang.T("tutorial_daily_step3")),
			gray(lang.T("tutorial_daily_push_cmd")),
		),

		// ── Page 4 : Branches ─────────────────────────────────────────────
		page(
			inf(lang.T("tutorial_branches_title")),
			sep(),
			warn(lang.T("tutorial_branch_start")),
			gray(lang.T("tutorial_branch_start_cmd")),
			warn(lang.T("tutorial_branch_finish")),
			gray(lang.T("tutorial_branch_finish_cmd")),
			warn(lang.T("tutorial_branch_list")),
			gray(lang.T("tutorial_branch_list_cmd")),
		),

		// ── Page 5 : Git <-> gitBook reference table ──────────────────────
		page(
			ok(lang.T("tutorial_git_vs_gitbook_title")),
			sep(),
			gray(lang.T("tutorial_git_vs_row_init")),
			gray(lang.T("tutorial_git_vs_row_status")),
			gray(lang.T("tutorial_git_vs_row_add")),
			gray(lang.T("tutorial_git_vs_row_add_file")),
			gray(lang.T("tutorial_git_vs_row_commit")),
			gray(lang.T("tutorial_git_vs_row_push")),
			gray(lang.T("tutorial_git_vs_row_pull")),
			gray(lang.T("tutorial_git_vs_row_branch")),
			gray(lang.T("tutorial_git_vs_row_checkout")),
			gray(lang.T("tutorial_git_vs_row_merge")),
		),

		// ── Page 6 : Git <-> gitBook reference table (cont.) ──────────────
		page(
			ok(lang.T("tutorial_git_vs_gitbook_title")),
			sep(),
			gray(lang.T("tutorial_git_vs_row_log")),
			gray(lang.T("tutorial_git_vs_row_diff")),
			gray(lang.T("tutorial_git_vs_row_stash")),
			gray(lang.T("tutorial_git_vs_row_stash_pop")),
			gray(lang.T("tutorial_git_vs_row_reset")),
			gray(lang.T("tutorial_git_vs_row_revert")),
			gray(lang.T("tutorial_git_vs_row_rebase")),
			gray(lang.T("tutorial_git_vs_row_clone")),
			gray(lang.T("tutorial_git_vs_row_remote")),
			gray(lang.T("tutorial_git_vs_row_tag")),
			gray(lang.T("tutorial_git_vs_row_blame")),
			gray(lang.T("tutorial_git_vs_row_config")),
			gray(lang.T("tutorial_git_vs_row_gitignore")),
			inf(lang.T("tutorial_git_vs_hint")),
		),

		// ── Page 7 : Autonomous Pipelines ─────────────────────────────────
		page(
			ok(lang.T("tutorial_auto_title")),
			sep(),
			gray(lang.T("tutorial_auto_intro")),
			inf(""),
			warn(lang.T("tutorial_auto_push")),
			gray(lang.T("tutorial_auto_push_cmd")),
			warn(lang.T("tutorial_auto_save")),
			gray(lang.T("tutorial_auto_save_cmd")),
			warn(lang.T("tutorial_auto_sync")),
			gray(lang.T("tutorial_auto_sync_cmd")),
			warn(lang.T("tutorial_auto_start")),
			gray(lang.T("tutorial_auto_start_cmd")),
			warn(lang.T("tutorial_auto_release")),
			gray(lang.T("tutorial_auto_release_cmd")),
			warn(lang.T("tutorial_auto_fresh")),
			gray(lang.T("tutorial_auto_fresh_cmd")),
		),

		// ── Page 8 : Tips ─────────────────────────────────────────────────
		page(
			ok(lang.T("tutorial_tips_title")),
			sep(),
			warn(lang.T("tutorial_tip1")),
			warn(lang.T("tutorial_tip2")),
			warn(lang.T("tutorial_tip3")),
			warn(lang.T("tutorial_tip4")),
			warn(lang.T("tutorial_tip5")),
		),
	}

	m.tutorialActive = true
	m.tutorialPage = 0
	m.showTutorialPage()
}



// ── File Browser ─────────────────────────────────────────────────────────────

func (m *Model) loadDirectory(path string) error {
	entries, err := os.ReadDir(path)
	if err != nil {
		return err
	}
	var items []FileInfo
	if path != filepath.VolumeName(path)+string(filepath.Separator) && path != "/" {
		items = append(items, FileInfo{Name: "..", Path: filepath.Dir(path), IsDir: true})
	}
	var dirs, files []FileInfo
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}
		item := FileInfo{
			Name:    entry.Name(),
			Path:    filepath.Join(path, entry.Name()),
			IsDir:   entry.IsDir(),
			Size:    info.Size(),
			ModTime: info.ModTime().Format("2006-01-02 15:04"),
		}
		if entry.IsDir() {
			dirs = append(dirs, item)
		} else {
			files = append(files, item)
		}
	}
	sort.Slice(dirs, func(i, j int) bool { return dirs[i].Name < dirs[j].Name })
	sort.Slice(files, func(i, j int) bool { return files[i].Name < files[j].Name })
	items = append(items, dirs...)
	items = append(items, files...)
	m.fileBrowserItems = items
	m.fileBrowserCurrentDir = path
	m.fileBrowserSelected = 0
	return nil
}

func (m *Model) enterFileBrowserMode() {
	cwd, err := os.Getwd()
	if err != nil {
		m.bad(lang.Tf("file_browser_failed_getcwd", err))
		return
	}
	if err := m.loadDirectory(cwd); err != nil {
		m.bad(lang.Tf("file_browser_failed_load", err))
		return
	}
	m.showFileBrowser = true
	m.gray(lang.T("msg_select_directory"))
}

func (m *Model) exitFileBrowserMode() {
	m.showFileBrowser = false
	m.fileBrowserItems = nil
	m.fileBrowserSelected = 0
}

func (m *Model) navigateToFileBrowser(direction string) {
	if len(m.fileBrowserItems) == 0 {
		return
	}
	switch direction {
	case "up":
		if m.fileBrowserSelected > 0 {
			m.fileBrowserSelected--
		}
	case "down":
		if m.fileBrowserSelected < len(m.fileBrowserItems)-1 {
			m.fileBrowserSelected++
		}
	}
}

func (m *Model) selectFileBrowserItem() {
	if len(m.fileBrowserItems) == 0 {
		return
	}
	item := m.fileBrowserItems[m.fileBrowserSelected]
	if item.IsDir {
		if err := m.loadDirectory(item.Path); err != nil {
			m.bad(lang.Tf("file_browser_failed_enter", err))
		}
	} else {
		m.warn(lang.T("msg_select_dir_not_file"))
	}
}

func (m *Model) changeToSelectedDirectory() {
	if len(m.fileBrowserItems) == 0 {
		return
	}
	item := m.fileBrowserItems[m.fileBrowserSelected]
	if !item.IsDir {
		m.warn(lang.T("msg_select_dir_not_file"))
		return
	}
	if err := os.Chdir(item.Path); err != nil {
		m.bad(lang.Tf("msg_cannot_change_to", item.Path, err))
		return
	}
	cwd, _ := os.Getwd()
	m.ok(lang.Tf("msg_working_directory", cwd))
	m.exitFileBrowserMode()
}
