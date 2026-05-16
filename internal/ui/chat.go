package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nowte/gitbook/internal/git"
	"github.com/nowte/gitbook/internal/lang"
)

// ── Line renderer ─────────────────────────────────────────────────────────────

func renderLine(ol outputLine, w int) string {
	// Spinner satırı: marker'ı soy, sarı renkte göster
	if strings.HasPrefix(ol.content, spinnerLineMarker) {
		visible := strings.TrimPrefix(ol.content, spinnerLineMarker)
		return renderSideBox("  "+visible, colorYellow, w)
	}

	switch ol.kind {
	case kindInput:
		dollar := lipgloss.NewStyle().Foreground(tipOrange).Render("$ ")
		text := lipgloss.NewStyle().Foreground(pureWhite).Render(ol.content)
		return lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), false, false, false, true).
			BorderForeground(subtleGrey).
			PaddingLeft(2).PaddingRight(2).
			Width(w - 1).
			Render(dollar + text)

	case kindPlain:
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("252")).
			PaddingLeft(2).
			Render(wordWrap(ol.content, w-3))

	case kindSep:
		bar := strings.Repeat("─", w-4)
		return lipgloss.NewStyle().
			Foreground(subtleGrey).
			PaddingLeft(2).
			Render(bar)

	case kindGray:
		return renderSideBox(ol.content, colorGray, w)
	case kindBlue:
		return renderSideBox(ol.content, colorBlue, w)
	case kindRed:
		return renderSideBox(ol.content, colorRed, w)
	case kindGreen:
		return renderSideBox(ol.content, colorGreen, w)
	case kindYellow:
		return renderSideBox(ol.content, colorYellow, w)
	}
	return ol.content
}

func renderSideBox(content string, accent lipgloss.Color, w int) string {
	inner := w - 6
	if inner < 10 {
		inner = 10
	}
	body := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252")).
		Render(wordWrap(content, inner))
	return lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, false, false, true).
		BorderForeground(accent).
		PaddingLeft(3).PaddingRight(2).
		Width(w - 1).
		Render(body)
}

// ── Viewport helpers ──────────────────────────────────────────────────────────

func (m *Model) rebuildViewport() {
	// Enhanced nil guards and validation
	if m == nil {
		return
	}
	if !m.vpReady {
		return
	}
	if m.vp.Width <= 0 || m.vp.Height <= 0 {
		return
	}

	// Validate lines slice
	if len(m.lines) == 0 {
		m.vp.SetContent("")
		return
	}

	vw := m.vp.Width
	if vw <= 0 {
		vw = 80 // fallback width
	}

	var parts []string
	for _, ol := range m.lines {
		if ol.content != "" {
			parts = append(parts, renderLine(ol, vw))
		}
	}

	// Final validation before setting content
	if len(parts) > 0 {
		content := strings.Join(parts, "\n")
		m.vp.SetContent(content)
		m.vp.GotoBottom()
	} else {
		m.vp.SetContent("")
	}
}

// ── Command dispatcher ────────────────────────────────────────────────────────

func (m *Model) appendLine(kind outputKind, text string) {
	m.lines = append(m.lines, outputLine{kind, text})
}

func (m *Model) ok(text string)   { m.appendLine(kindGreen, text); m.rebuildViewport() }
func (m *Model) info(text string) { m.appendLine(kindBlue, text); m.rebuildViewport() }
func (m *Model) warn(text string) { m.appendLine(kindYellow, text); m.rebuildViewport() }
func (m *Model) bad(text string)  { m.appendLine(kindRed, text); m.rebuildViewport() }
func (m *Model) gray(text string) { m.appendLine(kindGray, text); m.rebuildViewport() }

