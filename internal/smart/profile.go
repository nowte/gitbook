package smart

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ── Proje Profil Sistemi ──────────────────────────────────────────────────────
//
// .gitbook/profiles.json içinde proje bazlı ayarları saklar.
// Her proje farklı push kurallarına, onay gereksinimlerine ve
// otomatik davranışlara sahip olabilir.

const profilesFile = "profiles.json"

// ProfileType profilin hangi kullanım senaryosuna ait olduğunu belirtir.
type ProfileType string

const (
	ProfileWork     ProfileType = "work"
	ProfilePersonal ProfileType = "personal"
	ProfileExplore  ProfileType = "explore" // deney/playground
	ProfileOSS      ProfileType = "oss"     // açık kaynak
)

// PushPolicy push yapılırken uygulanacak kuralları tanımlar.
type PushPolicy struct {
	// Bu branch'lere push öncesi onay sor
	ConfirmBranches []string `json:"confirm_branches"`
	// Bu branch'lere push tamamen engellenir
	BlockedBranches []string `json:"blocked_branches"`
	// Push öncesi diff özeti göster
	ShowDiffOnPush bool `json:"show_diff_on_push"`
	// Push öncesi commit mesajı öneri sun
	SuggestCommitMsg bool `json:"suggest_commit_msg"`
}

// CommitPolicy commit yapılırken uygulanacak kuralları tanımlar.
type CommitPolicy struct {
	// Conventional Commits formatını zorunlu kıl
	RequireConventional bool `json:"require_conventional"`
	// Minimum commit mesajı uzunluğu
	MinMessageLength int `json:"min_message_length"`
	// Maksimum commit mesajı uzunluğu
	MaxMessageLength int `json:"max_message_length"`
	// Commit öncesi staged diff göster
	ShowStagedOnCommit bool `json:"show_staged_on_commit"`
}

// SmartFeatures akıllı sistemlerin açık/kapalı durumlarını tutar.
type SmartFeatures struct {
	DiffAnalysis     bool `json:"diff_analysis"`
	CommitSuggestion bool `json:"commit_suggestion"`
	GitignoreGen     bool `json:"gitignore_gen"`
}

