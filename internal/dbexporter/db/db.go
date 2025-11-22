package db

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/lib/pq"
)

// DB wraps database connection
type DB struct {
	*sql.DB
}

// Config holds database configuration
type Config struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

// NewDB creates a new database connection
func NewDB(config Config) (*DB, error) {
	connStr := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		config.Host, config.Port, config.User, config.Password, config.DBName, config.SSLMode,
	)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.Println("[DB] Connected to PostgreSQL database")

	return &DB{db}, nil
}

// NewDBFromEnv creates a database connection from environment variables
func NewDBFromEnv() (*DB, error) {
	config := Config{
		Host:     getEnv("DB_HOST", "localhost"),
		Port:     getEnv("DB_PORT", "5432"),
		User:     getEnv("DB_USER", "pmeuser"),
		Password: getEnv("DB_PASSWORD", "pmepass"),
		DBName:   getEnv("DB_NAME", "pmedb"),
		SSLMode:  getEnv("DB_SSLMODE", "disable"),
	}

	return NewDB(config)
}

// RunMigrations runs SQL migration files
func (db *DB) RunMigrations(migrationsPath string) error {
	log.Println("[DB] Running migrations...")

	// Read migration file
	content, err := os.ReadFile(migrationsPath)
	if err != nil {
		return fmt.Errorf("failed to read migration file: %w", err)
	}

	// Execute migration
	if _, err := db.Exec(string(content)); err != nil {
		return fmt.Errorf("failed to execute migration: %w", err)
	}

	log.Println("[DB] Migrations completed successfully")
	return nil
}

// Close closes the database connection
func (db *DB) Close() error {
	log.Println("[DB] Closing database connection")
	return db.DB.Close()
}

// Helper function to get environment variable with default
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
