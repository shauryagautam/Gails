package db

import (
	"database/sql"
	"fmt"

	"github.com/pressly/goose/v3"
	"gorm.io/gorm"
)

// Migrate runs all pending migrations.
func Migrate(db *gorm.DB, dir string) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}
	return goose.Up(sqlDB, dir)
}

// Rollback rolls back migrations.
func Rollback(db *gorm.DB, dir string, steps int) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}
	if steps <= 0 {
		steps = 1
	}
	for i := 0; i < steps; i++ {
		if err := goose.Down(sqlDB, dir); err != nil {
			return err
		}
	}
	return nil
}

// MigrationStatus prints the migration status.
func MigrationStatus(db *gorm.DB, dir string) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}
	return goose.Status(sqlDB, dir)
}

// CreateDB creates a database by connecting to the 'postgres' default DB first.
func CreateDB(name string, host string, port int, user, password, sslMode string) error {
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s sslmode=%s dbname=postgres",
		host, port, user, password, sslMode)
	conn, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("[Gails] ERROR: Cannot connect to PostgreSQL — %v", err)
	}
	defer conn.Close()

	_, err = conn.Exec(fmt.Sprintf("CREATE DATABASE %s", name))
	if err != nil {
		return fmt.Errorf("[Gails] ERROR: Cannot create database %s — %v", name, err)
	}
	fmt.Printf("[Gails] Created database: %s\n", name)
	return nil
}

// DropDB drops a database by connecting to the 'postgres' default DB first.
func DropDB(name string, host string, port int, user, password, sslMode string) error {
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s sslmode=%s dbname=postgres",
		host, port, user, password, sslMode)
	conn, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("[Gails] ERROR: Cannot connect to PostgreSQL — %v", err)
	}
	defer conn.Close()

	// Terminate existing connections
	conn.Exec(fmt.Sprintf("SELECT pg_terminate_backend(pg_stat_activity.pid) FROM pg_stat_activity WHERE pg_stat_activity.datname = '%s' AND pid <> pg_backend_pid()", name))

	_, err = conn.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", name))
	if err != nil {
		return fmt.Errorf("[Gails] ERROR: Cannot drop database %s — %v", name, err)
	}
	fmt.Printf("[Gails] Dropped database: %s\n", name)
	return nil
}

// Seed runs a seed function.
func Seed(db *gorm.DB, seedFn func(*gorm.DB) error) error {
	return seedFn(db)
}
