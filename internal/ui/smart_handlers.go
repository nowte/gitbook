package ui

import (
	"fmt"
	"os"
	"strings"

	"github.com/nowte/gitbook/internal/git"
	"github.com/nowte/gitbook/internal/lang"
	"github.com/nowte/gitbook/internal/smart"
)

// ── /analyze — Diff Analiz Motoru ────────────────────────────────────────────

func (m *Model) handleAnalyze(args []string) {
	if !git.IsGitRepo() {
		m.bad(lang.T("msg_not_git_repo"))
		return
	}

	// staged yoksa unstaged'a bak
	staged := git.GitDiffStagedFull()
	unstaged := git.GitDiffFull()

	hasStaged := staged.OK && strings.TrimSpace(staged.Output) != ""
	hasUnstaged := unstaged.OK && strings.TrimSpace(unstaged.Output) != ""

	if !hasStaged && !hasUnstaged {
		m.info(lang.T("smart_analyze_no_changes"))
		return
	}

	m.info(lang.T("smart_analyze_title"))

	if hasStaged {
		m.gray(lang.T("smart_analyze_staged_header"))
		statResult := git.GitDiffStaged()
		summary := smart.AnalyzeDiffStat(statResult.Output)
		m.gray(summary.Format())
	}

	if hasUnstaged {
		m.gray(lang.T("smart_analyze_unstaged_header"))
		statResult := git.GitDiffStat()
		summary := smart.AnalyzeDiffStat(statResult.Output)
		m.gray(summary.Format())
	}
}

// ── /suggest — Commit Mesajı Öneri Motoru ────────────────────────────────────

func (m *Model) handleSuggest(args []string) {
	if !git.IsGitRepo() {
		m.bad(lang.T("msg_not_git_repo"))
		return
	}

	// Staged değişiklikler varsa onları, yoksa unstaged'a bak
	statResult := git.GitDiffStaged()
	if !statResult.OK || strings.TrimSpace(statResult.Output) == "" {
		statResult = git.GitDiffStat()
	}

	if !statResult.OK || strings.TrimSpace(statResult.Output) == "" {
		m.warn(lang.T("smart_suggest_no_changes"))
		return
	}

	summary := smart.AnalyzeDiffStat(statResult.Output)
	suggestions := smart.SuggestCommitMessages(summary)

	m.info(lang.T("smart_suggest_title"))
	m.gray(smart.FormatSuggestions(suggestions))

	// Birincil öneriyi kolay kopyalanabilir şekilde göster
	if len(suggestions) > 0 {
		m.ok(fmt.Sprintf(lang.T("smart_suggest_copy_hint"), suggestions[0].Message))
	}
}

// ── /gitignore — Akıllı .gitignore Üreteci ───────────────────────────────────

func (m *Model) handleGitignore(args []string) {
	cwd, err := os.Getwd()
	if err != nil {
		m.bad(lang.Tf("smart_gitignore_cwd_error", err))
		return
	}

	// "preview" argümanı varsa sadece göster, yazma
	preview := len(args) > 0 && args[0] == "preview"

	result := smart.GenerateGitignore(cwd)

	m.info(lang.T("smart_gitignore_title"))
	m.gray(smart.FormatGitignoreResult(result))

	if preview {
		m.gray(lang.T("smart_gitignore_preview_header"))
		// İlk 30 satırı göster
		lines := strings.Split(result.Content, "\n")
		limit := 30
		if len(lines) < limit {
			limit = len(lines)
		}
		m.gray(strings.Join(lines[:limit], "\n"))
		if len(lines) > 30 {
			m.gray(fmt.Sprintf(lang.T("smart_gitignore_more_lines"), len(lines)-30))
		}
		m.warn(lang.T("smart_gitignore_preview_hint"))
		return
	}

	// .gitignore dosyasına yaz
	gitignorePath := ".gitignore"
	if err := os.WriteFile(gitignorePath, []byte(result.Content), 0644); err != nil {
		m.bad(lang.Tf("smart_gitignore_write_error", err))
		return
	}

	if result.AlreadyExist && result.WasMerged {
		m.ok(lang.T("smart_gitignore_merged"))
	} else if result.AlreadyExist && !result.WasMerged {
		m.ok(lang.T("smart_gitignore_already_uptodate"))
	} else {
		m.ok(lang.T("smart_gitignore_created"))
	}
}

// ── /profile — Proje Profil Sistemi ──────────────────────────────────────────

