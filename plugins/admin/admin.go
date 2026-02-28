package admin

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

// Config configures the admin panel.
type Config struct {
	Models []Resource
	Auth   func(http.Handler) http.Handler
}

// Resource describes a model registered in the admin panel.
type Resource struct {
	ModelType     reflect.Type
	ModelName     string
	DisplayFields []string
	SearchFields  []string
	ReadOnlyMode  bool
}

// NewResource creates a new admin resource for a model type.
func NewResource[T any]() Resource {
	var zero T
	t := reflect.TypeOf(zero)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	name := t.Name()

	// Default: display all exported fields
	var fields []string
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if f.PkgPath == "" && f.Name != "Model" && f.Name != "DeletedAt" {
			fields = append(fields, f.Name)
		}
	}

	return Resource{
		ModelType:     t,
		ModelName:     name,
		DisplayFields: fields,
	}
}

// SearchFields sets the fields that can be searched.
func (r Resource) WithSearchFields(fields ...string) Resource {
	r.SearchFields = fields
	return r
}

// DisplayColumns sets the fields to display in the index.
func (r Resource) WithDisplayFields(fields ...string) Resource {
	r.DisplayFields = fields
	return r
}

// ReadOnly marks the resource as read-only.
func (r Resource) ReadOnly() Resource {
	r.ReadOnlyMode = true
	return r
}

// Panel returns an http.Handler for the admin panel.
func Panel(cfg Config) http.Handler {
	a := &adminPanel{
		config: cfg,
		models: make(map[string]Resource),
	}
	for _, m := range cfg.Models {
		a.models[strings.ToLower(m.ModelName)] = m
	}

	r := chi.NewRouter()
	if cfg.Auth != nil {
		r.Use(cfg.Auth)
	}
	r.Get("/", a.dashboard)
	r.Get("/{model}", a.index)
	r.Get("/{model}/new", a.new)
	r.Post("/{model}", a.create)
	r.Get("/{model}/{id}", a.show)
	r.Get("/{model}/{id}/edit", a.edit)
	r.Post("/{model}/{id}", a.update)
	r.Post("/{model}/{id}/delete", a.delete)
	r.Get("/{model}/export.csv", a.exportCSV)

	return r
}

