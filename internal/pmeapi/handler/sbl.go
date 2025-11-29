package handler

import (
	"net/http"

	"pmeonline/pkg/ledger"
)

type SBLHandler struct {
	ledger *ledger.LedgerPoint
}

func NewSBLHandler(l *ledger.LedgerPoint) *SBLHandler {
	return &SBLHandler{ledger: l}
}

// SBLOrderInfo represents an order in the SBL
type SBLOrderInfo struct {
	NID               int     `json:"nid"`
	ParticipantCode   string  `json:"participant_code"`
	SID               string  `json:"sid,omitempty"`
	AccountCode       string  `json:"account_code"`
	InstrumentCode    string  `json:"instrument_code"`
	Side              string  `json:"side"`
	Quantity          float64 `json:"quantity"`
	DoneQuantity      float64 `json:"done_quantity"`
	RemainingQuantity float64 `json:"remaining_quantity"`
	Rate              float64 `json:"rate"`
	ARO               bool    `json:"aro"`
	SettlementDate    string  `json:"settlement_date"`
	ReimbursementDate string  `json:"reimbursement_date"`
	Periode           int     `json:"periode"`
	State             string  `json:"state"`
	EntryAt           string  `json:"entry_at"`
}

// SBLAggregateInfo represents aggregated SBL data per instrument
type SBLAggregateInfo struct {
	InstrumentCode string  `json:"instrument_code"`
	InstrumentName string  `json:"instrument_name"`
	BorrowQuantity float64 `json:"borrow_quantity"`
	LendQuantity   float64 `json:"lend_quantity"`
	NetQuantity    float64 `json:"net_quantity"`
	NetSide        string  `json:"net_side"` // "BORR" or "LEND"
}

// GetSBLDetail handles GET /api/sbl/detail
// Returns all Open and Partial orders (the lendable pool and borrowing needs)
func (h *SBLHandler) GetSBLDetail(w http.ResponseWriter, r *http.Request) {
	// Query parameters for filtering
	participantCode := r.URL.Query().Get("participant")
	instrumentCode := r.URL.Query().Get("instrument")
	side := r.URL.Query().Get("side")
	aroFilter := r.URL.Query().Get("aro")

	var orders []SBLOrderInfo

	h.ledger.ForEachOrder(func(order ledger.OrderEntity) bool {
		// Only include Open and Partial orders
		if order.State != "O" && order.State != "P" {
			return true
		}

		// Apply filters
		if participantCode != "" && order.ParticipantCode != participantCode {
			return true
		}

		if instrumentCode != "" && order.InstrumentCode != instrumentCode {
			return true
		}

		if side != "" && order.Side != side {
			return true
		}

		if aroFilter == "true" && !order.ARO {
			return true
		} else if aroFilter == "false" && order.ARO {
			return true
		}

		// Get account SID
		account, exists := h.ledger.GetAccount(order.AccountCode)
		sid := ""
		if exists {
			sid = account.SID
		}

		// Calculate remaining quantity
		remainingQty := order.Quantity - order.DoneQuantity

		orderInfo := SBLOrderInfo{
			NID:               order.NID,
			ParticipantCode:   order.ParticipantCode,
			SID:               sid,
			AccountCode:       order.AccountCode,
			InstrumentCode:    order.InstrumentCode,
			Side:              order.Side,
			Quantity:          order.Quantity,
			DoneQuantity:      order.DoneQuantity,
			RemainingQuantity: remainingQty,
			Rate:              order.Rate,
			ARO:               order.ARO,
			SettlementDate:    order.SettlementDate.Format("2006-01-02"),
			ReimbursementDate: order.ReimbursementDate.Format("2006-01-02"),
			Periode:           order.Periode,
			State:             order.State,
			EntryAt:           order.EntryAt.Format("2006-01-02 15:04:05"),
		}

		orders = append(orders, orderInfo)
		return true
	})

	respondSuccess(w, "SBL detail retrieved", map[string]interface{}{
		"count":  len(orders),
		"orders": orders,
	})
}

// GetSBLAggregate handles GET /api/sbl/aggregate
// Returns net position per instrument (only one side will have quantity)
func (h *SBLHandler) GetSBLAggregate(w http.ResponseWriter, r *http.Request) {
	// Query parameters for filtering
	instrumentFilter := r.URL.Query().Get("instrument")
	sideFilter := r.URL.Query().Get("side")

	// Aggregate by instrument
	type AggData struct {
		BorrowQty float64
		LendQty   float64
	}

	aggMap := make(map[string]*AggData)

	h.ledger.ForEachOrder(func(order ledger.OrderEntity) bool {
		// Only include Open and Partial orders
		if order.State != "O" && order.State != "P" {
			return true
		}

		// Apply instrument filter
		if instrumentFilter != "" && order.InstrumentCode != instrumentFilter {
			return true
		}

		// Initialize if not exists
		if _, exists := aggMap[order.InstrumentCode]; !exists {
			aggMap[order.InstrumentCode] = &AggData{}
		}

		// Add remaining quantity
		remainingQty := order.Quantity - order.DoneQuantity

		if order.Side == "BORR" {
			aggMap[order.InstrumentCode].BorrowQty += remainingQty
		} else if order.Side == "LEND" {
			aggMap[order.InstrumentCode].LendQty += remainingQty
		}
		return true
	})

	// Build result
	var aggregates []SBLAggregateInfo

	for instrCode, data := range aggMap {
		// Get instrument name
		instrName := instrCode
		if instrument, exists := h.ledger.GetInstrument(instrCode); exists {
			instrName = instrument.Name
		}

		// Calculate net (in PME, only one side should have quantity after matching)
		netQty := data.LendQty - data.BorrowQty
		netSide := "LEND"
		if netQty < 0 {
			netQty = -netQty
			netSide = "BORR"
		}

		// Apply side filter
		if sideFilter != "" && netSide != sideFilter {
			continue
		}

		agg := SBLAggregateInfo{
			InstrumentCode: instrCode,
			InstrumentName: instrName,
			BorrowQuantity: data.BorrowQty,
			LendQuantity:   data.LendQty,
			NetQuantity:    netQty,
			NetSide:        netSide,
		}

		aggregates = append(aggregates, agg)
	}

	respondSuccess(w, "SBL aggregate retrieved", map[string]interface{}{
		"count":      len(aggregates),
		"aggregates": aggregates,
	})
}
