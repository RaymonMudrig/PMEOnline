package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"pmeonline/pkg/ledger"
)

type EClearClient struct {
	baseURL    string
	httpClient *http.Client
	ledger     *ledger.LedgerPoint
}

func NewEClearClient(baseURL string, l *ledger.LedgerPoint) *EClearClient {
	return &EClearClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		ledger: l,
	}
}

// TradeMatchedPayload represents the payload sent to eClear for trade approval
type TradeMatchedPayload struct {
	PmeTradeReff   string             `json:"pme_trade_reff"`
	InstrumentCode string             `json:"instrument_code"`
	Quantity       float64            `json:"quantity"`
	Periode        int                `json:"periode"`
	AroStatus      bool               `json:"aro_status"`
	FeeFlatRate    float64            `json:"fee_flat_rate"`
	FeeBorrRate    float64            `json:"fee_borr_rate"`
	FeeLendRate    float64            `json:"fee_lend_rate"`
	MatchedAt      string             `json:"matched_at"`
	ReimburseAt    string             `json:"reimburse_at"`
	Lender         ContractInfo       `json:"lender"`
	Borrower       ContractInfo       `json:"borrower"`
}

type ContractInfo struct {
	PmeContractReff string  `json:"pme_contract_reff"`
	AccountCode     string  `json:"account_code"`
	SID             string  `json:"sid"`
	ParticipantCode string  `json:"participant_code"`
	FeeLender       float64 `json:"fee_lender,omitempty"`
	FeeFlat         float64 `json:"fee_flat,omitempty"`
	FeeBorrower     float64 `json:"fee_borrower,omitempty"`
}

// Start begins listening to Trade events and sends them to eClear
func (c *EClearClient) Start(ctx context.Context) {
	log.Println("🚀 Starting eClear outbound client...")

	// Create a sync handler to subscribe to Trade events
	syncHandler := &EClearSyncHandler{client: c}
	c.ledger.Sync <- syncHandler

	log.Println("✅ eClear outbound client started and subscribed to events")

	// Wait for context cancellation
	<-ctx.Done()
	log.Println("🛑 eClear outbound client stopped")
}

// EClearSyncHandler implements LedgerPointInterface to receive events
type EClearSyncHandler struct {
	client *EClearClient
}

func (h *EClearSyncHandler) SyncServiceStart(a ledger.ServiceStart)       {}
func (h *EClearSyncHandler) SyncParameter(a ledger.Parameter)             {}
func (h *EClearSyncHandler) SyncSessionTime(a ledger.SessionTime)         {}
func (h *EClearSyncHandler) SyncHoliday(a ledger.Holiday)                 {}
func (h *EClearSyncHandler) SyncAccount(a ledger.Account)                 {}
func (h *EClearSyncHandler) SyncAccountLimit(a ledger.AccountLimit)       {}
func (h *EClearSyncHandler) SyncParticipant(a ledger.Participant)         {}
func (h *EClearSyncHandler) SyncInstrument(a ledger.Instrument)           {}
func (h *EClearSyncHandler) SyncOrder(a ledger.Order)                     {}
func (h *EClearSyncHandler) SyncOrderAck(a ledger.OrderAck)               {}
func (h *EClearSyncHandler) SyncOrderNak(a ledger.OrderNak)               {}
func (h *EClearSyncHandler) SyncOrderWithdraw(a ledger.OrderWithdraw)     {}
func (h *EClearSyncHandler) SyncOrderWithdrawAck(a ledger.OrderWithdrawAck) {}
func (h *EClearSyncHandler) SyncOrderWithdrawNak(a ledger.OrderWithdrawNak) {}
func (h *EClearSyncHandler) SyncTradeWait(a ledger.TradeWait)             {}
func (h *EClearSyncHandler) SyncTradeAck(a ledger.TradeAck)               {}
func (h *EClearSyncHandler) SyncTradeNak(a ledger.TradeNak)               {}
func (h *EClearSyncHandler) SyncTradeReimburse(a ledger.TradeReimburse)   {}
func (h *EClearSyncHandler) SyncContract(a ledger.Contract)               {}

// SyncTrade is called when a new trade is created
func (h *EClearSyncHandler) SyncTrade(a ledger.Trade) {
	log.Printf("📤 New trade detected, preparing to send to eClear: %s", a.KpeiReff)

	// Note: This is a simplified implementation
	// In production, you would want to:
	// 1. Store this in a queue for retry handling
	// 2. Implement proper error handling
	// 3. Track submission status

	// For now, we'll just log that we would send it
	log.Printf("📨 Would send trade to eClear: %s", a.KpeiReff)

	// The actual sending would be done by the EClearClient.SendTrade method
}

