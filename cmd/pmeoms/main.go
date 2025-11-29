package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"pmeonline/internal/pmeoms"
	"pmeonline/pkg/ledger"
)

func main() {
	log.Println("[OMS] Starting OMS (Order Management System) Service...")

	// Configuration from environment variables
	kafkaURL := getEnv("KAFKA_URL", "localhost:9092")
	kafkaTopic := getEnv("KAFKA_TOPIC", "pme-ledger")

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize LedgerPoint
	log.Println("[OMS] Initializing LedgerPoint...")
	ledgerPoint := ledger.CreateLedgerPoint(kafkaURL, kafkaTopic, "pmeoms")

	// Initialize OMS
	log.Println("[OMS] Initializing OMS...")
	omsEngine := pmeoms.NewOMS(ledgerPoint)

	// Subscribe to events
	log.Println("[OMS] Subscribing to ledger events...")
	syncHandler := pmeoms.NewSyncHandler(omsEngine, ledgerPoint)

	ledgerPoint.Start([]ledger.LedgerPointInterface{syncHandler}, ctx)

	// Wait for LedgerPoint to be ready
	log.Println("[OMS] Waiting for LedgerPoint to be ready...")
	for !ledgerPoint.IsReady {
		time.Sleep(100 * time.Millisecond)
	}
	log.Println("[OMS] LedgerPoint is ready")

	// Initialize existing orders from ledger (process saved and open orders)
	omsEngine.InitOrders()

	log.Println("[OMS] Service started and ready to process orders")

	// Display statistics periodically
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				stats := omsEngine.GetStatistics()
				log.Printf("[OMS] Statistics: %+v", stats)
			case <-ctx.Done():
				return
			}
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("[OMS] Shutting down service...")
	cancel()
	log.Println("[OMS] Service stopped")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
