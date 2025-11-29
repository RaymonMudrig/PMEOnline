package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"pmeonline/internal/eclearapi/handler"
	"pmeonline/pkg/ledger"
)

func main() {
	log.Println("🚀 Starting eClear API Service...")

	// Configuration from environment variables
	kafkaURL := getEnv("KAFKA_URL", "localhost:9092")
	kafkaTopic := getEnv("KAFKA_TOPIC", "pme-ledger")
	apiPort := getEnv("API_PORT", "8081")
	eclearBaseURL := getEnv("ECLEAR_BASE_URL", "http://localhost:9000")

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize LedgerPoint
	log.Println("📊 Initializing LedgerPoint...")
	ledgerPoint := ledger.CreateLedgerPoint(kafkaURL, kafkaTopic, "eclearapi")

	// Initialize outbound client (for sending trades to eClear)
	// This must be created BEFORE starting LedgerPoint to receive all events
	log.Println("📤 Initializing eClear outbound client...")
	eclearClient := handler.NewEClearClient(eclearBaseURL, ledgerPoint)
	eclearSyncHandler := eclearClient.GetSyncHandler()

	// Collect all subscribers
	subscribers := []ledger.LedgerPointInterface{
		eclearSyncHandler,
	}

	// Start LedgerPoint with all subscribers
	log.Println("🚀 Starting LedgerPoint with subscribers...")
	ledgerPoint.Start(subscribers, ctx)

	// Wait for LedgerPoint to be ready
	log.Println("⏳ Waiting for LedgerPoint to be ready...")
	for !ledgerPoint.IsReady {
		time.Sleep(100 * time.Millisecond)
	}
	log.Println("✅ LedgerPoint is ready")

	// Start outbound client processing (after LedgerPoint is ready)
	log.Println("▶️  Starting eClear outbound client...")
	go eclearClient.RunProcessing(ctx)

	// Initialize handlers (these don't need event subscription)
	masterDataHandler := handler.NewMasterDataHandler(ledgerPoint)
	tradeHandler := handler.NewTradeHandler(ledgerPoint)
	queryHandler := handler.NewQueryHandler(ledgerPoint)
	settingsHandler := handler.NewSettingsHandler(ledgerPoint)

	// Setup HTTP router
	mux := http.NewServeMux()

	// Master Data endpoints (Inbound from eClear)
	mux.HandleFunc("POST /account/insert", masterDataHandler.InsertAccounts)
	mux.HandleFunc("POST /instrument/insert", masterDataHandler.InsertInstruments)
	mux.HandleFunc("POST /participant/insert", masterDataHandler.InsertParticipants)
	mux.HandleFunc("POST /account/limit", masterDataHandler.UpdateAccountLimit)

	// Trade Approval endpoints (Inbound from eClear)
	mux.HandleFunc("POST /contract/matched", tradeHandler.MatchedConfirm)
	mux.HandleFunc("POST /contract/reimburse", tradeHandler.Reimburse)
	mux.HandleFunc("POST /lender/recall", tradeHandler.LenderRecall)

	// Query endpoints (for dashboard and external queries)
	mux.HandleFunc("GET /participant/list", queryHandler.GetParticipants)
	mux.HandleFunc("GET /instrument/list", queryHandler.GetInstruments)
	mux.HandleFunc("GET /account/list", queryHandler.GetAccounts)

	// Settings endpoints (for parameter, holiday, sessiontime management)
	mux.HandleFunc("GET /parameter", settingsHandler.GetParameter)
	mux.HandleFunc("POST /parameter/update", settingsHandler.UpdateParameter)
	mux.HandleFunc("GET /holiday/list", settingsHandler.GetHolidays)
	mux.HandleFunc("POST /holiday/add", settingsHandler.AddHoliday)
	mux.HandleFunc("GET /sessiontime", settingsHandler.GetSessionTime)
	mux.HandleFunc("POST /sessiontime/update", settingsHandler.UpdateSessionTime)

	// Serve static files
	fs := http.FileServer(http.Dir("../../web/static/eclearapi"))
	mux.Handle("GET /static/", http.StripPrefix("/static/", fs))

	// Dashboard
	mux.HandleFunc("GET /", serveDashboard)
	mux.HandleFunc("GET /dashboard", serveDashboard)

	// Health check endpoint
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok","service":"eclearapi"}`))
	})

	// Create HTTP server with CORS and logging middleware
	server := &http.Server{
		Addr:         ":" + apiPort,
		Handler:      loggingMiddleware(corsMiddleware(mux)),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		log.Printf("🌐 eClear API listening on port %s\n", apiPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("❌ Server error: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("🛑 Shutting down server...")

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("❌ Server forced to shutdown: %v", err)
	}

	cancel() // Cancel main context
	log.Println("✅ Server stopped")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		log.Printf("📨 %s %s", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
		log.Printf("✅ %s %s completed in %v", r.Method, r.URL.Path, time.Since(start))
	})
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func serveDashboard(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "../../web/static/eclearapi/index.html")
}
