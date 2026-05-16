package ui

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nowte/gitbook/internal/config"
	"github.com/nowte/gitbook/internal/git"
	"github.com/nowte/gitbook/internal/lang"
)

// ── Colour palette ────────────────────────────────────────────────────────────

var (
	subtleGrey  = lipgloss.Color("241")
	pureWhite   = lipgloss.Color("255")
	accentBlue  = lipgloss.Color("33")
	tipOrange   = lipgloss.Color("208")
	bgDark      = lipgloss.Color("234")
	colorGray   = lipgloss.Color("240")
	colorBlue   = lipgloss.Color("33")
	colorRed    = lipgloss.Color("196")
	colorGreen  = lipgloss.Color("82")
	colorYellow = lipgloss.Color("220")
	logoStyle   = lipgloss.NewStyle().Foreground(subtleGrey).Bold(true)
)

// ── File browser ──────────────────────────────────────────────────────────────

type FileInfo struct {
	Name    string
	Path    string
	IsDir   bool
	Size    int64
	ModTime string
}

// ── Output lines ──────────────────────────────────────────────────────────────

type outputKind int

const (
	kindInput  outputKind = iota
	kindPlain
	kindGray
	kindBlue
	kindRed
	kindGreen
	kindYellow
	kindSep // thin separator — no side-bar, no padding
)

type outputLine struct {
	kind    outputKind
	content string
}

// ── App mode ──────────────────────────────────────────────────────────────────

type appMode int

const (
	modeHome appMode = iota
	modeChat
)

// ── Wizard ────────────────────────────────────────────────────────────────────
//
// setupStep enum KALDIRILDI. Wizard state artık sadece şu struct ile yönetilir.
// Yeni bir wizard komutu için başka hiçbir yerde kayıt gerekmez.

type wizardField struct {
	label       string
	placeholder string
	required    bool
}

type wizardState struct {
	fields    []wizardField
	collected []string
	idx       int
	cmdName   string // hangi komut başlattı
}

func (w wizardState) active() bool { return len(w.fields) > 0 }
func (w wizardState) done() bool   { return w.active() && w.idx >= len(w.fields) }

// ── Command registry ──────────────────────────────────────────────────────────
//
// TEK KAYNAK GERÇEK.
// Yeni komut = buraya bir satır. Başka hiçbir yere dokunmak gerekmez.
//
//   handler    -> args ile doğrudan çağrılır
//   fields     -> dolu ise listeden seçilince wizard başlar
//   needsInput -> true ise input'a "<name> " yazılır

type command struct {
	name       string
	desc       string
	handler    func(*Model, []string)
	fields     []wizardField
	needsInput bool
}

var commands []command // initCommands() ile doldurulur

