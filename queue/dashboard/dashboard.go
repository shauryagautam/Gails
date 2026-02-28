package dashboard

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/hibiken/asynq"
	"github.com/shaurya/gails/framework"
)

// DashboardConfig configures the job dashboard.
type DashboardConfig struct {
	Auth      func(http.Handler) http.Handler
	RedisAddr string
}

// Dashboard returns an http.Handler for the job monitoring dashboard.
func Dashboard(cfg DashboardConfig) http.Handler {
	d := &jobDashboard{
		inspector: asynq.NewInspector(asynq.RedisClientOpt{Addr: cfg.RedisAddr}),
	}

	r := chi.NewRouter()
	if cfg.Auth != nil {
		r.Use(cfg.Auth)
	}
	r.Get("/", d.Index)
	r.Get("/queues/{queue}", d.QueueInfo)
	r.Get("/api/stats", d.APIStats)
	return r
}

type jobDashboard struct {
	inspector *asynq.Inspector
}

func (d *jobDashboard) Index(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	queues, err := d.inspector.Queues()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var queueRows strings.Builder
	for _, q := range queues {
		info, infoErr := d.inspector.GetQueueInfo(q)
		if infoErr != nil {
			continue
		}
		queueRows.WriteString(fmt.Sprintf(`
			<tr>
				<td><a href="/jobs/queues/%s">%s</a></td>
				<td>%d</td>
				<td>%d</td>
				<td>%d</td>
				<td>%d</td>
			</tr>`, q, q, info.Pending, info.Active, info.Completed, info.Failed))
	}

	fmt.Fprintf(w, dashboardHTML, queueRows.String())
}

func (d *jobDashboard) QueueInfo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	qName := chi.URLParam(r, "queue")
	info, err := d.inspector.GetQueueInfo(qName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(w, queueDetailHTML, qName, qName, info.Pending, info.Active, info.Completed, info.Failed, info.Processed)
}

func (d *jobDashboard) APIStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	queues, err := d.inspector.Queues()
	if err != nil {
		json.NewEncoder(w).Encode(framework.H{"error": err.Error()})
		return
	}

	stats := make([]framework.H, 0)
	for _, q := range queues {
		info, infoErr := d.inspector.GetQueueInfo(q)
		if infoErr != nil {
			continue
		}
		stats = append(stats, framework.H{
			"queue":     q,
			"pending":   info.Pending,
			"active":    info.Active,
			"completed": info.Completed,
			"failed":    info.Failed,
		})
	}
	json.NewEncoder(w).Encode(stats)
}

const dashboardHTML = `<!DOCTYPE html>
<html>
<head>
	<title>Gails — Job Dashboard</title>
	<style>
		* { margin: 0; padding: 0; box-sizing: border-box; }
		body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; background: #1a1a2e; color: #e0e0e0; }
		.container { max-width: 1000px; margin: 0 auto; padding: 20px; }
		h1 { color: #00d2ff; margin-bottom: 20px; font-size: 24px; }
		.card { background: #16213e; border-radius: 8px; padding: 20px; margin-bottom: 20px; border: 1px solid #0f3460; }
		table { width: 100%%; border-collapse: collapse; }
		th, td { text-align: left; padding: 12px 16px; border-bottom: 1px solid #0f3460; }
		th { color: #00d2ff; font-weight: 600; text-transform: uppercase; font-size: 12px; }
		a { color: #00d2ff; text-decoration: none; }
		a:hover { text-decoration: underline; }
		.badge { display: inline-block; padding: 2px 8px; border-radius: 4px; font-size: 12px; }
		.refresh-note { text-align: center; color: #666; font-size: 12px; margin-top: 10px; }
	</style>
</head>
<body>
	<div class="container">
		<h1>🔧 Background Jobs</h1>
		<div class="card">
			<table>
				<thead>
					<tr><th>Queue</th><th>Pending</th><th>Active</th><th>Completed</th><th>Failed</th></tr>
				</thead>
				<tbody>%s</tbody>
			</table>
		</div>
		<p class="refresh-note">Auto-refreshes every 5 seconds</p>
	</div>
	<script>setTimeout(() => location.reload(), 5000);</script>
</body>
</html>`

const queueDetailHTML = `<!DOCTYPE html>
<html>
<head>
	<title>Queue: %s — Gails Jobs</title>
	<style>
		* { margin: 0; padding: 0; box-sizing: border-box; }
		body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; background: #1a1a2e; color: #e0e0e0; }
		.container { max-width: 1000px; margin: 0 auto; padding: 20px; }
		h1 { color: #00d2ff; margin-bottom: 20px; }
		.card { background: #16213e; border-radius: 8px; padding: 20px; margin-bottom: 20px; border: 1px solid #0f3460; }
		.stat { display: inline-block; margin-right: 30px; }
		.stat-label { color: #999; font-size: 12px; text-transform: uppercase; }
		.stat-value { font-size: 28px; font-weight: bold; color: #00d2ff; }
		a { color: #00d2ff; text-decoration: none; }
	</style>
</head>
<body>
	<div class="container">
		<p><a href="/jobs">← Back</a></p>
		<h1>Queue: %s</h1>
		<div class="card">
			<div class="stat"><div class="stat-label">Pending</div><div class="stat-value">%d</div></div>
			<div class="stat"><div class="stat-label">Active</div><div class="stat-value">%d</div></div>
			<div class="stat"><div class="stat-label">Completed</div><div class="stat-value">%d</div></div>
			<div class="stat"><div class="stat-label">Failed</div><div class="stat-value">%d</div></div>
			<div class="stat"><div class="stat-label">Total Processed</div><div class="stat-value">%d</div></div>
		</div>
	</div>
	<script>setTimeout(() => location.reload(), 5000);</script>
</body>
</html>`
