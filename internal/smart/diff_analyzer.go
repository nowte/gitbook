package smart

import (
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// ── Diff Analiz Motoru ────────────────────────────────────────────────────────
//
// Git diff çıktısını parse ederek insan tarafından okunabilir
// bir özet üretir. Push öncesi kullanıcıya ne değiştiğini açıklar.

// FileChange tek bir dosyadaki değişikliği temsil eder.
type FileChange struct {
	Path      string
	OldPath   string // rename durumunda önceki yol
	Status    string // "modified", "added", "deleted", "renamed", "copied"
	Additions int
	Deletions int
	Language  string
	Category  string // "source", "config", "docs", "test", "asset"
}

// DiffSummary tüm diff'in özet analizini tutar.
type DiffSummary struct {
	Files      []FileChange
	TotalAdded int
	TotalDel   int

	// Kategorize edilmiş sayılar
	SourceFiles int
	TestFiles   int
	ConfigFiles int
	DocFiles    int

	// Dikkat çekici bulgular
	Warnings []string
	// Önemli değişiklikler (büyük ekleme/silme, kritik dosyalar)
	Highlights []string
}

// AnalyzeDiff git diff --stat veya --numstat çıktısını analiz eder.
// numstat formatı: "<ekleme>\t<silme>\t<dosya>"
func AnalyzeDiff(numstatOutput string) *DiffSummary {
	summary := &DiffSummary{}
	if strings.TrimSpace(numstatOutput) == "" {
		return summary
	}

	for _, line := range strings.Split(numstatOutput, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fc := parseNumstatLine(line)
		if fc == nil {
			continue
		}
		summary.Files = append(summary.Files, *fc)
		summary.TotalAdded += fc.Additions
		summary.TotalDel += fc.Deletions

		switch fc.Category {
		case "test":
			summary.TestFiles++
		case "config":
			summary.ConfigFiles++
		case "docs":
			summary.DocFiles++
		default:
			summary.SourceFiles++
		}
	}

	summary.generateWarnings()
	summary.generateHighlights()
	return summary
}

// AnalyzeDiffStat git diff --stat çıktısını analiz eder.
// stat formatı: " dosya.go | 42 +++---"
func AnalyzeDiffStat(statOutput string) *DiffSummary {
	summary := &DiffSummary{}
	if strings.TrimSpace(statOutput) == "" {
		return summary
	}

	for _, line := range strings.Split(statOutput, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "...") {
			continue
		}
		// " dosya | 42 +++---" formatı
		parts := strings.Split(line, "|")
		if len(parts) != 2 {
			continue
		}
		path := strings.TrimSpace(parts[0])
		if path == "" {
			continue
		}
		fc := &FileChange{
			Path:     path,
			Status:   "modified",
			Language: detectLanguage(path),
			Category: categorizeFile(path),
		}

		changeStr := strings.TrimSpace(parts[1])
		// "42 +++---" -> sayıyı çıkar
		fields := strings.Fields(changeStr)
		if len(fields) > 0 {
			n, err := strconv.Atoi(fields[0])
			if err == nil {
				// +/- işaretlerine göre tahmin et
				plusMinus := ""
				if len(fields) > 1 {
					plusMinus = fields[1]
				}
				plusCount := strings.Count(plusMinus, "+")
				minusCount := strings.Count(plusMinus, "-")
				total := plusCount + minusCount
				if total > 0 {
					fc.Additions = n * plusCount / total
					fc.Deletions = n - fc.Additions
				} else {
					fc.Additions = n
				}
			}
		}

		summary.Files = append(summary.Files, *fc)
		summary.TotalAdded += fc.Additions
		summary.TotalDel += fc.Deletions

		switch fc.Category {
		case "test":
			summary.TestFiles++
		case "config":
			summary.ConfigFiles++
		case "docs":
			summary.DocFiles++
		default:
			summary.SourceFiles++
		}
	}

	summary.generateWarnings()
	summary.generateHighlights()
	return summary
}

