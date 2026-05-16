package smart

import (
	"fmt"
	"path/filepath"
	"strings"
)

// ── Commit Mesajı Öneri Motoru ────────────────────────────────────────────────
//
// Diff analizinden yola çıkarak Conventional Commits formatında
// commit mesajı önerileri üretir.
// Format: <type>(<scope>): <description>

// CommitSuggestion tek bir commit mesajı önerisini temsil eder.
type CommitSuggestion struct {
	Message     string // Tam mesaj
	Type        string // feat, fix, docs, refactor, test, chore, style
	Scope       string // Etkilenen modül/paket
	Description string // Kısa açıklama
	Confidence  int    // 1-3: düşük / orta / yüksek güven
	Reasoning   string // Neden bu öneri üretildi
}

// SuggestCommitMessages diff özetinden commit önerileri üretir.
// Birden fazla alternatif döner, en iyi tahmin başta gelir.
func SuggestCommitMessages(summary *DiffSummary) []CommitSuggestion {
	if summary == nil || len(summary.Files) == 0 {
		return nil
	}

	var suggestions []CommitSuggestion

	commitType := detectCommitType(summary)
	scope := detectScope(summary)
	desc := buildDescription(summary)

	// Birincil öneri
	primary := CommitSuggestion{
		Type:        commitType,
		Scope:       scope,
		Description: desc,
		Confidence:  calcConfidence(summary, commitType),
		Reasoning:   buildReasoning(summary, commitType),
	}
	primary.Message = formatConventional(primary)
	suggestions = append(suggestions, primary)

	// Alternatif: daha kısa/genel bir versiyon
	if len(summary.Files) > 1 {
		alt := buildAlternative(summary, commitType, scope)
		if alt.Message != primary.Message {
			suggestions = append(suggestions, alt)
		}
	}

	// Dosya adı bazlı sade öneri (her zaman son seçenek olarak)
	simple := buildSimple(summary)
	if simple.Message != primary.Message {
		suggestions = append(suggestions, simple)
	}

	return suggestions
}

// detectCommitType değişikliklerin türünü tahmin eder.
func detectCommitType(s *DiffSummary) string {
	// Sadece testler değiştiyse
	if s.TestFiles > 0 && s.SourceFiles == 0 && s.ConfigFiles == 0 {
		return "test"
	}

	// Sadece dokümantasyon değiştiyse
	if s.DocFiles > 0 && s.SourceFiles == 0 && s.TestFiles == 0 {
		return "docs"
	}

	// Sadece config değiştiyse
	if s.ConfigFiles > 0 && s.SourceFiles == 0 && s.TestFiles == 0 {
		return "chore"
	}

	// Ağırlıklı olarak silme varsa refactor olabilir
	if s.TotalDel > s.TotalAdded*2 {
		return "refactor"
	}

	// Yeni dosya eklenmişse feat
	for _, f := range s.Files {
		if f.Status == "added" {
			return "feat"
		}
	}

	// Büyük eklemeler var ve yeni kaynak dosya yoksa
	if s.TotalAdded > 100 {
		return "feat"
	}

	// Küçük değişiklikler genellikle fix ya da refactor
	if s.TotalAdded+s.TotalDel < 20 {
		return "fix"
	}

	return "feat"
}

// detectScope hangi modülün değiştiğini bulmaya çalışır.
func detectScope(s *DiffSummary) string {
	if len(s.Files) == 0 {
		return ""
	}

	// Tek dosya → dosyanın adı scope olabilir
	if len(s.Files) == 1 {
		base := filepath.Base(s.Files[0].Path)
		ext := filepath.Ext(base)
		return strings.TrimSuffix(base, ext)
	}

	// Ortak dizin bul
	commonDir := findCommonDir(s.Files)
	if commonDir != "" && commonDir != "." {
		// İç içe dizinlerin son anlamlı parçasını al
		parts := strings.Split(commonDir, "/")
		for i := len(parts) - 1; i >= 0; i-- {
			p := parts[i]
			if p != "" && p != "." && p != "internal" && p != "src" && p != "pkg" {
				return p
			}
		}
		return parts[len(parts)-1]
	}

	// Dosyaların büyük çoğunluğu aynı kategorideyse
	if s.TestFiles > s.SourceFiles {
		return "tests"
	}
	if s.ConfigFiles > s.SourceFiles {
		return "config"
	}

	return ""
}

