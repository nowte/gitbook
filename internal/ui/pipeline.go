package ui

// ── Pipeline Durum Göstergesi ─────────────────────────────────────────────────
//
// Bu dosya pipeline adımlarını çalıştıran ve UI'ya ilerleme mesajları
// ileten sistemi tanımlar.
//
// Akış:
//   1. handleAutoXxx()  ->  m.runPipeline(steps)
//   2. runPipeline       ->  goroutine: her adımı sırayla çalıştır
//   3. Her adım başı     ->  PipeStepStartMsg  (spinner başlar)
//   4. Her adım sonu     ->  PipeStepDoneMsg   (ms cinsinden süre)
//   5. Hata              ->  PipeStepFailMsg   -> awaitingPipelineContinue = true
//   6. Kullanıcı cevabı  ->  "devam" / "vazgeç"
//   7. Pipeline biter    ->  PipelineDoneMsg
//
// Spinner karakter dizisi — saf ASCII:
//   |  /  -  \
// ─────────────────────────────────────────────────────────────────────────────

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/nowte/gitbook/internal/git"
	"github.com/nowte/gitbook/internal/lang"
)

// ── Spinner ───────────────────────────────────────────────────────────────────

var spinnerFrames = []string{"|", "/", "-", "\\"}

// ── Tea Mesajları ─────────────────────────────────────────────────────────────

// SpinTickMsg — spinner'ı bir kare ilerletir.
type SpinTickMsg struct{}

// PipeStepStartMsg — yeni bir adım başladı; spinner satırı eklenir.
type PipeStepStartMsg struct {
	Label string
}

// PipeStepDoneMsg — adım başarıyla bitti.
type PipeStepDoneMsg struct {
	Label   string
	Elapsed time.Duration
}

// PipelineDoneMsg — tüm pipeline tamamlandı (başarıyla).
type PipelineDoneMsg struct {
	Summary string // son ok mesajı
}

// PipelineCancelledMsg — kullanıcı "vazgeç" dedi.
type PipelineCancelledMsg struct{}

// ── Pipeline Adımı ────────────────────────────────────────────────────────────

// PipeStep tek bir pipeline adımını tanımlar.
type PipeStep struct {
	Label string                  // ekranda gösterilecek isim
	Run   func() git.GitResult    // çalıştırılacak fonksiyon
}

// ── Spinner tick komutu ───────────────────────────────────────────────────────

func spinTickCmd() tea.Cmd {
	return tea.Tick(80*time.Millisecond, func(_ time.Time) tea.Msg {
		return SpinTickMsg{}
	})
}

// ── Pipeline State (Model'e gömülü) ──────────────────────────────────────────

type pipelineState struct {
	active       bool
	spinnerFrame int
	stepLabel    string // aktif adım etiketi

	// Hata sonrası bekleme
	awaitingContinue bool
	failedLabel      string
	failedErr        string

	// Tamamlandığında gösterilecek mesaj
	doneMsg string

	// Hata sonrası kalan adımları çalıştırmak için kanal
	continueCh chan bool // true = devam, false = vazgeç

	// Tea mesaj kuyruğu — goroutine'den gelen mesajlar birikir
	msgQueue []tea.Msg

	// handler içinden döndürülemeyen Cmd'i taşır;
	// UpdateChat tarafından bir sonraki döngüde alınır
	pendingStartCmd tea.Cmd
}

func newPipelineState() pipelineState {
	return pipelineState{}
}

// ── runPipeline: pipeline'ı goroutine'de başlatır ─────────────────────────────
//
// steps listesindeki her adımı sırayla çalıştırır.
// Her adım için UI'ya Msg gönderir.
// Hata durumunda continueCh üzerinden kullanıcı cevabını bekler.



// ── Gerçek kanal tabanlı implementasyon ──────────────────────────────────────
//
// runPipelineWithContinue — hata durumunda kullanıcıdan onay alır.
// Model.pipeline.continueCh üzerinden true/false bekler.

