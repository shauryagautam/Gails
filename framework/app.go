package framework

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/shaurya/gails/cache"
	"github.com/shaurya/gails/config"
	"github.com/shaurya/gails/framework/assets"
	"github.com/shaurya/gails/framework/i18n"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// App is the main Gails application struct, holding all framework services.
type App struct {
	DB       *gorm.DB
	Redis    *redis.Client
	Cache    cache.Cache
	Config   *config.Config
	Router   *Router
	Renderer *Renderer
	Plugins  []Plugin
	Log      *zap.Logger
}

// New creates a new Gails application instance.
func New() *App {
	cfg, err := LoadConfig()
	if err != nil {
		// In case the config file is not found, use defaults
		cfg = &config.Config{
			App: config.AppConfig{
				Name: "MyApp",
				Port: 3000,
				Env:  "development",
			},
		}
	}

	env := os.Getenv("APP_ENV")
	if env == "" {
		env = cfg.App.Env
		if env == "" {
			env = "development"
		}
		os.Setenv("APP_ENV", env)
	}
	cfg.App.Env = env

	router := NewRouter()

	app := &App{
		Config:  cfg,
		Router:  router,
		Plugins: make([]Plugin, 0),
	}

	router.app = app

	return app
}

// Register adds a plugin to the application.
func (a *App) Register(p Plugin) {
	a.Plugins = append(a.Plugins, p)
}

// Routes configures the application routes using a callback function.
func (a *App) Routes(fn func(r *Router)) {
	fn(a.Router)
}

// Boot initializes all application subsystems in order.
func (a *App) Boot() {
	// 1. Load config (already done in New())

	// 2. Initialize logger
	InitLogger()
	a.Log = Log
	Log.Info("Booting Gails...")

	// 3. Initialize i18n
	if err := i18n.Init("config/locales"); err != nil {
		Log.Warn("Failed to initialize i18n", zap.Error(err))
	}

	// 4. Initialize assets
	if err := assets.Init("public/assets/manifest.json"); err != nil {
		Log.Warn("Failed to initialize assets", zap.Error(err))
	}

	// 5. Initialize session store
	InitSessionStore(a.Config.App.SecretKeyBase)

	// 6. Boot all registered plugins
	a.bootPlugins()

	// 7. Register default middleware
	a.Router.Use(RequestID())
	a.Router.Use(Logger())
	a.Router.Use(Recovery())
	a.Router.Mux.Use(SecureHeaders)

	// 8. Initialize renderer
	a.Renderer = NewRenderer(a.Config)

	// 9. Mount metrics endpoint
	a.Router.Mux.Handle("/metrics", MetricsHandler())
	a.Router.addRoute("GET", "/metrics", "Prometheus")

	Log.Info("Gails booted successfully")
}

// bootPlugins initializes all registered plugins.
func (a *App) bootPlugins() {
	for _, p := range a.Plugins {
		Log.Info("Booting plugin...", zap.String("name", p.Name()), zap.String("version", p.Version()))
		if err := p.Boot(a); err != nil {
			Log.Error("Failed to boot plugin",
				zap.String("name", p.Name()),
				zap.Error(err))
			continue
		}

		// Register plugin routes
		p.Routes(a.Router)
	}
}

// Run boots the app and starts the HTTP server.
func (a *App) Run() {
	a.Boot()

	port := a.Config.App.Port
	if port == 0 {
		port = 3000
	}

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: a.Router.Mux,
	}

	// Boot banner
	dbStatus := "✗"
	if a.DB != nil {
		dbStatus = "✓ postgres"
	}
	redisStatus := "✗"
	if a.Redis != nil {
		redisStatus = "✓"
	}
	pluginCount := len(a.Plugins)

	fmt.Println("┌─────────────────────────────────────────┐")
	fmt.Printf("│  🚀  Gails v1.0.0 — %-18s  │\n", a.Config.App.Name)
	fmt.Printf("│  Env: %-14s Port: %-6d      │\n", a.Config.App.Env, port)
	fmt.Printf("│  DB:  %-15s Redis: %-6s    │\n", dbStatus, redisStatus)
	fmt.Printf("│  Jobs: ✓ asynq       Plugins: %-3d       │\n", pluginCount)
	fmt.Println("└─────────────────────────────────────────┘")

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			Log.Fatal("Server failed", zap.Error(err))
		}
	}()

	// Graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	Log.Info("Shutting down server...")

	drainTimeout := 30 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), drainTimeout)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		Log.Error("Shutdown failed", zap.Error(err))
	}

	// Close DB pool
	if a.DB != nil {
		if sqlDB, err := a.DB.DB(); err == nil {
			sqlDB.Close()
		}
	}

	// Close Redis
	if a.Redis != nil {
		a.Redis.Close()
	}

	Log.Info("Gails stopped gracefully")
}