// buildDescription değişikliğin ne olduğunu açıklar.
func buildDescription(s *DiffSummary) string {
	if len(s.Files) == 1 {
		f := s.Files[0]
		base := filepath.Base(f.Path)
		ext := filepath.Ext(base)
		name := strings.TrimSuffix(base, ext)

		switch f.Status {
		case "added":
			return "add " + toSnakeHuman(name)
		case "deleted":
			return "remove " + toSnakeHuman(name)
		case "renamed":
			return "rename " + filepath.Base(f.OldPath) + " to " + base
		default:
			if f.Deletions > f.Additions*3 {
				return "simplify " + toSnakeHuman(name)
			}
			return "update " + toSnakeHuman(name)
		}
	}

	// Çok dosya
	addedCount := 0
	deletedCount := 0
	for _, f := range s.Files {
		if f.Status == "added" {
			addedCount++
		} else if f.Status == "deleted" {
			deletedCount++
		}
	}

	if addedCount > 0 && deletedCount == 0 {
		return fmt.Sprintf("add %d new files", addedCount)
	}
	if deletedCount > 0 && addedCount == 0 {
		return fmt.Sprintf("remove %d files", deletedCount)
	}

	scope := detectScope(s)
	if scope != "" {
		return fmt.Sprintf("update %s", toSnakeHuman(scope))
	}

	return fmt.Sprintf("update %d files", len(s.Files))
}

// buildAlternative daha kısa genel bir alternatif öneri üretir.
func buildAlternative(s *DiffSummary, commitType, scope string) CommitSuggestion {
	var desc string
	if scope != "" {
		desc = fmt.Sprintf("update %s module", scope)
	} else {
		desc = fmt.Sprintf("update %d files across %d categories",
			len(s.Files), countCategories(s))
	}

	alt := CommitSuggestion{
		Type:        commitType,
		Scope:       scope,
		Description: desc,
		Confidence:  1,
		Reasoning:   "Genel kapsam önerisi",
	}
	alt.Message = formatConventional(alt)
	return alt
}

// buildSimple sade, dosya adı bazlı bir öneri üretir.
func buildSimple(s *DiffSummary) CommitSuggestion {
	var desc string
	if len(s.Files) == 1 {
		desc = "update " + s.Files[0].Path
	} else {
		desc = fmt.Sprintf("update %d files", len(s.Files))
	}

	simple := CommitSuggestion{
		Type:        "chore",
		Description: desc,
		Confidence:  1,
		Reasoning:   "Temel dosya listesi önerisi",
	}
	simple.Message = simple.Type + ": " + simple.Description
	return simple
}

// formatConventional Conventional Commits formatında mesaj üretir.
func formatConventional(s CommitSuggestion) string {
	if s.Scope != "" {
		return fmt.Sprintf("%s(%s): %s", s.Type, s.Scope, s.Description)
	}
	return fmt.Sprintf("%s: %s", s.Type, s.Description)
}

// calcConfidence güven puanı hesaplar (1-3).
func calcConfidence(s *DiffSummary, commitType string) int {
	// Tek dosya → yüksek güven
	if len(s.Files) == 1 {
		return 3
	}
	// Homojen kategori → orta güven
	total := s.SourceFiles + s.TestFiles + s.ConfigFiles + s.DocFiles
	dominant := maxOf(s.SourceFiles, s.TestFiles, s.ConfigFiles, s.DocFiles)
	if total > 0 && dominant*100/total >= 80 {
		return 2
	}
	return 1
}

