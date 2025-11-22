package handler

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"

	"pmeonline/pkg/ledger"
)

type TradeHandler struct {
	ledger *ledger.LedgerPoint
}

func NewTradeHandler(l *ledger.LedgerPoint) *TradeHandler {
	return &TradeHandler{ledger: l}
}

// MatchedConfirmRequest represents trade confirmation from eClear
type MatchedConfirmRequest struct {
	PmeTradeReff     string    `json:"pme_trade_reff"`
	State            string    `json:"state"` // "OK"
	BorrContractReff string    `json:"borr_contract_reff"`
	LendContractReff string    `json:"lend_contract_reff"`
	OpenTime         time.Time `json:"open_time"`
}

// ReimburseRequest represents reimbursement instruction from eClear
type ReimburseRequest struct {
	PmeTradeReff     string    `json:"pme_trade_reff"`
	State            string    `json:"state"` // "REIM" or "ARO"
	BorrContractReff string    `json:"borr_contract_reff"`
	LendContractReff string    `json:"lend_contract_reff"`
	CloseTime        time.Time `json:"close_time"`
}

// LenderRecallRequest represents lender recall instruction from eClear
type LenderRecallRequest struct {
	ContractReff string `json:"contract_reff"`
	KpeiReff     string `json:"kpei_reff"`
}

// MatchedConfirm handles POST /contract/matched
// This is called by eClear to confirm a trade has been approved
func (h *TradeHandler) MatchedConfirm(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("❌ Failed to read request body: %v", err)
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var confirm MatchedConfirmRequest
	if err := json.Unmarshal(body, &confirm); err != nil {
		log.Printf("❌ Failed to parse JSON: %v", err)
		http.Error(w, "Invalid JSON format", http.StatusBadRequest)
		return
	}

	log.Printf("📥 Received trade confirmation from eClear: %s", confirm.PmeTradeReff)

	// Find the trade by KpeiReff (which is the PmeTradeReff)
	var tradeNID int
	var found bool
	for nid, trade := range h.ledger.Trades {
		if trade.KpeiReff == confirm.PmeTradeReff {
			tradeNID = nid
			found = true
			break
		}
	}

	if !found {
		log.Printf("❌ Trade not found: %s", confirm.PmeTradeReff)
		http.Error(w, "Trade not found", http.StatusNotFound)
		return
	}

	// Check if state is OK
	if confirm.State != "OK" {
		log.Printf("⚠️  Trade confirmation state is not OK: %s", confirm.State)
		// You might want to handle different states here
	}

	// Commit TradeAck event to update trade state to Open
	tradeAck := ledger.TradeAck{
		TradeNID: tradeNID,
	}
	h.ledger.Commit <- tradeAck
	log.Printf("✅ Trade approved and opened: %s (NID: %d)", confirm.PmeTradeReff, tradeNID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"message": "Trade confirmed",
	})
}

// Reimburse handles POST /contract/reimburse
// This is called by eClear to instruct reimbursement of a trade
func (h *TradeHandler) Reimburse(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("❌ Failed to read request body: %v", err)
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var reimburse ReimburseRequest
	if err := json.Unmarshal(body, &reimburse); err != nil {
		log.Printf("❌ Failed to parse JSON: %v", err)
		http.Error(w, "Invalid JSON format", http.StatusBadRequest)
		return
	}

	log.Printf("📥 Received reimbursement instruction from eClear: %s (State: %s)",
		reimburse.PmeTradeReff, reimburse.State)

	// Find the trade by KpeiReff
	var tradeNID int
	var trade ledger.TradeEntity
	var found bool
	for nid, t := range h.ledger.Trades {
		if t.KpeiReff == reimburse.PmeTradeReff {
			tradeNID = nid
			trade = t
			found = true
			break
		}
	}

	if !found {
		log.Printf("❌ Trade not found: %s", reimburse.PmeTradeReff)
		http.Error(w, "Trade not found", http.StatusNotFound)
		return
	}

	// Handle ARO (Auto Roll-Over)
	if reimburse.State == "ARO" {
		log.Printf("🔄 Processing ARO for trade: %s", reimburse.PmeTradeReff)

		// Create new order for borrower with ARO flag
		for _, contract := range trade.Borrower {
			// Get the original order details
			if origOrder, exists := h.ledger.Orders[contract.OrderNID]; exists {
				newOrder := ledger.Order{
					NID:               int(ledger.GetCurrentTimeMillis()),
					PrevNID:           0, // ARO order has no previous order
					ReffRequestID:     reimburse.PmeTradeReff + "-ARO",
					AccountNID:        contract.AccountNID,
					AccountCode:       contract.AccountCode,
					ParticipantNID:    contract.AccountParticipantNID,
					ParticipantCode:   contract.AccountParticipantCode,
					InstrumentNID:     contract.InstrumentNID,
					InstrumentCode:    contract.InstrumentCode,
					Side:              "BORR",
					Quantity:          contract.Quantity,
					SettlementDate:    time.Now().AddDate(0, 0, 1), // T+1
					ReimbursementDate: origOrder.ReimbursementDate.AddDate(0, 0, trade.Periode),
					Periode:           trade.Periode,
					State:             "S",
					MarketPrice:       origOrder.MarketPrice,
					Rate:              origOrder.Rate,
					Instruction:       "ARO from " + reimburse.PmeTradeReff,
					ARO:               true,
				}

				// Commit new order
				h.ledger.Commit <- newOrder
				log.Printf("✅ Created ARO order for account %s", contract.AccountCode)
			}
		}
	}

	// Commit TradeReimburse event to close the trade
	tradeReimburse := ledger.TradeReimburse{
		TradeNID: tradeNID,
	}
	h.ledger.Commit <- tradeReimburse
	log.Printf("✅ Trade reimbursed: %s (NID: %d)", reimburse.PmeTradeReff, tradeNID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"message": "Trade reimbursed",
		"aro":     reimburse.State == "ARO",
	})
}

