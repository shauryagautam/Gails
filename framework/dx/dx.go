package dx

import (
	"fmt"

	"go.uber.org/zap"
)

// Logger is used by DX to output warnings. Set during Boot.
var Logger *zap.Logger

// BootValidations runs startup checks and prints warnings.
func BootValidations(cfg BootConfig) {
	if Logger == nil {
		return
	}

	if cfg.SecretKeyBase == "" && cfg.Env == "production" {
		Logger.Warn("[Gails] WARN: config.app.secret_key_base is empty — required in production")
	}

	if cfg.SecretKeyBase == "" && cfg.Env != "production" {
		Logger.Info("[Gails] INFO: secret_key_base not set, using default (not secure for production)")
	}

	if len(cfg.RegisteredControllers) > 0 {
		for _, name := range cfg.RegisteredControllers {
			if name == "" {
				continue
			}
			// Controller validation would be done here
		}
	}
}

// BootConfig holds the configuration for boot-time validations.
type BootConfig struct {
	Env                   string
	SecretKeyBase         string
	DBConnected           bool
	RedisConnected        bool
	RegisteredControllers []string
	RegisteredJobs        []string
}

// PrintGoFileChange prints a notice when a Go file changes (for dev mode).
func PrintGoFileChange(path string) {
	fmt.Printf("[Gails] Change detected: %s\n", path)
	fmt.Println("[Gails] Run `gails server` to apply Go changes  (or use `air` for auto-restart)")
}

// GenerateAirConfig returns a default .air.toml configuration.
func GenerateAirConfig() string {
	return `root = "."
tmp_dir = "tmp"

[build]
  bin = "./tmp/main"
  cmd = "go build -o ./tmp/main ."
  delay = 1000
  exclude_dir = ["assets", "tmp", "vendor", "node_modules"]
  exclude_file = []
  exclude_regex = ["_test.go"]
  exclude_unchanged = false
  follow_symlink = false
  full_bin = ""
  include_dir = []
  include_ext = ["go", "tpl", "tmpl", "html"]
  kill_delay = "0s"
  log = "build-errors.log"
  send_interrupt = false
  stop_on_error = true

[color]
  app = ""
  build = "yellow"
  main = "magenta"
  runner = "green"
  watcher = "cyan"

[log]
  time = false

[misc]
  clean_on_exit = false
`
}
