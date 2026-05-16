package ui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/nowte/gitbook/internal/lang"
)

// ── Auto Pipeline Commands — Disabled ────────────────────────────────────────
//
// /auto-push, /auto-save, /auto-sync, /auto-start, /auto-release, /auto-fresh
// komutları şimdilik devre dışı bırakılmıştır.
//
// Bu dosyadaki tüm handler'lar kullanıcıya bilgi mesajı gösterir ve döner.
// pipeline.go sistemi hâlâ derleniyor; yalnızca bu entry-point'ler pasif.
// ─────────────────────────────────────────────────────────────────────────────

// withPipelineCmd is a no-op shim kept for compilation compatibility.
func (m *Model) withPipelineCmd(cmd tea.Cmd) (tea.Model, tea.Cmd) {
	return m, cmd
}

func (m *Model) handleAutoPush(_ []string) {
	m.info(lang.T("auto_coming_soon"))
}

func (m *Model) handleAutoSave(_ []string) {
	m.info(lang.T("auto_coming_soon"))
}

func (m *Model) handleAutoSync(_ []string) {
	m.info(lang.T("auto_coming_soon"))
}

func (m *Model) handleAutoStart(_ []string) {
	m.info(lang.T("auto_coming_soon"))
}

func (m *Model) handleAutoRelease(_ []string) {
	m.info(lang.T("auto_coming_soon"))
}

func (m *Model) handleAutoFresh(_ []string) {
	m.info(lang.T("auto_coming_soon"))
}
