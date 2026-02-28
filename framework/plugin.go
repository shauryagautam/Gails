package framework

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// Plugin is the base interface for Gails extensions.
type Plugin interface {
	Name() string
	Version() string
	Boot(app *App) error
	Routes(r *Router)
}

// PluginWithMigrations is a plugin that provides its own migrations.
type PluginWithMigrations interface {
	Plugin
	Migrations() []string // Returns paths or SQL migration content
}

// MiddlewareProvider is a plugin that can inject global middleware.
type MiddlewareProvider interface {
	Plugin
	Middleware() []func(http.Handler) http.Handler
}

// Engine is a legacy compatibility interface for plugins that mount chi routes.
type Engine interface {
	Name() string
	Init(app *App) error
	Routes(r chi.Router)
}
