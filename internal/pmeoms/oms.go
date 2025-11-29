package pmeoms

import (
	"log"
	"sync"

	"pmeonline/pkg/ledger"
	"pmeonline/pkg/ledger/risk"
)

// OMS represents the Order Management System
type OMS struct {
	ledger        *ledger.LedgerPoint
	validator     *risk.Validator
	checker       *risk.Checker
	matcher       *Matcher
	tradeGen      *TradeGenerator
	mu            sync.RWMutex
	instrumentMap map[string]bool // Track instrument eligibility
}

// NewOMS creates a new OMS instance
func NewOMS(l *ledger.LedgerPoint) *OMS {
	calculator := risk.NewCalculator(l)
	validator := risk.NewValidator(l)
	checker := risk.NewChecker(l)
	matcher := NewMatcher()
	tradeGen := NewTradeGenerator(calculator)

	oms := &OMS{
		ledger:        l,
		validator:     validator,
		checker:       checker,
		matcher:       matcher,
		tradeGen:      tradeGen,
		instrumentMap: make(map[string]bool),
	}

	// Set up eligibility handlers
	checker.SetInstrumentIneligibleHandler(oms.handleInstrumentIneligible)
	checker.SetInstrumentEligibleHandler(oms.handleInstrumentEligible)

	return oms
}

// InitOrders processes all existing orders after ledger sync completes
// This should be called after ledger.IsReady becomes true
func (oms *OMS) InitOrders() {
	log.Printf("🔄 Initializing orders from ledger...")

	// Process all orders in state "S" (Saved - not yet acknowledged)
	savedCount := 0
	oms.ledger.ForEachOrder(func(order ledger.OrderEntity) bool {
		if order.State == "S" {
			savedCount++
			log.Printf("[OMS] Processing saved order: %d", order.NID)
			oms.ProcessOrder(order.NID)
		}
		return true // Continue iteration
	})

	// Match all orders in state "O" (Open - acknowledged and ready for matching)
	openCount := 0
	oms.ledger.ForEachOrder(func(order ledger.OrderEntity) bool {
		if order.State == "O" {
			openCount++
			log.Printf("[OMS] Matching open order: %d", order.NID)
			oms.MatchOrder(order.NID)
		}
		return true // Continue iteration
	})

	log.Printf("✅ Order initialization complete - Processed %d saved orders, matched %d open orders",
		savedCount, openCount)
}

// ProcessOrder handles a new order by OrderNID
func (oms *OMS) ProcessOrder(orderNID int) {
	// Get order from ledger
	orderEntity, exists := oms.ledger.GetOrder(orderNID)
	if !exists {
		log.Printf("❌ Order %d not found in ledger", orderNID)
		return
	}

	log.Printf("📥 Processing order: %d (%s %s %.0f shares)",
		orderEntity.NID, orderEntity.Side, orderEntity.InstrumentCode, orderEntity.Quantity)

	// Step 1: Validate order
	if err := oms.validator.ValidateOrder(orderEntity); err != nil {
		log.Printf("❌ Order %d validation failed: %v", orderNID, err)
		oms.ledger.Commit <- ledger.OrderNak{
			OrderNID: orderNID,
			Message:  err.Error(),
		}
		return
	}

	// Step 2: Check if order should be pending (future settlement date)
	if oms.validator.IsPendingNew(orderEntity) {
		log.Printf("⏰ Order %d is pending (settlement date: %s)",
			orderNID, orderEntity.SettlementDate.Format("2006-01-02"))

		oms.ledger.Commit <- ledger.OrderPending{OrderNID: orderNID}

		// Pending orders stay in state "G" (Pending) and will be acknowledged during SOD
		// when their settlement date arrives
		log.Printf("📋 Order %d will be acknowledged during SOD on %s",
			orderNID, orderEntity.SettlementDate.Format("2006-01-02"))
		return
	}

	// Step 3: Acknowledge order (risk checks passed)
	oms.ledger.Commit <- ledger.OrderAck{OrderNID: orderNID}
	log.Printf("✅ Order %d acknowledged", orderNID)
	// Note: Matching will be performed when SyncOrderAck is received
}