// buildReasoning neden bu tahmine varıldığını açıklar.
func buildReasoning(s *DiffSummary, commitType string) string {
	switch commitType {
	case "test":
		return fmt.Sprintf("%d test dosyası değişiyor, kaynak dosya yok", s.TestFiles)
	case "docs":
		return fmt.Sprintf("%d dokümantasyon dosyası değişiyor", s.DocFiles)
	case "chore":
		return fmt.Sprintf("%d konfigürasyon dosyası değişiyor", s.ConfigFiles)
	case "refactor":
		return fmt.Sprintf("Silinen satır (%d) eklenen satırın (%d) 2 katından fazla", s.TotalDel, s.TotalAdded)
	case "feat":
		addedCount := 0
		for _, f := range s.Files {
			if f.Status == "added" {
				addedCount++
			}
		}
		if addedCount > 0 {
			return fmt.Sprintf("%d yeni dosya ekleniyor", addedCount)
		}
		return fmt.Sprintf("+%d satır ekleniyor (%d dosyada)", s.TotalAdded, len(s.Files))
	case "fix":
		return fmt.Sprintf("Küçük değişiklik (%d satır toplam)", s.TotalAdded+s.TotalDel)
	}
	return ""
}

// ── Yardımcı fonksiyonlar ─────────────────────────────────────────────────────

// findCommonDir dosyaların ortak dizinini bulur.
func findCommonDir(files []FileChange) string {
	if len(files) == 0 {
		return ""
	}

	dirCount := make(map[string]int)
	for _, f := range files {
		dir := filepath.Dir(f.Path)
		for dir != "." && dir != "/" {
			dirCount[dir]++
			dir = filepath.Dir(dir)
		}
	}

	// Tüm dosyaların bulunduğu en derin ortak dizini bul
	bestDir := ""
	bestDepth := 0
	for dir, count := range dirCount {
		if count == len(files) {
			depth := strings.Count(dir, "/")
			if depth > bestDepth {
				bestDir = dir
				bestDepth = depth
			}
		}
	}
	return bestDir
}

// toSnakeHuman snake_case veya camelCase'i okunabilir yapar.
func toSnakeHuman(s string) string {
	// camelCase → kelimelerine ayır
	var result []rune
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result = append(result, ' ')
		}
		result = append(result, r)
	}
	s = string(result)

	// snake_case'i temizle
	s = strings.ReplaceAll(s, "_", " ")
	s = strings.ReplaceAll(s, "-", " ")
	return strings.ToLower(strings.TrimSpace(s))
}

func countCategories(s *DiffSummary) int {
	count := 0
	if s.SourceFiles > 0 {
		count++
	}
	if s.TestFiles > 0 {
		count++
	}
	if s.ConfigFiles > 0 {
		count++
	}
	if s.DocFiles > 0 {
		count++
	}
	return count
}

func maxOf(vals ...int) int {
	m := 0
	for _, v := range vals {
		if v > m {
			m = v
		}
	}
	return m
}

// FormatSuggestions commit önerilerini ekrana yazdırılacak formata çevirir.
func FormatSuggestions(suggestions []CommitSuggestion) string {
	if len(suggestions) == 0 {
		return "  Öneri üretilemedi."
	}

	var sb strings.Builder
	sb.WriteString("  [!] Commit Mesajı Önerileri:\n\n")

	confidenceLabel := map[int]string{
		1: "•",
		2: "o",
		3: "*",
	}

	for i, s := range suggestions {
		label := confidenceLabel[s.Confidence]
		if i == 0 {
			sb.WriteString(fmt.Sprintf("  %s  \"%s\"  ← önerilen\n", label, s.Message))
		} else {
			sb.WriteString(fmt.Sprintf("  %s  \"%s\"\n", label, s.Message))
		}
		if s.Reasoning != "" {
			sb.WriteString(fmt.Sprintf("     ↳ %s\n", s.Reasoning))
		}
	}

	sb.WriteString("\n  * yüksek güven  o orta güven  • düşük güven\n")
	return sb.String()
}
