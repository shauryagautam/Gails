package requestlog

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/shaurya/gails/framework"
	"gorm.io/gorm"
)

// RequestLog is the database model for storing request logs.
type RequestLog struct {
	ID        uint   `gorm:"primarykey"`
	Method    string `gorm:"size:10;not null"`
	Path      string `gorm:"size:500;not null"`
	Status    int
	Duration  float64 // milliseconds
	IP        string  `gorm:"size:50"`
	UserAgent string  `gorm:"size:500"`
	UserID    *uint
	CreatedAt time.Time
}

// Plugin stores every HTTP request in a request_logs Postgres table.
type Plugin struct {
	app *framework.App
	db  *gorm.DB
}

func (p *Plugin) Name() string    { return "requestlog" }
func (p *Plugin) Version() string { return "1.0.0" }

func (p *Plugin) Boot(app *framework.App) error {
	p.app = app
	p.db = app.DB
	if p.db != nil {
		p.db.AutoMigrate(&RequestLog{})
	}
	return nil
}

func (p *Plugin) Routes(r *framework.Router) {
	// This plugin adds middleware, not routes
	if p.db != nil {
		r.Use(p.logMiddleware())
	}
}

func (p *Plugin) logMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			next.ServeHTTP(ww, r)

			go func() {
				if p.db == nil {
					return
				}
				log := RequestLog{
					Method:    r.Method,
					Path:      r.URL.Path,
					Status:    ww.Status(),
					Duration:  float64(time.Since(start).Microseconds()) / 1000.0,
					IP:        r.RemoteAddr,
					UserAgent: r.UserAgent(),
					CreatedAt: time.Now(),
				}
				p.db.Create(&log)
			}()
		})
	}
}

// GetRouteStats returns per-route hit counts.
func GetRouteStats(db *gorm.DB) ([]RouteStats, error) {
	var stats []RouteStats
	err := db.Model(&RequestLog{}).
		Select("method, path, COUNT(*) as hit_count, AVG(duration) as avg_duration_ms").
		Group("method, path").
		Order("hit_count DESC").
		Find(&stats).Error
	return stats, err
}

// RouteStats holds per-route statistics.
type RouteStats struct {
	Method        string
	Path          string
	HitCount      int64
	AvgDurationMs float64
}