func initCommands(m *Model) {
	wf := func(label, ph string, req bool) wizardField {
		return wizardField{label: label, placeholder: ph, required: req}
	}
	commands = []command{
		// ── Wizard komutları ──────────────────────────────────────────────────
		{
			name:    "/amend",
			desc:    "Fix the description of your last save",
			fields:  []wizardField{wf("New commit message", "feat: updated ...", true)},
			handler: func(m *Model, a []string) { m.handleAmend(a) },
		},
		{
			name:    "/blame",
			desc:    "See who changed each line of a file",
			fields:  []wizardField{wf("File path", "src/main.go", true)},
			handler: func(m *Model, a []string) { m.handleBlame(a) },
		},
		{
			name: "/clone",
			desc: "Download a project from GitHub",
			fields: []wizardField{
				wf("Repository URL", "https://github.com/user/repo.git", true),
				wf("Target directory (optional)", "my-repo", false),
			},
			handler: func(m *Model, a []string) { m.handleClone(a) },
		},
		{
			name:    "/github",
			desc:    "Connect this project to GitHub",
			fields:  []wizardField{wf("GitHub URL", "https://github.com/user/repo.git", true)},
			handler: func(m *Model, a []string) { m.handleGitHub(a) },
		},
		{
			name: "/setup",
			desc: "Set your name and email",
			fields: []wizardField{
				wf("Your name", "Ada Lovelace", true),
				wf("Your email", "ada@example.com", true),
			},
			// handler: dispatchWizard'da özel ele alınır
		},

		// ── needsInput komutları (wizard destekli, args varsa direkt çalışır) ──
		{name: "/branch",      desc: "List or create a workspace",              needsInput: true, fields: []wizardField{wf("Workspace name (optional)", "feature/my-feature", false)}, handler: func(m *Model, a []string) { m.handleBranch(a) }},
		{name: "/cd",          desc: "Open a different folder",          needsInput: true, fields: []wizardField{wf("Directory path", "/path/to/dir", true)}, handler: func(m *Model, a []string) { m.handleCd(a) }},
		{name: "/cherry-pick", desc: "Apply a past save to this workspace",  needsInput: true, fields: []wizardField{wf("Save ID", "abc1234", true)}, handler: func(m *Model, a []string) { m.handleCherryPick(a) }},
		{name: "/cleanup",     desc: "Remove a finished workspace",      needsInput: true, fields: []wizardField{wf("Workspace name (optional)", "feature/done", false)}, handler: func(m *Model, a []string) { m.handleCleanup(a) }},
		{name: "/commit",      desc: "Save your marked changes with a description", needsInput: true, fields: []wizardField{wf("Save description", "e.g. improved login page", true)}, handler: func(m *Model, a []string) { m.handleCommitSmart(a) }},
		{name: "/diff",        desc: "See what changed in your files", needsInput: true, fields: []wizardField{wf("Workspace to compare (optional)", "main", false)}, handler: func(m *Model, a []string) { m.handleDiff(a) }},
		{name: "/language",    desc: "Change the interface language",         needsInput: true, fields: []wizardField{wf("Language code", "en / tr", true)}, handler: func(m *Model, a []string) { m.handleLanguage(a) }},
		{name: "/log",         desc: "View your save history",               needsInput: true, fields: []wizardField{wf("How many to show (optional)", "10", false)}, handler: func(m *Model, a []string) { m.handleLog(a) }},
		{name: "/path",        desc: "Browse and pick a folder",   needsInput: true, handler: func(m *Model, a []string) { m.handlePath(a) }},
		{name: "/push",        desc: "Send your saved work to GitHub",  needsInput: true, fields: []wizardField{wf("Destination (optional, default: origin)", "origin", false)}, handler: func(m *Model, a []string) { m.handlePushSmart(a) }},
		{name: "/rebase",      desc: "Move your workspace onto another",  needsInput: true, fields: []wizardField{wf("Target workspace", "main", true)}, handler: func(m *Model, a []string) { m.handleRebase(a) }},
		{name: "/reset",       desc: "Undo last save(s) — keep your changes",         needsInput: true, fields: []wizardField{wf("Number of saves to undo", "1", true)}, handler: func(m *Model, a []string) { m.handleReset(a) }},
		{name: "/reset-hard",  desc: "Undo last save(s) — discard your changes",         needsInput: true, fields: []wizardField{wf("Number of saves to undo", "1", true)}, handler: func(m *Model, a []string) { m.handleResetHard(a) }},
		{name: "/revert",      desc: "Cancel a specific past save (safely)",            needsInput: true, fields: []wizardField{wf("Save ID to undo", "abc1234", true)}, handler: func(m *Model, a []string) { m.handleRevert(a) }},
		{name: "/stage",       desc: "Mark files as ready to save",                       needsInput: true, fields: []wizardField{wf("File path (optional — leave empty for all)", "src/main.go", false)}, handler: func(m *Model, a []string) { m.handleStage(a) }},
		{name: "/start",       desc: "Open a new workspace for a feature",        needsInput: true, fields: []wizardField{wf("Workspace name", "my-feature", true)}, handler: func(m *Model, a []string) { m.handleStart(a) }},
		{name: "/tag",         desc: "List or mark a version",              needsInput: true, fields: []wizardField{wf("Tag name (optional, list if empty)", "v1.0.0", false)}, handler: func(m *Model, a []string) { m.handleTag(a) }},
		{name: "/tag-push",    desc: "Send version markers to GitHub",           needsInput: true, fields: []wizardField{wf("Destination (optional, default: origin)", "origin", false)}, handler: func(m *Model, a []string) { m.handleTagPush(a) }},
		{name: "/unstage",     desc: "Unmark a file from ready-to-save list",                    needsInput: true, fields: []wizardField{wf("File path", "src/main.go", true)}, handler: func(m *Model, a []string) { m.handleUnstage(a) }},

		// ── Direkt komutlar ───────────────────────────────────────────────────
		{name: "/config",     desc: "Show your account info and project details",         handler: func(m *Model, a []string) { m.handleConfig(a) }},
		{name: "/exit",       desc: "Exit the app"},
		{name: "/fetch",      desc: "Check for updates (without downloading)",                       handler: func(m *Model, a []string) { m.handleFetch(a) }},
		{name: "/finish",     desc: "Merge your feature into the main workspace",      handler: func(m *Model, a []string) { m.handleFinish(a) }},
		{name: "/help",       desc: "Show all available commands",                       handler: func(m *Model, a []string) { m.info(buildHelpText()) }},
		{name: "/info",       desc: "Show project and app information",  handler: func(m *Model, a []string) { m.handleInfo(a) }},
		{name: "/init",       desc: "Set up version tracking for this project",                handler: func(m *Model, a []string) { m.handleInit(a) }},
		{name: "/new",        desc: "Clear the screen and start fresh",                           handler: func(m *Model, a []string) { m.lines = nil; m.gray(lang.T("msg_session_cleared")) }},
		{name: "/pull",       desc: "Download the latest changes from GitHub",                     handler: func(m *Model, a []string) { m.handlePull(a) }},
		{name: "/pwd",        desc: "Show which folder you are in",                 handler: func(m *Model, a []string) { cwd, _ := os.Getwd(); m.info(cwd) }},
		{name: "/remote",     desc: "Show where your project is synced online",                 handler: func(m *Model, a []string) { m.handleRemote(a) }},
		{name: "/review",     desc: "Preview all changes before saving",        handler: func(m *Model, a []string) { m.handleReview(a) }},
		{name: "/stash",      desc: "Set aside unfinished changes temporarily",               needsInput: true, fields: []wizardField{wf("Note about what you set aside (optional)", "e.g. paused login redesign", false)}, handler: func(m *Model, a []string) { m.handleStash(a) }},
		{name: "/stash-list", desc: "See your set-aside changes",                        handler: func(m *Model, a []string) { m.handleStashList(a) }},
		{name: "/stash-pop",  desc: "Bring back your last set-aside changes",      handler: func(m *Model, a []string) { m.handleStashPop(a) }},
		{name: "/status",     desc: "See what has changed in your project",         handler: func(m *Model, a []string) { m.handleStatus(a) }},
		{name: "/sync",       desc: "Check how far ahead/behind from GitHub",       handler: func(m *Model, a []string) { m.handleSync(a) }},
		{name: "/tutorial",   desc: "Step-by-step guide for beginners",       handler: func(m *Model, a []string) { m.handleTutorial(a) }},
		{name: "/undo",       desc: "Go back to the last checkpoint",       handler: func(m *Model, a []string) { m.handleUndo(a) }},
		{name: "/snapshots",  desc: "List all saved checkpoints",                    handler: func(m *Model, a []string) { m.handleSnapshots(a) }},

		// ── Otonom Pipeline Komutları ─────────────────────────────────────────
		// Tek komutla birden fazla adım çalıştırır. /help auto ile listele.
		{
			name: "/auto-push",
			desc: "Mark + save + upload to GitHub in one step",
			fields: []wizardField{
				wf("Commit mesajı", "e.g. giriş sayfası düzeltildi", true),
				wf("Remote (opsiyonel)", "origin", false),
			},
			handler: func(m *Model, a []string) { m.handleAutoPush(a) },
		},
		{
			name: "/auto-save",
			desc: "Mark + save locally — no upload (offline)",
			fields: []wizardField{
				wf("Commit mesajı", "e.g. henüz bitmemiş özellik", true),
			},
			handler: func(m *Model, a []string) { m.handleAutoSave(a) },
		},
		{
			name: "/auto-sync",
			desc: "Check + download + show status",
			handler: func(m *Model, a []string) { m.handleAutoSync(a) },
		},
		{
			name: "/auto-start",
			desc: "Set up + identity + GitHub + first upload",
			fields: []wizardField{
				wf("Adınız", "Ada Lovelace", true),
				wf("E-postanız", "ada@example.com", true),
				wf("GitHub repo URL'si", "https://github.com/kullanici/repo.git", true),
			},
			handler: func(m *Model, a []string) { m.handleAutoStart(a) },
		},
		{
			name: "/auto-release",
			desc: "Save + mark version + upload — publish a release",
			fields: []wizardField{
				wf("Release commit mesajı", "release: v1.0.0", true),
				wf("Tag adı", "v1.0.0", true),
				wf("Remote (opsiyonel)", "origin", false),
			},
			handler: func(m *Model, a []string) { m.handleAutoRelease(a) },
		},
		{
			name: "/auto-fresh",
			desc: "Set up + ignore files + save — fresh start",
			fields: []wizardField{
				wf("İlk commit mesajı (opsiyonel)", "chore: initial setup", false),
			},
			handler: func(m *Model, a []string) { m.handleAutoFresh(a) },
		},

		// ── Akıllı Sistemler (v1.02.00) ──────────────────────────────────────
		{name: "/analyze",   desc: "Analyze what changed — file types, scope, warnings",
			handler: func(m *Model, a []string) { m.handleAnalyze(a) }},
		{name: "/suggest",   desc: "Get a save description suggestion",
			handler: func(m *Model, a []string) { m.handleSuggest(a) }},
		{name: "/gitignore", desc: "Auto-generate a list of files to never track",
			needsInput: true,
			fields:     []wizardField{wf("Mod: boş=yaz, preview=önizle", "preview", false)},
			handler:    func(m *Model, a []string) { m.handleGitignore(a) }},
		{name: "/profile",   desc: "Manage profiles (work / personal / open-source)",
			needsInput: true,
			fields: []wizardField{
				wf("Alt komut: list / set <isim> / show / init", "list", false),
				wf("Profil adı (set için)", "work", false),
			},
			handler: func(m *Model, a []string) { m.handleProfile(a) }},
	}
}

