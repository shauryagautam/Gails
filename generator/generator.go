package generator

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"
)

// Generator handles code generation from templates.
type Generator struct {
	TemplatesDir string
}

// NewGenerator creates a new Generator.
func NewGenerator(templatesDir string) *Generator {
	return &Generator{TemplatesDir: templatesDir}
}

// Field represents a model field parsed from CLI arguments.
type Field struct {
	Name        string
	Type        string
	GormTag     string
	ValidateTag string
}

// TypeMap maps field type shorthand to Go types.
var TypeMap = map[string]string{
	"string":   "string",
	"text":     "string",
	"integer":  "int",
	"int":      "int",
	"float":    "float64",
	"boolean":  "bool",
	"bool":     "bool",
	"date":     "time.Time",
	"datetime": "time.Time",
	"uuid":     "string",
}

// ParseFields parses field definitions from CLI arguments.
// Format: name:type or name:type:modifier1:modifier2
func (g *Generator) ParseFields(args []string) []Field {
	fields := []Field{}
	for _, arg := range args {
		parts := strings.Split(arg, ":")
		if len(parts) < 2 {
			continue
		}

		name := parts[0]
		typeName := parts[1]

		goType, ok := TypeMap[typeName]
		if !ok {
			goType = "string"
		}

		f := Field{
			Name: capitalize(name),
			Type: goType,
		}

		// Handle modifiers
		if len(parts) > 2 {
			var gormTags []string
			var valTags []string
			for _, mod := range parts[2:] {
				switch mod {
				case "unique":
					gormTags = append(gormTags, "uniqueIndex")
				case "index":
					gormTags = append(gormTags, "index")
				case "required":
					valTags = append(valTags, "required")
				case "notnull":
					gormTags = append(gormTags, "not null")
				}
			}
			if len(gormTags) > 0 {
				f.GormTag = strings.Join(gormTags, ";")
			}
			if len(valTags) > 0 {
				f.ValidateTag = strings.Join(valTags, ",")
			}
		}
		fields = append(fields, f)
	}
	return fields
}