func runPipelineWithContinue(steps []PipeStep, doneMsg string,
	continueCh <-chan bool, send func(tea.Msg)) {

	for i, step := range steps {
		_ = i
		send(PipeStepStartMsg{Label: step.Label})

		start := time.Now()
		result := step.Run()
		elapsed := time.Since(start)

		if result.OK {
			send(PipeStepDoneMsg{Label: step.Label, Elapsed: elapsed})
			continue
		}

		// Hata — kullanıcıdan sor
		send(pipeStepFailWithChanMsg{
			Label:   step.Label,
			Err:     result.Err,
			Elapsed: elapsed,
		})

		// Kullanıcı cevabını bekle
		goOn := <-continueCh
		if !goOn {
			send(PipelineCancelledMsg{})
			return
		}
		// "devam" dedi — sonraki adıma geç (hatalı adım atlandı)
	}
	send(PipelineDoneMsg{Summary: doneMsg})
}

// pipeStepFailWithChanMsg — PipeStepFailMsg'in kanal içermeyen kopyası;
// gerçek kanal Model'de tutulur.
type pipeStepFailWithChanMsg struct {
	Label   string
	Err     string
	Elapsed time.Duration
}

// ── Yardımcı: ms formatı ─────────────────────────────────────────────────────

func fmtElapsed(d time.Duration) string {
	ms := d.Milliseconds()
	if ms < 1000 {
		return fmt.Sprintf("%dms", ms)
	}
	return fmt.Sprintf("%.1fs", d.Seconds())
}

// ── Model metodları ───────────────────────────────────────────────────────────

// startPipeline — pipeline'ı goroutine'de başlatır ve spinner'ı aktive eder.
// İlk PipeStepStartMsg'i tea.Cmd olarak döndürür; sonrakiler msgQueue'ya girer.
func (m *Model) startPipeline(steps []PipeStep, doneMsg string) tea.Cmd {
	ch := make(chan bool, 1)
	m.pipeline = pipelineState{
		active:     true,
		doneMsg:    doneMsg,
		continueCh: ch,
	}

	// tea.Cmd olarak döndür — bubbletea bunu goroutine'de çalıştırır
	return func() tea.Msg {
		msgCh := make(chan tea.Msg, 64)
		sendFn := func(msg tea.Msg) { msgCh <- msg }
		go func() {
			runPipelineWithContinue(steps, doneMsg, ch, sendFn)
			close(msgCh)
		}()
		// İlk mesajı döndür
		first, ok := <-msgCh
		if !ok {
			return PipelineDoneMsg{Summary: doneMsg}
		}
		// Geri kalan mesajları kuyrukla
		go func() {
			for msg := range msgCh {
				m.pipeline.msgQueue = append(m.pipeline.msgQueue, msg)
			}
		}()
		return first
	}
}

// handlePipelineMsg — gelen pipeline mesajını işler, yeni Cmd döndürür.
func (m *Model) handlePipelineMsg(msg tea.Msg) tea.Cmd {
	switch v := msg.(type) {

	case PipeStepStartMsg:
		m.pipeline.stepLabel = v.Label
		// Spinner satırı — mutable, her tick güncellenir (son satırı değiştiririz)
		m.appendSpinnerLine(v.Label)
		m.rebuildViewport()
		return spinTickCmd()

	case SpinTickMsg:
		if !m.pipeline.active || m.pipeline.awaitingContinue {
			return nil
		}
		m.pipeline.spinnerFrame = (m.pipeline.spinnerFrame + 1) % len(spinnerFrames)
		m.updateLastSpinnerLine(m.pipeline.stepLabel)
		m.rebuildViewport()
		return spinTickCmd()

	case PipeStepDoneMsg:
		elapsed := fmtElapsed(v.Elapsed)
		m.replaceLastLine(kindGreen,
			fmt.Sprintf("  v  %-38s  [%s]", v.Label, elapsed))
		m.rebuildViewport()
		// Kuyruktan sonraki mesajı al
		return m.drainPipelineQueue()

	case pipeStepFailWithChanMsg:
		elapsed := fmtElapsed(v.Elapsed)
		m.replaceLastLine(kindRed,
			fmt.Sprintf("  x  %-38s  [%s]", v.Label, elapsed))
		if hint := gitErrorHint(v.Err); hint != "" {
			m.bad(hint)
		} else {
			m.bad(lang.T("err_hint_generic"))
		}
		m.pipeline.awaitingContinue = true
		m.pipeline.failedLabel = v.Label
		m.pipeline.failedErr = v.Err
		m.warn(lang.T("pipe_fail_prompt"))
		m.rebuildViewport()
		return nil // kullanıcı girişini bekle

	case PipelineDoneMsg:
		m.pipeline.active = false
		m.pipeline.awaitingContinue = false
		if v.Summary != "" {
			m.ok(v.Summary)
		}
		m.rebuildViewport()
		return nil

	case PipelineCancelledMsg:
		m.pipeline.active = false
		m.pipeline.awaitingContinue = false
		m.warn(lang.T("pipe_cancelled"))
		m.rebuildViewport()
		return nil
	}
	return nil
}

