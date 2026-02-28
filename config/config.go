package config

type Config struct {
	App      AppConfig      `mapstructure:"app"`
	Database DatabaseConfig `mapstructure:"database"`
	Redis    RedisConfig    `mapstructure:"redis"`
	Queue    QueueConfig    `mapstructure:"queue"`
	Mailer   MailerConfig   `mapstructure:"mailer"`
	Cache    CacheConfig    `mapstructure:"cache"`
	Sessions SessionConfig  `mapstructure:"sessions"`
}

type AppConfig struct {
	Name          string `mapstructure:"name"`
	Port          int    `mapstructure:"port"`
	SecretKeyBase string `mapstructure:"secret_key_base"`
	AutoMigrate   bool   `mapstructure:"auto_migrate"`
	Env           string `mapstructure:"env"`
}

type DatabaseConfig struct {
	Host        string `mapstructure:"host"`
	Port        int    `mapstructure:"port"`
	Name        string `mapstructure:"name"`
	User        string `mapstructure:"user"`
	Password    string `mapstructure:"password"`
	Pool        int    `mapstructure:"pool"`
	SSLMode     string `mapstructure:"ssl_mode"`
	SlowQueryMs int    `mapstructure:"slow_query_ms"`
}

type RedisConfig struct {
	URL  string `mapstructure:"url"`
	Pool int    `mapstructure:"pool"`
	DB   int    `mapstructure:"db"`
}

type SessionConfig struct {
	Store     string `mapstructure:"store"`
	TTL       int    `mapstructure:"ttl"`
	KeyPrefix string `mapstructure:"key_prefix"`
}

type QueueConfig struct {
	Concurrency int           `mapstructure:"concurrency"`
	Queues      []QueueOption `mapstructure:"queues"`
}

type QueueOption struct {
	Name   string `mapstructure:"name"`
	Weight int    `mapstructure:"weight"`
}

type MailerConfig struct {
	SMTPHost string `mapstructure:"smtp_host"`
	SMTPPort int    `mapstructure:"smtp_port"`
	From     string `mapstructure:"from"`
}

type CacheConfig struct {
	TTL    int    `mapstructure:"ttl"`
	Prefix string `mapstructure:"prefix"`
}
