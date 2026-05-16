package lang

import (
	"fmt"
	"os"
	"strings"
)

// Language interface defines the contract for language implementations
type Language interface {
	GetTranslations() map[string]string
}

// LanguageManager handles language loading and translation
type LanguageManager struct {
	currentLanguage string
	translations   map[string]string
}

// NewLanguageManager creates a new language manager
func NewLanguageManager() *LanguageManager {
	lm := &LanguageManager{
		currentLanguage: "en", // default to English
		translations:   make(map[string]string),
	}
	
	// Try to detect system language or use environment variable
	if lang := os.Getenv("GITBOOK_LANG"); lang != "" {
		lm.SetLanguage(lang)
	} else {
		lm.SetLanguage("en") // default to English
	}
	
	return lm
}

// SetLanguage changes the current language
func (lm *LanguageManager) SetLanguage(langCode string) error {
	var lang Language
	
	switch strings.ToLower(langCode) {
	case "tr", "turkish":
		lang = TurkishLanguage{}
		lm.currentLanguage = "tr"
	case "en", "english":
		lang = EnglishLanguage{}
		lm.currentLanguage = "en"
	case "example":
		lang = ExampleLanguage{}
		lm.currentLanguage = "example"
	default:
		// Fallback to English if language not found
		lang = EnglishLanguage{}
		lm.currentLanguage = "en"
	}
	
	lm.translations = lang.GetTranslations()
	return nil
}

// GetTranslation returns a translated string for the given key
func (lm *LanguageManager) GetTranslation(key string) string {
	if translation, exists := lm.translations[key]; exists {
		return translation
	}
	
	// Fallback to key if translation not found
	return key
}

// T is a shorthand for GetTranslation
func (lm *LanguageManager) T(key string) string {
	return lm.GetTranslation(key)
}

// Tf returns a formatted translation
func (lm *LanguageManager) Tf(key string, args ...interface{}) string {
	translation := lm.GetTranslation(key)
	if len(args) > 0 {
		return fmt.Sprintf(translation, args...)
	}
	return translation
}

// GetCurrentLanguage returns the current language code
func (lm *LanguageManager) GetCurrentLanguage() string {
	return lm.currentLanguage
}

// GetAvailableLanguages returns a list of available language codes
func GetAvailableLanguages() []string {
	return []string{"en", "tr", "example"}
}

// Global language manager instance
var globalLangManager *LanguageManager

// InitLanguage initializes the global language manager
func InitLanguage() {
	globalLangManager = NewLanguageManager()
}

// GetGlobalLang returns the global language manager
func GetGlobalLang() *LanguageManager {
	if globalLangManager == nil {
		InitLanguage()
	}
	return globalLangManager
}

// Global translation functions for convenience
func T(key string) string {
	return GetGlobalLang().T(key)
}

func Tf(key string, args ...interface{}) string {
	return GetGlobalLang().Tf(key, args...)
}