// BasicAuth returns a middleware for HTTP Basic Authentication.
func BasicAuth(username, password string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			u, p, ok := r.BasicAuth()
			if !ok || u != username || p != password {
				w.Header().Set("WWW-Authenticate", `Basic realm="Gails Admin"`)
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

type adminPanel struct {
	config Config
	models map[string]Resource
	db     *gorm.DB
}

func (a *adminPanel) render(w http.ResponseWriter, title, body string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	// Build sidebar links
	var sidebar strings.Builder
	for name := range a.models {
		sidebar.WriteString(fmt.Sprintf(`<a href="/admin/%s" class="nav-link">%s</a>`, name, strings.Title(name)))
	}

	fmt.Fprintf(w, adminLayout, title, sidebar.String(), body)
}

func (a *adminPanel) dashboard(w http.ResponseWriter, r *http.Request) {
	var cards strings.Builder
	for name, res := range a.models {
		cards.WriteString(fmt.Sprintf(`
			<div class="card">
				<div class="card-title">%s</div>
				<div class="card-info">%d fields</div>
				<a href="/admin/%s" class="card-link">View Records →</a>
			</div>`, strings.Title(res.ModelName), len(res.DisplayFields), name))
	}
	a.render(w, "Dashboard", fmt.Sprintf(`<h2>Dashboard</h2><div class="card-grid">%s</div>`, cards.String()))
}

func (a *adminPanel) index(w http.ResponseWriter, r *http.Request) {
	modelName := chi.URLParam(r, "model")
	res, ok := a.models[modelName]
	if !ok {
		http.NotFound(w, r)
		return
	}

	search := r.URL.Query().Get("q")
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}

	// Build table header
	var thead strings.Builder
	thead.WriteString("<tr>")
	for _, f := range res.DisplayFields {
		thead.WriteString(fmt.Sprintf("<th>%s</th>", f))
	}
	thead.WriteString("<th>Actions</th></tr>")

	body := fmt.Sprintf(`
		<div class="toolbar">
			<h2>%s</h2>
			<div>
				<form method="get" class="search-form">
					<input type="text" name="q" placeholder="Search..." value="%s" class="search-input">
					<button type="submit" class="btn">Search</button>
				</form>
				<a href="/admin/%s/export.csv" class="btn btn-secondary">CSV Export</a>
				%s
			</div>
		</div>
		<table><thead>%s</thead><tbody><tr><td colspan="%d" class="empty">Connect database to view records</td></tr></tbody></table>
		<div class="pagination">Page %d</div>`,
		strings.Title(res.ModelName),
		search,
		modelName,
		func() string {
			if !res.ReadOnlyMode {
				return fmt.Sprintf(`<a href="/admin/%s/new" class="btn btn-primary">+ New</a>`, modelName)
			}
			return ""
		}(),
		thead.String(),
		len(res.DisplayFields)+1,
		page)

	a.render(w, strings.Title(res.ModelName), body)
}

func (a *adminPanel) show(w http.ResponseWriter, r *http.Request) {
	modelName := chi.URLParam(r, "model")
	id := chi.URLParam(r, "id")
	res, ok := a.models[modelName]
	if !ok {
		http.NotFound(w, r)
		return
	}

	var details strings.Builder
	for _, f := range res.DisplayFields {
		details.WriteString(fmt.Sprintf(`<div class="detail-row"><span class="detail-label">%s</span><span class="detail-value">—</span></div>`, f))
	}

	body := fmt.Sprintf(`<h2>%s #%s</h2><div class="details">%s</div><a href="/admin/%s" class="btn">← Back</a>`,
		strings.Title(res.ModelName), id, details.String(), modelName)
	a.render(w, fmt.Sprintf("%s #%s", res.ModelName, id), body)
}

func (a *adminPanel) new(w http.ResponseWriter, r *http.Request) {
	modelName := chi.URLParam(r, "model")
	res, ok := a.models[modelName]
	if !ok {
		http.NotFound(w, r)
		return
	}

	var fields strings.Builder
	for _, f := range res.DisplayFields {
		if f == "ID" || f == "CreatedAt" || f == "UpdatedAt" {
			continue
		}
		fields.WriteString(fmt.Sprintf(`
			<div class="form-group">
				<label>%s</label>
				<input type="text" name="%s" class="form-input">
			</div>`, f, f))
	}

	body := fmt.Sprintf(`
		<h2>New %s</h2>
		<form method="post" action="/admin/%s">
			%s
			<button type="submit" class="btn btn-primary">Create</button>
		</form>`, strings.Title(res.ModelName), modelName, fields.String())
	a.render(w, "New "+res.ModelName, body)
}

func (a *adminPanel) create(w http.ResponseWriter, r *http.Request) {
	modelName := chi.URLParam(r, "model")
	http.Redirect(w, r, "/admin/"+modelName, http.StatusFound)
}

func (a *adminPanel) edit(w http.ResponseWriter, r *http.Request) {
	modelName := chi.URLParam(r, "model")
	id := chi.URLParam(r, "id")
	res, ok := a.models[modelName]
	if !ok {
		http.NotFound(w, r)
		return
	}

	var fields strings.Builder
	for _, f := range res.DisplayFields {
		if f == "ID" || f == "CreatedAt" || f == "UpdatedAt" {
			continue
		}
		fields.WriteString(fmt.Sprintf(`
			<div class="form-group">
				<label>%s</label>
				<input type="text" name="%s" class="form-input">
			</div>`, f, f))
	}

	body := fmt.Sprintf(`
		<h2>Edit %s #%s</h2>
		<form method="post" action="/admin/%s/%s">
			%s
			<button type="submit" class="btn btn-primary">Update</button>
		</form>`, strings.Title(res.ModelName), id, modelName, id, fields.String())
	a.render(w, "Edit "+res.ModelName, body)
}

func (a *adminPanel) update(w http.ResponseWriter, r *http.Request) {
	modelName := chi.URLParam(r, "model")
	http.Redirect(w, r, "/admin/"+modelName, http.StatusFound)
}

func (a *adminPanel) delete(w http.ResponseWriter, r *http.Request) {
	modelName := chi.URLParam(r, "model")
	http.Redirect(w, r, "/admin/"+modelName, http.StatusFound)
}

func (a *adminPanel) exportCSV(w http.ResponseWriter, r *http.Request) {
	modelName := chi.URLParam(r, "model")
	res, ok := a.models[modelName]
	if !ok {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s.csv", modelName))

	writer := csv.NewWriter(w)
	defer writer.Flush()

	// Write header
	writer.Write(res.DisplayFields)
	// Data rows would be written here with DB integration
}

func (a *adminPanel) respondJSON(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

const adminLayout = `<!DOCTYPE html>
<html>
<head>
	<title>Gails Admin — %s</title>
	<style>
		* { margin: 0; padding: 0; box-sizing: border-box; }
		body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; background: #0f0f23; color: #e0e0e0; display: flex; min-height: 100vh; }
		.sidebar { width: 220px; background: #1a1a2e; border-right: 1px solid #16213e; padding: 20px 0; flex-shrink: 0; }
		.sidebar-title { color: #00d2ff; font-size: 18px; font-weight: bold; padding: 0 20px 20px; border-bottom: 1px solid #16213e; }
		.nav-link { display: block; padding: 10px 20px; color: #b8b8cc; text-decoration: none; font-size: 14px; transition: all 0.2s; }
		.nav-link:hover { background: #16213e; color: #fff; }
		.main { flex: 1; padding: 30px; }
		h2 { color: #fff; margin-bottom: 20px; font-size: 22px; }
		.card-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(250px, 1fr)); gap: 20px; }
		.card { background: #1a1a2e; border: 1px solid #0f3460; border-radius: 8px; padding: 20px; }
		.card-title { font-size: 18px; font-weight: bold; color: #fff; margin-bottom: 8px; }
		.card-info { color: #888; font-size: 13px; margin-bottom: 12px; }
		.card-link { color: #00d2ff; text-decoration: none; font-size: 14px; }
		table { width: 100%%; border-collapse: collapse; background: #1a1a2e; border-radius: 8px; overflow: hidden; }
		th { background: #16213e; color: #00d2ff; font-weight: 600; text-transform: uppercase; font-size: 11px; padding: 12px 16px; text-align: left; }
		td { padding: 10px 16px; border-bottom: 1px solid #16213e; font-size: 14px; }
		.empty { text-align: center; color: #666; padding: 40px; }
		.toolbar { display: flex; justify-content: space-between; align-items: center; margin-bottom: 20px; flex-wrap: wrap; gap: 10px; }
		.toolbar > div { display: flex; gap: 10px; align-items: center; }
		.btn { display: inline-block; padding: 8px 16px; border-radius: 6px; text-decoration: none; font-size: 13px; cursor: pointer; border: 1px solid #0f3460; background: #16213e; color: #fff; transition: all 0.2s; }
		.btn:hover { background: #0f3460; }
		.btn-primary { background: #0066ff; border-color: #0066ff; }
		.btn-primary:hover { background: #0055dd; }
		.btn-secondary { background: transparent; }
		.search-form { display: flex; gap: 8px; }
		.search-input { padding: 8px 12px; border-radius: 6px; border: 1px solid #0f3460; background: #16213e; color: #fff; font-size: 13px; }
		.form-group { margin-bottom: 16px; }
		.form-group label { display: block; margin-bottom: 6px; font-size: 13px; color: #b8b8cc; }
		.form-input { width: 100%%; padding: 10px 12px; border-radius: 6px; border: 1px solid #0f3460; background: #16213e; color: #fff; font-size: 14px; }
		.details { background: #1a1a2e; border-radius: 8px; padding: 20px; margin-bottom: 20px; }
		.detail-row { display: flex; padding: 10px 0; border-bottom: 1px solid #16213e; }
		.detail-label { width: 200px; color: #888; font-size: 13px; }
		.detail-value { color: #fff; font-size: 14px; }
		.pagination { text-align: center; padding: 20px; color: #888; font-size: 13px; }
	</style>
</head>
<body>
	<div class="sidebar">
		<div class="sidebar-title">⚡ Gails Admin</div>
		<a href="/admin" class="nav-link">Dashboard</a>
		%s
	</div>
	<div class="main">%s</div>
</body>
</html>`