// LenderRecall handles POST /lender/recall
// This is called by eClear to instruct a lender recall (find new lender)
func (h *TradeHandler) LenderRecall(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("❌ Failed to read request body: %v", err)
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var recall LenderRecallRequest
	if err := json.Unmarshal(body, &recall); err != nil {
		log.Printf("❌ Failed to parse JSON: %v", err)
		http.Error(w, "Invalid JSON format", http.StatusBadRequest)
		return
	}

	log.Printf("📥 Received lender recall instruction from eClear: %s", recall.ContractReff)

	// Find the contract
	var contract ledger.ContractEntity
	var found bool
	for _, c := range h.ledger.Contracts {
		if c.KpeiReff == recall.ContractReff {
			contract = c
			found = true
			break
		}
	}

	if !found {
		log.Printf("❌ Contract not found: %s", recall.ContractReff)
		http.Error(w, "Contract not found", http.StatusNotFound)
		return
	}

	// Contract must be a lending contract
	if contract.Side != "LEND" {
		log.Printf("❌ Contract is not a lending contract: %s", recall.ContractReff)
		http.Error(w, "Contract is not a lending contract", http.StatusBadRequest)
		return
	}

	// Get the trade
	trade, exists := h.ledger.Trades[contract.TradeNID]
	if !exists {
		log.Printf("❌ Trade not found for contract: %s", recall.ContractReff)
		http.Error(w, "Trade not found", http.StatusInternalServerError)
		return
	}

	// Create a new "matching order" for the borrower to find a new lender
	// This will be treated like a regular borrow order and matched by the OMS
	for _, borrowContract := range trade.Borrower {
		newOrder := ledger.Order{
			NID:               int(ledger.GetCurrentTimeMillis()),
			PrevNID:           0,
			ReffRequestID:     recall.ContractReff + "-RECALL",
			AccountNID:        borrowContract.AccountNID,
			AccountCode:       borrowContract.AccountCode,
			ParticipantNID:    borrowContract.AccountParticipantNID,
			ParticipantCode:   borrowContract.AccountParticipantCode,
			InstrumentNID:     borrowContract.InstrumentNID,
			InstrumentCode:    borrowContract.InstrumentCode,
			Side:              "BORR",
			Quantity:          contract.Quantity,
			SettlementDate:    time.Now(), // Immediate
			ReimbursementDate: contract.ReimburseAt,
			Periode:           contract.Periode,
			State:             "S",
			MarketPrice:       0, // Will be determined at matching
			Rate:              0,  // Will be determined at matching
			Instruction:       "Lender Recall from " + recall.ContractReff,
			ARO:               false,
		}

		// Commit new order
		h.ledger.Commit <- newOrder
		log.Printf("✅ Created recall order for borrower %s", borrowContract.AccountCode)
	}

	// Note: The old contract will be terminated when the new trade is matched
	// This should be handled by the OMS

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"message": "Lender recall processed",
	})
}
