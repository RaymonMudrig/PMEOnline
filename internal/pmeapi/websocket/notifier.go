package websocket

import (
	"log"

	"pmeonline/pkg/ledger"
)

// Notifier listens to ledger events and sends WebSocket notifications
type Notifier struct {
	hub    *Hub
	ledger *ledger.LedgerPoint
}

// NewNotifier creates a new notifier
func NewNotifier(hub *Hub, l *ledger.LedgerPoint) *Notifier {
	return &Notifier{
		hub:    hub,
		ledger: l,
	}
}

// Send a notification to all connected clients
func (n *Notifier) sendNotification(notifType string, data map[string]interface{}) {
	// Add timestamp to data
	data["timestamp"] = ledger.GetCurrentTimeMillis()

	// Broadcast using sequenced notification
	seq := n.hub.BroadcastNotification(notifType, data)

	log.Printf("[NOTIFIER] Sent %s notification (seq: %d) to %d clients", notifType, seq, n.hub.ClientCount())
}

// Implement LedgerPointInterface methods

func (n *Notifier) SyncServiceStart(a ledger.ServiceStart) {
	// Don't notify on service start
}

func (n *Notifier) SyncParameter(a ledger.Parameter) {
	// Don't notify on parameter updates
}

func (n *Notifier) SyncSessionTime(a ledger.SessionTime) {
	// Don't notify on session time updates
}

func (n *Notifier) SyncHoliday(a ledger.Holiday) {
	// Don't notify on holiday updates
}

func (n *Notifier) SyncAccount(a ledger.Account) {
	// Don't notify on account updates
}

func (n *Notifier) SyncAccountLimit(a ledger.AccountLimit) {
	n.sendNotification("account_limit_updated", map[string]interface{}{
		"account_code": a.Code,
		"trade_limit":  a.TradeLimit,
		"pool_limit":   a.PoolLimit,
	})
}

func (n *Notifier) SyncParticipant(a ledger.Participant) {
	// Don't notify on participant updates
}

func (n *Notifier) SyncInstrument(a ledger.Instrument) {
	// Notify on instrument eligibility changes
	status := "eligible"
	if !a.Status {
		status = "ineligible"
	}

	n.sendNotification("instrument_status_changed", map[string]interface{}{
		"instrument_code": a.Code,
		"instrument_name": a.Name,
		"status":          status,
	})
}

func (n *Notifier) SyncOrder(a ledger.Order) {
	n.sendNotification("order_created", map[string]interface{}{
		"order_nid":       a.NID,
		"account_code":    a.AccountCode,
		"instrument":      a.InstrumentCode,
		"side":            a.Side,
		"quantity":        a.Quantity,
		"reff_request_id": a.ReffRequestID,
		"state":           "S",
	})
}

func (n *Notifier) SyncOrderAck(a ledger.OrderAck) {
	order, exists := n.ledger.GetOrder(a.OrderNID)
	if !exists {
		return
	}

	n.sendNotification("order_acknowledged", map[string]interface{}{
		"order_nid":    a.OrderNID,
		"account_code": order.AccountCode,
		"state":        "O",
	})
}

func (n *Notifier) SyncOrderNak(a ledger.OrderNak) {
	order, exists := n.ledger.GetOrder(a.OrderNID)
	if !exists {
		return
	}

	n.sendNotification("order_rejected", map[string]interface{}{
		"order_nid":    a.OrderNID,
		"account_code": order.AccountCode,
		"state":        "R",
		"message":      a.Message,
	})
}

func (n *Notifier) SyncOrderPending(a ledger.OrderPending) {
	order, exists := n.ledger.GetOrder(a.OrderNID)
	if !exists {
		return
	}

	n.sendNotification("order_pending", map[string]interface{}{
		"order_nid":    a.OrderNID,
		"account_code": order.AccountCode,
		"state":        "G",
	})
}

func (n *Notifier) SyncOrderWithdraw(a ledger.OrderWithdraw) {
	// Don't notify on withdrawal request (will notify on ack/nak)
}

