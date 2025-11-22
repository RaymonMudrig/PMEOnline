package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"pmeonline/pkg/ledger"
	"pmeonline/pkg/oms"
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
	ledgerPoint := ledger.CreateLedgerPoint(kafkaURL, kafkaTopic, "pmeoms", ctx)

	// Wait for LedgerPoint to be ready
	log.Println("[OMS] Waiting for LedgerPoint to be ready...")
	for !ledgerPoint.IsReady {
		time.Sleep(100 * time.Millisecond)
	}
	log.Println("[OMS] LedgerPoint is ready")

	// Initialize OMS
	log.Println("[OMS] Initializing OMS...")
	omsEngine := oms.NewOMS(ledgerPoint)

	// Subscribe to events
	log.Println("[OMS] Subscribing to ledger events...")
	syncHandler := &OMSSyncHandler{oms: omsEngine}
	ledgerPoint.Sync <- syncHandler

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

// OMSSyncHandler implements LedgerPointInterface to receive events
type OMSSyncHandler struct {
	oms *oms.OMS
}

func (h *OMSSyncHandler) SyncServiceStart(a ledger.ServiceStart) {
	log.Printf("[OMS] Service started: %s", a.StartID)
}

func (h *OMSSyncHandler) SyncParameter(a ledger.Parameter) {
	log.Printf("[OMS] Parameter updated: FlatFee=%.4f, BorrowFee=%.4f, LendFee=%.4f",
		a.FlatFee, a.BorrowingFee, a.LendingFee)
}

func (h *OMSSyncHandler) SyncSessionTime(a ledger.SessionTime) {
	log.Printf("[OMS] Session time updated: %s", a.Description)
}

func (h *OMSSyncHandler) SyncHoliday(a ledger.Holiday) {
	log.Printf("[OMS] Holiday added: %s (%s)", a.Date.Format("2006-01-02"), a.Description)
}

func (h *OMSSyncHandler) SyncAccount(a ledger.Account) {
	log.Printf("[OMS] Account synced: %s (%s)", a.Code, a.Name)
}

func (h *OMSSyncHandler) SyncAccountLimit(a ledger.AccountLimit) {
	log.Printf("[OMS] Account limit updated: %s - TradeLimit=%.2f, PoolLimit=%.2f",
		a.Code, a.TradeLimit, a.PoolLimit)
}

func (h *OMSSyncHandler) SyncParticipant(a ledger.Participant) {
	log.Printf("[OMS] Participant synced: %s (%s) - Borr:%v, Lend:%v",
		a.Code, a.Name, a.BorrEligibility, a.LendEligibility)
}

func (h *OMSSyncHandler) SyncInstrument(a ledger.Instrument) {
	log.Printf("[OMS] Instrument synced: %s (%s) - Eligible:%v",
		a.Code, a.Name, a.Status)
}

func (h *OMSSyncHandler) SyncOrder(a ledger.Order) {
	log.Printf("[OMS] New order received: %d (%s %s %.0f shares)",
		a.NID, a.Side, a.InstrumentCode, a.Quantity)

	// Process the order
	h.oms.ProcessOrder(a)
}

func (h *OMSSyncHandler) SyncOrderAck(a ledger.OrderAck) {
	log.Printf("[OMS] Order acknowledged: %d", a.OrderNID)
}

func (h *OMSSyncHandler) SyncOrderNak(a ledger.OrderNak) {
	log.Printf("[OMS] Order rejected: %d - %s", a.OrderNID, a.Message)
}

func (h *OMSSyncHandler) SyncOrderWithdraw(a ledger.OrderWithdraw) {
	log.Printf("[OMS] Order withdrawal requested: %d", a.OrderNID)
	h.oms.ProcessOrderWithdraw(a)
}

func (h *OMSSyncHandler) SyncOrderWithdrawAck(a ledger.OrderWithdrawAck) {
	log.Printf("[OMS] Order withdrawal acknowledged: %d", a.OrderNID)
}

func (h *OMSSyncHandler) SyncOrderWithdrawNak(a ledger.OrderWithdrawNak) {
	log.Printf("[OMS] Order withdrawal rejected: %d - %s", a.OrderNID, a.Message)
}

func (h *OMSSyncHandler) SyncTrade(a ledger.Trade) {
	log.Printf("[OMS] Trade created: %s (%.0f shares)", a.KpeiReff, a.Quantity)
}

func (h *OMSSyncHandler) SyncTradeWait(a ledger.TradeWait) {
	log.Printf("[OMS] Trade waiting for eClear approval: NID %d", a.TradeNID)
}

func (h *OMSSyncHandler) SyncTradeAck(a ledger.TradeAck) {
	log.Printf("[OMS] Trade approved by eClear: NID %d", a.TradeNID)
}

func (h *OMSSyncHandler) SyncTradeNak(a ledger.TradeNak) {
	log.Printf("[OMS] Trade rejected by eClear: NID %d - %s", a.TradeNID, a.Message)
}

func (h *OMSSyncHandler) SyncTradeReimburse(a ledger.TradeReimburse) {
	log.Printf("[OMS] Trade reimbursed: NID %d", a.TradeNID)
}

func (h *OMSSyncHandler) SyncContract(a ledger.Contract) {
	log.Printf("[OMS] Contract created: %s (%s side, %.0f shares)",
		a.KpeiReff, a.Side, a.Quantity)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
