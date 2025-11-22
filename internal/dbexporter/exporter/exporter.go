package exporter

import (
	"database/sql"
	"log"

	"pmeonline/internal/dbexporter/repository"
	"pmeonline/pkg/ledger"
)

// Exporter implements LedgerPointInterface to export events to database
type Exporter struct {
	participantRepo *repository.ParticipantRepository
	instrumentRepo  *repository.InstrumentRepository
	accountRepo     *repository.AccountRepository
	orderRepo       *repository.OrderRepository
	tradeRepo       *repository.TradeRepository
	contractRepo    *repository.ContractRepository
	otherRepo       *repository.OtherRepository
}

// NewExporter creates a new exporter
func NewExporter(db *sql.DB) *Exporter {
	return &Exporter{
		participantRepo: repository.NewParticipantRepository(db),
		instrumentRepo:  repository.NewInstrumentRepository(db),
		accountRepo:     repository.NewAccountRepository(db),
		orderRepo:       repository.NewOrderRepository(db),
		tradeRepo:       repository.NewTradeRepository(db),
		contractRepo:    repository.NewContractRepository(db),
		otherRepo:       repository.NewOtherRepository(db),
	}
}

// SyncServiceStart handles ServiceStart events
func (e *Exporter) SyncServiceStart(s ledger.ServiceStart) {
	if err := e.otherRepo.InsertServiceStart(s); err != nil {
		log.Printf("[EXPORTER] Error inserting service start: %v", err)
		return
	}
	log.Printf("[EXPORTER] Service start recorded: %s", s.ID)
	e.logEvent("ServiceStart", s, ledger.GetCurrentTimeMillis())
}

// SyncParameter handles Parameter events
func (e *Exporter) SyncParameter(p ledger.Parameter) {
	if err := e.otherRepo.UpsertParameter(p); err != nil {
		log.Printf("[EXPORTER] Error upserting parameter: %v", err)
		return
	}
	log.Printf("[EXPORTER] Parameter updated")
	e.logEvent("Parameter", p, ledger.GetCurrentTimeMillis())
}

// SyncSessionTime handles SessionTime events
func (e *Exporter) SyncSessionTime(s ledger.SessionTime) {
	if err := e.otherRepo.UpsertSessionTime(s); err != nil {
		log.Printf("[EXPORTER] Error upserting session time: %v", err)
		return
	}
	log.Printf("[EXPORTER] Session time updated")
	e.logEvent("SessionTime", s, ledger.GetCurrentTimeMillis())
}

// SyncHoliday handles Holiday events
func (e *Exporter) SyncHoliday(h ledger.Holiday) {
	if err := e.otherRepo.UpsertHoliday(h); err != nil {
		log.Printf("[EXPORTER] Error upserting holiday: %v", err)
		return
	}
	log.Printf("[EXPORTER] Holiday upserted: %s - %s", h.Date.Format("2006-01-02"), h.Description)
	e.logEvent("Holiday", h, ledger.GetCurrentTimeMillis())
}

// SyncAccount handles Account events
func (e *Exporter) SyncAccount(a ledger.Account) {
	if err := e.accountRepo.Upsert(a); err != nil {
		log.Printf("[EXPORTER] Error upserting account: %v", err)
		return
	}
	log.Printf("[EXPORTER] Account upserted: %s - %s", a.Code, a.Name)
	e.logEvent("Account", a, ledger.GetCurrentTimeMillis())
}

// SyncAccountLimit handles AccountLimit events
func (e *Exporter) SyncAccountLimit(a ledger.AccountLimit) {
	if err := e.accountRepo.UpdateLimit(a); err != nil {
		log.Printf("[EXPORTER] Error updating account limit: %v", err)
		return
	}
	log.Printf("[EXPORTER] Account limit updated: %s", a.Code)
	e.logEvent("AccountLimit", a, ledger.GetCurrentTimeMillis())
}

// SyncParticipant handles Participant events
func (e *Exporter) SyncParticipant(p ledger.Participant) {
	if err := e.participantRepo.Upsert(p); err != nil {
		log.Printf("[EXPORTER] Error upserting participant: %v", err)
		return
	}
	log.Printf("[EXPORTER] Participant upserted: %s - %s", p.Code, p.Name)
	e.logEvent("Participant", p, ledger.GetCurrentTimeMillis())
}

// SyncInstrument handles Instrument events
func (e *Exporter) SyncInstrument(i ledger.Instrument) {
	if err := e.instrumentRepo.Upsert(i); err != nil {
		log.Printf("[EXPORTER] Error upserting instrument: %v", err)
		return
	}
	log.Printf("[EXPORTER] Instrument upserted: %s - %s (Status: %v)", i.Code, i.Name, i.Status)
	e.logEvent("Instrument", i, ledger.GetCurrentTimeMillis())
}

// SyncOrder handles Order events
func (e *Exporter) SyncOrder(o ledger.Order) {
	if err := e.orderRepo.Insert(o); err != nil {
		log.Printf("[EXPORTER] Error inserting order: %v", err)
		return
	}
	log.Printf("[EXPORTER] Order inserted: NID=%d, %s-%s, State=%s", o.NID, o.Side, o.InstrumentCode, o.State)
	e.logEvent("Order", o, ledger.GetCurrentTimeMillis())
}