func lookupCommand(name string) *command {
	for i := range commands {
		if commands[i].name == name {
			return &commands[i]
		}
	}
	return nil
}

// ── Main model ────────────────────────────────────────────────────────────────

type Model struct {
	mode      appMode
	textInput textinput.Model
	width     int
	height    int

	showCommands bool
	cmdSelected  int
	filteredCmds []command

	lines   []outputLine
	vp      viewport.Model
	vpReady bool

	// Wizard (setupStep enum'u kaldırıldı)
	wizard       wizardState
	setupContext string // confirmation promptları için bağlam

	gitVersion    string
	latestVersion string
	lastInputValue string

	// Cache
	cachedIsGitRepo   bool
	cachedGitBranch   string
	cachedWorkingDir  string
	cacheInitialized  bool
	cachedTopBar      string
	cachedEngineState string
	cachedFooterLeft  string
	cachedInputBox    string
	cacheWidth        int

	// File browser
	showFileBrowser       bool
	fileBrowserItems      []FileInfo
	fileBrowserSelected   int
	fileBrowserCurrentDir string

	// Tutorial pagination
	tutorialActive bool
	tutorialPage   int  // 0-based current page index
	tutorialPages  [][]outputLine // each page is a slice of lines

	// Input history (up/down ok tuşları)
	inputHistory []string // geçmiş komutlar (en yeni sonda)
	historyIdx   int      // geçerli pozisyon; len(inputHistory) = "aktif giriş"
	historySaved string   // ok tuşuna basılmadan önceki geçici giriş

	// Update confirmation prompt
	awaitingUpdateConfirm bool

	// Pipeline durum göstergesi
	pipeline pipelineState
}

