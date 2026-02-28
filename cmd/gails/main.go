package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/shaurya/gails/db"
	"github.com/shaurya/gails/framework"
	"github.com/shaurya/gails/generator"
	"github.com/shaurya/gails/queue"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "gails",
	Short: "Gails — a Ruby on Rails-like web framework in Go",
	Long:  `Gails is an opinionated, batteries-included, production-grade web framework for Go.`,
}

func main() {
	// Runtime
	rootCmd.AddCommand(serverCmd())
	rootCmd.AddCommand(workerCmd())

	// Generators
	rootCmd.AddCommand(generateCmd())
	rootCmd.AddCommand(newAppCmd())

	// Database
	rootCmd.AddCommand(dbCmd())

	// Introspection
	rootCmd.AddCommand(routesCmd())
	rootCmd.AddCommand(versionCmd())

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// --- Server ---

func serverCmd() *cobra.Command {
	var port int
	var env string
	cmd := &cobra.Command{
		Use:   "server",
		Short: "Start the Gails HTTP server",
		Run: func(cmd *cobra.Command, args []string) {
			if env != "" {
				os.Setenv("APP_ENV", env)
			}
			app := framework.New()
			if port != 0 {
				app.Config.App.Port = port
			}
			app.Run()
		},
	}
	cmd.Flags().IntVarP(&port, "port", "p", 0, "Port to listen on")
	cmd.Flags().StringVarP(&env, "env", "e", "", "Environment (development, production, test)")
	return cmd
}

// --- Worker ---

func workerCmd() *cobra.Command {
	var queueName string
	var concurrency int
	cmd := &cobra.Command{
		Use:   "worker",
		Short: "Start background job worker",
		Run: func(cmd *cobra.Command, args []string) {
			cfg, _ := framework.LoadConfig()
			if concurrency > 0 {
				cfg.Queue.Concurrency = concurrency
			}
			w := queue.NewWorker(cfg.Redis, cfg.Queue)
			w.Run()
		},
	}
	cmd.Flags().StringVar(&queueName, "queue", "", "Queue name to process")
	cmd.Flags().IntVar(&concurrency, "concurrency", 0, "Number of concurrent workers")
	return cmd
}

// --- Generators ---

func generateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "generate [type] [name] [fields...]",
		Aliases: []string{"g"},
		Short:   "Generate code (model, controller, scaffold, migration, mailer, job)",
		Args:    cobra.MinimumNArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			genType := args[0]
			name := args[1]
			g := generator.NewGenerator("generator/templates")

			switch genType {
			case "model":
				fields := g.ParseFields(args[2:])
				generateModel(g, name, fields)

			case "controller":
				generateController(g, name, args[2:])

			case "scaffold":
				fields := g.ParseFields(args[2:])
				generateModel(g, name, fields)
				generateController(g, name, []string{"index", "show", "new", "create", "edit", "update", "destroy"})
				generateViews(g, name, fields)
				generateMigration(g, name, fields)
				fmt.Printf("\n[Gails] Scaffold complete for %s\n", name)
				fmt.Printf("[Gails] Add to your routes: r.Resources(\"%s\", &%sController{})\n", strings.ToLower(name)+"s", name)

			case "migration":
				fields := g.ParseFields(args[2:])
				generateMigration(g, name, fields)

			case "mailer":
				actions := args[2:]
				generateMailer(g, name, actions)

			case "job":
				generateJob(g, name)

			default:
				fmt.Printf("Unknown generator type: %s\n", genType)
				fmt.Println("Available: model, controller, scaffold, migration, mailer, job")
			}
		},
	}
	return cmd
}

func generateModel(g *generator.Generator, name string, fields []generator.Field) {
	tmpl := `package models

import (
	"github.com/shaurya/gails/orm"
)

type {{.Name}} struct {
	orm.Model
{{range .Fields}}	{{.Name}} {{.Type}}` + " `" + `{{if .GormTag}}gorm:"{{.GormTag}}"{{end}}{{if .ValidateTag}} validate:"{{.ValidateTag}}"{{end}}` + "`" + `
{{end}}}
`
	data := map[string]any{"Name": name, "Fields": fields}
	g.GenerateInline(tmpl, data, fmt.Sprintf("app/models/%s.go", strings.ToLower(name)))
}

