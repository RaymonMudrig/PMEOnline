package oms

import (
	"fmt"
	"log"

	"pmeonline/pkg/ledger"
)

// MatchResult represents the result of a matching operation
type MatchResult struct {
	Matches       []Match
	RemainingQty  float64
	FullyMatched  bool
}

// Match represents a single match between two orders
type Match struct {
	BorrowerOrder ledger.OrderEntity
	LenderOrder   ledger.OrderEntity
	Quantity      float64
}

// Matcher handles order matching logic
type Matcher struct {
	orderBooks map[string]*OrderBook // Map of instrument code to order book
}

// NewMatcher creates a new matcher instance
func NewMatcher() *Matcher {
	return &Matcher{
		orderBooks: make(map[string]*OrderBook),
	}
}

// GetOrCreateOrderBook gets or creates an order book for an instrument
func (m *Matcher) GetOrCreateOrderBook(instrumentCode string) *OrderBook {
	if ob, exists := m.orderBooks[instrumentCode]; exists {
		return ob
	}

	ob := NewOrderBook(instrumentCode)
	m.orderBooks[instrumentCode] = ob
	return ob
}

// AddOrder adds an order to the order book
func (m *Matcher) AddOrder(order ledger.OrderEntity) {
	ob := m.GetOrCreateOrderBook(order.InstrumentCode)
	ob.AddOrder(order, order.ParticipantCode)
}

// RemoveOrder removes an order from the order book
func (m *Matcher) RemoveOrder(order ledger.OrderEntity) bool {
	ob, exists := m.orderBooks[order.InstrumentCode]
	if !exists {
		return false
	}

	return ob.RemoveOrder(order.NID, order.Side)
}

// Match attempts to match an order against the order book
// Returns a MatchResult with all matches found
func (m *Matcher) Match(order ledger.OrderEntity) *MatchResult {
	ob := m.GetOrCreateOrderBook(order.InstrumentCode)

	result := &MatchResult{
		Matches:      make([]Match, 0),
		RemainingQty: order.Quantity,
	}

	// Get matchable orders (sorted by priority)
	matchableOrders := ob.GetMatchableOrders(order)

	log.Printf("🔍 Attempting to match order %d (%s %s %.0f shares) - Found %d potential matches",
		order.NID, order.Side, order.InstrumentCode, order.Quantity, len(matchableOrders))

	// Try to match with each order
	for _, queuedOrder := range matchableOrders {
		if result.RemainingQty <= 0 {
			break
		}

		matchOrder := queuedOrder.Order

		// Calculate match quantity (minimum of remaining and available)
		availableQty := matchOrder.Quantity - matchOrder.DoneQuantity
		matchQty := min(result.RemainingQty, availableQty)

		if matchQty <= 0 {
			continue
		}

		// Create match
		var match Match
		if order.Side == "BORR" {
			match = Match{
				BorrowerOrder: order,
				LenderOrder:   matchOrder,
				Quantity:      matchQty,
			}
		} else {
			match = Match{
				BorrowerOrder: matchOrder,
				LenderOrder:   order,
				Quantity:      matchQty,
			}
		}

		result.Matches = append(result.Matches, match)
		result.RemainingQty -= matchQty

		log.Printf("✅ Matched %.0f shares: Order %d (%s) <-> Order %d (%s)",
			matchQty, order.NID, order.Side, matchOrder.NID, matchOrder.Side)
	}

	result.FullyMatched = result.RemainingQty <= 0

	if result.FullyMatched {
		log.Printf("🎯 Order %d FULLY matched (%.0f shares)", order.NID, order.Quantity)
	} else if len(result.Matches) > 0 {
		log.Printf("⚡ Order %d PARTIALLY matched (%.0f/%.0f shares)",
			order.NID, order.Quantity-result.RemainingQty, order.Quantity)
	} else {
		log.Printf("📋 Order %d queued (no matches found)", order.NID)
	}

	return result
}

// GetSBLData returns all open orders for an instrument
func (m *Matcher) GetSBLData(instrumentCode string) (borrowOrders, lendOrders []*QueuedOrder) {
	ob, exists := m.orderBooks[instrumentCode]
	if !exists {
		return nil, nil
	}

	ob.mu.RLock()
	defer ob.mu.RUnlock()

	return ob.BorrowOrders.GetAllOrders(), ob.LendOrders.GetAllOrders()
}

// GetAllInstruments returns all instruments with orders
func (m *Matcher) GetAllInstruments() []string {
	instruments := make([]string, 0, len(m.orderBooks))
	for code := range m.orderBooks {
		instruments = append(instruments, code)
	}
	return instruments
}

// GetOrderBookStats returns statistics for an order book
func (m *Matcher) GetOrderBookStats(instrumentCode string) string {
	ob, exists := m.orderBooks[instrumentCode]
	if !exists {
		return fmt.Sprintf("No order book for %s", instrumentCode)
	}

	ob.mu.RLock()
	defer ob.mu.RUnlock()

	borrowCount := ob.BorrowOrders.Count()
	lendCount := ob.LendOrders.Count()

	return fmt.Sprintf("Instrument: %s - Borrow: %d orders, Lend: %d orders",
		instrumentCode, borrowCount, lendCount)
}

// min returns the minimum of two float64 values
func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