// SyncOrderAck handles OrderAck events
func (e *Exporter) SyncOrderAck(a ledger.OrderAck) {
	if err := e.orderRepo.UpdateState(a.OrderNID, "O", 0); err != nil {
		log.Printf("[EXPORTER] Error updating order ack: %v", err)
		return
	}
	log.Printf("[EXPORTER] Order acknowledged: NID=%d", a.OrderNID)
	e.logEvent("OrderAck", a, ledger.GetCurrentTimeMillis())
}

// SyncOrderNak handles OrderNak events
func (e *Exporter) SyncOrderNak(a ledger.OrderNak) {
	if err := e.orderRepo.UpdateState(a.OrderNID, "R", 0); err != nil {
		log.Printf("[EXPORTER] Error updating order nak: %v", err)
		return
	}
	log.Printf("[EXPORTER] Order rejected: NID=%d, Message=%s", a.OrderNID, a.Message)
	e.logEvent("OrderNak", a, ledger.GetCurrentTimeMillis())
}

// SyncOrderWithdraw handles OrderWithdraw events
func (e *Exporter) SyncOrderWithdraw(w ledger.OrderWithdraw) {
	log.Printf("[EXPORTER] Order withdraw request: NID=%d", w.OrderNID)
	e.logEvent("OrderWithdraw", w, ledger.GetCurrentTimeMillis())
}

// SyncOrderWithdrawAck handles OrderWithdrawAck events
func (e *Exporter) SyncOrderWithdrawAck(a ledger.OrderWithdrawAck) {
	if err := e.orderRepo.UpdateState(a.OrderNID, "W", 0); err != nil {
		log.Printf("[EXPORTER] Error updating order withdraw ack: %v", err)
		return
	}
	log.Printf("[EXPORTER] Order withdrawn: NID=%d", a.OrderNID)
	e.logEvent("OrderWithdrawAck", a, ledger.GetCurrentTimeMillis())
}

// SyncOrderWithdrawNak handles OrderWithdrawNak events
func (e *Exporter) SyncOrderWithdrawNak(a ledger.OrderWithdrawNak) {
	log.Printf("[EXPORTER] Order withdraw rejected: NID=%d, Message=%s", a.OrderNID, a.Message)
	e.logEvent("OrderWithdrawNak", a, ledger.GetCurrentTimeMillis())
}

// SyncTrade handles Trade events
func (e *Exporter) SyncTrade(t ledger.Trade) {
	if err := e.tradeRepo.Insert(t); err != nil {
		log.Printf("[EXPORTER] Error inserting trade: %v", err)
		return
	}
	log.Printf("[EXPORTER] Trade inserted: NID=%d, Reff=%s, State=%s", t.NID, t.KpeiReff, t.State)
	e.logEvent("Trade", t, ledger.GetCurrentTimeMillis())
}

// SyncTradeWait handles TradeWait events
func (e *Exporter) SyncTradeWait(w ledger.TradeWait) {
	if err := e.tradeRepo.UpdateState(w.TradeNID, "E"); err != nil {
		log.Printf("[EXPORTER] Error updating trade wait: %v", err)
		return
	}
	log.Printf("[EXPORTER] Trade waiting approval: NID=%d", w.TradeNID)
	e.logEvent("TradeWait", w, ledger.GetCurrentTimeMillis())
}

// SyncTradeAck handles TradeAck events
func (e *Exporter) SyncTradeAck(a ledger.TradeAck) {
	if err := e.tradeRepo.UpdateState(a.TradeNID, "O"); err != nil {
		log.Printf("[EXPORTER] Error updating trade ack: %v", err)
		return
	}
	log.Printf("[EXPORTER] Trade approved: NID=%d", a.TradeNID)
	e.logEvent("TradeAck", a, ledger.GetCurrentTimeMillis())
}

// SyncTradeNak handles TradeNak events
func (e *Exporter) SyncTradeNak(a ledger.TradeNak) {
	if err := e.tradeRepo.UpdateState(a.TradeNID, "R"); err != nil {
		log.Printf("[EXPORTER] Error updating trade nak: %v", err)
		return
	}
	log.Printf("[EXPORTER] Trade rejected: NID=%d, Message=%s", a.TradeNID, a.Message)
	e.logEvent("TradeNak", a, ledger.GetCurrentTimeMillis())
}

// SyncTradeReimburse handles TradeReimburse events
func (e *Exporter) SyncTradeReimburse(r ledger.TradeReimburse) {
	if err := e.tradeRepo.UpdateState(r.TradeNID, "C"); err != nil {
		log.Printf("[EXPORTER] Error updating trade reimburse: %v", err)
		return
	}
	log.Printf("[EXPORTER] Trade reimbursed: NID=%d", r.TradeNID)
	e.logEvent("TradeReimburse", r, ledger.GetCurrentTimeMillis())
}

// SyncContract handles Contract events
func (e *Exporter) SyncContract(c ledger.Contract) {
	if err := e.contractRepo.Insert(c); err != nil {
		log.Printf("[EXPORTER] Error inserting contract: %v", err)
		return
	}
	log.Printf("[EXPORTER] Contract inserted: NID=%d, Side=%s, State=%s", c.NID, c.Side, c.State)
	e.logEvent("Contract", c, ledger.GetCurrentTimeMillis())
}

// Helper function to log events
func (e *Exporter) logEvent(eventType string, eventData interface{}, timestamp int64) {
	if err := e.otherRepo.LogEvent(eventType, eventData, timestamp); err != nil {
		log.Printf("[EXPORTER] Error logging event: %v", err)
	}
}
