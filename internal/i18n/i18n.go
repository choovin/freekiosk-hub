package i18n

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/labstack/echo/v4"
)

// TranslationStore holds all translations
type TranslationStore struct {
	mu           sync.RWMutex
	translations map[string]map[string]string
	currentLang  string
}

var store *TranslationStore
var once sync.Once

// GetStore returns the singleton TranslationStore
func GetStore() *TranslationStore {
	once.Do(func() {
		store = &TranslationStore{
			translations: make(map[string]map[string]string),
			currentLang:  "zh", // Default to Chinese
		}
	})
	return store
}

// LoadLanguage loads translations from a JSON file
func (s *TranslationStore) LoadLanguage(lang string, filePath string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read translation file: %w", err)
	}

	var translations map[string]string
	if err := json.Unmarshal(data, &translations); err != nil {
		return fmt.Errorf("failed to parse translation file: %w", err)
	}

	s.translations[lang] = translations
	return nil
}

// LoadTranslations loads all supported languages
func (s *TranslationStore) LoadTranslations(basePath string) error {
	languages := []string{"en", "zh", "es", "fr", "de", "ja", "ko", "pt", "ru", "ar"}
	for _, lang := range languages {
		filePath := fmt.Sprintf("%s/%s.json", basePath, lang)
		if err := s.LoadLanguage(lang, filePath); err != nil {
			// Log but continue - will fallback to English
			fmt.Printf("Warning: Could not load %s translations: %v\n", lang, err)
		}
	}
	return nil
}

// SetLanguage sets the current language
func (s *TranslationStore) SetLanguage(lang string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.translations[lang]; ok {
		s.currentLang = lang
	}
}

// Get retrieves a translation by key
func (s *TranslationStore) Get(key string) string {
	return s.GetLang(s.currentLang, key)
}

// GetLang retrieves a translation by key and language
func (s *TranslationStore) GetLang(lang, key string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if translations, ok := s.translations[lang]; ok {
		if value, ok := translations[key]; ok {
			return value
		}
	}

	// Fallback to Chinese
	if translations, ok := s.translations["zh"]; ok {
		if value, ok := translations[key]; ok {
			return value
		}
	}

	// Return key if no translation found
	return key
}

// T is a convenience function for translations
func T(key string) string {
	return GetStore().Get(key)
}

// TL is a convenience function for translations with language
func TL(lang, key string) string {
	return GetStore().GetLang(lang, key)
}

// SetLang is a convenience function to set the current language
func SetLang(lang string) {
	GetStore().SetLanguage(lang)
}

// LanguageMiddleware extracts language from Accept-Language header or query param
func LanguageMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		lang := c.QueryParam("lang")
		if lang == "" {
			lang = c.Request().Header.Get("Accept-Language")
			if lang != "" {
				// Extract first language code (e.g., "en-US" -> "en")
				if idx := strings.Index(lang, ","); idx != -1 {
					lang = lang[:idx]
				}
				if idx := strings.Index(lang, "-"); idx != -1 {
					lang = lang[:idx]
				}
			}
		}
		if lang == "" {
			lang = "zh" // Default to Chinese
		}
		c.Set("lang", lang)
		return next(c)
	}
}

// TFromContext gets translation using language from context
func TFromContext(c echo.Context, key string) string {
	lang, ok := c.Get("lang").(string)
	if !ok {
		lang = "zh"
	}
	return GetStore().GetLang(lang, key)
}
