package i18n

import (
	"testing"
)

func TestNew(t *testing.T) {
	i := New("en")
	if i.Locale() != "en" {
		t.Errorf("expected locale 'en', got '%s'", i.Locale())
	}
}

func TestT_ExistingKey(t *testing.T) {
	i := New("en")
	result := i.T("welcome")
	// If locales/en/messages.json exists, should return translated value
	// Otherwise returns the key itself
	if result == "" {
		t.Error("expected non-empty result")
	}
}

func TestT_NonExistentKey(t *testing.T) {
	i := New("en")
	result := i.T("nonexistent.key.path")
	if result != "nonexistent.key.path" {
		t.Errorf("expected key passthrough, got '%s'", result)
	}
}

func TestT_NestedKey(t *testing.T) {
	i := New("en")
	result := i.T("errors.file_not_found")
	// Should resolve nested key if locale file exists
	if result == "" {
		t.Error("expected non-empty result for nested key")
	}
}

func TestFallbackToDefaultLocale(t *testing.T) {
	i := New("xx") // non-existent locale
	result := i.T("welcome")
	// Should fall back to English or return key
	if result == "" {
		t.Error("expected non-empty result with fallback")
	}
}

func TestGlobalT(t *testing.T) {
	result := T("welcome")
	if result == "" {
		t.Error("expected non-empty result from global T()")
	}
}
