package assets

import (
	"encoding/json"
	"fmt"
	"html/template"
	"os"
)

type Manifest map[string]struct {
	File    string `json:"file"`
	Src     string `json:"src"`
	IsEntry bool   `json:"isEntry"`
}

var manifest Manifest

func Init(manifestPath string) error {
	if os.Getenv("APP_ENV") == "development" {
		return nil
	}

	data, err := os.ReadFile(manifestPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No manifest yet
		}
		return err
	}

	return json.Unmarshal(data, &manifest)
}

func AssetPath(name string) string {
	if os.Getenv("APP_ENV") == "development" {
		// Proxy to Vite dev server (default port 5173)
		return fmt.Sprintf("http://localhost:5173/%s", name)
	}

	if m, ok := manifest[name]; ok {
		return "/assets/" + m.File
	}
	return "/assets/" + name
}

func StylesheetTag(name string) template.HTML {
	path := AssetPath(name)
	return template.HTML(fmt.Sprintf(`<link rel="stylesheet" href="%s">`, path))
}

func JavascriptTag(name string) template.HTML {
	path := AssetPath(name)
	if os.Getenv("APP_ENV") == "development" {
		// Include Vite client in dev for HMR
		return template.HTML(fmt.Sprintf(`
			<script type="module" src="http://localhost:5173/@vite/client"></script>
			<script type="module" src="%s"></script>
		`, path))
	}
	return template.HTML(fmt.Sprintf(`<script type="module" src="%s"></script>`, path))
}
