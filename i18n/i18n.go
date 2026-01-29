package i18n

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

//go:embed langs/*.yml langs/*.yaml
var translationFiles embed.FS

type Translations struct {
	Common  map[string]string `yaml:"common"`
	Host    map[string]string `yaml:"host"`
	Client  map[string]string `yaml:"client"`
	Errors  map[string]string `yaml:"errors"`
	Backend map[string]string `yaml:"backend"`
}

type I18n struct {
	mu          sync.RWMutex
	lang        string
	translations map[string]*Translations
}

var (
	instance *I18n
	once     sync.Once
)

// GetInstance returns singleton i18n instance
func GetInstance() *I18n {
	once.Do(func() {
		instance = &I18n{
			translations: make(map[string]*Translations),
		}
	})
	return instance
}

// SetLanguage sets current language
func (i *I18n) SetLanguage(lang string) error {
	i.mu.Lock()
	defer i.mu.Unlock()

	// Load translations for this language if not already loaded
	if _, ok := i.translations[lang]; !ok {
		if err := i.loadLanguage(lang); err != nil {
			return fmt.Errorf("failed to load language %s: %w", lang, err)
		}
	}

	i.lang = lang
	return nil
}

// GetLanguage returns current language
func (i *I18n) GetLanguage() string {
	i.mu.RLock()
	defer i.mu.RUnlock()
	return i.lang
}

// T translates a key in current language
// The key format is "section.key", e.g., "host.title"
func (i *I18n) T(key string, args ...any) string {
	return i.Translate(key, args...)
}

// Translate translates a key with optional arguments
func (i *I18n) Translate(key string, args ...any) string {
	i.mu.RLock()
	defer i.mu.RUnlock()

	translations, ok := i.translations[i.lang]
	if !ok {
		// Fall back to English if current language not loaded
		translations = i.translations["en"]
		if translations == nil {
			return key
		}
	}

	// Parse key format: "section.key" or just "key"
	var section, k string
	if dot := strings.Index(key, "."); dot >= 0 {
		section = key[:dot]
		k = key[dot+1:]
	} else {
		k = key
	}

	var str string
	switch section {
	case "common":
		str = translations.Common[k]
	case "host":
		str = translations.Host[k]
	case "client":
		str = translations.Client[k]
	case "errors":
		str = translations.Errors[k]
	case "backend":
		str = translations.Backend[k]
	default:
		// Try common keys if no section specified
		str = translations.Common[k]
		if str == "" {
			str = translations.Host[k]
		}
		if str == "" {
			str = translations.Client[k]
		}
		if str == "" {
			str = translations.Errors[k]
		}
		if str == "" {
			str = translations.Backend[k]
		}
	}

	if str == "" {
		return key
	}

	if len(args) > 0 {
		return fmt.Sprintf(str, args...)
	}
	return str
}

// GetTranslations returns full translations map for current language (as JSON)
// This is used to send translations to frontend
func (i *I18n) GetTranslations() (map[string]any, error) {
	i.mu.RLock()
	defer i.mu.RUnlock()

	translations, ok := i.translations[i.lang]
	if !ok {
		translations = i.translations["en"]
		if translations == nil {
			return nil, fmt.Errorf("no translations loaded")
		}
	}

	// Convert to a map suitable for JSON serialization
	result := make(map[string]any)
	result["common"] = translations.Common
	result["host"] = translations.Host
	result["client"] = translations.Client
	result["errors"] = translations.Errors
	result["backend"] = translations.Backend

	return result, nil
}

// loadLanguage loads translations for a specific language from embedded files
func (i *I18n) loadLanguage(lang string) error {
	// Try both .yml and .yaml extensions
	filenames := []string{
		fmt.Sprintf("langs/%s.yml", lang),
		fmt.Sprintf("langs/%s.yaml", lang),
	}

	var data []byte
	var err error

	for _, filename := range filenames {
		data, err = translationFiles.ReadFile(filename)
		if err == nil {
			break
		}
	}

	if err != nil {
		return fmt.Errorf("translation file not found for language %s", lang)
	}

	var translations Translations
	if err := yaml.Unmarshal(data, &translations); err != nil {
		return fmt.Errorf("failed to parse translations: %w", err)
	}

	// Initialize maps if nil
	if translations.Common == nil {
		translations.Common = make(map[string]string)
	}
	if translations.Host == nil {
		translations.Host = make(map[string]string)
	}
	if translations.Client == nil {
		translations.Client = make(map[string]string)
	}
	if translations.Errors == nil {
		translations.Errors = make(map[string]string)
	}
	if translations.Backend == nil {
		translations.Backend = make(map[string]string)
	}

	i.translations[lang] = &translations
	log.Printf("Loaded translations for language: %s", lang)
	return nil
}

// LoadAllLanguages loads all available translation files
func (i *I18n) LoadAllLanguages() error {
	entries, err := fs.ReadDir(translationFiles, "langs")
	if err != nil {
		return fmt.Errorf("failed to read langs directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// Extract language code from filename (e.g., "en.yml" -> "en", "pt-BR.yaml" -> "pt-BR")
		name := entry.Name()
		var lang string
		if strings.HasSuffix(name, ".yml") {
			lang = strings.TrimSuffix(name, ".yml")
		} else if strings.HasSuffix(name, ".yaml") {
			lang = strings.TrimSuffix(name, ".yaml")
		} else {
			continue // Skip non-YAML files
		}

		if len(lang) < 2 {
			continue
		}

		if err := i.loadLanguage(lang); err != nil {
			log.Printf("Warning: failed to load language %s: %v", lang, err)
		}
	}

	return nil
}

// GetAvailableLanguages returns list of available language codes
func (i *I18n) GetAvailableLanguages() []string {
	i.mu.RLock()
	defer i.mu.RUnlock()

	langs := make([]string, 0, len(i.translations))
	for lang := range i.translations {
		langs = append(langs, lang)
	}
	return langs
}

// ToJSON converts translations to JSON format for frontend use
func (i *I18n) ToJSON() ([]byte, error) {
	translations, err := i.GetTranslations()
	if err != nil {
		return nil, err
	}
	return json.Marshal(translations)
}