// ── processInput: home ve chat'in ortak giriş noktası ────────────────────────

func (m *Model) processInput(val string) (quit bool) {
	val = strings.TrimSpace(val)

	// Wizard aktifken boş Enter = optional alanı geç
	if m.wizard.active() {
		m.advanceWizard(val)
		return false
	}

	if val == "" {
		return false
	}
	if val == "/exit" {
		return true
	}

	m.appendLine(kindInput, val)

	// Geçmişe ekle (aynı komut art arda tekrarlanmasın)
	if len(m.inputHistory) == 0 || m.inputHistory[len(m.inputHistory)-1] != val {
		m.inputHistory = append(m.inputHistory, val)
	}
	m.historyIdx = len(m.inputHistory) // sona sıfırla
	m.historySaved = ""

	// Tutorial navigation (.next / .prev) — intercepted before the "/" check
	// so that the dot-prefix doesn't trigger "not a command".
	if val == ".next" || strings.HasPrefix(val, ".next ") {
		if m.tutorialActive {
			m.advanceTutorialPage()
		} else {
			m.bad(lang.Tf("msg_unknown_command", ".next"))
		}
		return false
	}
	if val == ".prev" || strings.HasPrefix(val, ".prev ") {
		if m.tutorialActive {
			m.prevTutorialPage()
		} else {
			m.bad(lang.Tf("msg_unknown_command", ".prev"))
		}
		return false
	}

	if !strings.HasPrefix(val, "/") {
		m.appendLine(kindGray, lang.T("msg_not_a_command"))
		// Any non-command message also cancels the tutorial
		if m.tutorialActive {
			m.tutorialActive = false
			m.tutorialPages = nil
		}
		return false
	}

	parts := strings.Fields(val)
	cmdName := parts[0]
	args := parts[1:]

	// Any /command typed while tutorial is active also cancels it
	if m.tutorialActive {
		m.tutorialActive = false
		m.tutorialPages = nil
	}

	cmd := lookupCommand(cmdName)
	if cmd == nil {
		m.bad(lang.Tf("msg_unknown_command", cmdName))
		return false
	}

	// fields varsa ve args yoksa -> wizard başlat
	if len(cmd.fields) > 0 && len(args) == 0 {
		m.beginWizard(cmd)
		return false
	}

	// fields var ama args da var -> wizard'ı atla, direkt çalıştır
	if cmd.handler != nil {
		cmd.handler(m, args)
	}
	return false
}

