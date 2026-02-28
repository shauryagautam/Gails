package framework

import (
	"fmt"
	"html/template"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/shaurya/gails/config"
	"github.com/shaurya/gails/framework/assets"
	"github.com/shaurya/gails/framework/helpers"
	"github.com/shaurya/gails/framework/i18n"
)

// Renderer manages HTML template rendering with layouts and hot-reload.
type Renderer struct {
	Templates *template.Template
	Config    *config.Config
	mu        sync.RWMutex
	compiled  bool
}

// NewRenderer creates a new Renderer.
func NewRenderer(cfg *config.Config) *Renderer {
	r := &Renderer{Config: cfg}
	r.CompileTemplates()
	return r
}

// CompileTemplates parses all templates from views/ directory.
func (r *Renderer) CompileTemplates() {
	r.mu.Lock()
	defer r.mu.Unlock()

	funcs := r.templateFuncs()
	tmpl := template.New("").Funcs(funcs)

	// Walk through views directory and parse templates
	viewsDir := "views"
	if _, err := os.Stat(viewsDir); err == nil {
		filepath.Walk(viewsDir, func(path string, info os.FileInfo, err error) error {
			if err == nil && !info.IsDir() && filepath.Ext(path) == ".html" {
				_, parseErr := tmpl.ParseFiles(path)
				if parseErr != nil && Log != nil {
					Log.Warn(fmt.Sprintf("Failed to parse template %s: %v", path, parseErr))
				}
			}
			return nil
		})
	}

	r.Templates = tmpl
	r.compiled = true
}

// templateFuncs returns the FuncMap with all built-in template helpers.
func (r *Renderer) templateFuncs() template.FuncMap {
	return template.FuncMap{
		"urlFor": func(resource string, id any) string {
			return "/" + resource + "/" + fmt.Sprint(id)
		},
		"linkTo": func(text, url string) template.HTML {
			return template.HTML(fmt.Sprintf(`<a href="%s">%s</a>`, url, text))
		},
		"csrfToken": func() template.HTML {
			return template.HTML(`<input type="hidden" name="csrf_token" value="token">`)
		},
		"flashMessages": func() template.HTML {
			return template.HTML("")
		},
		"currentUser": func() any {
			return nil
		},
		"env": func() string {
			env := os.Getenv("APP_ENV")
			if env == "" {
				env = "development"
			}
			return env
		},
		"formFor":       helpers.FormFor,
		"inputFor":      helpers.InputFor,
		"labelFor":      helpers.LabelFor,
		"textareaFor":   helpers.TextareaFor,
		"checkboxFor":   helpers.CheckboxFor,
		"selectFor":     helpers.SelectFor,
		"submitButton":  helpers.SubmitButton,
		"hiddenField":   helpers.HiddenField,
		"errorMessages": helpers.ErrorMessages,
		"safeHTML": func(s string) template.HTML {
			return template.HTML(s)
		},
		"t": func(key string, args ...any) string {
			vars := make(i18n.Vars)
			if len(args) == 1 {
				vars["name"] = args[0]
			} else {
				for idx := 0; idx < len(args)-1; idx += 2 {
					if k, ok := args[idx].(string); ok {
						vars[k] = args[idx+1]
					}
				}
			}
			return i18n.T(key, vars)
		},
		"stylesheetInclude": assets.StylesheetTag,
		"javascriptInclude": assets.JavascriptTag,
		"assetPath":         assets.AssetPath,
		"stylesheetTag":     assets.StylesheetTag,
		"javascriptTag":     assets.JavascriptTag,
	}
}

// Render renders a named template.
// In development, templates are hot-reloaded on every request.
func (r *Renderer) Render(w io.Writer, name string, data any) error {
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "development"
	}

	// Hot-reload in development
	if env != "production" {
		r.CompileTemplates()
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.Templates == nil {
		return fmt.Errorf("templates not compiled")
	}

	return r.Templates.ExecuteTemplate(w, name, data)
}