func (m *Model) gitResultLine(r git.GitResult, successMsg string) {
	if r.OK {
		msg := successMsg
		if msg == "" && r.Output != "" {
			msg = r.Output
		}
		m.ok(msg)
	} else {
		errMsg := r.Err
		if errMsg == "" {
			errMsg = r.Output
		}
		hint := gitErrorHint(errMsg)
		if hint != "" {
			m.bad(hint)
		} else {
			m.bad(lang.T("err_hint_generic"))
		}
	}
}

// gitErrorHint returns a gitBook /command suggestion based on the git error message.
func gitErrorHint(errMsg string) string {
	lower := strings.ToLower(errMsg)

	type hintRule struct {
		keywords []string
		key      string
	}

	rules := []hintRule{
		{[]string{"nothing to commit", "nothing added to commit"}, "err_hint_nothing_to_commit"},
		{[]string{"not a git repository", "not a git repo"}, "err_hint_not_a_repo"},
		{[]string{"please tell me who you are", "user.email", "user.name"}, "err_hint_no_identity"},
		{[]string{"rejected", "failed to push", "non-fast-forward"}, "err_hint_push_rejected"},
		{[]string{"conflict", "merge conflict", "automatic merge failed"}, "err_hint_merge_conflict"},
		{[]string{"does not have a commit", "no commits yet"}, "err_hint_no_commits"},
		{[]string{"uncommitted changes", "unstaged changes"}, "err_hint_uncommitted"},
		{[]string{"already exists", "already up to date"}, "err_hint_already_uptodate"},
		{[]string{"detached head"}, "err_hint_detached_head"},
		{[]string{"authentication", "could not read username", "permission denied"}, "err_hint_auth_failed"},
		{[]string{"no remote", "no such remote"}, "err_hint_no_remote"},
		{[]string{"untracked files"}, "err_hint_untracked"},
		{[]string{"stash", "stash pop"}, "err_hint_stash"},
		{[]string{"rebase", "cannot rebase"}, "err_hint_rebase"},
		{[]string{"timed out", "timeout", "deadline exceeded"}, "err_hint_timeout"},
		{[]string{"empty repository", "no branches", "did not send all necessary"}, "err_hint_empty_repo"},
		{[]string{"another git process", "index.lock", "lock file"}, "err_hint_multi_instance"},
		{[]string{"refusing to merge unrelated histories"}, "err_hint_unrelated_histories"},
		{[]string{"repository not found"}, "err_hint_repo_not_found"},
		{[]string{"cannot lock ref"}, "err_hint_lock_conflict"},
		{[]string{"pathspec did not match"}, "err_hint_pathspec"},
		{[]string{"shallow update not allowed"}, "err_hint_shallow"},
	}

	for _, rule := range rules {
		for _, kw := range rule.keywords {
			if strings.Contains(lower, kw) {
				return lang.T(rule.key)
			}
		}
	}
	return ""
}

// execCommand kaldırıldı — processInput (types.go) registry üzerinden halleder.

// ── Tutorial pagination ───────────────────────────────────────────────────────

// tutorialPageCount returns how many pages the tutorial has.
func (m *Model) tutorialPageCount() int { return len(m.tutorialPages) }

// showTutorialPage renders the current tutorial page into the viewport.
func (m *Model) showTutorialPage() {
	if !m.tutorialActive || len(m.tutorialPages) == 0 {
		return
	}
	pg := m.tutorialPages[m.tutorialPage]
	total := m.tutorialPageCount()
	cur := m.tutorialPage + 1

	// Clear viewport lines and inject page content
	m.lines = nil
	for _, ol := range pg {
		m.lines = append(m.lines, ol)
	}
	// Footer nav hint
	nav := fmt.Sprintf("  [%d/%d]", cur, total)
	if cur < total {
		nav += "  ->  .next  |  next page"
	}
	if cur > 1 {
		nav += "  |  .prev  |  previous page"
	}
	m.lines = append(m.lines, outputLine{kindSep, ""})
	m.lines = append(m.lines, outputLine{kindGray, nav})
	m.rebuildViewport()
}