func (m *Model) handleProfile(args []string) {
	if len(args) == 0 {
		// Profil listesini göster
		store, err := smart.LoadProfiles()
		if err != nil {
			m.bad(lang.Tf("smart_profile_load_error", err))
			return
		}
		m.info(smart.FormatProfileList(store))
		return
	}

	subCmd := strings.ToLower(args[0])
	switch subCmd {

	case "set":
		// /profile set <isim>
		if len(args) < 2 {
			m.bad(lang.T("smart_profile_set_usage"))
			return
		}
		profileName := args[1]
		if err := smart.SetActiveProfile(profileName); err != nil {
			m.bad(err.Error())
			return
		}
		m.ok(lang.Tf("smart_profile_activated", profileName))

	case "show":
		// /profile show — aktif profili detaylı göster
		p := smart.GetActiveProfile()
		m.info(smart.FormatActiveProfile(p))

	case "init":
		// /profile init — .gitbook/profiles.json oluştur (varsayılan profiller)
		if !git.IsGitRepo() {
			m.bad(lang.T("msg_not_git_repo"))
			return
		}
		store, err := smart.LoadProfiles()
		if err != nil {
			m.bad(lang.Tf("smart_profile_load_error", err))
			return
		}
		if err := smart.SaveProfiles(store); err != nil {
			m.bad(lang.Tf("smart_profile_save_error", err))
			return
		}
		m.ok(lang.T("smart_profile_init_done"))

	default:
		m.bad(lang.Tf("smart_profile_unknown_cmd", subCmd))
	}
}

// ── /push (akıllı versiyon) ───────────────────────────────────────────────────
//
// Mevcut handlePush'u genişletmek yerine profilden gelen ek kontrolleri
// yapan bir wrapper: handlePushSmart
//
// types.go'daki /push komutu bunu çağırmalı.

func (m *Model) handlePushSmart(args []string) {
	if !git.IsGitRepo() {
		m.bad(lang.T("msg_not_git_repo"))
		return
	}

	branchResult := git.GitBranch()
	if !branchResult.OK {
		m.bad(lang.T("msg_not_git_repo"))
		return
	}
	branch := branchResult.Output

	// Profil tabanlı push kontrolü
	check := smart.CheckPush(branch)
	if !check.Allowed {
		m.bad(check.Message)
		return
	}

	// Push öncesi diff özeti (profil ayarlıysa)
	if check.ShowDiff {
		statResult := git.GitDiffStaged()
		if !statResult.OK || strings.TrimSpace(statResult.Output) == "" {
			statResult = git.GitDiffStat()
		}
		if statResult.OK && strings.TrimSpace(statResult.Output) != "" {
			summary := smart.AnalyzeDiffStat(statResult.Output)
			m.gray(lang.T("smart_push_diff_header"))
			m.gray(summary.Format())
		}
	}

	// Onay gerekiyor mu?
	if check.NeedsConfirm {
		m.warn(check.Message)
		m.setupContext = branch
		m.beginWizard(&command{
			name: "/__confirm_push",
			fields: []wizardField{
				{label: lang.Tf("smart_push_confirm_prompt", branch), placeholder: "push / iptal", required: true},
			},
			handler: func(m *Model, a []string) {
				if len(a) > 0 && strings.ToLower(a[0]) == "push" {
					m.executePush(m.setupContext, args)
				} else {
					m.info(lang.T("smart_push_cancelled"))
				}
				m.setupContext = ""
			},
		})
		return
	}

	// Normal push
	m.executePush(branch, args)
}

// executePush asıl push işlemini yapar.
func (m *Model) executePush(branch string, args []string) {
	remote := "origin"
	if len(args) > 0 && args[0] != "" {
		remote = args[0]
	}

	ahead := git.GitCountAhead()
	if ahead == 0 {
		m.warn(lang.T("msg_nothing_to_push"))
		return
	}

	r := git.GitPush(remote, branch)
	if !r.OK {
		r = git.GitPushSetUpstream(remote, branch)
	}
	m.gitResultLine(r, lang.Tf("msg_pushed", branch, remote))
}

// ── /commit (akıllı versiyon) ─────────────────────────────────────────────────
//
// Profil commit politikasını uygulayan wrapper.

func (m *Model) handleCommitSmart(args []string) {
	if !git.IsGitRepo() {
		m.bad(lang.T("msg_not_git_repo"))
		return
	}
	if len(args) == 0 {
		m.bad(lang.T("usage_commit"))
		return
	}

	commitMsg := args[0]

	// Profil tabanlı mesaj doğrulama
	if err := smart.CheckCommitMessage(commitMsg); err != nil {
		m.bad(err.Error())
		return
	}

	// Staged diff göster (profil ayarlıysa)
	p := smart.GetActiveProfile()
	if p.Commit.ShowStagedOnCommit {
		staged := git.GitDiffStaged()
		if staged.OK && strings.TrimSpace(staged.Output) != "" {
			summary := smart.AnalyzeDiffStat(staged.Output)
			m.gray(lang.T("smart_commit_staged_header"))
			m.gray(summary.Format())
		}
	}

	r := git.GitCommit(commitMsg)
	m.gitResultLine(r, lang.Tf("msg_committed", commitMsg))
}
