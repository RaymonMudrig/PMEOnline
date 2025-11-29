package pmeoms

import (
	"log"

	"pmeonline/pkg/ledger"
)

// SyncHandler implements LedgerPointInterface to receive events
type SyncHandler struct {
	oms    *OMS
	ledger *ledger.LedgerPoint
}

// NewSyncHandler creates a new sync handler
func NewSyncHandler(oms *OMS, ledger *ledger.LedgerPoint) *SyncHandler {
	return &SyncHandler{
		oms:    oms,
		ledger: ledger,
	}
}

func (h *SyncHandler) SyncServiceStart(a ledger.ServiceStart) {
	log.Printf("[OMS] Service started: %s", a.StartID)
	// Note: Order initialization is handled by InitOrders() called after ledger is ready
}

func (h *SyncHandler) SyncParameter(a ledger.Parameter) {
	log.Printf("[OMS] Parameter updated: FlatFee=%.4f, BorrowFee=%.4f, LendFee=%.4f",
		a.FlatFee, a.BorrowingFee, a.LendingFee)
}

func (h *SyncHandler) SyncSessionTime(a ledger.SessionTime) {
	log.Printf("[OMS] Session time updated: %s", a.Description)
}

func (h *SyncHandler) SyncHoliday(a ledger.Holiday) {
	log.Printf("[OMS] Holiday added: %s (%s)", a.Date.Format("2006-01-02"), a.Description)
}

func (h *SyncHandler) SyncAccount(a ledger.Account) {
	log.Printf("[OMS] Account synced: %s (%s)", a.Code, a.Name)
}

func (h *SyncHandler) SyncAccountLimit(a ledger.AccountLimit) {
	log.Printf("[OMS] Account limit updated: %s - TradeLimit=%.2f, PoolLimit=%.2f",
		a.Code, a.TradeLimit, a.PoolLimit)
}

func (h *SyncHandler) SyncParticipant(a ledger.Participant) {
	log.Printf("[OMS] Participant synced: %s (%s) - Borr:%v, Lend:%v",
		a.Code, a.Name, a.BorrEligibility, a.LendEligibility)
}

func (h *SyncHandler) SyncInstrument(a ledger.Instrument) {
	log.Printf("[OMS] Instrument synced: %s (%s) - Eligible:%v",
		a.Code, a.Name, a.Status)
}

func (h *SyncHandler) SyncOrder(a ledger.Order) {
	log.Printf("[OMS] Order event: %d (%s %s %.0f shares)",
		a.NID, a.Side, a.InstrumentCode, a.Quantity)

	// Only process orders after initial sync is complete
	// During initial sync (IsReady=false), we're replaying historical events
	if h.ledger.IsReady {
		log.Printf("[OMS] Processing new order: %d", a.NID)
		h.oms.ProcessOrder(a.NID)
	}
}

func (h *SyncHandler) SyncOrderAck(a ledger.OrderAck) {
	log.Printf("[OMS] Order acknowledged: %d", a.OrderNID)

	// Only perform matching after initial sync is complete
	if h.ledger.IsReady {
		log.Printf("[OMS] Attempting to match acknowledged order: %d", a.OrderNID)
		h.oms.MatchOrder(a.OrderNID)
	}
}

func (h *SyncHandler) SyncOrderNak(a ledger.OrderNak) {
	log.Printf("[OMS] Order rejected: %d - %s", a.OrderNID, a.Message)
}

func (h *SyncHandler) SyncOrderPending(a ledger.OrderPending) {
	log.Printf("[OMS] Order pending: %d", a.OrderNID)
}

func (h *SyncHandler) SyncOrderWithdraw(a ledger.OrderWithdraw) {
	log.Printf("[OMS] Order withdrawal event: %d", a.OrderNID)

	// Only process withdrawals after initial sync is complete
	if h.ledger.IsReady {
		log.Printf("[OMS] Processing order withdrawal: %d", a.OrderNID)
		h.oms.ProcessOrderWithdraw(a.OrderNID)
	}
}

func (h *SyncHandler) SyncOrderWithdrawAck(a ledger.OrderWithdrawAck) {
	log.Printf("[OMS] Order withdrawal acknowledged: %d", a.OrderNID)
}

func (h *SyncHandler) SyncOrderWithdrawNak(a ledger.OrderWithdrawNak) {
	log.Printf("[OMS] Order withdrawal rejected: %d - %s", a.OrderNID, a.Message)
}

func (h *SyncHandler) SyncTrade(a ledger.Trade) {
	log.Printf("[OMS] Trade created: %s (%.0f shares)", a.KpeiReff, a.Quantity)
}

func (h *SyncHandler) SyncTradeWait(a ledger.TradeWait) {
	log.Printf("[OMS] Trade waiting for eClear approval: NID %d", a.TradeNID)
}

func (h *SyncHandler) SyncTradeAck(a ledger.TradeAck) {
	log.Printf("[OMS] Trade approved by eClear: NID %d", a.TradeNID)
}

func (h *SyncHandler) SyncTradeNak(a ledger.TradeNak) {
	log.Printf("[OMS] Trade rejected by eClear: NID %d - %s", a.TradeNID, a.Message)
}

func (h *SyncHandler) SyncTradeReimburse(a ledger.TradeReimburse) {
	log.Printf("[OMS] Trade reimbursed: NID %d", a.TradeNID)
}

func (h *SyncHandler) SyncContract(a ledger.Contract) {
	log.Printf("[OMS] Contract created: %s (%s side, %.0f shares)",
		a.KpeiReff, a.Side, a.Quantity)
}

func (h *SyncHandler) SyncSod(a ledger.Sod) {
	log.Printf("[OMS] 🌅 Start of Day: %s", a.Date.Format("2006-01-02"))
	// TODO: Implement SOD processing
	// - Activate orders with today's settlement date
	// - Process pending orders
	log.Printf("[OMS] SOD processing complete")
}

func (h *SyncHandler) SyncEod(a ledger.Eod) {
	log.Printf("[OMS] 🌆 End of Day: %s", a.Date.Format("2006-01-02"))
	// TODO: Implement EOD processing
	// - Calculate daily fees
	// - Drop open BORR orders (BORR orders valid for 1 day only)
	// - Generate EOD reports
	log.Printf("[OMS] EOD processing complete")
}