// advanceTutorialPage moves forward one page; exits tutorial on last page.
func (m *Model) advanceTutorialPage() {
	if !m.tutorialActive {
		return
	}
	if m.tutorialPage < m.tutorialPageCount()-1 {
		m.tutorialPage++
		m.showTutorialPage()
	} else {
		m.tutorialActive = false
		m.tutorialPages = nil
		m.ok(lang.T("tutorial_completed"))
		m.rebuildViewport()
	}
}

// prevTutorialPage moves back one page.
func (m *Model) prevTutorialPage() {
	if !m.tutorialActive || m.tutorialPage == 0 {
		return
	}
	m.tutorialPage--
	m.showTutorialPage()
}

// ── Chat update & view ────────────────────────────────────────────────────────

func (m Model) UpdateChat(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	// ── Pipeline mesajları ────────────────────────────────────────────────────
	case PipeStepStartMsg, SpinTickMsg, PipeStepDoneMsg,
		pipeStepFailWithChanMsg, PipelineDoneMsg, PipelineCancelledMsg:
		cmd := m.handlePipelineMsg(msg)
		// Kuyrukta başka mesaj varsa zincirle
		if cmd == nil {
			cmd = m.drainPipelineQueue()
		}
		return m, cmd

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		vpH := chatViewportHeight(m.height)
		if !m.vpReady {
			m.vp = viewport.New(m.width, vpH)
			m.vpReady = true
		} else {
			m.vp.Width = m.width
			m.vp.Height = vpH
		}
		m.rebuildViewport()
		return m, nil

	case tea.MouseMsg:
		if m.vpReady {
			var cmd tea.Cmd
			m.vp, cmd = m.vp.Update(msg)
			return m, cmd
		}

	case tea.KeyMsg:
		if msg.Type == tea.KeyCtrlC {
			return m, tea.Quit
		}

		if m.showFileBrowser {
			switch msg.Type {
			case tea.KeyEsc:
				m.exitFileBrowserMode()
				m.rebuildViewport()
				return m, nil
			case tea.KeyUp:
				m.navigateToFileBrowser("up")
				m.rebuildViewport()
				return m, nil
			case tea.KeyDown:
				m.navigateToFileBrowser("down")
				m.rebuildViewport()
				return m, nil
			case tea.KeyEnter:
				m.selectFileBrowserItem()
				m.rebuildViewport()
				return m, nil
			case tea.KeyTab:
				m.changeToSelectedDirectory()
				m.rebuildViewport()
				return m, nil
			}
			return m, nil
		}

		if m.showCommands {
			res := m.handleCmdListKey(msg)
			m.rebuildViewport()
			if res.execNow {
				val := strings.TrimSpace(m.textInput.Value())
				if val != "" {
					if quit := m.processInput(val); quit {
						return m, tea.Quit
					}
					m.textInput.SetValue("")
					m.showCommands = false
					m.rebuildViewport()
					if m.pipeline.pendingStartCmd != nil {
						cmd := m.pipeline.pendingStartCmd
						m.pipeline.pendingStartCmd = nil
						return m, cmd
					}
				}
			}
			return m, res.cmd
		}

		switch msg.Type {
		case tea.KeyEsc:
			if m.wizard.active() {
				m.cancelWizard()
				m.rebuildViewport()
				return m, nil
			}
			m.mode = modeHome
			m.refreshCache()
			m.textInput.SetValue("")
			m.showCommands = false
			return m, nil

		case tea.KeyUp:
			if len(m.inputHistory) > 0 && m.historyIdx > 0 {
				// İlk kez yukarı basılıyorsa mevcut girişi sakla
				if m.historyIdx == len(m.inputHistory) {
					m.historySaved = m.textInput.Value()
				}
				m.historyIdx--
				m.textInput.SetValue(m.inputHistory[m.historyIdx])
				m.textInput.CursorEnd()
			} else if m.vpReady {
				m.vp.LineUp(3)
			}
			return m, nil

		case tea.KeyDown:
			if len(m.inputHistory) > 0 && m.historyIdx < len(m.inputHistory) {
				m.historyIdx++
				if m.historyIdx == len(m.inputHistory) {
					// En sona döndük: kayıtlı geçici girişi geri yükle
					m.textInput.SetValue(m.historySaved)
				} else {
					m.textInput.SetValue(m.inputHistory[m.historyIdx])
				}
				m.textInput.CursorEnd()
			} else if m.vpReady {
				m.vp.LineDown(3)
			}
			return m, nil

		case tea.KeyPgUp:
			if m.vpReady {
				m.vp.HalfViewUp()
			}
			return m, nil

		case tea.KeyPgDown:
			if m.vpReady {
				m.vp.HalfViewDown()
			}
			return m, nil

		case tea.KeyEnter:
			val := strings.TrimSpace(m.textInput.Value())
			// Wizard aktifken boş Enter = optional alanı atla
			if val == "" && !m.wizard.active() {
				return m, nil
			}

			// Pipeline hata onayı bekleniyorsa girişi yakala
			if consumed, cmd := m.handlePipelineContinueInput(val); consumed {
				m.textInput.SetValue("")
				m.showCommands = false
				return m, cmd
			}

			// Güncelleme sorusu bekleniyorsa her girişi yakala
			if m.awaitingUpdateConfirm {
				m.textInput.SetValue("")
				m.showCommands = false
				lower := strings.ToLower(val)
				switch lower {
				case "yes", "y", "evet", "e":
					m.awaitingUpdateConfirm = false
					m.ok(lang.T("msg_update_continue"))
				case "no", "n", "hayır", "hayir", "h":
					m.awaitingUpdateConfirm = false
					m.info(lang.T("msg_update_opening_browser"))
					openBrowser("https://github.com/nowte/gitbook/releases/latest")
				default:
					// Yanlış cevap — sadece input'u temizle, chate hiçbir şey ekleme
				}
				m.rebuildViewport()
				return m, nil
			}

			if quit := m.processInput(val); quit {
				return m, tea.Quit
			}
			m.textInput.SetValue("")
			m.showCommands = false
			m.rebuildViewport()
			// Pipeline başlatıldıysa Cmd'i döndür
			if m.pipeline.pendingStartCmd != nil {
				cmd := m.pipeline.pendingStartCmd
				m.pipeline.pendingStartCmd = nil
				return m, cmd
			}
			return m, nil
		}

		var cmd tea.Cmd
		m.textInput, cmd = m.textInput.Update(msg)
		nv := m.textInput.Value()

		if shouldShowCommands(nv) {
			m.showCommands = true
			m.filteredCmds = filterCommands(nv)
			if m.cmdSelected >= len(m.filteredCmds) {
				m.cmdSelected = 0
			}
		} else {
			m.showCommands = false
		}
		return m, cmd
	}

	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m Model) ViewChat() string {
	w := m.width

	var topBar string
	switch {
	case m.latestVersion == "":
		topBar = lipgloss.NewStyle().
			Width(w).Align(lipgloss.Right).
			Foreground(subtleGrey).
			Render("v" + GitbookVersion + " ")
	case !UpdateAvailable(m.latestVersion):
		// Up to date
		topBar = lipgloss.NewStyle().
			Width(w).Align(lipgloss.Right).
			Foreground(subtleGrey).
			Render("v" + GitbookVersion + " · up to date ")
	default:
		// Update available
		topBar = lipgloss.NewStyle().
			Width(w).Align(lipgloss.Right).
			Foreground(tipOrange).
			Render("v" + GitbookVersion + " · outdated  ↑  " + m.latestVersion + " available ")
	}
	topBarH := lipgloss.Height(topBar)

	inputBar := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, false, false, true).
		BorderForeground(accentBlue).
		Padding(1, 4).
		Background(bgDark).
		Width(w).
		Render(m.textInput.View())
	inputBarH := lipgloss.Height(inputBar)

	spacer := lipgloss.NewStyle().Width(w).Render("")

	vpH := m.height - topBarH - inputBarH - 1 // 1 = spacer satırı
	if vpH < 1 {
		vpH = 1
	}
	if m.vpReady && m.vp.Height != vpH {
		m.vp.Height = vpH
		m.vp.GotoBottom()
	}

	vpView := ""
	if m.vpReady {
		vpView = m.vp.View()
	}

	if m.showFileBrowser {
		fileBrowserView := renderFileBrowser(m, w)
		return lipgloss.JoinVertical(lipgloss.Left, topBar, vpView, fileBrowserView, spacer, inputBar)
	}

	if m.showCommands {
		maxVisible := (m.height - inputBarH) / 3
		if maxVisible < 1 {
			maxVisible = 1
		}
		if maxVisible > cmdListMaxVisible {
			maxVisible = cmdListMaxVisible
		}
		cmdListView := renderCmdListFixed(m.filteredCmds, m.cmdSelected, true, w, maxVisible)
		cmdListH := lipgloss.Height(cmdListView)
		vpLines := strings.Split(vpView, "\n")
		if len(vpLines) > cmdListH {
			vpView = strings.Join(vpLines[:len(vpLines)-cmdListH], "\n")
		} else {
			vpView = ""
		}
		return lipgloss.JoinVertical(lipgloss.Left, topBar, vpView, cmdListView, spacer, inputBar)
	}

	return lipgloss.JoinVertical(lipgloss.Left, topBar, vpView, spacer, inputBar)
}

