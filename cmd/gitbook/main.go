package main

import (
	"fmt"
	"os"
	"os/signal"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/nowte/gitbook/internal/config"
	"github.com/nowte/gitbook/internal/lang"
	"github.com/nowte/gitbook/internal/ui"
)

func main() {
	// Initialize language system
	lang.InitLanguage()

	// Initialize logger for audit trail
	if err := ui.InitLogger(); err != nil {
		fmt.Printf("Warning: Failed to initialize logger: %v\n", err)
		// Continue without logging rather than exit
	}

	// Advisory instance lock — warn if another gitBook is running on this repo
	if ok, existingPID := config.AcquireInstanceLock(); !ok {
		fmt.Printf("[!] Another gitBook instance (PID %s) may be running on this repository.\n    Running multiple instances concurrently can cause conflicts.\n    Press Enter to continue anyway, or Ctrl+C to exit.\n", existingPID)
		var buf [1]byte
		_, _ = os.Stdin.Read(buf[:])
	}

	// Setup cleanup on exit
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c
		config.ReleaseInstanceLock()
		ui.CleanupLogger()
		os.Exit(0)
	}()

	p := tea.NewProgram(
		ui.InitialModel(),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)
	if _, err := p.Run(); err != nil {
		fmt.Printf("Hata: %v", err)
	}

	// Cleanup on normal exit
	config.ReleaseInstanceLock()
	ui.CleanupLogger()
}
