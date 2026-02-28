package healthcheck

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/shaurya/gails/framework"
)

var bootTime = time.Now()

// Plugin provides health check endpoints.
type Plugin struct {
	app *framework.App
}

func (p *Plugin) Name() string    { return "healthcheck" }
func (p *Plugin) Version() string { return "1.0.0" }

func (p *Plugin) Boot(app *framework.App) error {
	p.app = app
	return nil
}

func (p *Plugin) Routes(r *framework.Router) {
	r.GET("/health", p.healthHandler)
	r.GET("/health/ready", p.readyHandler)
}

func (p *Plugin) healthHandler(ctx *framework.Context) error {
	uptime := time.Since(bootTime).Round(time.Second).String()

	dbStatus := "not configured"
	if p.app != nil && p.app.DB != nil {
		if sqlDB, err := p.app.DB.DB(); err == nil {
			if err := sqlDB.Ping(); err == nil {
				dbStatus = "ok"
			} else {
				dbStatus = "error: " + err.Error()
			}
		}
	}

	redisStatus := "not configured"
	if p.app != nil && p.app.Redis != nil {
		if err := p.app.Redis.Ping(ctx.Request.Context()).Err(); err == nil {
			redisStatus = "ok"
		} else {
			redisStatus = "error: " + err.Error()
		}
	}

	data := map[string]any{
		"status":  "ok",
		"db":      dbStatus,
		"redis":   redisStatus,
		"uptime":  uptime,
		"version": "1.0.0",
	}

	return ctx.JSON(http.StatusOK, data)
}

func (p *Plugin) readyHandler(ctx *framework.Context) error {
	ready := true
	checks := make(map[string]string)

	if p.app != nil && p.app.DB != nil {
		if sqlDB, err := p.app.DB.DB(); err == nil {
			if err := sqlDB.Ping(); err != nil {
				ready = false
				checks["db"] = "error"
			} else {
				checks["db"] = "ok"
			}
		}
	}

	if p.app != nil && p.app.Redis != nil {
		if err := p.app.Redis.Ping(ctx.Request.Context()).Err(); err != nil {
			ready = false
			checks["redis"] = "error"
		} else {
			checks["redis"] = "ok"
		}
	}

	status := http.StatusOK
	statusText := "ready"
	if !ready {
		status = http.StatusServiceUnavailable
		statusText = "not ready"
	}

	data := map[string]any{
		"status": statusText,
		"checks": checks,
	}

	ctx.Response.Header().Set("Content-Type", "application/json")
	ctx.Response.WriteHeader(status)
	return json.NewEncoder(ctx.Response).Encode(data)
}