func chatViewportHeight(totalH int) int {
	inputBarH := 3
	spacerH := 1
	h := totalH - inputBarH - spacerH
	if h < 1 {
		h = 1
	}
	return h
}

func forDash(s string) string {
	if s == "" {
		return "—"
	}
	return s
}

// ── File Browser UI ───────────────────────────────────────────────────────────

func renderFileBrowser(m Model, w int) string {
	if !m.showFileBrowser || len(m.fileBrowserItems) == 0 {
		return ""
	}

	maxVisible := 10
	if maxVisible > len(m.fileBrowserItems) {
		maxVisible = len(m.fileBrowserItems)
	}

	// Calculate visible window
	start := 0
	if m.fileBrowserSelected >= maxVisible {
		start = m.fileBrowserSelected - maxVisible + 1
	}
	end := start + maxVisible
	if end > len(m.fileBrowserItems) {
		end = len(m.fileBrowserItems)
		start = end - maxVisible
		if start < 0 {
			start = 0
		}
	}

	visible := m.fileBrowserItems[start:end]
	selectedIndex := m.fileBrowserSelected - start

	// Header
	header := lipgloss.NewStyle().
		Foreground(pureWhite).
		Background(accentBlue).
		Bold(true).
		Padding(0, 2).
		Render(" [D] " + m.fileBrowserCurrentDir)

	// File list
	var rows []string
	nameWidth := w - 30
	if nameWidth < 20 {
		nameWidth = 20
	}

	for i, item := range visible {
		isSelected := i == selectedIndex

		var icon string
		var nameColor, sizeColor lipgloss.Color
		if item.IsDir {
			icon = "[D]"
			nameColor = colorBlue
			sizeColor = colorGray
		} else {
			icon = "[f]"
			nameColor = pureWhite
			sizeColor = colorGray
		}

		// Truncate name if too long
		name := item.Name
		if len([]rune(name)) > nameWidth {
			name = string([]rune(name)[:nameWidth-1]) + "…"
		}

		// Format size
		sizeStr := ""
		if !item.IsDir {
			if item.Size < 1024 {
				sizeStr = fmt.Sprintf("%dB", item.Size)
			} else if item.Size < 1024*1024 {
				sizeStr = fmt.Sprintf("%.1fK", float64(item.Size)/1024)
			} else {
				sizeStr = fmt.Sprintf("%.1fM", float64(item.Size)/(1024*1024))
			}
		}

		// Build row
		var bgColor lipgloss.Color
		if isSelected {
			bgColor = tipOrange
		} else {
			bgColor = bgDark
		}

		iconStyle := lipgloss.NewStyle().Foreground(colorYellow).Background(bgColor)
		nameStyle := lipgloss.NewStyle().Foreground(nameColor).Background(bgColor)
		sizeStyle := lipgloss.NewStyle().Foreground(sizeColor).Background(bgColor)
		timeStyle := lipgloss.NewStyle().Foreground(subtleGrey).Background(bgColor)

		row := lipgloss.JoinHorizontal(lipgloss.Left,
			iconStyle.Render(icon+" "),
			nameStyle.Render(padRight(name, nameWidth)),
			sizeStyle.Render(padRight(sizeStr, 8)),
			timeStyle.Render(item.ModTime),
		)

		rows = append(rows, row)
	}

	// Footer with help
	footer := lipgloss.NewStyle().
		Foreground(pureWhite).
		Background(colorBlue).
		Bold(true).
		Padding(0, 1).
		Render(lang.T("file_browser_help"))

	// Combine all parts
	content := lipgloss.JoinVertical(lipgloss.Left,
		header,
		strings.Join(rows, "\n"),
		"",
		footer,
	)

	return lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), true, true, true, true).
		BorderForeground(accentBlue).
		Padding(1).
		Background(lipgloss.Color("236")).
		Width(w).
		Render(content)
}
func (m *Model) EnterChat(firstInput string) {
	m.mode = modeChat
	vpH := chatViewportHeight(m.height)
	m.vp = viewport.New(m.width, vpH)
	m.vp.Width = m.width
	m.vpReady = true

	m.lines = []outputLine{
		{kindGray, buildStartupStatus()},
	}

	if UpdateAvailable(m.latestVersion) {
		m.lines = append(m.lines, outputLine{kindYellow,
			lang.Tf("msg_update_prompt_notice", GitbookVersion, m.latestVersion)})
		m.lines = append(m.lines, outputLine{kindYellow,
			lang.T("msg_update_prompt_question")})
		m.awaitingUpdateConfirm = true
	}

	if firstInput != "" {
		// Direkt komut: processInput çağır
		_ = m.processInput(firstInput)
	} else if m.wizard.active() {
		// Home'dan wizard başlatıldı: wizard panelini yeniden göster
		m.showWizardPanel()
		m.promptWizardField()
	}

	m.textInput.SetValue("")
	m.showCommands = false
	m.rebuildViewport()
}