func (n *Notifier) SyncOrderWithdrawAck(a ledger.OrderWithdrawAck) {
	order, exists := n.ledger.GetOrder(a.OrderNID)
	if !exists {
		return
	}

	n.sendNotification("order_withdrawn", map[string]interface{}{
		"order_nid":    a.OrderNID,
		"account_code": order.AccountCode,
		"state":        "W",
	})
}

func (n *Notifier) SyncOrderWithdrawNak(a ledger.OrderWithdrawNak) {
	order, exists := n.ledger.GetOrder(a.OrderNID)
	if !exists {
		return
	}

	n.sendNotification("order_withdrawal_rejected", map[string]interface{}{
		"order_nid":    a.OrderNID,
		"account_code": order.AccountCode,
		"message":      a.Message,
	})
}

func (n *Notifier) SyncTrade(a ledger.Trade) {
	// Extract account codes from contracts
	var borrowerAccount, lenderAccount string

	for _, contract := range a.Borrower {
		borrowerAccount = contract.AccountCode
		break
	}

	for _, contract := range a.Lender {
		lenderAccount = contract.AccountCode
		break
	}

	n.sendNotification("trade_matched", map[string]interface{}{
		"trade_nid":        a.NID,
		"kpei_reff":        a.KpeiReff,
		"instrument":       a.InstrumentCode,
		"quantity":         a.Quantity,
		"borrower_account": borrowerAccount,
		"lender_account":   lenderAccount,
		"matched_at":       a.MatchedAt,
	})
}

func (n *Notifier) SyncTradeWait(a ledger.TradeWait) {
	trade, exists := n.ledger.GetTrade(a.TradeNID)
	if !exists {
		return
	}

	n.sendNotification("trade_pending_approval", map[string]interface{}{
		"trade_nid": a.TradeNID,
		"kpei_reff": trade.KpeiReff,
		"status":    "waiting_eclear_approval",
	})
}

func (n *Notifier) SyncTradeAck(a ledger.TradeAck) {
	trade, exists := n.ledger.GetTrade(a.TradeNID)
	if !exists {
		return
	}

	n.sendNotification("trade_approved", map[string]interface{}{
		"trade_nid": a.TradeNID,
		"kpei_reff": trade.KpeiReff,
		"status":    "approved",
	})
}

func (n *Notifier) SyncTradeNak(a ledger.TradeNak) {
	trade, exists := n.ledger.GetTrade(a.TradeNID)
	if !exists {
		return
	}

	n.sendNotification("trade_rejected", map[string]interface{}{
		"trade_nid": a.TradeNID,
		"kpei_reff": trade.KpeiReff,
		"message":   a.Message,
	})
}

func (n *Notifier) SyncTradeReimburse(a ledger.TradeReimburse) {
	trade, exists := n.ledger.GetTrade(a.TradeNID)
	if !exists {
		return
	}

	n.sendNotification("trade_reimbursed", map[string]interface{}{
		"trade_nid": a.TradeNID,
		"kpei_reff": trade.KpeiReff,
		"status":    "reimbursed",
	})
}

func (n *Notifier) SyncContract(a ledger.Contract) {
	n.sendNotification("contract_created", map[string]interface{}{
		"contract_nid": a.NID,
		"trade_nid":    a.TradeNID,
		"kpei_reff":    a.KpeiReff,
		"side":         a.Side,
		"account_code": a.AccountCode,
		"instrument":   a.InstrumentCode,
		"quantity":     a.Quantity,
		"fee_daily":    a.FeeValDaily,
	})
}

func (n *Notifier) SyncSod(s ledger.Sod) {
	n.sendNotification("sod", map[string]interface{}{
		"date":    s.Date.Format("2006-01-02"),
		"message": "Start of Day - Market Opening",
	})
}

func (n *Notifier) SyncEod(e ledger.Eod) {
	n.sendNotification("eod", map[string]interface{}{
		"date":    e.Date.Format("2006-01-02"),
		"message": "End of Day - Market Closing",
	})
}
