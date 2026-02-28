# 🚀 Gails

**A Ruby on Rails-like web framework in Go.**

Opinionated · Batteries-included · Production-grade

```
go get github.com/shaurya/gails
```

---

## Features

| Category | What You Get |
|----------|-------------|
| **Routing** | RESTful resources, nested routes, namespaces, PATCH/WebSocket/Mount, formatted `Inspect()` |
| **Controllers** | `func(*Context) error` actions, automatic error-to-HTTP mapping, panic recovery |
| **ORM** | Generic `QueryBuilder[T]`, pagination (`Page`/`PerPage`), named scopes, callbacks, validations, counter cache |
| **Database** | PostgreSQL via pgx/GORM, slow query logging, migrations (goose), `CreateDB`/`DropDB`, seeds (`Once`/`Fake[T]`) |
| **Cache** | Redis + in-memory adapters, `SetModel`/`GetModel`, fragment caching, pub/sub |
| **Sessions** | Cookie & Redis-backed sessions, flash messages, CSRF protection (double-submit cookie) |
| **Auth** | JWT (HS256) with context injection, session auth with `Required()` / `RequireRole()`, bcrypt passwords |
| **Background Jobs** | Asynq-powered workers, per-job logging + Prometheus counters, embedded monitoring dashboard |
| **Mailer** | HTML+text multipart, template rendering, dev email interception, `DeliverLater` |
| **WebSocket** | `Channel` interface (OnConnect/OnMessage/OnDisconnect), rooms, broadcast |
| **i18n** | YAML-backed, dot-notation keys, `%{var}` interpolation, per-request locale |
| **Templates** | Hot-reload in dev, layout wrapping, rich helper functions (forms, links, assets) |
| **Plugins** | Healthcheck (`/health`, `/health/ready`), request logger, full admin panel |
| **CLI** | `gails new`, `generate scaffold`, `db:migrate`, `db:rollback`, `routes`, `console`, and more |
| **Observability** | Prometheus metrics (`/metrics`), structured logging (zap), panic breadcrumbs |
| **Testing** | Test suite with request helpers, factory pattern, assertions |

---

## Quick Start

### Create a New App

```bash
gails new myapp
cd myapp
```

### Or Add to an Existing Project

```go
package main

import (
    "net/http"
    "github.com/shaurya/gails/framework"
    "github.com/shaurya/gails/plugins/healthcheck"
)

func main() {
    app := framework.New()
    app.Register(&healthcheck.Plugin{})

    app.Routes(func(r *framework.Router) {
        r.GET("/", func(ctx *framework.Context) error {
            return ctx.JSON(http.StatusOK, framework.H{
                "message": "Hello from Gails!",
            })
        })
    })

    app.Run()
}
```

### Run It

```bash
go run .
```

```
┌─────────────────────────────────────────┐
│  🚀  Gails v1.0.0 — MyApp              │
│  Env: development    Port: 3000         │
│  DB:  ✗               Redis: ✗          │
│  Jobs: ✓ asynq       Plugins: 1         │
└─────────────────────────────────────────┘
```

---

## Routing

```go
app.Routes(func(r *framework.Router) {
    // RESTful resources (auto-generates index, show, create, update, destroy)
    r.Resources("users", &UsersController{})

    // Nested resources
    r.Resources("posts", &PostsController{}, func(r *framework.Router) {
        r.Resources("comments", &CommentsController{})
    })

    // Namespaced routes
    r.Namespace("/api/v1", func(r *framework.Router) {
        r.GET("/status", statusHandler)
    })

    // WebSocket
    r.WebSocket("/ws/chat", hub.HandleChannel(&ChatChannel{}))

    // Mount sub-handlers
    r.Mount("/admin", adminPanel)
})
```

Print all routes:
```bash
gails routes
```

---

## Controllers

```go
type UsersController struct{ framework.Controller }

func (c *UsersController) Index(ctx *framework.Context) error {
    users, err := orm.Query[User](db).Page(1).PerPage(20).All()
    if err != nil {
        return ctx.InternalError(err)
    }
    return ctx.JSON(http.StatusOK, framework.H{"users": users})
}

func (c *UsersController) Create(ctx *framework.Context) error {
    var input CreateUserInput
    if err := ctx.Bind(&input); err != nil {
        return err // Automatically returns 422 with field errors
    }
    // ...
}
```