// handlePipelineContinueInput — awaitingContinue modunda kullanıcı girdisini işler.
// true dönerse input tüketilmiştir (başka işlem yapma).
func (m *Model) handlePipelineContinueInput(val string) (consumed bool, cmd tea.Cmd) {
	if !m.pipeline.active || !m.pipeline.awaitingContinue {
		return false, nil
	}
	lower := strings.ToLower(strings.TrimSpace(val))
	switch lower {
	case lang.T("pipe_continue_word"), "devam", "continue", "c", "d":
		m.pipeline.awaitingContinue = false
		m.ok(lang.T("pipe_continuing"))
		m.pipeline.continueCh <- true
		cmd = m.drainPipelineQueue()
	case lang.T("pipe_abort_word"), "vazgec", "vazgeç", "abort", "v", "a":
		m.pipeline.awaitingContinue = false
		m.pipeline.continueCh <- false
		cmd = m.drainPipelineQueue()
	default:
		m.warn(lang.T("pipe_fail_prompt"))
	}
	m.rebuildViewport()
	return true, cmd
}

// drainPipelineQueue — msgQueue'daki bir sonraki mesajı döndüren Cmd.
func (m *Model) drainPipelineQueue() tea.Cmd {
	if len(m.pipeline.msgQueue) == 0 {
		return nil
	}
	msg := m.pipeline.msgQueue[0]
	m.pipeline.msgQueue = m.pipeline.msgQueue[1:]
	return func() tea.Msg { return msg }
}

// ── Spinner satır yönetimi ────────────────────────────────────────────────────

const spinnerLineMarker = "\x00spinner"

// appendSpinnerLine — output'a spinner satırı ekler.
func (m *Model) appendSpinnerLine(label string) {
	frame := spinnerFrames[m.pipeline.spinnerFrame%len(spinnerFrames)]
	m.appendLine(kindYellow, spinnerLineMarker+frame+"  "+label)
}

// updateLastSpinnerLine — son spinner satırını günceller.
func (m *Model) updateLastSpinnerLine(label string) {
	frame := spinnerFrames[m.pipeline.spinnerFrame%len(spinnerFrames)]
	for i := len(m.lines) - 1; i >= 0; i-- {
		if strings.HasPrefix(m.lines[i].content, spinnerLineMarker) {
			m.lines[i].content = spinnerLineMarker + frame + "  " + label
			return
		}
	}
}

// replaceLastLine — son spinner satırını kalıcı sonuçla değiştirir.
func (m *Model) replaceLastLine(kind outputKind, content string) {
	for i := len(m.lines) - 1; i >= 0; i-- {
		if strings.HasPrefix(m.lines[i].content, spinnerLineMarker) {
			m.lines[i] = outputLine{kind: kind, content: content}
			return
		}
	}
	// Spinner satırı yoksa normal ekle
	m.appendLine(kind, content)
}
