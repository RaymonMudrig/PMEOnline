package handler

import (
	"net/http"

	"pmeonline/pkg/ledger"
)

type QueryHandler struct {
	ledger *ledger.LedgerPoint
}

func NewQueryHandler(l *ledger.LedgerPoint) *QueryHandler {
	return &QueryHandler{ledger: l}
}

// AccountInfoResponse represents account information
type AccountInfoResponse struct {
	Code            string  `json:"code"`
	SID             string  `json:"sid"`
	Name            string  `json:"name"`
	Address         string  `json:"address"`
	ParticipantCode string  `json:"participant_code"`
	ParticipantName string  `json:"participant_name"`
	TradeLimit      float64 `json:"trade_limit"`
	PoolLimit       float64 `json:"pool_limit"`
}

// OrderInfo represents order information
type OrderInfo struct {
	NID               int     `json:"nid"`
	ReffRequestID     string  `json:"reff_request_id"`
	AccountCode       string  `json:"account_code"`
	ParticipantCode   string  `json:"participant_code"`
	InstrumentCode    string  `json:"instrument_code"`
	Side              string  `json:"side"`
	Quantity          float64 `json:"quantity"`
	DoneQuantity      float64 `json:"done_quantity"`
	SettlementDate    string  `json:"settlement_date"`
	ReimbursementDate string  `json:"reimbursement_date"`
	Periode           int     `json:"periode"`
	State             string  `json:"state"`
	MarketPrice       float64 `json:"market_price"`
	Rate              float64 `json:"rate"`
	ARO               bool    `json:"aro"`
	Message           string  `json:"message"`
	EntryAt           string  `json:"entry_at"`
}

// ContractInfo represents contract information
type ContractInfo struct {
	NID                    int     `json:"nid"`
	TradeNID               int     `json:"trade_nid"`
	KpeiReff               string  `json:"kpei_reff"`
	Side                   string  `json:"side"`
	AccountCode            string  `json:"account_code"`
	AccountSID             string  `json:"account_sid"`
	AccountParticipantCode string  `json:"account_participant_code"`
	OrderNID               int     `json:"order_nid"`
	InstrumentCode         string  `json:"instrument_code"`
	Quantity               float64 `json:"quantity"`
	Periode                int     `json:"periode"`
	State                  string  `json:"state"`
	FeeFlatVal             float64 `json:"fee_flat_val"`
	FeeValDaily            float64 `json:"fee_val_daily"`
	FeeValAccumulated      float64 `json:"fee_val_accumulated"`
	MatchedAt              string  `json:"matched_at"`
	ReimburseAt            string  `json:"reimburse_at"`
}

// GetAccountInfo handles GET /api/account/info?sid={sid}
func (h *QueryHandler) GetAccountInfo(w http.ResponseWriter, r *http.Request) {
	sid := r.URL.Query().Get("sid")
	if sid == "" {
		respondError(w, http.StatusBadRequest, "sid parameter is required")
		return
	}

	// Find account by SID
	var foundAccount *ledger.AccountEntity
	for _, account := range h.ledger.Account {
		if account.SID == sid {
			foundAccount = &account
			break
		}
	}

	if foundAccount == nil {
		respondError(w, http.StatusNotFound, "Account not found")
		return
	}

	// Get participant information
	participant, exists := h.ledger.Participant[foundAccount.ParticipantCode]
	if !exists {
		respondError(w, http.StatusInternalServerError, "Participant not found")
		return
	}

	// Build response
	response := AccountInfoResponse{
		Code:            foundAccount.Code,
		SID:             foundAccount.SID,
		Name:            foundAccount.Name,
		Address:         foundAccount.Address,
		ParticipantCode: foundAccount.ParticipantCode,
		ParticipantName: participant.Name,
		TradeLimit:      foundAccount.TradeLimit,
		PoolLimit:       foundAccount.PoolLimit,
	}

	respondSuccess(w, "Account information retrieved", response)
}