// Generate renders a template and writes it to targetPath.
func (g *Generator) Generate(templateName string, data any, targetPath string) error {
	tmplPath := filepath.Join(g.TemplatesDir, templateName+".tmpl")

	funcMap := template.FuncMap{
		"Title":     strings.Title,
		"Lower":     strings.ToLower,
		"Plural":    pluralize,
		"Singular":  singularize,
		"Timestamp": func() string { return time.Now().Format("20060102150405") },
	}

	tmpl := template.New(templateName + ".tmpl").Funcs(funcMap)
	tmpl, err := tmpl.ParseFiles(tmplPath)
	if err != nil {
		return fmt.Errorf("failed to parse template %s: %w", tmplPath, err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
		return err
	}

	fmt.Printf("[Gails] Created: %s\n", targetPath)
	return os.WriteFile(targetPath, buf.Bytes(), 0644)
}

// GenerateInline renders a template string and writes it to targetPath.
func (g *Generator) GenerateInline(tmplStr string, data any, targetPath string) error {
	funcMap := template.FuncMap{
		"Title":     strings.Title,
		"Lower":     strings.ToLower,
		"Plural":    pluralize,
		"Singular":  singularize,
		"Timestamp": func() string { return time.Now().Format("20060102150405") },
	}

	tmpl, err := template.New("inline").Funcs(funcMap).Parse(tmplStr)
	if err != nil {
		return err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
		return err
	}

	fmt.Printf("[Gails] Created: %s\n", targetPath)
	return os.WriteFile(targetPath, buf.Bytes(), 0644)
}

// GenerateSkeleton generates a complete new Gails application skeleton.
func (g *Generator) GenerateSkeleton(name string) error {
	dirs := []string{
		"app/controllers",
		"app/models",
		"app/jobs",
		"app/mailers",
		"config/environments",
		"config/locales",
		"db/migrations",
		"views/layouts",
		"views/home",
		"views/mailers",
		"frontend",
		"public",
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(filepath.Join(name, dir), 0755); err != nil {
			return err
		}
	}

	// Generate main.go
	mainGo := `package main

import (
	"github.com/shaurya/gails/framework"
	"github.com/shaurya/gails/plugins/healthcheck"
)

func main() {
	app := framework.New()
	app.Register(&healthcheck.Plugin{})

	app.Routes(func(r *framework.Router) {
		r.GET("/", func(ctx *framework.Context) error {
			return ctx.JSON(200, framework.H{"message": "Welcome to Gails!"})
		})
	})

	app.Run()
}
`
	writeFile(filepath.Join(name, "main.go"), mainGo)

	// Generate go.mod
	goMod := fmt.Sprintf(`module %s

go 1.22

require github.com/shaurya/gails v0.0.0
`, name)
	writeFile(filepath.Join(name, "go.mod"), goMod)

	// Generate config/app.yaml
	appYaml := fmt.Sprintf(`app:
  name: %s
  port: 3000
  secret_key_base: ""
  auto_migrate: false
  env: development

database:
  host: localhost
  port: 5432
  name: %s_development
  user: postgres
  password: ""
  pool: 10
  ssl_mode: disable
  slow_query_ms: 200

redis:
  url: redis://localhost:6379
  pool: 10
  db: 0

sessions:
  store: cookie
  ttl: 86400
  key_prefix: "sess:"

queue:
  concurrency: 10
  queues:
    - name: critical
      weight: 6
    - name: default
      weight: 3
    - name: low
      weight: 1

mailer:
  smtp_host: localhost
  smtp_port: 1025
  from: noreply@%s.com

cache:
  ttl: 3600
  prefix: "%s:"
`, name, strings.ToLower(name), strings.ToLower(name), strings.ToLower(name))
	writeFile(filepath.Join(name, "config/app.yaml"), appYaml)

	// Environment configs
	writeFile(filepath.Join(name, "config/environments/development.yaml"), "# Development overrides\n")
	writeFile(filepath.Join(name, "config/environments/production.yaml"), "app:\n  env: production\n")
	writeFile(filepath.Join(name, "config/environments/test.yaml"), "app:\n  env: test\ndatabase:\n  name: "+strings.ToLower(name)+"_test\n")

	// Locale file
	writeFile(filepath.Join(name, "config/locales/en.yaml"), `en:
  welcome: "Welcome to `+name+`!"
`)

	// Layout
	writeFile(filepath.Join(name, "views/layouts/application.html"), `<!DOCTYPE html>
<html>
<head>
    <title>`+name+`</title>
    {{stylesheetInclude "app.css"}}
</head>
<body>
    {{template "content" .}}
    {{javascriptInclude "app.js"}}
</body>
</html>
`)

	// Home view
	writeFile(filepath.Join(name, "views/home/index.html"), `{{define "content"}}
<h1>Welcome to `+name+`</h1>
<p>Your Gails application is running!</p>
{{end}}
`)

	// Seeds
	writeFile(filepath.Join(name, "db/seeds.go"), `package main

import "gorm.io/gorm"

func SeedDB(db *gorm.DB) error {
	// Add your seed data here
	return nil
}
`)

	// Controllers
	writeFile(filepath.Join(name, "app/controllers/home_controller.go"), `package controllers

import "github.com/shaurya/gails/framework"

type HomeController struct{ framework.Controller }

func (c *HomeController) Index(ctx *framework.Context) error {
	return ctx.Render("index", framework.H{"title": "Home"})
}
`)

	// Frontend
	writeFile(filepath.Join(name, "frontend/app.js"), "console.log('Gails app loaded')\n")
	writeFile(filepath.Join(name, "frontend/app.css"), "/* App styles */\nbody { font-family: sans-serif; }\n")

	// .air.toml for hot reload
	writeFile(filepath.Join(name, ".air.toml"), airConfig())

	// .gitkeep
	writeFile(filepath.Join(name, "public/.gitkeep"), "")

	fmt.Printf("\n✅ Created new Gails app: %s\n\n", name)
	fmt.Println("Next steps:")
	fmt.Printf("  cd %s\n", name)
	fmt.Println("  gails server")
	fmt.Println()

	return nil
}

func writeFile(path, content string) {
	os.MkdirAll(filepath.Dir(path), 0755)
	os.WriteFile(path, []byte(content), 0644)
}

func airConfig() string {
	return `root = "."
tmp_dir = "tmp"

[build]
  bin = "./tmp/main"
  cmd = "go build -o ./tmp/main ."
  delay = 1000
  exclude_dir = ["assets", "tmp", "vendor", "node_modules"]
  include_ext = ["go", "tpl", "tmpl", "html"]
`
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func pluralize(s string) string {
	if strings.HasSuffix(s, "y") {
		return s[:len(s)-1] + "ies"
	}
	if strings.HasSuffix(s, "s") || strings.HasSuffix(s, "x") {
		return s + "es"
	}
	return s + "s"
}

func singularize(s string) string {
	if strings.HasSuffix(s, "ies") {
		return s[:len(s)-3] + "y"
	}
	if strings.HasSuffix(s, "ses") || strings.HasSuffix(s, "xes") {
		return s[:len(s)-2]
	}
	if strings.HasSuffix(s, "s") {
		return s[:len(s)-1]
	}
	return s
}
