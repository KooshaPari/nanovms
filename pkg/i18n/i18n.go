package i18n

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

const DefaultLocale = "en"

var SupportedLocales = []string{"en"}

// I18n manages translations for multi-language support.
type I18n struct {
	locale       string
	translations map[string]interface{}
}

// New creates a new I18n instance for the given locale.
func New(locale string) *I18n {
	i := &I18n{
		locale:       locale,
		translations: make(map[string]interface{}),
	}
	i.loadTranslations()
	return i
}

func (i *I18n) loadTranslations() {
	localeDir := filepath.Join("locales", i.locale)
	if _, err := os.Stat(localeDir); err == nil {
		data, err := os.ReadFile(filepath.Join(localeDir, "messages.json"))
		if err == nil {
			json.Unmarshal(data, &i.translations)
		}
	}

	// Fallback to default locale
	if i.locale != DefaultLocale {
		defaultFile := filepath.Join("locales", DefaultLocale, "messages.json")
		data, err := os.ReadFile(defaultFile)
		if err == nil {
			var defaultTranslations map[string]interface{}
			if json.Unmarshal(data, &defaultTranslations) == nil {
				for k, v := range defaultTranslations {
					if _, exists := i.translations[k]; !exists {
						i.translations[k] = v
					}
				}
			}
		}
	}
}

// T translates a key, returning the key itself if not found.
func (i *I18n) T(key string) string {
	parts := strings.Split(key, ".")
	var value interface{} = i.translations

	for _, part := range parts {
		if m, ok := value.(map[string]interface{}); ok {
			value = m[part]
		} else {
			return key
		}
	}

	if value == nil {
		return key
	}

	if s, ok := value.(string); ok {
		return s
	}

	return key
}

// Locale returns the current locale.
func (i *I18n) Locale() string {
	return i.locale
}

// T is a global convenience function for translation.
func T(key string) string {
	i := New(DefaultLocale)
	return i.T(key)
}