func generateController(g *generator.Generator, name string, actions []string) {
	tmpl := `package controllers

import (
	"net/http"
	"github.com/shaurya/gails/framework"
)

type {{.Name}}Controller struct {
	framework.Controller
}
{{range .Actions}}
func (c *{{$.Name}}Controller) {{.}}(ctx *framework.Context) error {
	return ctx.JSON(http.StatusOK, framework.H{"action": "{{.}}"})
}
{{end}}`

	// Capitalize action names
	var caps []string
	for _, a := range actions {
		caps = append(caps, strings.Title(a))
	}

	data := map[string]any{"Name": name, "Actions": caps}
	g.GenerateInline(tmpl, data, fmt.Sprintf("app/controllers/%s_controller.go", strings.ToLower(name)))
}

func generateViews(g *generator.Generator, name string, fields []generator.Field) {
	lower := strings.ToLower(name)

	// Index view
	indexTmpl := `{{define "content"}}
<h1>` + name + `s</h1>
<a href="/` + lower + `s/new">New ` + name + `</a>
<table>
<thead><tr>` + func() string {
		var h string
		for _, f := range fields {
			h += "<th>" + f.Name + "</th>"
		}
		return h
	}() + `<th>Actions</th></tr></thead>
<tbody></tbody>
</table>
{{end}}`
	writeFile(fmt.Sprintf("views/%s/index.html", lower+"s"), indexTmpl)

	// Show, New, Edit views
	writeFile(fmt.Sprintf("views/%s/show.html", lower+"s"), `{{define "content"}}<h1>`+name+` Details</h1>{{end}}`)
	writeFile(fmt.Sprintf("views/%s/new.html", lower+"s"), `{{define "content"}}<h1>New `+name+`</h1>{{end}}`)
	writeFile(fmt.Sprintf("views/%s/edit.html", lower+"s"), `{{define "content"}}<h1>Edit `+name+`</h1>{{end}}`)
}

func generateMigration(g *generator.Generator, name string, fields []generator.Field) {
	timestamp := time.Now().Format("20060102150405")
	lower := strings.ToLower(name) + "s"

	var columns string
	for _, f := range fields {
		sqlType := goTypeToSQL(f.Type)
		columns += fmt.Sprintf("\t\t%s %s", strings.ToLower(f.Name), sqlType)
		if f.GormTag != "" && strings.Contains(f.GormTag, "uniqueIndex") {
			columns += " UNIQUE"
		}
		if f.GormTag != "" && strings.Contains(f.GormTag, "not null") {
			columns += " NOT NULL"
		}
		columns += ",\n"
	}

	migrationSQL := fmt.Sprintf(`-- +goose Up
CREATE TABLE %s (
	id SERIAL PRIMARY KEY,
%s	created_at TIMESTAMP DEFAULT NOW(),
	updated_at TIMESTAMP DEFAULT NOW(),
	deleted_at TIMESTAMP
);

-- +goose Down
DROP TABLE IF EXISTS %s;
`, lower, columns, lower)

	path := fmt.Sprintf("db/migrations/%s_create_%s.sql", timestamp, lower)
	writeFile(path, migrationSQL)
}

func generateMailer(g *generator.Generator, name string, actions []string) {
	tmpl := `package mailers

import (
	"github.com/shaurya/gails/mailer"
	"github.com/shaurya/gails/framework"
)

type {{.Name}}Mailer struct {
	mailer.Mailer
}
{{range .Actions}}
func (m {{$.Name}}Mailer) {{.}}(data framework.H) *mailer.Email {
	return m.NewEmail().
		Subject("{{.}}").
		Template("{{$.LowerName}}/{{.}}", data)
}
{{end}}`
	data := map[string]any{
		"Name":      name,
		"LowerName": strings.ToLower(name),
		"Actions":   actions,
	}
	g.GenerateInline(tmpl, data, fmt.Sprintf("app/mailers/%s_mailer.go", strings.ToLower(name)))

	// Create template files for each action
	for _, action := range actions {
		writeFile(fmt.Sprintf("views/mailers/%s/%s.html", strings.ToLower(name), action), fmt.Sprintf("<h1>%s</h1>\n<p>Email content here.</p>", action))
	}
}

func generateJob(g *generator.Generator, name string) {
	tmpl := `package jobs

import (
	"context"
	"fmt"
)

type {{.Name}}Job struct {
	// Add job fields here
}

func (j *{{.Name}}Job) Perform(ctx context.Context) error {
	fmt.Println("Performing {{.Name}}Job...")
	return nil
}
`
	data := map[string]any{"Name": name}
	g.GenerateInline(tmpl, data, fmt.Sprintf("app/jobs/%s_job.go", strings.ToLower(name)))
}

