package oms

import (
	"log"
	"sync"

	"pmeonline/pkg/ledger"
	"pmeonline/pkg/risk"
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

// ProcessOrder handles a new order
func (oms *OMS) ProcessOrder(order ledger.Order) {
	log.Printf("📥 Processing order: %d (%s %s %.0f shares)",
		order.NID, order.Side, order.InstrumentCode, order.Quantity)

	// Step 1: Validate order
	if err := oms.validator.ValidateOrder(order); err != nil {
		log.Printf("❌ Order %d validation failed: %v", order.NID, err)
		oms.ledger.Commit <- ledger.OrderNak{
			OrderNID: order.NID,
			Message:  err.Error(),
		}
		return
	}

	// Step 2: Check if order should be pending (future settlement date)
	if oms.validator.IsPendingNew(order) {
		log.Printf("⏰ Order %d is pending (settlement date: %s)",
			order.NID, order.SettlementDate.Format("2006-01-02"))
		// TODO: Implement pending order scheduler
		// For now, just acknowledge it
		oms.ledger.Commit <- ledger.OrderAck{OrderNID: order.NID}
		return
	}

	// Step 3: Acknowledge order (risk checks passed)
	oms.ledger.Commit <- ledger.OrderAck{OrderNID: order.NID}
	log.Printf("✅ Order %d acknowledged", order.NID)

	// Step 4: Convert to OrderEntity for matching
	orderEntity := oms.getOrderEntity(order.NID)
	if orderEntity == nil {
		log.Printf("⚠️  Order %d not found in ledger after ACK", order.NID)
		return
	}

	// Step 5: Attempt matching
	oms.MatchOrder(*orderEntity)
}

// MatchOrder attempts to match an order
func (oms *OMS) MatchOrder(order ledger.OrderEntity) {
	oms.mu.Lock()
	defer oms.mu.Unlock()

	log.Printf("🔄 Matching order: %d (%s %s %.0f shares)",
		order.NID, order.Side, order.InstrumentCode, order.Quantity)

	// Check if instrument is eligible
	if !oms.isInstrumentEligible(order.InstrumentCode) {
		log.Printf("⚠️  Instrument %s is ineligible, order %d cannot be matched",
			order.InstrumentCode, order.NID)
		// Add to order book but don't match
		oms.matcher.AddOrder(order)
		return
	}

	// Perform matching
	matchResult := oms.matcher.Match(order)

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
		oms.matcher.AddOrder(order)
		log.Printf("📋 Order %d added to order book (%.0f shares remaining)",
			order.NID, matchResult.RemainingQty)
	}
}

// ProcessOrderWithdraw handles order withdrawal
func (oms *OMS) ProcessOrderWithdraw(withdraw ledger.OrderWithdraw) {
	log.Printf("📥 Processing withdrawal for order: %d", withdraw.OrderNID)

	orderEntity := oms.getOrderEntity(withdraw.OrderNID)
	if orderEntity == nil {
		log.Printf("❌ Order %d not found", withdraw.OrderNID)
		oms.ledger.Commit <- ledger.OrderWithdrawNak{
			OrderNID: withdraw.OrderNID,
			Message:  "Order not found",
		}
		return
	}

	// Check if order can be withdrawn (must be Open or Partial)
	if orderEntity.State != "O" && orderEntity.State != "P" {
		log.Printf("❌ Order %d cannot be withdrawn (state: %s)", withdraw.OrderNID, orderEntity.State)
		oms.ledger.Commit <- ledger.OrderWithdrawNak{
			OrderNID: withdraw.OrderNID,
			Message:  "Order cannot be withdrawn in current state",
		}
		return
	}

	// Remove from order book
	oms.mu.Lock()
	removed := oms.matcher.RemoveOrder(*orderEntity)
	oms.mu.Unlock()

	if removed {
		log.Printf("✅ Order %d removed from order book", withdraw.OrderNID)
	}

	// Acknowledge withdrawal
	oms.ledger.Commit <- ledger.OrderWithdrawAck{OrderNID: withdraw.OrderNID}
	log.Printf("✅ Order %d withdrawal acknowledged", withdraw.OrderNID)
}

// handleInstrumentIneligible handles instrument becoming ineligible
func (oms *OMS) handleInstrumentIneligible(instrumentCode string) {
	oms.mu.Lock()
	defer oms.mu.Unlock()

	log.Printf("⚠️  Instrument %s became ineligible - blocking matching", instrumentCode)
	oms.instrumentMap[instrumentCode] = false

	// Find all open orders for this instrument and mark them as blocked
	ineligibleOrders := oms.checker.GetIneligibleOrders(oms.ledger.Orders)

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
	instrument, exists := oms.ledger.Instrument[instrumentCode]
	if !exists {
		return false
	}

	oms.instrumentMap[instrumentCode] = instrument.Status
	return instrument.Status
}

// getOrderEntity retrieves an order entity from the ledger
func (oms *OMS) getOrderEntity(nid int) *ledger.OrderEntity {
	if order, exists := oms.ledger.Orders[nid]; exists {
		return &order
	}
	return nil
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
