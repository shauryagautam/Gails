package framework

import (
	"fmt"
	"os"
	"strings"

	"github.com/shaurya/gails/config"
	"github.com/spf13/viper"
)

func LoadConfig() (*config.Config, error) {
	v := viper.New()

	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "development"
	}

	// Set configuration path
	v.AddConfigPath("config")
	v.SetConfigName("app")
	v.SetConfigType("yaml")

	// Read base config
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read app.yaml: %w", err)
	}

	// Environment-specific override
	v.SetConfigName("environments/" + env)
	if err := v.MergeInConfig(); err != nil {
		// It's okay if environment config doesn't exist, we fallback to app.yaml
		fmt.Printf("[Gails] Warning: No environment-specific config for %s, using defaults\n", env)
	}

	// Read environment variables
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	var cfg config.Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &cfg, nil
}