// showWizardPanel wizard'ın alan listesini chat output'una yeniden basar.
func (m *Model) showWizardPanel() {
	cmd := lookupCommand(m.wizard.cmdName)
	if cmd == nil {
		return
	}
	var sb strings.Builder
	sb.WriteString(lang.T("wizard_menu_header") + "\n")
	for i, f := range m.wizard.fields {
		marker := "  "
		if i == m.wizard.idx {
			marker = "▶ "
		}
		req := ""
		if f.required {
			req = " *"
		}
		sb.WriteString(fmt.Sprintf("%s[%d] %s%s\n", marker, i+1, f.label, req))
	}
	sb.WriteString("\n" + lang.T("wizard_menu_footer"))
	m.info(sb.String())
}

// buildHelpText returns the full help text for /help.
func buildHelpText() string {
	sections := []struct {
		title string
		cmds  [][2]string
	}{
		{lang.T("help_session"), [][2]string{
			{"/new", lang.T("cmd_new")},
			{"/exit", lang.T("cmd_exit")},
			{"/help", lang.T("cmd_help")},
			{"/tutorial", lang.T("cmd_tutorial")},
		}},
		{lang.T("help_setup_config"), [][2]string{
			{"/setup", lang.T("cmd_setup")},
			{"/config", lang.T("cmd_config")},
			{"/language", lang.T("cmd_language")},
			{"/info", lang.T("cmd_info")},
			{"/cd <path>", lang.T("cmd_cd")},
			{"/path", lang.T("cmd_path")},
			{"/pwd", lang.T("cmd_pwd")},
		}},
		{lang.T("help_repository"), [][2]string{
			{"/init", lang.T("cmd_init")},
			{"/status", lang.T("cmd_status")},
			{"/branch", lang.T("cmd_branch")},
			{"/log [n]", lang.T("cmd_log")},
			{"/review", lang.T("cmd_review")},
			{"/diff [workspace]", lang.T("cmd_diff")},
			{"/blame <file>", lang.T("cmd_blame")},
		}},
		{lang.T("help_feature_workflow"), [][2]string{
			{"/start <name>", lang.T("cmd_start")},
			{"/finish", lang.T("cmd_finish")},
			{"/cleanup [workspace-name]", lang.T("cmd_cleanup")},
		}},
		{lang.T("help_staging_committing"), [][2]string{
			{"/stage [path]", lang.T("cmd_stage")},
			{"/unstage <path>", lang.T("cmd_unstage")},
			{"/commit <description>", lang.T("cmd_commit")},
			{"/amend <msg>", lang.T("cmd_amend")},
		}},
		{lang.T("help_remote_github"), [][2]string{
			{"/github <url>", lang.T("cmd_github")},
			{"/remote", lang.T("cmd_remote")},
			{"/push", lang.T("cmd_push")},
			{"/pull", lang.T("cmd_pull")},
			{"/fetch", lang.T("cmd_fetch")},
			{"/clone <url> [dir]", lang.T("cmd_clone")},
			{"/sync", lang.T("cmd_sync")},
			{"/tag-push", lang.T("cmd_tag_push")},
		}},
		{lang.T("help_stash"), [][2]string{
			{"/stash [note]", lang.T("cmd_stash")},
			{"/stash-pop", lang.T("cmd_stash_pop")},
			{"/stash-list", lang.T("cmd_stash_list")},
		}},
		{lang.T("help_tags"), [][2]string{
			{"/tag [name]", lang.T("cmd_tag")},
			{"/tag-push", lang.T("cmd_tag_push")},
		}},
		{lang.T("help_undo_history"), [][2]string{
			{"/reset [n]", lang.T("cmd_reset")},
			{"/reset-hard [n]", lang.T("cmd_reset_hard")},
			{"/revert <hash>", lang.T("cmd_revert")},
			{"/cherry-pick <hash>", lang.T("cmd_cherry_pick")},
			{"/rebase <workspace>", lang.T("cmd_rebase")},
		}},
		{lang.T("help_smart_systems"), [][2]string{
			{"/analyze", lang.T("cmd_analyze")},
			{"/suggest", lang.T("cmd_suggest")},
			{"/gitignore [preview]", lang.T("cmd_gitignore")},
			{"/profile [set|show]", lang.T("cmd_profile")},
		}},
		{lang.T("help_auto_section"), [][2]string{
			{"/auto-push", lang.T("cmd_auto_push")},
			{"/auto-save", lang.T("cmd_auto_save")},
			{"/auto-sync", lang.T("cmd_auto_sync")},
			{"/auto-start", lang.T("cmd_auto_start")},
			{"/auto-release", lang.T("cmd_auto_release")},
			{"/auto-fresh", lang.T("cmd_auto_fresh")},
			{"", lang.T("help_auto_pipeline_note")},
		}},
	}

	var sb strings.Builder
	sb.WriteString(lang.T("help_title") + "\n")
	sb.WriteString(lang.T("help_separator") + "\n")
	for _, sec := range sections {
		sb.WriteString("\n" + sec.title + "\n")
		for _, c := range sec.cmds {
			sb.WriteString(fmt.Sprintf("  %-26s %s\n", c[0], c[1]))
		}
	}
	return sb.String()
}