// parseNumstatLine tek bir numstat satırını parse eder.
func parseNumstatLine(line string) *FileChange {
	parts := strings.Fields(line)
	if len(parts) < 3 {
		return nil
	}

	fc := &FileChange{}

	// Binary dosyalar "-" gösterir
	if parts[0] == "-" {
		fc.Additions = 0
		fc.Deletions = 0
	} else {
		add, err := strconv.Atoi(parts[0])
		if err != nil {
			return nil
		}
		del, err := strconv.Atoi(parts[1])
		if err != nil {
			return nil
		}
		fc.Additions = add
		fc.Deletions = del
	}

	// Rename formatı: "eski.go => yeni.go" veya "{eski => yeni}/dosya.go"
	path := strings.Join(parts[2:], " ")
	if strings.Contains(path, " => ") {
		fc.Status = "renamed"
		pieces := strings.SplitN(path, " => ", 2)
		fc.OldPath = strings.Trim(pieces[0], "{}")
		fc.Path = strings.Trim(pieces[1], "{}")
	} else {
		fc.Path = path
		fc.Status = "modified"
	}

	fc.Language = detectLanguage(fc.Path)
	fc.Category = categorizeFile(fc.Path)
	return fc
}

// detectLanguage dosya uzantısından dil adını döner.
func detectLanguage(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	langMap := map[string]string{
		".go":    "Go",
		".js":    "JavaScript",
		".ts":    "TypeScript",
		".jsx":   "React",
		".tsx":   "React/TS",
		".py":    "Python",
		".rs":    "Rust",
		".java":  "Java",
		".kt":    "Kotlin",
		".swift": "Swift",
		".c":     "C",
		".cpp":   "C++",
		".h":     "C/C++",
		".cs":    "C#",
		".rb":    "Ruby",
		".php":   "PHP",
		".sh":    "Shell",
		".yaml":  "YAML",
		".yml":   "YAML",
		".json":  "JSON",
		".toml":  "TOML",
		".md":    "Markdown",
		".sql":   "SQL",
		".html":  "HTML",
		".css":   "CSS",
		".scss":  "SCSS",
	}
	if lang, ok := langMap[ext]; ok {
		return lang
	}
	return ""
}

// categorizeFile dosya yolu ve adına göre kategori döner.
func categorizeFile(path string) string {
	lower := strings.ToLower(path)
	base := strings.ToLower(filepath.Base(path))

	// Test dosyaları
	if strings.Contains(lower, "_test.") ||
		strings.Contains(lower, ".test.") ||
		strings.Contains(lower, "/test/") ||
		strings.Contains(lower, "/tests/") ||
		strings.Contains(lower, "__tests__") ||
		strings.HasSuffix(base, "_test.go") ||
		strings.HasPrefix(base, "test_") {
		return "test"
	}

	// Konfigürasyon dosyaları
	configFiles := map[string]bool{
		".env": true, ".gitignore": true, ".gitbookrc": true,
		"go.mod": true, "go.sum": true, "package.json": true,
		"package-lock.json": true, "yarn.lock": true, "cargo.toml": true,
		"dockerfile": true, "docker-compose.yml": true,
		".eslintrc": true, ".prettierrc": true, "tsconfig.json": true,
		"makefile": true, "cmakelists.txt": true,
	}
	if configFiles[base] {
		return "config"
	}
	ext := strings.ToLower(filepath.Ext(path))
	if ext == ".yaml" || ext == ".yml" || ext == ".toml" || ext == ".ini" || ext == ".cfg" {
		return "config"
	}

	// Dokümantasyon
	if ext == ".md" || ext == ".rst" || ext == ".txt" ||
		strings.Contains(lower, "/docs/") ||
		base == "readme.md" || base == "changelog.md" || base == "license" {
		return "docs"
	}

	return "source"
}

// generateWarnings olası sorunları saptar.
func (s *DiffSummary) generateWarnings() {
	// Çok büyük değişiklik
	if len(s.Files) > 20 {
		s.Warnings = append(s.Warnings,
			fmt.Sprintf("(!)  Çok fazla dosya değişiyor (%d). Daha küçük commit'lere bölmeyi düşün.", len(s.Files)))
	}

	// Hassas dosyalar
	for _, f := range s.Files {
		base := strings.ToLower(filepath.Base(f.Path))
		if base == ".env" || base == ".env.local" || base == ".env.production" {
			s.Warnings = append(s.Warnings,
				fmt.Sprintf("[!!] Hassas dosya değişiyor: %s — gizli bilgi içerdiğinden emin ol!", f.Path))
		}
		if strings.HasSuffix(base, ".key") || strings.HasSuffix(base, ".pem") ||
			strings.HasSuffix(base, ".p12") || strings.HasSuffix(base, ".pfx") {
			s.Warnings = append(s.Warnings,
				fmt.Sprintf("[!!] Anahtar/sertifika dosyası değişiyor: %s", f.Path))
		}
		// go.sum veya package-lock.json tek başına değişiyorsa
		if (base == "go.sum" || base == "package-lock.json" || base == "yarn.lock") &&
			len(s.Files) == 1 {
			s.Warnings = append(s.Warnings,
				"[!] Sadece bağımlılık lock dosyası değişiyor. Kaynak kodu değişikliği var mı?")
		}
	}
}