// GetOrderList handles GET /api/order/list?participant={code}&sid={sid}&state={state}
func (h *QueryHandler) GetOrderList(w http.ResponseWriter, r *http.Request) {
	participantCode := r.URL.Query().Get("participant")
	sid := r.URL.Query().Get("sid")
	stateFilter := r.URL.Query().Get("state")

	// Build filter criteria
	var orders []OrderInfo

	for _, order := range h.ledger.Orders {
		// Apply filters
		if participantCode != "" && order.ParticipantCode != participantCode {
			continue
		}

		if sid != "" {
			// Find account by SID
			account, exists := h.ledger.Account[order.AccountCode]
			if !exists || account.SID != sid {
				continue
			}
		}

		if stateFilter != "" && order.State != stateFilter {
			continue
		}

		// Add to results
		orderInfo := OrderInfo{
			NID:               order.NID,
			ReffRequestID:     order.ReffRequestID,
			AccountCode:       order.AccountCode,
			ParticipantCode:   order.ParticipantCode,
			InstrumentCode:    order.InstrumentCode,
			Side:              order.Side,
			Quantity:          order.Quantity,
			DoneQuantity:      order.DoneQuantity,
			SettlementDate:    order.SettlementDate.Format("2006-01-02"),
			ReimbursementDate: order.ReimbursementDate.Format("2006-01-02"),
			Periode:           order.Periode,
			State:             order.State,
			MarketPrice:       order.MarketPrice,
			Rate:              order.Rate,
			ARO:               order.ARO,
			Message:           order.Message,
			EntryAt:           order.EntryAt.Format("2006-01-02 15:04:05"),
		}
		orders = append(orders, orderInfo)
	}

	respondSuccess(w, "Order list retrieved", map[string]interface{}{
		"count":  len(orders),
		"orders": orders,
	})
}

// GetContractList handles GET /api/contract/list?participant={code}&sid={sid}&state={state}
func (h *QueryHandler) GetContractList(w http.ResponseWriter, r *http.Request) {
	participantCode := r.URL.Query().Get("participant")
	sid := r.URL.Query().Get("sid")
	stateFilter := r.URL.Query().Get("state")

	// Build filter criteria
	var contracts []ContractInfo

	for _, contract := range h.ledger.Contracts {
		// Apply filters
		if participantCode != "" && contract.AccountParticipantCode != participantCode {
			continue
		}

		if sid != "" {
			// Find account by SID
			account, exists := h.ledger.Account[contract.AccountCode]
			if !exists || account.SID != sid {
				continue
			}
		}

		if stateFilter != "" && contract.State != stateFilter {
			continue
		}

		// Add to results
		contractInfo := ContractInfo{
			NID:                    contract.NID,
			TradeNID:               contract.TradeNID,
			KpeiReff:               contract.KpeiReff,
			Side:                   contract.Side,
			AccountCode:            contract.AccountCode,
			AccountSID:             contract.AccountSID,
			AccountParticipantCode: contract.AccountParticipantCode,
			OrderNID:               contract.OrderNID,
			InstrumentCode:         contract.InstrumentCode,
			Quantity:               contract.Quantity,
			Periode:                contract.Periode,
			State:                  contract.State,
			FeeFlatVal:             contract.FeeFlatVal,
			FeeValDaily:            contract.FeeValDaily,
			FeeValAccumulated:      contract.FeeValAccumulated,
			MatchedAt:              contract.MatchedAt.Format("2006-01-02 15:04:05"),
			ReimburseAt:            contract.ReimburseAt.Format("2006-01-02"),
		}
		contracts = append(contracts, contractInfo)
	}

	respondSuccess(w, "Contract list retrieved", map[string]interface{}{
		"count":     len(contracts),
		"contracts": contracts,
	})
}

// QueryResponse represents a generic query response
type QueryResponse struct {
	Status  string      `json:"status"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}
