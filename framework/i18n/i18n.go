package i18n

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

// Vars is a map of interpolation variables for translations.
type Vars map[string]any

var (
	translations     = make(map[string]map[string]any)
	defaultLocale    = "en"
	mu               sync.RWMutex
	currentLocale    = "en"
	availableLocales []string
)

// Init loads all locale files from the given directory.
func Init(localesDir string) error {
	entries, err := os.ReadDir(localesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No locales to load
		}
		return err
	}

	mu.Lock()
	defer mu.Unlock()

	for _, f := range entries {
		if f.IsDir() {
			continue
		}
		ext := filepath.Ext(f.Name())
		if ext != ".yaml" && ext != ".yml" {
			continue
		}

		locale := strings.TrimSuffix(f.Name(), ext)

		data, err := os.ReadFile(filepath.Join(localesDir, f.Name()))
		if err != nil {
			return err
		}

		var nested map[string]any
		if err := yaml.Unmarshal(data, &nested); err != nil {
			return err
		}

		// Locales are optionally nested under the language code: en: { ... }
		if localeData, ok := nested[locale].(map[string]any); ok {
			translations[locale] = flatten(localeData, "")
		} else {
			translations[locale] = flatten(nested, "")
		}
		availableLocales = append(availableLocales, locale)
	}
	return nil
}

// T translates a key with optional variable interpolation.
func T(key string, vars Vars) string {
	mu.RLock()
	locale := currentLocale
	mu.RUnlock()

	data, ok := translations[locale]
	if !ok {
		data, ok = translations[defaultLocale]
		if !ok {
			return key
		}
	}

	val, ok := data[key].(string)
	if !ok {
		// Fallback to default locale
		if locale != defaultLocale {
			data, ok = translations[defaultLocale]
			if ok {
				val, ok = data[key].(string)
				if !ok {
					return key
				}
			} else {
				return key
			}
		} else {
			return key
		}
	}

	for k, v := range vars {
		placeholder := fmt.Sprintf("%%{%s}", k)
		val = strings.ReplaceAll(val, placeholder, fmt.Sprint(v))
	}

	return val
}

// GetLocale returns the current locale.
func GetLocale() string {
	mu.RLock()
	defer mu.RUnlock()
	return currentLocale
}

// SetLocale sets the current locale.
func SetLocale(locale string) {
	mu.Lock()
	defer mu.Unlock()
	currentLocale = locale
}

// Locale returns the current locale (alias for GetLocale).
func Locale() string {
	return GetLocale()
}

// AvailableLocales returns all loaded locales.
func AvailableLocales() []string {
	mu.RLock()
	defer mu.RUnlock()
	return availableLocales
}

// flatten takes a nested map and flattens it into dot-notation keys.
func flatten(nested map[string]any, prefix string) map[string]any {
	flat := make(map[string]any)
	for k, v := range nested {
		key := k
		if prefix != "" {
			key = prefix + "." + k
		}

		if sub, ok := v.(map[string]any); ok {
			for sk, sv := range flatten(sub, key) {
				flat[sk] = sv
			}
		} else {
			flat[key] = v
		}
	}
	return flat
}
