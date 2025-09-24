package i18n

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"

	"booking-platform/shared/config"
)

type Localizer struct {
	translations map[string]map[string]string
	defaultLang  string
}

var globalLocalizer *Localizer

func Initialize(cfg *config.Config) error {
	globalLocalizer = &Localizer{
		translations: make(map[string]map[string]string),
		defaultLang:  cfg.I18n.DefaultLanguage,
	}

	// Load translation files
	for _, lang := range cfg.I18n.SupportedLanguages {
		if err := loadLanguageFile(lang); err != nil {
			return fmt.Errorf("failed to load language %s: %w", lang, err)
		}
	}

	return nil
}

func loadLanguageFile(lang string) error {
	filename := filepath.Join("shared", "i18n", "locales", fmt.Sprintf("%s.json", lang))
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}

	var translations map[string]string
	if err := json.Unmarshal(data, &translations); err != nil {
		return err
	}

	globalLocalizer.translations[lang] = translations
	return nil
}

func T(lang, key string, args ...interface{}) string {
	if globalLocalizer == nil {
		return key
	}

	// Try requested language
	if langMap, ok := globalLocalizer.translations[lang]; ok {
		if translation, ok := langMap[key]; ok {
			if len(args) > 0 {
				return fmt.Sprintf(translation, args...)
			}
			return translation
		}
	}

	// Fall back to default language
	if langMap, ok := globalLocalizer.translations[globalLocalizer.defaultLang]; ok {
		if translation, ok := langMap[key]; ok {
			if len(args) > 0 {
				return fmt.Sprintf(translation, args...)
			}
			return translation
		}
	}

	// Return key if no translation found
	return key
}

func GetSupportedLanguages() []string {
	if globalLocalizer == nil {
		return []string{"en"}
	}

	langs := make([]string, 0, len(globalLocalizer.translations))
	for lang := range globalLocalizer.translations {
		langs = append(langs, lang)
	}
	return langs
}