// generateHighlights önemli noktaları üretir.
func (s *DiffSummary) generateHighlights() {
	if len(s.Files) == 0 {
		return
	}

	// En çok değişen dosyaları bul
	sorted := make([]FileChange, len(s.Files))
	copy(sorted, s.Files)
	sort.Slice(sorted, func(i, j int) bool {
		return (sorted[i].Additions + sorted[i].Deletions) > (sorted[j].Additions + sorted[j].Deletions)
	})

	if len(sorted) > 0 {
		top := sorted[0]
		total := top.Additions + top.Deletions
		if total > 50 {
			s.Highlights = append(s.Highlights,
				fmt.Sprintf("[~] En çok değişen: %s (+%d/-%d satır)", top.Path, top.Additions, top.Deletions))
		}
	}

	// Dil dağılımı
	langCount := make(map[string]int)
	for _, f := range s.Files {
		if f.Language != "" {
			langCount[f.Language]++
		}
	}
	if len(langCount) > 1 {
		langs := make([]string, 0, len(langCount))
		for l := range langCount {
			langs = append(langs, l)
		}
		sort.Strings(langs)
		s.Highlights = append(s.Highlights,
			fmt.Sprintf("[A] Diller: %s", strings.Join(langs, ", ")))
	}
}

// Format okunabilir bir özet metni döner.
func (s *DiffSummary) Format() string {
	if len(s.Files) == 0 {
		return "  Değişiklik bulunamadı."
	}

	var sb strings.Builder

	// Başlık satırı
	sb.WriteString(fmt.Sprintf("  [%] %d dosya  •  +%d  -%d satır\n",
		len(s.Files), s.TotalAdded, s.TotalDel))

	// Kategori özeti
	if s.SourceFiles > 0 || s.TestFiles > 0 || s.ConfigFiles > 0 || s.DocFiles > 0 {
		var cats []string
		if s.SourceFiles > 0 {
			cats = append(cats, fmt.Sprintf("%d kaynak", s.SourceFiles))
		}
		if s.TestFiles > 0 {
			cats = append(cats, fmt.Sprintf("%d test", s.TestFiles))
		}
		if s.ConfigFiles > 0 {
			cats = append(cats, fmt.Sprintf("%d config", s.ConfigFiles))
		}
		if s.DocFiles > 0 {
			cats = append(cats, fmt.Sprintf("%d dokümantasyon", s.DocFiles))
		}
		sb.WriteString("  [/] " + strings.Join(cats, "  •  ") + "\n")
	}

	// Dosya listesi (max 8)
	sb.WriteString("\n")
	limit := len(s.Files)
	if limit > 8 {
		limit = 8
	}
	for _, f := range s.Files[:limit] {
		icon := fileIcon(f.Status)
		langTag := ""
		if f.Language != "" {
			langTag = " [" + f.Language + "]"
		}
		sb.WriteString(fmt.Sprintf("  %s %s%s", icon, f.Path, langTag))
		if f.Additions > 0 || f.Deletions > 0 {
			sb.WriteString(fmt.Sprintf("  +%d/-%d", f.Additions, f.Deletions))
		}
		sb.WriteString("\n")
	}
	if len(s.Files) > 8 {
		sb.WriteString(fmt.Sprintf("  ... ve %d dosya daha\n", len(s.Files)-8))
	}

	// Vurgular
	if len(s.Highlights) > 0 {
		sb.WriteString("\n")
		for _, h := range s.Highlights {
			sb.WriteString("  " + h + "\n")
		}
	}

	// Uyarılar
	if len(s.Warnings) > 0 {
		sb.WriteString("\n")
		for _, w := range s.Warnings {
			sb.WriteString("  " + w + "\n")
		}
	}

	return sb.String()
}

func fileIcon(status string) string {
	switch status {
	case "added":
		return "[+]"
	case "deleted":
		return "[-] "
	case "renamed":
		return "[e] "
	case "copied":
		return "[=]"
	default:
		return "[~]"
	}
}