// SendTrade sends a trade to eClear for approval
func (c *EClearClient) SendTrade(trade ledger.Trade) error {
	log.Printf("📤 Sending trade to eClear: %s", trade.KpeiReff)

	// Find borrower and lender contracts
	var borrowerContract ledger.Contract
	var lenderContract ledger.Contract
	var hasBorrower, hasLender bool

	for _, contract := range trade.Borrower {
		borrowerContract = contract
		hasBorrower = true
		break // Take first borrower
	}

	for _, contract := range trade.Lender {
		lenderContract = contract
		hasLender = true
		break // Take first lender
	}

	if !hasBorrower || !hasLender {
		return fmt.Errorf("trade %s missing borrower or lender contract", trade.KpeiReff)
	}

	// Get account information for SID
	borrowerAccount, exists := c.ledger.Account[borrowerContract.AccountCode]
	if !exists {
		return fmt.Errorf("borrower account not found: %s", borrowerContract.AccountCode)
	}

	lenderAccount, exists := c.ledger.Account[lenderContract.AccountCode]
	if !exists {
		return fmt.Errorf("lender account not found: %s", lenderContract.AccountCode)
	}

	// Determine ARO status from any of the orders
	aroStatus := false
	if borrowerOrder, exists := c.ledger.Orders[borrowerContract.OrderNID]; exists {
		aroStatus = borrowerOrder.ARO
	}

	// Build payload
	payload := TradeMatchedPayload{
		PmeTradeReff:   trade.KpeiReff,
		InstrumentCode: trade.InstrumentCode,
		Quantity:       trade.Quantity,
		Periode:        trade.Periode,
		AroStatus:      aroStatus,
		FeeFlatRate:    trade.FeeFlatRate,
		FeeBorrRate:    trade.FeeBorrRate,
		FeeLendRate:    trade.FeeLendRate,
		MatchedAt:      trade.MatchedAt.Format("2006-01-02 15:04:05"),
		ReimburseAt:    trade.ReimburseAt.Format("2006-01-02 15:04:05"),
		Lender: ContractInfo{
			PmeContractReff: lenderContract.KpeiReff,
			AccountCode:     lenderContract.AccountCode,
			SID:             lenderAccount.SID,
			ParticipantCode: lenderContract.AccountParticipantCode,
			FeeLender:       lenderContract.FeeValDaily * float64(trade.Periode),
		},
		Borrower: ContractInfo{
			PmeContractReff: borrowerContract.KpeiReff,
			AccountCode:     borrowerContract.AccountCode,
			SID:             borrowerAccount.SID,
			ParticipantCode: borrowerContract.AccountParticipantCode,
			FeeFlat:         borrowerContract.FeeFlatVal,
			FeeBorrower:     borrowerContract.FeeValDaily * float64(trade.Periode),
		},
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %v", err)
	}

	// Send HTTP POST request
	url := c.baseURL + "/contract/matched"
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		log.Printf("❌ Failed to send trade to eClear: %v", err)
		// Commit TradeWait event (waiting for approval)
		c.ledger.Commit <- ledger.TradeWait{TradeNID: trade.NID}
		return fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("❌ eClear returned non-OK status: %d", resp.StatusCode)
		// Commit TradeWait event
		c.ledger.Commit <- ledger.TradeWait{TradeNID: trade.NID}
		return fmt.Errorf("eClear returned status: %d", resp.StatusCode)
	}

	// Commit TradeWait event (trade submitted, waiting for eClear approval)
	c.ledger.Commit <- ledger.TradeWait{TradeNID: trade.NID}
	log.Printf("✅ Trade sent to eClear successfully: %s", trade.KpeiReff)

	return nil
}

// CheckPendingTrades checks for trades in Wait state that haven't been approved by EOD
// This should be called by a scheduler at EOD
func (c *EClearClient) CheckPendingTrades() {
	log.Println("🔍 Checking for pending trades at EOD...")

	for nid, trade := range c.ledger.Trades {
		// Check if trade is in Wait state (E = Approval/Wait)
		if trade.State == "E" {
			// Check if matched today (simplified - should check against session time)
			if time.Since(trade.MatchedAt) > 24*time.Hour {
				log.Printf("⚠️  Trade %s not approved by EOD, dropping trade", trade.KpeiReff)

				// Commit TradeNak to drop the trade
				c.ledger.Commit <- ledger.TradeNak{
					TradeNID: nid,
					Message:  "Trade not approved by eClear by EOD",
				}
			}
		}
	}

	log.Println("✅ Pending trades check completed")
}