// ── Wizard engine ─────────────────────────────────────────────────────────────

func (m *Model) beginWizard(cmd *command) {
	m.wizard = wizardState{
		fields:    cmd.fields,
		collected: make([]string, 0, len(cmd.fields)),
		idx:       0,
		cmdName:   cmd.name,
	}
	// Home modunda paneli gösterme — EnterChat geçişi showWizardPanel çağıracak
	if m.mode == modeChat {
		var sb strings.Builder
		sb.WriteString(lang.T("wizard_menu_header") + "\n")
		for i, f := range cmd.fields {
			marker := "  "
			if i == 0 {
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
		m.promptWizardField()
	}
}

func (m *Model) promptWizardField() {
	if !m.wizard.active() || m.wizard.done() {
		return
	}
	f := m.wizard.fields[m.wizard.idx]
	total := len(m.wizard.fields)
	m.gray(fmt.Sprintf(lang.T("wizard_prompt_field"), m.wizard.idx+1, total, f.label))
	if !f.required {
		m.gray(lang.T("wizard_skip_hint"))
	}
	m.textInput.Placeholder = f.placeholder
	m.textInput.SetValue("")
}

func (m *Model) advanceWizard(input string) {
	if !m.wizard.active() {
		return
	}
	f := m.wizard.fields[m.wizard.idx]
	trimmed := strings.TrimSpace(input)
	if f.required && trimmed == "" {
		m.bad(lang.T("wizard_error_required"))
		return
	}
	// Optional field boş bırakıldıysa collected'a ekleme (args temiz kalır)
	if trimmed != "" {
		m.wizard.collected = append(m.wizard.collected, trimmed)
	}
	m.wizard.idx++
	if !m.wizard.done() {
		m.promptWizardField()
		return
	}
	m.dispatchWizard()
}

func (m *Model) dispatchWizard() {
	collected := m.wizard.collected
	cmdName := m.wizard.cmdName
	cmd := lookupCommand(cmdName)

	m.wizard = wizardState{}
	m.textInput.Placeholder = lang.T("placeholder_run_command")
	m.textInput.SetValue("")

	switch cmdName {
	case "/setup":
		name, email := "", ""
		if len(collected) > 0 { name = collected[0] }
		if len(collected) > 1 { email = collected[1] }
		if r := git.GitConfig("user.name", name); !r.OK {
			m.bad(lang.Tf("msg_git_name_failed", r.Err))
			return
		}
		if r := git.GitConfig("user.email", email); !r.OK {
			m.bad(lang.Tf("msg_git_email_failed", r.Err))
			return
		}
		m.ok(lang.T("msg_git_identity_configured"))
	default:
		if cmd != nil && cmd.handler != nil {
			cmd.handler(m, collected)
		}
	}
}

func (m *Model) cancelWizard() {
	m.wizard = wizardState{}
	m.textInput.Placeholder = lang.T("placeholder_run_command")
	m.textInput.SetValue("")
	m.appendLine(kindGray, lang.T("msg_cancelled"))
}

// ── handleCmdListKey (*Model receiver — artık value receiver değil) ───────────

type cmdListResult struct {
	execNow bool
	cmd     tea.Cmd
}

func (m *Model) handleCmdListKey(msg tea.KeyMsg) cmdListResult {
	switch msg.Type {
	case tea.KeyEsc:
		m.showCommands = false
		m.filteredCmds = commands
		m.cmdSelected = 0
		return cmdListResult{}

	case tea.KeyUp:
		if m.cmdSelected > 0 {
			m.cmdSelected--
		}
		return cmdListResult{}

	case tea.KeyDown:
		if m.cmdSelected < len(m.filteredCmds)-1 {
			m.cmdSelected++
		}
		return cmdListResult{}

	case tea.KeyEnter:
		if len(m.filteredCmds) == 0 {
			return cmdListResult{}
		}
		sel := m.filteredCmds[m.cmdSelected]
		m.showCommands = false
		m.filteredCmds = commands
		m.cmdSelected = 0

		if len(sel.fields) > 0 {
			m.beginWizard(&sel)
			return cmdListResult{}
		}
		if sel.needsInput {
			m.textInput.SetValue(sel.name + " ")
			m.textInput.CursorEnd()
			return cmdListResult{}
		}
		m.textInput.SetValue(sel.name)
		m.textInput.CursorEnd()
		return cmdListResult{execNow: true}

	case tea.KeyBackspace:
		val := m.textInput.Value()
		if len([]rune(val)) <= 1 {
			m.textInput.SetValue("")
			m.showCommands = false
			m.filteredCmds = commands
			m.cmdSelected = 0
			return cmdListResult{}
		}
		var tc tea.Cmd
		m.textInput, tc = m.textInput.Update(msg)
		nv := m.textInput.Value()
		m.filteredCmds = filterCommands(nv)
		if m.cmdSelected >= len(m.filteredCmds) {
			m.cmdSelected = maxInt(0, len(m.filteredCmds)-1)
		}
		m.showCommands = shouldShowCommands(nv)
		return cmdListResult{cmd: tc}

	default:
		var tc tea.Cmd
		m.textInput, tc = m.textInput.Update(msg)
		nv := m.textInput.Value()
		m.filteredCmds = filterCommands(nv)
		if m.cmdSelected >= len(m.filteredCmds) {
			m.cmdSelected = 0
		}
		m.showCommands = shouldShowCommands(nv)
		return cmdListResult{cmd: tc}
	}
}

// ── Helpers ───────────────────────────────────────────────────────────────────

var _ = logoStyle

const cmdListMaxVisible = 6

func newTextInput() textinput.Model {
	ti := textinput.New()
	ti.Placeholder = "What do you want to do? Type /help to see all options…"
	ti.Focus()
	ti.Prompt = ""
	ti.TextStyle = lipgloss.NewStyle().Foreground(pureWhite).Background(bgDark)
	ti.PlaceholderStyle = lipgloss.NewStyle().Foreground(subtleGrey).Background(bgDark)
	return ti
}

type versionCheckMsg struct {
	latest string
	err    error
}

type cacheInitializedMsg struct{}

func doVersionCheck() tea.Cmd {
	return func() tea.Msg {
		ch := make(chan versionCheckMsg, 1)
		go func() {
			defer func() {
				if r := recover(); r != nil {
					ch <- versionCheckMsg{err: fmt.Errorf("version check panicked: %v", r)}
				}
			}()
			latest, err := checkLatestVersion()
			ch <- versionCheckMsg{latest: latest, err: err}
		}()
		select {
		case msg := <-ch:
			return msg
		case <-time.After(10 * time.Second):
			return versionCheckMsg{err: fmt.Errorf("version check timeout after 10 seconds")}
		}
	}
}

func (m *Model) initCache() {
	if m.cacheInitialized && m.cacheWidth == m.width {
		return
	}
	if wd, err := os.Getwd(); err == nil {
		m.cachedWorkingDir = wd
	}
	m.cacheInitialized = true
	m.cacheWidth = m.width
	m.renderCachedComponents()
}

func (m *Model) initCacheAsync() tea.Cmd {
	return func() tea.Msg {
		m.cachedIsGitRepo = git.IsGitRepo()
		if m.cachedIsGitRepo {
			if branch := git.GitBranch(); branch.OK {
				m.cachedGitBranch = branch.Output
			}
		}
		m.renderCachedComponents()
		return cacheInitializedMsg{}
	}
}

func (m *Model) renderCachedComponents() {
	dw := 80
	if m.width < 85 {
		dw = m.width - 5
	}
	if dw < 30 {
		dw = 30
	}

	asciiLogo := "        _ _   ____              _    \n" +
		"   __ _(_) |_| __ )  ___   ___ | | __\n" +
		"  / _` | | __|  _ \\ / _ \\ / _ \\| |/ /\n" +
		" | (_| | | |_| |_) | (_) | (_) |   < \n" +
		"  \\__, |_|\\__|____/ \\___/ \\___/|_|\\_\\\n" +
		"  |___/                               \n"
	m.cachedTopBar = lipgloss.NewStyle().
		Foreground(subtleGrey).Width(dw).Align(lipgloss.Center).
		Render(asciiLogo)

	engineLabel := lipgloss.NewStyle().Foreground(accentBlue).Bold(true).Background(bgDark).Render("ENGINE  ")
	gitLabel := lipgloss.NewStyle().Foreground(pureWhite).Background(bgDark).Render("Git  ")
	if m.gitVersion != "" {
		activeLabel := lipgloss.NewStyle().Foreground(lipgloss.Color("82")).Background(bgDark).Render("● active")
		m.cachedEngineState = lipgloss.JoinHorizontal(lipgloss.Top, engineLabel, gitLabel,
			lipgloss.NewStyle().Foreground(subtleGrey).Background(bgDark).Render(m.gitVersion+"  "), activeLabel)
	} else {
		m.cachedEngineState = lipgloss.JoinHorizontal(lipgloss.Top, engineLabel, gitLabel,
			lipgloss.NewStyle().Foreground(colorRed).Background(bgDark).Render("◌ not found"))
	}

	footerPath := lipgloss.NewStyle().Foreground(subtleGrey).Bold(true).Render(m.cachedWorkingDir)
	repoStatus := ""
	if m.cachedIsGitRepo && m.cachedGitBranch != "" {
		label := m.cachedGitBranch
		if config.IsProtectedBranch(label) {
			label = "(!) " + label
		}
		repoStatus = lipgloss.NewStyle().Foreground(accentBlue).Render("  git:" + label)
	}
	m.cachedFooterLeft = footerPath + repoStatus

	m.cachedInputBox = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, false, false, true).
		BorderForeground(accentBlue).Padding(2, 4).Background(bgDark).Width(dw).
		Render("")
}

func (m *Model) refreshCache() {
	m.cachedIsGitRepo = git.IsGitRepo()
	if m.cachedIsGitRepo {
		if branch := git.GitBranch(); branch.OK {
			m.cachedGitBranch = branch.Output
		} else {
			m.cachedGitBranch = ""
		}
	} else {
		m.cachedGitBranch = ""
	}
	if wd, err := os.Getwd(); err == nil {
		m.cachedWorkingDir = wd
	}
}

func InitialModel() Model {
	gitVer := git.GitVersion()
	shortVer := ""
	if gitVer.OK {
		raw := strings.TrimPrefix(strings.TrimSpace(gitVer.Output), "git version ")
		parts := strings.Fields(raw)
		if len(parts) > 0 {
			shortVer = parts[0]
		}
	}
	m := Model{
		mode:       modeHome,
		textInput:  newTextInput(),
		gitVersion: shortVer,
	}
	initCommands(&m)
	m.filteredCmds = commands
	return m
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, doVersionCheck())
}

