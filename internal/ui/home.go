package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ── Home Update ────────────────────────────────────────────────────────────────

func (m Model) UpdateHome(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, m.initCacheAsync()

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit
		}

		// Komut listesi açıksa: ortak handler
		if m.showCommands {
			res := m.handleCmdListKey(msg)

			// Wizard başlatıldıysa chat'e geç
			if m.wizard.active() {
				m.EnterChat("")   // boş: wizard zaten başlatıldı
				m.rebuildViewport()
				return m, nil
			}

			// Direkt çalıştır sinyali
			if res.execNow {
				val := strings.TrimSpace(m.textInput.Value())
				if val != "" {
					m.EnterChat(val)
					return m, nil
				}
			}
			return m, res.cmd
		}

		// Normal tuş: Enter veya yazma
		if msg.Type == tea.KeyEnter {
			val := strings.TrimSpace(m.textInput.Value())
			if val == "" {
				return m, nil
			}
			m.showCommands = false
			m.EnterChat(val)
			return m, nil
		}

		m.textInput, cmd = m.textInput.Update(msg)

	case tea.MouseMsg:
		if m.showCommands && msg.Action == tea.MouseActionRelease {
			listLen := len(m.filteredCmds)
			if listLen > 0 {
				mainUIHeight := 14 + listLen
				startY := (m.height-1)/2 - mainUIHeight/2
				listTop := startY + 3
				clickedRow := msg.Y - listTop
				if clickedRow >= 0 && clickedRow < listLen {
					sel := m.filteredCmds[clickedRow]
					if len(sel.fields) > 0 {
						m.beginWizard(&sel)
						m.EnterChat("")
						m.rebuildViewport()
						return m, nil
					}
					m.textInput.SetValue(sel.name + " ")
					m.textInput.CursorEnd()
					m.showCommands = false
					m.filteredCmds = commands
					return m, nil
				}
			}
		}
	}

	// Input değişimini izle → komut filtrele
	if !m.showCommands {
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
	}

	return m, cmd
}

// ── Home View ──────────────────────────────────────────────────────────────────

func (m Model) ViewHome() string {
	m.initCache()

	dw := 80
	if m.width < 85 {
		dw = m.width - 5
	}
	if dw < 30 {
		dw = 30
	}

	topBar := m.cachedTopBar
	gitState := m.cachedEngineState

	inputStyle := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, false, false, true).
		BorderForeground(accentBlue).
		Padding(2, 4).
		Background(bgDark).
		Width(dw)
	mainBox := inputStyle.Render(m.textInput.View() + "\n\n" + gitState)

	shortcuts := lipgloss.NewStyle().Width(dw).Align(lipgloss.Right).Render(
		lipgloss.JoinHorizontal(lipgloss.Top,
			lipgloss.NewStyle().Foreground(pureWhite).Render("esc"),
			lipgloss.NewStyle().Foreground(subtleGrey).Render(" quit"),
		),
	)

	tipLine := lipgloss.JoinHorizontal(lipgloss.Top,
		lipgloss.NewStyle().Foreground(tipOrange).Render(" ● Tip "),
		lipgloss.NewStyle().Foreground(subtleGrey).Render("New here? "),
		lipgloss.NewStyle().Foreground(pureWhite).Bold(true).Render("/tutorial"),
		lipgloss.NewStyle().Foreground(subtleGrey).Render(" — or /help to see everything."),
	)

	logoH := lipgloss.Height(topBar)
	logoPad := 2
	maxVisible := logoH + logoPad - 2
	if maxVisible < 1 {
		maxVisible = 1
	}
	if maxVisible > cmdListMaxVisible {
		maxVisible = cmdListMaxVisible
	}

	cmdSlot := ""
	if m.showCommands {
		cmdSlot = renderCmdListFixed(m.filteredCmds, m.cmdSelected, true, dw, maxVisible)
	}
	cmdSlotH := lipgloss.Height(cmdSlot)

	logoAndGap := lipgloss.JoinVertical(lipgloss.Left, topBar, strings.Repeat("\n", logoPad-1))
	logoAndGapLines := strings.Split(logoAndGap, "\n")

	if m.showCommands && cmdSlotH > 0 {
		cmdLines := strings.Split(strings.TrimRight(cmdSlot, "\n"), "\n")
		insertAt := len(logoAndGapLines) - len(cmdLines)
		if insertAt < 0 {
			insertAt = 0
		}
		for i, cl := range cmdLines {
			idx := insertAt + i
			if idx < len(logoAndGapLines) {
				logoAndGapLines[idx] = cl
			}
		}
	}
	logoBlock := strings.Join(logoAndGapLines, "\n")

	group := lipgloss.JoinVertical(lipgloss.Left,
		logoBlock,
		mainBox,
		shortcuts,
		lipgloss.NewStyle().MarginTop(1).Width(dw).Align(lipgloss.Center).Render(tipLine),
	)

	totalH := m.height - 1
	centeredGroup := lipgloss.Place(m.width, totalH, lipgloss.Center, lipgloss.Center, group)

	footerLeft := m.cachedFooterLeft
	versionLabel := GitbookVersion
	if UpdateAvailable(m.latestVersion) {
		versionLabel = GitbookVersion +
			lipgloss.NewStyle().Foreground(tipOrange).Render("  ^ update available: "+m.latestVersion)
	}
	versionStr := lipgloss.NewStyle().Foreground(subtleGrey).Render(versionLabel)
	padLen := m.width - lipgloss.Width(footerLeft) - lipgloss.Width(versionStr)
	if padLen < 0 {
		padLen = 0
	}
	footer := footerLeft + strings.Repeat(" ", padLen) + versionStr

	return centeredGroup + "\n" + footer
}
