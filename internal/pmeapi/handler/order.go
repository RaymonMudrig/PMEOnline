package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"pmeonline/pkg/idgen"
	"pmeonline/pkg/ledger"
)

type OrderHandler struct {
	ledger *ledger.LedgerPoint
	idgen  *idgen.Generator
}

func NewOrderHandler(l *ledger.LedgerPoint, idGenerator *idgen.Generator) *OrderHandler {
	return &OrderHandler{
		ledger: l,
		idgen:  idGenerator,
	}
}

// OrderRequest represents a new order request
type OrderRequest struct {
	ReffRequestID     string    `json:"reff_request_id"`
	AccountCode       string    `json:"account_code"`
	ParticipantCode   string    `json:"participant_code"`
	InstrumentCode    string    `json:"instrument_code"`
	Side              string    `json:"side"` // "BORR" or "LEND"
	Quantity          float64   `json:"quantity"`
	SettlementDate    time.Time `json:"settlement_date"`
	ReimbursementDate time.Time `json:"reimbursement_date"`
	Periode           int       `json:"periode"`
	MarketPrice       float64   `json:"market_price"`
	Rate              float64   `json:"rate"`
	Instruction       string    `json:"instruction"`
	ARO               bool      `json:"aro"`
}

// AmendOrderRequest represents an order amendment request
type AmendOrderRequest struct {
	OrderNID          int       `json:"order_nid"`
	ReffRequestID     string    `json:"reff_request_id"`
	Quantity          float64   `json:"quantity,omitempty"`
	SettlementDate    time.Time `json:"settlement_date,omitempty"`
	ReimbursementDate time.Time `json:"reimbursement_date,omitempty"`
	Periode           int       `json:"periode,omitempty"`
	ARO               *bool     `json:"aro,omitempty"`
	Instruction       string    `json:"instruction,omitempty"`
}

// WithdrawOrderRequest represents an order withdrawal request
type WithdrawOrderRequest struct {
	OrderNID      int    `json:"order_nid"`
	ReffRequestID string `json:"reff_request_id"`
}

