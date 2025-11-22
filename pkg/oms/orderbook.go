package oms

import (
	"sort"
	"sync"
	"time"

	"pmeonline/pkg/ledger"
)

// OrderBook maintains buy and sell orders for a specific instrument
type OrderBook struct {
	InstrumentCode string
	BorrowOrders   *OrderQueue // Borrowing orders
	LendOrders     *OrderQueue // Lending orders
	mu             sync.RWMutex
}

// OrderQueue holds orders with priority handling
type OrderQueue struct {
	SameParticipant  []*QueuedOrder // In-house orders (same participant)
	CrossParticipant []*QueuedOrder // Cross-participant orders
	mu               sync.RWMutex
}

// QueuedOrder wraps an order entity with additional queue information
type QueuedOrder struct {
	Order     ledger.OrderEntity
	QueuedAt  time.Time
	Priority  int // Used for sorting
}

// NewOrderBook creates a new order book for an instrument
func NewOrderBook(instrumentCode string) *OrderBook {
	return &OrderBook{
		InstrumentCode: instrumentCode,
		BorrowOrders: &OrderQueue{
			SameParticipant:  make([]*QueuedOrder, 0),
			CrossParticipant: make([]*QueuedOrder, 0),
		},
		LendOrders: &OrderQueue{
			SameParticipant:  make([]*QueuedOrder, 0),
			CrossParticipant: make([]*QueuedOrder, 0),
		},
	}
}

// AddOrder adds an order to the appropriate queue
func (ob *OrderBook) AddOrder(order ledger.OrderEntity, targetParticipant string) {
	ob.mu.Lock()
	defer ob.mu.Unlock()

	queuedOrder := &QueuedOrder{
		Order:    order,
		QueuedAt: time.Now(),
	}

	if order.Side == "BORR" {
		ob.BorrowOrders.Add(queuedOrder, targetParticipant)
	} else if order.Side == "LEND" {
		ob.LendOrders.Add(queuedOrder, targetParticipant)
	}
}

// RemoveOrder removes an order by NID
func (ob *OrderBook) RemoveOrder(nid int, side string) bool {
	ob.mu.Lock()
	defer ob.mu.Unlock()

	if side == "BORR" {
		return ob.BorrowOrders.Remove(nid)
	} else if side == "LEND" {
		return ob.LendOrders.Remove(nid)
	}

	return false
}

// GetMatchableOrders returns orders that can match with the given order
// according to the matching rules in F.2
func (ob *OrderBook) GetMatchableOrders(order ledger.OrderEntity) []*QueuedOrder {
	ob.mu.RLock()
	defer ob.mu.RUnlock()

	var matchableOrders []*QueuedOrder

	if order.Side == "BORR" {
		// For borrow orders, match with lend orders
		// Priority: Same participant first, sorted by quantity DESC
		matchableOrders = ob.LendOrders.GetSorted(order.ParticipantCode, order.Side)
	} else if order.Side == "LEND" {
		// For lend orders, match with borrow orders
		// Priority: Same participant first, sorted by time ASC (FIFO)
		matchableOrders = ob.BorrowOrders.GetSorted(order.ParticipantCode, order.Side)
	}

	return matchableOrders
}

// Add adds an order to the queue
func (q *OrderQueue) Add(order *QueuedOrder, targetParticipant string) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if order.Order.ParticipantCode == targetParticipant {
		q.SameParticipant = append(q.SameParticipant, order)
	} else {
		q.CrossParticipant = append(q.CrossParticipant, order)
	}
}

// Remove removes an order by NID
func (q *OrderQueue) Remove(nid int) bool {
	q.mu.Lock()
	defer q.mu.Unlock()

	// Try same participant queue
	for i, order := range q.SameParticipant {
		if order.Order.NID == nid {
			q.SameParticipant = append(q.SameParticipant[:i], q.SameParticipant[i+1:]...)
			return true
		}
	}

	// Try cross participant queue
	for i, order := range q.CrossParticipant {
		if order.Order.NID == nid {
			q.CrossParticipant = append(q.CrossParticipant[:i], q.CrossParticipant[i+1:]...)
			return true
		}
	}

	return false
}

// GetSorted returns sorted orders for matching
// For Borrow (incoming Lend): Sort by Quantity DESC (prefer larger lenders)
// For Lend (incoming Borrow): Sort by Time ASC (FIFO)
func (q *OrderQueue) GetSorted(participantCode string, incomingSide string) []*QueuedOrder {
	q.mu.RLock()
	defer q.mu.RUnlock()

	// Combine same participant and cross participant orders
	// Same participant orders have priority
	result := make([]*QueuedOrder, 0)

	// Add same participant orders first
	for _, order := range q.SameParticipant {
		if order.Order.ParticipantCode == participantCode {
			result = append(result, order)
		}
	}

	// Sort same participant orders
	if incomingSide == "LEND" {
		// Incoming is LEND, so we're matching against BORR
		// Sort by time ASC (FIFO)
		sort.Slice(result, func(i, j int) bool {
			return result[i].Order.EntryAt.Before(result[j].Order.EntryAt)
		})
	} else {
		// Incoming is BORR, so we're matching against LEND
		// Sort by quantity DESC (prefer larger lenders)
		sort.Slice(result, func(i, j int) bool {
			if result[i].Order.Quantity != result[j].Order.Quantity {
				return result[i].Order.Quantity > result[j].Order.Quantity
			}
			// If quantities are equal, use time priority
			return result[i].Order.EntryAt.Before(result[j].Order.EntryAt)
		})
	}

	// If same participant matching didn't fully fill, add cross participant orders
	crossOrders := make([]*QueuedOrder, len(q.CrossParticipant))
	copy(crossOrders, q.CrossParticipant)

	// Sort cross participant orders
	if incomingSide == "LEND" {
		// Sort by time ASC (FIFO)
		sort.Slice(crossOrders, func(i, j int) bool {
			return crossOrders[i].Order.EntryAt.Before(crossOrders[j].Order.EntryAt)
		})
	} else {
		// Sort by quantity DESC
		sort.Slice(crossOrders, func(i, j int) bool {
			if crossOrders[i].Order.Quantity != crossOrders[j].Order.Quantity {
				return crossOrders[i].Order.Quantity > crossOrders[j].Order.Quantity
			}
			return crossOrders[i].Order.EntryAt.Before(crossOrders[j].Order.EntryAt)
		})
	}

	result = append(result, crossOrders...)

	return result
}

// GetAllOrders returns all orders in the queue
func (q *OrderQueue) GetAllOrders() []*QueuedOrder {
	q.mu.RLock()
	defer q.mu.RUnlock()

	result := make([]*QueuedOrder, 0)
	result = append(result, q.SameParticipant...)
	result = append(result, q.CrossParticipant...)

	return result
}

// Count returns the total number of orders in the queue
func (q *OrderQueue) Count() int {
	q.mu.RLock()
	defer q.mu.RUnlock()

	return len(q.SameParticipant) + len(q.CrossParticipant)
}
