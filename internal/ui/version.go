package ui

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

const GitbookVersion = "1.02.1"

// githubRelease is the shape of the GitHub releases API response.
type githubRelease struct {
	TagName string `json:"tag_name"`
}

// checkLatestVersion fetches the latest release tag from GitHub.
// Returns ("", err) on failure; (tag, nil) on success.
func checkLatestVersion() (string, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get("https://api.github.com/repos/nowte/gitbook/releases/latest")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API returned %d", resp.StatusCode)
	}

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", err
	}
	return strings.TrimSpace(release.TagName), nil
}

// UpdateAvailable returns true if latest != current version.
// "v1.02.1" ve "1.02.1" gibi prefix farklılıklarını normalize eder.
func UpdateAvailable(latest string) bool {
	if latest == "" {
		return false
	}
	normalize := func(s string) string {
		return strings.TrimPrefix(strings.TrimSpace(s), "v")
	}
	return normalize(latest) != normalize(GitbookVersion)
}

// buildStartupStatus returns the welcome message shown when chat opens.
func buildStartupStatus() string {
	var sb strings.Builder
	sb.WriteString("gitBook ready.\n")
	sb.WriteString("Type /help for available commands, or / to browse them.")
	return sb.String()
}

// openBrowser opens the given URL in the default system browser.
func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default: // linux
		cmd = exec.Command("xdg-open", url)
	}
	_ = cmd.Start()
}
