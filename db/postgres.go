package db

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/shaurya/gails/config"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// DB is the global database connection.
var DB *gorm.DB

// Connect establishes a PostgreSQL connection using GORM + pgx.
func Connect(cfg config.DatabaseConfig) (*gorm.DB, error) {
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=%s",
		cfg.Host, cfg.User, cfg.Password, cfg.Name, cfg.Port, cfg.SSLMode)

	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "development"
	}

	// Configure GORM logger based on environment
	var logLevel gormlogger.LogLevel
	if env == "development" {
		logLevel = gormlogger.Info // Log every SQL query
	} else {
		logLevel = gormlogger.Warn // Log only warnings + slow queries
	}

	slowThreshold := time.Duration(cfg.SlowQueryMs) * time.Millisecond
	if slowThreshold == 0 {
		slowThreshold = 200 * time.Millisecond
	}

	gormLogger := gormlogger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		gormlogger.Config{
			SlowThreshold:             slowThreshold,
			LogLevel:                  logLevel,
			IgnoreRecordNotFoundError: true,
			Colorful:                  env == "development",
		},
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: gormLogger,
	})
	if err != nil {
		return nil, fmt.Errorf("[Gails] ERROR: Cannot connect to PostgreSQL at %s:%d — %v", cfg.Host, cfg.Port, err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	sqlDB.SetMaxIdleConns(cfg.Pool / 2)
	sqlDB.SetMaxOpenConns(cfg.Pool)
	sqlDB.SetConnMaxLifetime(time.Hour)

	// Ping and fail fast
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("[Gails] ERROR: Cannot connect to PostgreSQL at %s:%d — %v", cfg.Host, cfg.Port, err)
	}

	DB = db
	return db, nil
}

// MustConnect connects or panics.
func MustConnect(cfg config.DatabaseConfig) *gorm.DB {
	db, err := Connect(cfg)
	if err != nil {
		log.Fatalf("%s", err.Error())
	}
	return db
}