// OrderResponse represents the API response
type OrderResponse struct {
	Status  string      `json:"status"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// NewOrder handles POST /api/order/new
func (h *OrderHandler) NewOrder(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("[APME-API] Failed to read request body: %v", err)
		respondError(w, http.StatusBadRequest, "Failed to read request body")
		return
	}
	defer r.Body.Close()

	var req OrderRequest
	if err := json.Unmarshal(body, &req); err != nil {
		log.Printf("[APME-API] Failed to parse JSON: %v", err)
		respondError(w, http.StatusBadRequest, "Invalid JSON format")
		return
	}

	// Validate required fields
	if err := validateOrderRequest(req); err != nil {
		log.Printf("[APME-API] Validation failed: %v", err)
		respondError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	// Get account and participant NIDs
	account, accountExists := h.ledger.GetAccount(req.AccountCode)
	if !accountExists {
		respondError(w, http.StatusNotFound, "Account not found")
		return
	}

	participant, participantExists := h.ledger.GetParticipant(req.ParticipantCode)
	if !participantExists {
		respondError(w, http.StatusNotFound, "Participant not found")
		return
	}

	instrument, instrumentExists := h.ledger.GetInstrument(req.InstrumentCode)
	if !instrumentExists {
		respondError(w, http.StatusNotFound, "Instrument not found")
		return
	}

	// Generate unique order NID using Snowflake ID
	nid, err := h.idgen.NextID()
	if err != nil {
		log.Printf("[APME-API] Failed to generate order NID: %v", err)
		respondError(w, http.StatusInternalServerError, "Failed to generate order ID")
		return
	}
	orderNID := int(nid)

	// Create order event
	order := ledger.Order{
		NID:               orderNID,
		PrevNID:           0,
		ReffRequestID:     req.ReffRequestID,
		AccountNID:        account.NID,
		AccountCode:       req.AccountCode,
		ParticipantNID:    participant.NID,
		ParticipantCode:   req.ParticipantCode,
		InstrumentNID:     instrument.NID,
		InstrumentCode:    req.InstrumentCode,
		Side:              req.Side,
		Quantity:          req.Quantity,
		SettlementDate:    req.SettlementDate,
		ReimbursementDate: req.ReimbursementDate,
		Periode:           req.Periode,
		State:             "S",
		MarketPrice:       req.MarketPrice,
		Rate:              req.Rate,
		Instruction:       req.Instruction,
		ARO:               req.ARO,
	}

	// Commit to Kafka
	h.ledger.Commit <- order
	log.Printf("[APME-API] Order submitted: %d (%s %s %.0f shares)",
		orderNID, req.Side, req.InstrumentCode, req.Quantity)

	// Return success response
	respondSuccess(w, "Order submitted successfully", map[string]interface{}{
		"order_nid":       orderNID,
		"reff_request_id": req.ReffRequestID,
		"status":          "submitted",
	})
}

// AmendOrder handles POST /api/order/amend
func (h *OrderHandler) AmendOrder(w http.ResponseWriter, r *http.Request) {
	log.Printf("[APME-API] AmendOrder handler called")

	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("[APME-API] Failed to read request body: %v", err)
		respondError(w, http.StatusBadRequest, "Failed to read request body")
		return
	}
	defer r.Body.Close()

	log.Printf("[APME-API] Request body: %s", string(body))

	var req AmendOrderRequest
	if err := json.Unmarshal(body, &req); err != nil {
		log.Printf("[APME-API] Failed to parse JSON: %v", err)
		respondError(w, http.StatusBadRequest, "Invalid JSON format")
		return
	}

	log.Printf("[APME-API] Parsed request: %+v", req)

	// Validate order exists
	log.Printf("[APME-API] Attempting to amend order NID: %d", req.OrderNID)

	// Debug: List all orders in ledger
	orderCount := 0
	foundMatch := false
	h.ledger.ForEachOrder(func(o ledger.OrderEntity) bool {
		log.Printf("[APME-API] Ledger Order: NID=%d, State=%s", o.NID, o.State)
		orderCount++
		if o.NID == req.OrderNID {
			foundMatch = true
			log.Printf("[APME-API] Found order in iteration: NID=%d, State=%s", o.NID, o.State)
		}
		return true
	})
	log.Printf("[APME-API] Total orders in ledger: %d, Found via iteration: %v", orderCount, foundMatch)

	originalOrder, exists := h.ledger.GetOrder(req.OrderNID)
	if !exists {
		log.Printf("[APME-API] Order not found via GetOrder: %d", req.OrderNID)
		respondError(w, http.StatusNotFound, "Order not found: "+fmt.Sprintf("%d", req.OrderNID))
		return
	}

	// Check if order can be amended (must be Open or Partial)
	log.Printf("[APME-API] Order %d state: %s", req.OrderNID, originalOrder.State)
	if originalOrder.State != "O" && originalOrder.State != "P" {
		respondError(w, http.StatusUnprocessableEntity, "Order cannot be amended in current state: "+originalOrder.State+" (must be O or P)")
		return
	}

	// Generate unique order NID using Snowflake ID
	nid, err := h.idgen.NextID()
	if err != nil {
		log.Printf("[APME-API] Failed to generate amended order NID: %v", err)
		respondError(w, http.StatusInternalServerError, "Failed to generate order ID")
		return
	}
	newOrderNID := int(nid)

	amendedOrder := ledger.Order{
		NID:               newOrderNID,
		PrevNID:           req.OrderNID,
		ReffRequestID:     req.ReffRequestID,
		AccountNID:        originalOrder.AccountNID,
		AccountCode:       originalOrder.AccountCode,
		ParticipantNID:    originalOrder.ParticipantNID,
		ParticipantCode:   originalOrder.ParticipantCode,
		InstrumentNID:     originalOrder.InstrumentNID,
		InstrumentCode:    originalOrder.InstrumentCode,
		Side:              originalOrder.Side,
		Quantity:          originalOrder.Quantity,
		SettlementDate:    originalOrder.SettlementDate,
		ReimbursementDate: originalOrder.ReimbursementDate,
		Periode:           originalOrder.Periode,
		State:             "S",
		MarketPrice:       originalOrder.MarketPrice,
		Rate:              originalOrder.Rate,
		Instruction:       originalOrder.Instruction,
		ARO:               originalOrder.ARO,
	}

	// Apply amendments
	if req.Quantity > 0 {
		amendedOrder.Quantity = req.Quantity
	}
	if !req.SettlementDate.IsZero() {
		amendedOrder.SettlementDate = req.SettlementDate
	}
	if !req.ReimbursementDate.IsZero() {
		amendedOrder.ReimbursementDate = req.ReimbursementDate
	}
	if req.Periode > 0 {
		amendedOrder.Periode = req.Periode
	}
	if req.ARO != nil {
		amendedOrder.ARO = *req.ARO
	}
	if req.Instruction != "" {
		amendedOrder.Instruction = req.Instruction
	}

	// Commit to Kafka
	h.ledger.Commit <- amendedOrder
	log.Printf("[APME-API] Order amended: %d -> %d", req.OrderNID, newOrderNID)

	respondSuccess(w, "Order amended successfully", map[string]interface{}{
		"original_order_nid": req.OrderNID,
		"new_order_nid":      newOrderNID,
		"status":             "submitted",
	})
}

// WithdrawOrder handles POST /api/order/withdraw
func (h *OrderHandler) WithdrawOrder(w http.ResponseWriter, r *http.Request) {
	log.Printf("[APME-API] WithdrawOrder handler called")

	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("[APME-API] Failed to read request body: %v", err)
		respondError(w, http.StatusBadRequest, "Failed to read request body")
		return
	}
	defer r.Body.Close()

	log.Printf("[APME-API] Request body: %s", string(body))

	var req WithdrawOrderRequest
	if err := json.Unmarshal(body, &req); err != nil {
		log.Printf("[APME-API] Failed to parse JSON: %v", err)
		respondError(w, http.StatusBadRequest, "Invalid JSON format")
		return
	}

	log.Printf("[APME-API] Parsed request: %+v", req)

	// Validate order exists
	log.Printf("[APME-API] Attempting to withdraw order NID: %d", req.OrderNID)
	order, exists := h.ledger.GetOrder(req.OrderNID)
	if !exists {
		log.Printf("[APME-API] Order not found: %d", req.OrderNID)
		respondError(w, http.StatusNotFound, "Order not found: "+fmt.Sprintf("%d", req.OrderNID))
		return
	}

	// Check if order can be withdrawn
	log.Printf("[APME-API] Order %d state: %s", req.OrderNID, order.State)
	if order.State != "O" && order.State != "P" {
		respondError(w, http.StatusUnprocessableEntity, "Order cannot be withdrawn in current state: "+order.State+" (must be O or P)")
		return
	}

	// Create withdrawal event
	withdraw := ledger.OrderWithdraw{
		OrderNID:      req.OrderNID,
		ReffRequestID: req.ReffRequestID,
	}

	// Commit to Kafka
	h.ledger.Commit <- withdraw
	log.Printf("[APME-API] Order withdrawal requested: %d", req.OrderNID)

	respondSuccess(w, "Order withdrawal submitted", map[string]interface{}{
		"order_nid": req.OrderNID,
		"status":    "withdrawal_pending",
	})
}

// validateOrderRequest validates the order request
func validateOrderRequest(req OrderRequest) error {
	if req.AccountCode == "" {
		return &ValidationError{Field: "account_code", Message: "is required"}
	}
	if req.ParticipantCode == "" {
		return &ValidationError{Field: "participant_code", Message: "is required"}
	}
	if req.InstrumentCode == "" {
		return &ValidationError{Field: "instrument_code", Message: "is required"}
	}
	if req.Side != "BORR" && req.Side != "LEND" {
		return &ValidationError{Field: "side", Message: "must be BORR or LEND"}
	}
	if req.Quantity <= 0 {
		return &ValidationError{Field: "quantity", Message: "must be greater than 0"}
	}

	// BORR-specific validations (LEND orders don't need settlement dates, periode, ARO)
	if req.Side == "BORR" {
		if req.Periode <= 0 {
			return &ValidationError{Field: "periode", Message: "must be greater than 0"}
		}
		if req.SettlementDate.IsZero() {
			return &ValidationError{Field: "settlement_date", Message: "is required"}
		}
		if req.ReimbursementDate.IsZero() {
			return &ValidationError{Field: "reimbursement_date", Message: "is required"}
		}
	}

	return nil
}

// ValidationError represents a validation error
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return e.Field + ": " + e.Message
}

// Helper functions for responses
func respondSuccess(w http.ResponseWriter, message string, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(OrderResponse{
		Status:  "success",
		Message: message,
		Data:    data,
	})
}

func respondError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(OrderResponse{
		Status:  "error",
		Message: message,
	})
}