func (m Model) Update(msg tea.Msg) (model tea.Model, cmd tea.Cmd) {
	// Graceful panic recovery — log the panic and keep the TUI alive
	defer func() {
		if r := recover(); r != nil {
			m.bad(fmt.Sprintf("[!] Internal error (recovered): %v", r))
			m.gray("    Please report this at https://github.com/nowte/gitbook/issues")
			m.rebuildViewport()
			model = m
			cmd = nil
		}
	}()

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if m.mode == modeChat {
			return m.UpdateChat(msg)
		}
		return m, nil

	case versionCheckMsg:
		if msg.err == nil {
			m.latestVersion = msg.latest
			if m.mode == modeChat && UpdateAvailable(m.latestVersion) && !m.awaitingUpdateConfirm {
				m.lines = append(m.lines, outputLine{kindYellow,
					lang.Tf("msg_update_prompt_notice", GitbookVersion, m.latestVersion)})
				m.lines = append(m.lines, outputLine{kindYellow,
					lang.T("msg_update_prompt_question")})
				m.awaitingUpdateConfirm = true
				m.rebuildViewport()
			}
		}
		return m, nil

	case cacheInitializedMsg:
		return m, nil
	}

	if m.mode == modeChat {
		return m.UpdateChat(msg)
	}
	return m.UpdateHome(msg)
}