func goTypeToSQL(goType string) string {
	switch goType {
	case "string":
		return "VARCHAR(255)"
	case "int":
		return "INTEGER"
	case "float64":
		return "DOUBLE PRECISION"
	case "bool":
		return "BOOLEAN DEFAULT false"
	case "time.Time":
		return "TIMESTAMP"
	default:
		return "VARCHAR(255)"
	}
}

func writeFile(path, content string) {
	os.MkdirAll(fmt.Sprintf("%s", path[:strings.LastIndex(path, "/")]), 0755)
	os.WriteFile(path, []byte(content), 0644)
}

// --- Database ---

func dbCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "db",
		Short: "Database management commands",
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "migrate",
		Short: "Run pending migrations",
		Run: func(cmd *cobra.Command, args []string) {
			cfg, _ := framework.LoadConfig()
			database := db.MustConnect(cfg.Database)
			if err := db.Migrate(database, "db/migrations"); err != nil {
				fmt.Fprintf(os.Stderr, "Migration failed: %v\n", err)
				os.Exit(1)
			}
			fmt.Println("[Gails] Migrations complete")
		},
	})

	var steps int
	rollbackCmd := &cobra.Command{
		Use:   "rollback",
		Short: "Rollback migrations",
		Run: func(cmd *cobra.Command, args []string) {
			cfg, _ := framework.LoadConfig()
			database := db.MustConnect(cfg.Database)
			if err := db.Rollback(database, "db/migrations", steps); err != nil {
				fmt.Fprintf(os.Stderr, "Rollback failed: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("[Gails] Rolled back %d migration(s)\n", steps)
		},
	}
	rollbackCmd.Flags().IntVar(&steps, "steps", 1, "Number of migrations to roll back")
	cmd.AddCommand(rollbackCmd)

	cmd.AddCommand(&cobra.Command{
		Use:   "status",
		Short: "Print migration status",
		Run: func(cmd *cobra.Command, args []string) {
			cfg, _ := framework.LoadConfig()
			database := db.MustConnect(cfg.Database)
			db.MigrationStatus(database, "db/migrations")
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "create",
		Short: "Create the database",
		Run: func(cmd *cobra.Command, args []string) {
			cfg, _ := framework.LoadConfig()
			if err := db.CreateDB(cfg.Database.Name, cfg.Database.Host, cfg.Database.Port, cfg.Database.User, cfg.Database.Password, cfg.Database.SSLMode); err != nil {
				fmt.Fprintf(os.Stderr, "%v\n", err)
				os.Exit(1)
			}
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "drop",
		Short: "Drop the database",
		Run: func(cmd *cobra.Command, args []string) {
			cfg, _ := framework.LoadConfig()
			if err := db.DropDB(cfg.Database.Name, cfg.Database.Host, cfg.Database.Port, cfg.Database.User, cfg.Database.Password, cfg.Database.SSLMode); err != nil {
				fmt.Fprintf(os.Stderr, "%v\n", err)
				os.Exit(1)
			}
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "seed",
		Short: "Run seed data",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("[Gails] Run your seed file manually or call db.Seed(db, seedFn)")
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "reset",
		Short: "Drop, create, migrate, and seed the database",
		Run: func(cmd *cobra.Command, args []string) {
			cfg, _ := framework.LoadConfig()
			db.DropDB(cfg.Database.Name, cfg.Database.Host, cfg.Database.Port, cfg.Database.User, cfg.Database.Password, cfg.Database.SSLMode)
			db.CreateDB(cfg.Database.Name, cfg.Database.Host, cfg.Database.Port, cfg.Database.User, cfg.Database.Password, cfg.Database.SSLMode)
			database := db.MustConnect(cfg.Database)
			db.Migrate(database, "db/migrations")
			fmt.Println("[Gails] Database reset complete")
		},
	})

	return cmd
}

// --- Routes ---

func routesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "routes",
		Short: "Print all registered routes",
		Run: func(cmd *cobra.Command, args []string) {
			app := framework.New()
			fmt.Println(app.Router.Inspect())
		},
	}
}

// --- New App ---

func newAppCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "new [name]",
		Short: "Create a new Gails application",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			name := args[0]
			g := generator.NewGenerator("generator/templates")
			if err := g.GenerateSkeleton(name); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to create app: %v\n", err)
				os.Exit(1)
			}
		},
	}
}

// --- Version ---

func versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the Gails version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Gails v1.0.0")
		},
	}
}