// MatchOrder attempts to match an order by OrderNID
func (oms *OMS) MatchOrder(orderNID int) {
	// Get order from ledger
	orderEntity, exists := oms.ledger.GetOrder(orderNID)
	if !exists {
		log.Printf("❌ Order %d not found in ledger", orderNID)
		return
	}

	oms.mu.Lock()
	defer oms.mu.Unlock()

	log.Printf("🔄 Matching order: %d (%s %s %.0f shares)",
		orderEntity.NID, orderEntity.Side, orderEntity.InstrumentCode, orderEntity.Quantity)

	// Check if instrument is eligible
	if !oms.isInstrumentEligible(orderEntity.InstrumentCode) {
		log.Printf("⚠️  Instrument %s is ineligible, order %d cannot be matched",
			orderEntity.InstrumentCode, orderEntity.NID)
		// Add to order book but don't match
		oms.matcher.AddOrder(orderEntity)
		return
	}

	// Perform matching
	matchResult := oms.matcher.Match(orderEntity)

	// Generate trades for matches
	if len(matchResult.Matches) > 0 {
		trades := oms.tradeGen.GenerateTrades(matchResult.Matches)

		for _, trade := range trades {
			log.Printf("📝 Generated trade: %s (%.0f shares)", trade.KpeiReff, trade.Quantity)
			oms.ledger.Commit <- trade
		}
	}

	// If order is not fully matched, add remaining to order book
	if !matchResult.FullyMatched {
		oms.matcher.AddOrder(orderEntity)
		log.Printf("📋 Order %d added to order book (%.0f shares remaining)",
			orderEntity.NID, matchResult.RemainingQty)
	}
}

// ProcessOrderWithdraw handles order withdrawal by OrderNID
func (oms *OMS) ProcessOrderWithdraw(orderNID int) {
	log.Printf("📥 Processing withdrawal for order: %d", orderNID)

	// Get order from ledger
	orderEntity, exists := oms.ledger.GetOrder(orderNID)
	if !exists {
		log.Printf("❌ Order %d not found", orderNID)
		oms.ledger.Commit <- ledger.OrderWithdrawNak{
			OrderNID: orderNID,
			Message:  "Order not found",
		}
		return
	}

	// Check if order can be withdrawn (must be Open or Partial)
	if orderEntity.State != "O" && orderEntity.State != "P" {
		log.Printf("❌ Order %d cannot be withdrawn (state: %s)", orderNID, orderEntity.State)
		oms.ledger.Commit <- ledger.OrderWithdrawNak{
			OrderNID: orderNID,
			Message:  "Order cannot be withdrawn in current state",
		}
		return
	}

	// Remove from order book
	oms.mu.Lock()
	removed := oms.matcher.RemoveOrder(orderEntity)
	oms.mu.Unlock()

	if removed {
		log.Printf("✅ Order %d removed from order book", orderNID)
	}

	// Acknowledge withdrawal
	oms.ledger.Commit <- ledger.OrderWithdrawAck{OrderNID: orderNID}
	log.Printf("✅ Order %d withdrawal acknowledged", orderNID)
}

// handleInstrumentIneligible handles instrument becoming ineligible
func (oms *OMS) handleInstrumentIneligible(instrumentCode string) {
	oms.mu.Lock()
	defer oms.mu.Unlock()

	log.Printf("⚠️  Instrument %s became ineligible - blocking matching", instrumentCode)
	oms.instrumentMap[instrumentCode] = false

	// Find all open orders for this instrument and mark them as blocked
	ineligibleOrders := oms.checker.GetIneligibleOrders()

	for _, nid := range ineligibleOrders {
		log.Printf("🚫 Blocking order %d due to instrument ineligibility", nid)
		// In a real implementation, you would emit OrderBlock event
		// For now, we just log it
	}
}

// handleInstrumentEligible handles instrument becoming eligible again
func (oms *OMS) handleInstrumentEligible(instrumentCode string) {
	oms.mu.Lock()
	defer oms.mu.Unlock()

	log.Printf("✅ Instrument %s is now eligible - enabling matching", instrumentCode)
	oms.instrumentMap[instrumentCode] = true

	// Re-match any orders that were blocked
	log.Printf("🔄 Re-matching blocked orders for %s", instrumentCode)
	// In a real implementation, you would trigger re-matching
}

// isInstrumentEligible checks if an instrument is eligible for matching
func (oms *OMS) isInstrumentEligible(instrumentCode string) bool {
	// Check cached status first
	if status, exists := oms.instrumentMap[instrumentCode]; exists {
		return status
	}

	// Check from ledger
	instrument, exists := oms.ledger.GetInstrument(instrumentCode)
	if !exists {
		return false
	}

	oms.instrumentMap[instrumentCode] = instrument.Status
	return instrument.Status
}

// GetSBLData returns SBL data for display
func (oms *OMS) GetSBLData(instrumentCode string) (borrowOrders, lendOrders []*QueuedOrder) {
	oms.mu.RLock()
	defer oms.mu.RUnlock()

	return oms.matcher.GetSBLData(instrumentCode)
}

// GetStatistics returns OMS statistics
func (oms *OMS) GetStatistics() map[string]interface{} {
	oms.mu.RLock()
	defer oms.mu.RUnlock()

	instruments := oms.matcher.GetAllInstruments()
	stats := make(map[string]interface{})

	stats["total_instruments"] = len(instruments)
	stats["instruments"] = make([]string, 0)

	for _, code := range instruments {
		stats["instruments"] = append(stats["instruments"].([]string),
			oms.matcher.GetOrderBookStats(code))
	}

	return stats
}
