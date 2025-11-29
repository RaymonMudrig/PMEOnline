package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"pmeonline/internal/pmeapi/handler"
	"pmeonline/internal/pmeapi/middleware"
	"pmeonline/internal/pmeapi/websocket"
	"pmeonline/pkg/idgen"
	"pmeonline/pkg/ledger"
)

func main() {
	log.Println("[APME-API] Starting APME API Service...")

	// Configuration from environment variables
	kafkaURL := getEnv("KAFKA_URL", "localhost:9092")
	kafkaTopic := getEnv("KAFKA_TOPIC", "pme-ledger")
	apiPort := getEnv("API_PORT", "8080")
	instanceIDStr := getEnv("INSTANCE_ID", "0")

	// Parse instance ID
	instanceID, err := strconv.ParseInt(instanceIDStr, 10, 64)
	if err != nil || instanceID < 0 || instanceID > 1023 {
		log.Fatalf("[APME-API] Invalid INSTANCE_ID: must be 0-1023, got '%s'", instanceIDStr)
	}
	log.Printf("[APME-API] Instance ID: %d", instanceID)

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize LedgerPoint
	log.Println("[APME-API] Initializing LedgerPoint...")
	ledgerPoint := ledger.CreateLedgerPoint(kafkaURL, kafkaTopic, "pmeapi", ctx)

	// Wait for LedgerPoint to be ready
	log.Println("[APME-API] Waiting for LedgerPoint to be ready...")
	for !ledgerPoint.IsReady {
		time.Sleep(100 * time.Millisecond)
	}
	log.Println("[APME-API] LedgerPoint is ready")

	// Initialize Snowflake ID generator
	idGenerator, err := idgen.NewGenerator(instanceID)
	if err != nil {
		log.Fatalf("[APME-API] Failed to create ID generator: %v", err)
	}
	log.Printf("[APME-API] Snowflake ID generator initialized (instance: %d)", instanceID)

	// Initialize WebSocket hub
	hub := websocket.NewHub()
	go hub.Run(ctx)

	// Subscribe to events for notifications
	notifier := websocket.NewNotifier(hub, ledgerPoint)
	ledgerPoint.Sync <- notifier

	// Initialize handlers
	orderHandler := handler.NewOrderHandler(ledgerPoint, idGenerator)
	queryHandler := handler.NewQueryHandler(ledgerPoint)
	sblHandler := handler.NewSBLHandler(ledgerPoint)

	// Setup HTTP router
	mux := http.NewServeMux()

	// Order Management endpoints
	mux.HandleFunc("POST /api/order/new", orderHandler.NewOrder)
	mux.HandleFunc("POST /api/order/amend", orderHandler.AmendOrder)
	mux.HandleFunc("POST /api/order/withdraw", orderHandler.WithdrawOrder)

	// Query endpoints
	mux.HandleFunc("GET /api/account/info", queryHandler.GetAccountInfo)
	mux.HandleFunc("GET /api/order/list", queryHandler.GetOrderList)
	mux.HandleFunc("GET /api/contract/list", queryHandler.GetContractList)

	// SBL endpoints
	mux.HandleFunc("GET /api/sbl/detail", sblHandler.GetSBLDetail)
	mux.HandleFunc("GET /api/sbl/aggregate", sblHandler.GetSBLAggregate)

	// WebSocket endpoint
	mux.HandleFunc("GET /ws/notifications", func(w http.ResponseWriter, r *http.Request) {
		websocket.ServeWs(hub, w, r)
	})

	// Serve static files
	fs := http.FileServer(http.Dir("../../web/static/pmeapi"))
	mux.Handle("GET /static/", http.StripPrefix("/static/", fs))

	// Dashboard
	mux.HandleFunc("GET /", serveDashboard)
	mux.HandleFunc("GET /dashboard", serveDashboard)

	// Health check endpoint
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok","service":"pmeapi"}`))
	})

	// Apply middleware
	handler := middleware.LoggingMiddleware(
		middleware.CORSMiddleware(mux),
	)

	// Create HTTP server
	server := &http.Server{
		Addr:         ":" + apiPort,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		log.Printf("[APME-API] API listening on port %s\n", apiPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("[APME-API] Server error: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("[APME-API] Shutting down server...")

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("[APME-API] Server forced to shutdown: %v", err)
	}

	cancel() // Cancel main context
	log.Println("[APME-API] Server stopped")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func serveDashboard(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "../../web/static/pmeapi/index.html")
}