func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}
	if m.mode == modeChat {
		return m.ViewChat()
	}
	return m.ViewHome()
}

// ── Command filtering ─────────────────────────────────────────────────────────

func filterCommands(input string) []command {
	if input == "" || input == "/" {
		return commands
	}
	query := strings.ToLower(strings.TrimPrefix(input, "/"))
	if query == "" {
		return commands
	}
	var exact, partial []command
	for _, c := range commands {
		if strings.ToLower(c.name) == "/"+query {
			exact = append(exact, c)
		}
	}
	if len(exact) > 0 {
		return exact
	}
	for _, c := range commands {
		nameCore := strings.TrimPrefix(strings.ToLower(c.name), "/")
		if strings.Contains(nameCore, query) || strings.Contains(strings.ToLower(c.desc), query) {
			partial = append(partial, c)
		}
	}
	return partial
}

func shouldShowCommands(val string) bool {
	return strings.HasPrefix(val, "/")
}

// ── Layout helpers ────────────────────────────────────────────────────────────

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func wordWrap(s string, width int) string {
	if width <= 0 {
		return s
	}
	var out []string
	for _, line := range strings.Split(s, "\n") {
		words := strings.Fields(line)
		if len(words) == 0 {
			out = append(out, "")
			continue
		}
		cur := ""
		for _, w := range words {
			if cur == "" {
				cur = w
			} else if len(cur)+1+len(w) <= width {
				cur += " " + w
			} else {
				out = append(out, cur)
				cur = w
			}
		}
		if cur != "" {
			out = append(out, cur)
		}
	}
	return strings.Join(out, "\n")
}