// Profile tek bir proje profilini temsil eder.
type Profile struct {
	Name        string        `json:"name"`
	Type        ProfileType   `json:"type"`
	Description string        `json:"description,omitempty"`
	Push        PushPolicy    `json:"push"`
	Commit      CommitPolicy  `json:"commit"`
	Smart       SmartFeatures `json:"smart"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
}

// ProfileStore tüm profilleri ve aktif profili tutar.
type ProfileStore struct {
	ActiveProfile string             `json:"active_profile"`
	Profiles      map[string]Profile `json:"profiles"`
}

// ── Hazır şablon profiller ────────────────────────────────────────────────────

// DefaultProfiles sık kullanılan profil şablonlarını döner.
func DefaultProfiles() map[string]Profile {
	now := time.Now().UTC()
	return map[string]Profile{
		"work": {
			Name:        "work",
			Type:        ProfileWork,
			Description: "İş projesi — main/master'a push onay gerektirir",
			Push: PushPolicy{
				ConfirmBranches:  []string{"main", "master", "release", "production"},
				BlockedBranches:  []string{"production"},
				ShowDiffOnPush:   true,
				SuggestCommitMsg: true,
			},
			Commit: CommitPolicy{
				RequireConventional: true,
				MinMessageLength:    10,
				MaxMessageLength:    72,
				ShowStagedOnCommit:  true,
			},
			Smart: SmartFeatures{
				DiffAnalysis:     true,
				CommitSuggestion: true,
				GitignoreGen:     true,
			},
			CreatedAt: now,
			UpdatedAt: now,
		},
		"personal": {
			Name:        "personal",
			Type:        ProfilePersonal,
			Description: "Kişisel proje — esnek kurallar",
			Push: PushPolicy{
				ConfirmBranches:  []string{"main"},
				BlockedBranches:  []string{},
				ShowDiffOnPush:   true,
				SuggestCommitMsg: true,
			},
			Commit: CommitPolicy{
				RequireConventional: false,
				MinMessageLength:    3,
				MaxMessageLength:    72,
				ShowStagedOnCommit:  false,
			},
			Smart: SmartFeatures{
				DiffAnalysis:     true,
				CommitSuggestion: true,
				GitignoreGen:     true,
			},
			CreatedAt: now,
			UpdatedAt: now,
		},
		"explore": {
			Name:        "explore",
			Type:        ProfileExplore,
			Description: "Deney/playground — hiçbir engel yok",
			Push: PushPolicy{
				ConfirmBranches:  []string{},
				BlockedBranches:  []string{},
				ShowDiffOnPush:   false,
				SuggestCommitMsg: false,
			},
			Commit: CommitPolicy{
				RequireConventional: false,
				MinMessageLength:    1,
				MaxMessageLength:    200,
				ShowStagedOnCommit:  false,
			},
			Smart: SmartFeatures{
				DiffAnalysis:     false,
				CommitSuggestion: false,
				GitignoreGen:     true,
			},
			CreatedAt: now,
			UpdatedAt: now,
		},
		"oss": {
			Name:        "oss",
			Type:        ProfileOSS,
			Description: "Açık kaynak — sıkı Conventional Commits, korumalı main",
			Push: PushPolicy{
				ConfirmBranches:  []string{"main", "master"},
				BlockedBranches:  []string{"main", "master"},
				ShowDiffOnPush:   true,
				SuggestCommitMsg: true,
			},
			Commit: CommitPolicy{
				RequireConventional: true,
				MinMessageLength:    15,
				MaxMessageLength:    72,
				ShowStagedOnCommit:  true,
			},
			Smart: SmartFeatures{
				DiffAnalysis:     true,
				CommitSuggestion: true,
				GitignoreGen:     true,
			},
			CreatedAt: now,
			UpdatedAt: now,
		},
	}
}

// ── Yükleme / Kaydetme ────────────────────────────────────────────────────────

func profilesPath() string {
	return filepath.Join(".gitbook", profilesFile)
}

// LoadProfiles mevcut profiles.json dosyasını okur.
// Yoksa varsayılan mağazayı döner.
func LoadProfiles() (*ProfileStore, error) {
	data, err := os.ReadFile(profilesPath())
	if err != nil {
		if os.IsNotExist(err) {
			return defaultStore(), nil
		}
		return nil, fmt.Errorf("profiller okunamadı: %w", err)
	}

	var store ProfileStore
	if err := json.Unmarshal(data, &store); err != nil {
		return nil, fmt.Errorf("profiller parse edilemedi: %w", err)
	}
	if store.Profiles == nil {
		store.Profiles = make(map[string]Profile)
	}
	return &store, nil
}

// SaveProfiles mağazayı profiles.json'a yazar.
func SaveProfiles(store *ProfileStore) error {
	if err := os.MkdirAll(".gitbook", 0755); err != nil {
		return fmt.Errorf(".gitbook dizini oluşturulamadı: %w", err)
	}
	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return fmt.Errorf("profiller yazılamadı: %w", err)
	}
	return os.WriteFile(profilesPath(), data, 0644)
}

func defaultStore() *ProfileStore {
	return &ProfileStore{
		ActiveProfile: "personal",
		Profiles:      DefaultProfiles(),
	}
}

// ── Aktif Profil İşlemleri ────────────────────────────────────────────────────

// GetActiveProfile aktif profili döner.
// .gitbook yoksa ya da hata varsa explore profilini döner (sıfır engel).
func GetActiveProfile() *Profile {
	store, err := LoadProfiles()
	if err != nil {
		p := DefaultProfiles()["explore"]
		return &p
	}
	p, ok := store.Profiles[store.ActiveProfile]
	if !ok {
		p = DefaultProfiles()["personal"]
	}
	return &p
}

// SetActiveProfile aktif profili değiştirir.
func SetActiveProfile(name string) error {
	store, err := LoadProfiles()
	if err != nil {
		return err
	}
	if _, ok := store.Profiles[name]; !ok {
		return fmt.Errorf("'%s' adlı profil bulunamadı. Mevcut profiller: %s",
			name, strings.Join(ProfileNames(store), ", "))
	}
	store.ActiveProfile = name
	return SaveProfiles(store)
}

// ── Push Politikası Kontrolleri ───────────────────────────────────────────────

// PushCheckResult push işleminin geçip geçmeyeceğini söyler.
type PushCheckResult struct {
	Allowed        bool
	NeedsConfirm   bool
	Message        string
	ShowDiff       bool
	SuggestMessage bool
}

// CheckPush aktif profilin push politikasına göre sonuç döner.
func CheckPush(branch string) PushCheckResult {
	p := GetActiveProfile()

	// Engellenen branch?
	for _, b := range p.Push.BlockedBranches {
		if b == branch {
			return PushCheckResult{
				Allowed: false,
				Message: fmt.Sprintf("[x] '%s' profili '%s' branch'ine push'u engelliyor.\n   Doğrudan push yerine Pull Request kullanın.", p.Name, branch),
			}
		}
	}

	// Onay gereken branch?
	needsConfirm := false
	for _, b := range p.Push.ConfirmBranches {
		if b == branch {
			needsConfirm = true
			break
		}
	}

	msg := ""
	if needsConfirm {
		msg = fmt.Sprintf("(!)  '%s' korumalı bir branch. '%s' profilinde onay gerekiyor.", branch, p.Name)
	}

	return PushCheckResult{
		Allowed:        true,
		NeedsConfirm:   needsConfirm,
		Message:        msg,
		ShowDiff:       p.Push.ShowDiffOnPush,
		SuggestMessage: p.Push.SuggestCommitMsg,
	}
}

// CheckCommitMessage aktif profilin commit politikasına göre mesajı doğrular.
func CheckCommitMessage(msg string) error {
	p := GetActiveProfile()
	trimmed := strings.TrimSpace(msg)

	if len(trimmed) < p.Commit.MinMessageLength {
		return fmt.Errorf("commit mesajı çok kısa (min %d karakter, şu an %d)",
			p.Commit.MinMessageLength, len(trimmed))
	}
	if len(trimmed) > p.Commit.MaxMessageLength {
		return fmt.Errorf("commit mesajı çok uzun (max %d karakter, şu an %d)",
			p.Commit.MaxMessageLength, len(trimmed))
	}
	if p.Commit.RequireConventional {
		if !isConventionalCommit(trimmed) {
			return fmt.Errorf(
				"'%s' profili Conventional Commits formatı gerektiriyor.\n   Örnek: feat(auth): add login page\n   Tipler: feat, fix, docs, style, refactor, test, chore",
				p.Name)
		}
	}
	return nil
}

// isConventionalCommit mesajın Conventional Commits formatına uyup uymadığını kontrol eder.
func isConventionalCommit(msg string) bool {
	validTypes := []string{
		"feat", "fix", "docs", "style", "refactor",
		"perf", "test", "chore", "ci", "build", "revert",
	}
	lower := strings.ToLower(msg)
	for _, t := range validTypes {
		if strings.HasPrefix(lower, t+":") || strings.HasPrefix(lower, t+"(") {
			return true
		}
	}
	return false
}

// ── Profil Listesi ve Ekran ───────────────────────────────────────────────────

// ProfileNames mağazadaki profil adlarını döner.
func ProfileNames(store *ProfileStore) []string {
	names := make([]string, 0, len(store.Profiles))
	for name := range store.Profiles {
		names = append(names, name)
	}
	return names
}

// FormatProfileList profilleri ekrana yazdırılacak formata çevirir.
func FormatProfileList(store *ProfileStore) string {
	var sb strings.Builder
	sb.WriteString("  [=] Kayıtlı Profiller:\n\n")

	for name, p := range store.Profiles {
		marker := "  "
		if name == store.ActiveProfile {
			marker = "▶ "
		}
		sb.WriteString(fmt.Sprintf("  %s%s", marker, name))
		if p.Description != "" {
			sb.WriteString(fmt.Sprintf("  —  %s", p.Description))
		}
		sb.WriteString("\n")

		// Aktif profil için detay göster
		if name == store.ActiveProfile {
			if len(p.Push.ConfirmBranches) > 0 {
				sb.WriteString(fmt.Sprintf("     [lock] Onay gereken: %s\n", strings.Join(p.Push.ConfirmBranches, ", ")))
			}
			if len(p.Push.BlockedBranches) > 0 {
				sb.WriteString(fmt.Sprintf("     [x] Engellenen: %s\n", strings.Join(p.Push.BlockedBranches, ", ")))
			}
			if p.Push.ShowDiffOnPush {
				sb.WriteString("     [%] Push öncesi diff özeti: açık\n")
			}
			if p.Commit.RequireConventional {
				sb.WriteString("     [ok] Conventional Commits: zorunlu\n")
			}
		}
	}

	sb.WriteString(fmt.Sprintf("\n  Aktif: %s\n", store.ActiveProfile))
	sb.WriteString("  Değiştirmek için: /profile set <isim>\n")
	return sb.String()
}

// FormatActiveProfile aktif profili özetler.
func FormatActiveProfile(p *Profile) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("  [tag]  Aktif profil: %s", p.Name))
	if p.Description != "" {
		sb.WriteString(fmt.Sprintf("  (%s)", p.Description))
	}
	sb.WriteString("\n")
	return sb.String()
}