---

## ORM

```go
// Query builder with pagination
users, _ := orm.Query[User](db).
    Where("role = ?", "admin").
    Order("created_at DESC").
    Page(2).PerPage(25).
    All()

// Named scopes
func Active(db *gorm.DB) *gorm.DB {
    return db.Where("active = ?", true)
}
users, _ := orm.Query[User](db).Scope(Active).All()

// Find by ID
user, _ := orm.Query[User](db).Find(42)
```

---

## Database

```bash
gails db create          # Create the database
gails db migrate         # Run pending migrations
gails db rollback --steps=2
gails db status          # Print migration status
gails db seed            # Run seed data
gails db reset           # Drop + create + migrate + seed
```

### Seeds

```go
// Run once (tracked in DB)
db.Once(database, "initial_admin", func() error {
    return database.Create(&User{Name: "Admin", Email: "admin@example.com"}).Error
})

// Generate fake data
db.Fake(database, 50, func(f *db.Faker) User {
    return User{Name: f.Name(), Email: f.Email()}
})
```

---

## Auth

```go
// JWT
token, _ := auth.GenerateToken(userID)
r.Use(auth.JWTMiddleware())

// Session auth with role checking
r.GET("/admin", auth.Required(adminHandler))
r.GET("/superadmin", auth.RequireRole("admin", superHandler))

// Password hashing
hash, _ := auth.HashPassword("secret123")
ok := auth.CheckPassword("secret123", hash)
```

---

## Background Jobs

```go
// Enqueue
enqueuer.Enqueue("email:welcome", map[string]any{"user_id": 42})

// Handle (in worker)
worker.HandleFunc("email:welcome", func(ctx context.Context, task *asynq.Task) error {
    // Process job...
    return nil
})
```

```bash
gails worker                    # Start processing jobs
gails worker --concurrency=20   # With custom concurrency
```

Job dashboard available at `/jobs` with auto-refresh.

---

## Generators

```bash
gails generate model User name:string email:string:unique role:string
gails generate controller Users index show create
gails generate scaffold Post title:string body:text:required user_id:integer
gails generate migration AddAgeToUsers age:integer
gails generate mailer Welcome welcome_email confirmation
gails generate job SendNewsletter
```

---

## Plugins

### Built-in

```go
app.Register(&healthcheck.Plugin{})  // /health, /health/ready

r.Mount("/admin", admin.Panel(admin.Config{
    Models: []admin.Resource{
        admin.NewResource[User]().WithSearchFields("Name", "Email"),
        admin.NewResource[Post](),
    },
    Auth: admin.BasicAuth("admin", "password"),
}))
```

### Custom Plugins

```go
type MyPlugin struct{}

func (p *MyPlugin) Name() string    { return "myplugin" }
func (p *MyPlugin) Version() string { return "1.0.0" }
func (p *MyPlugin) Boot(app *framework.App) error { return nil }
func (p *MyPlugin) Routes(r *framework.Router) {
    r.GET("/myplugin", myHandler)
}
```

---

## Configuration

`config/app.yaml`:

```yaml
app:
  name: MyApp
  port: 3000
  secret_key_base: "your-secret-here"
  env: development

database:
  host: localhost
  port: 5432
  name: myapp_development
  user: postgres
  password: ""
  pool: 10
  slow_query_ms: 200

redis:
  url: redis://localhost:6379
  pool: 10

queue:
  concurrency: 10
  queues:
    - name: critical
      weight: 6
    - name: default
      weight: 3
```

Environment overrides via `config/environments/{env}.yaml`.

---

## Project Structure

```
myapp/
├── app/
│   ├── controllers/       # Request handlers
│   ├── models/            # ORM models
│   ├── jobs/              # Background jobs
│   └── mailers/           # Email mailers
├── config/
│   ├── app.yaml           # Main configuration
│   ├── environments/      # Per-environment overrides
│   └── locales/           # i18n translation files
├── db/
│   └── migrations/        # SQL migration files (goose)
├── views/
│   ├── layouts/           # Layout templates
│   └── mailers/           # Email templates
├── frontend/              # JS/CSS assets (Vite)
├── public/                # Static files
└── main.go                # Entry point
```

---

## Requirements

- **Go** 1.22+
- **PostgreSQL** (required)
- **Redis** (required for cache, sessions, jobs)

---

## License

MIT