// ── Command list renderer ─────────────────────────────────────────────────────

func renderCmdListFixed(filteredCmds []command, cmdSelected int, show bool, width int, maxRows int) string {
	if !show {
		return ""
	}
	if len(filteredCmds) == 0 {
		notFound := lipgloss.NewStyle().Foreground(subtleGrey).Render("no such command")
		return lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), false, false, false, true).
			BorderForeground(subtleGrey).Padding(1, 4).Background(bgDark).Width(width).
			Render(notFound)
	}

	start := 0
	if cmdSelected >= maxRows {
		start = cmdSelected - maxRows + 1
	}
	end := start + maxRows
	if end > len(filteredCmds) {
		end = len(filteredCmds)
		start = end - maxRows
		if start < 0 {
			start = 0
		}
	}
	visible := filteredCmds[start:end]
	visSelected := cmdSelected - start
	ciw := width - 9
	if ciw < 10 {
		ciw = 10
	}

	var rows []string
	for i, c := range visible {
		sel := i == visSelected
		var rowBg, nFg, dFg lipgloss.Color
		if sel {
			rowBg, nFg, dFg = tipOrange, pureWhite, pureWhite
		} else {
			rowBg, nFg, dFg = bgDark, pureWhite, subtleGrey
		}
		ns := lipgloss.NewStyle().Foreground(nFg).Background(rowBg).Bold(true).
			Render(padRight(c.name, 18))
		da := ciw - 20
		if da < 5 {
			da = 5
		}
		dt := c.desc
		if rr := []rune(dt); len(rr) > da {
			dt = string(rr[:da-1]) + "…"
		}
		dr := lipgloss.NewStyle().Foreground(dFg).Background(rowBg).Render(dt)

		hintColor := subtleGrey
		if sel {
			hintColor = pureWhite
		}
		hs := lipgloss.NewStyle().Foreground(hintColor).Background(rowBg)
		var hint string
		if len(c.fields) > 0 {
			hint = hs.Render(" ◈")
		} else if c.needsInput {
			hint = hs.Render(" ›")
		}

		pad := maxInt(0, ciw-18-lipgloss.Width(dt)-lipgloss.Width(hint))
		tr := lipgloss.NewStyle().Background(rowBg).Render(strings.Repeat(" ", pad))
		rows = append(rows, ns+dr+tr+hint)
	}

	scrollLine := ""
	if len(filteredCmds) > maxRows {
		shown := strconv.Itoa(end) + "/" + strconv.Itoa(len(filteredCmds))
		scrollLine = "\n" + lipgloss.NewStyle().Foreground(subtleGrey).Render("  ↑↓ "+shown)
	}

	inner := strings.Join(rows, "\n") + scrollLine
	return lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, false, false, true).
		BorderForeground(subtleGrey).Padding(1, 4).Background(bgDark).Width(width).
		Render(inner)
}

func padRight(s string, n int) string {
	runes := []rune(s)
	if len(runes) >= n {
		return s
	}
	return s + strings.Repeat(" ", n-len(runes))
}
