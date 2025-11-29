package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"pmeonline/internal/dbexporter/db"
	"pmeonline/internal/dbexporter/exporter"
	"pmeonline/pkg/ledger"
)

func main() {
	log.Println("[DB-EXPORTER] Starting Database Exporter Service...")

	// Get configuration from environment
	kafkaURL := getEnv("KAFKA_URL", "localhost:9092")
	kafkaTopic := getEnv("KAFKA_TOPIC", "pme-ledger")

	log.Printf("[DB-EXPORTER] Kafka URL: %s", kafkaURL)
	log.Printf("[DB-EXPORTER] Kafka Topic: %s", kafkaTopic)

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Connect to database
	database, err := db.NewDBFromEnv()
	if err != nil {
		log.Fatalf("[DB-EXPORTER] Failed to connect to database: %v", err)
	}
	defer database.Close()

	// Run migrations
	migrationsPath := filepath.Join("migrations", "001_create_tables.sql")
	if err := database.RunMigrations(migrationsPath); err != nil {
		log.Fatalf("[DB-EXPORTER] Failed to run migrations: %v", err)
	}

	// Create exporter
	exp := exporter.NewExporter(database.DB)

	// Create LedgerPoint
	log.Println("[DB-EXPORTER] Initializing LedgerPoint...")
	ledgerPoint := ledger.CreateLedgerPoint(kafkaURL, kafkaTopic, "dbexporter")

	// Collect all subscribers
	subscribers := []ledger.LedgerPointInterface{
		exp,
	}

	// Start LedgerPoint with all subscribers
	log.Println("[DB-EXPORTER] Starting LedgerPoint with subscribers...")
	ledgerPoint.Start(subscribers, ctx)

	log.Println("[DB-EXPORTER] Database Exporter Service started successfully")
	log.Println("[DB-EXPORTER] Listening for Kafka events...")
	log.Println("[DB-EXPORTER] Press Ctrl+C to stop")

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	<-sigChan
	log.Println("[DB-EXPORTER] Shutdown signal received")

	// Cancel context to stop LedgerPoint
	cancel()

	log.Println("[DB-EXPORTER] Database Exporter Service stopped")
}

// Helper function to get environment variable with default
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
