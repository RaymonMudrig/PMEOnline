package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"pmeonline/pkg/ledger"
)

type MasterDataHandler struct {
	ledger *ledger.LedgerPoint
}

func NewMasterDataHandler(l *ledger.LedgerPoint) *MasterDataHandler {
	return &MasterDataHandler{ledger: l}
}

// AccountRequest represents the account data from eClear
type AccountRequest struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	SID         string `json:"sid"`
	Email       string `json:"email"`
	Address     string `json:"address"`
	Participant string `json:"participant"`
}

// InstrumentRequest represents the instrument data from eClear
type InstrumentRequest struct {
	Code   string `json:"code"`
	Name   string `json:"name"`
	Status bool   `json:"status"`
}

// ParticipantRequest represents the participant data from eClear
type ParticipantRequest struct {
	Code            string `json:"code"`
	Name            string `json:"name"`
	BorrEligibility bool   `json:"borr_eligibility"`
	LendEligibility bool   `json:"lend_eligibility"`
}

// AccountLimitRequest represents the account limit data from eClear
type AccountLimitRequest struct {
	Code      string  `json:"code"`
	BorrLimit float64 `json:"borr_limit"`
	PoolLimit float64 `json:"pool_limit"`
}

// InsertAccounts handles POST /account/insert
func (h *MasterDataHandler) InsertAccounts(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("❌ Failed to read request body: %v", err)
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var accounts []AccountRequest
	if err := json.Unmarshal(body, &accounts); err != nil {
		log.Printf("❌ Failed to parse JSON: %v", err)
		http.Error(w, "Invalid JSON format", http.StatusBadRequest)
		return
	}

	log.Printf("📥 Received %d accounts from eClear", len(accounts))

	// Generate unique NIDs and commit to Kafka
	for i, acc := range accounts {
		// Validate required fields
		if acc.Code == "" || acc.SID == "" || acc.Participant == "" {
			log.Printf("⚠️  Skipping account with missing required fields: %+v", acc)
			continue
		}

		// Check if account already exists
		if _, exists := h.ledger.Account[acc.Code]; exists {
			log.Printf("⚠️  Account %s already exists, skipping", acc.Code)
			continue
		}

		// Check if participant exists
		if _, exists := h.ledger.Participant[acc.Participant]; !exists {
			log.Printf("⚠️  Participant %s not found for account %s", acc.Participant, acc.Code)
			continue
		}

		participantEntity := h.ledger.Participant[acc.Participant]

		account := ledger.Account{
			NID:             generateNID("account", i),
			Code:            acc.Code,
			SID:             acc.SID,
			Name:            acc.Name,
			Address:         acc.Address,
			ParticipantNID:  participantEntity.NID,
			ParticipantCode: participantEntity.Code,
		}

		// Commit to Kafka
		h.ledger.Commit <- account
		log.Printf("✅ Account committed: %s (SID: %s)", acc.Code, acc.SID)
	}

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"message": fmt.Sprintf("Processed %d accounts", len(accounts)),
	})
}

// InsertInstruments handles POST /instrument/insert
func (h *MasterDataHandler) InsertInstruments(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("❌ Failed to read request body: %v", err)
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var instruments []InstrumentRequest
	if err := json.Unmarshal(body, &instruments); err != nil {
		log.Printf("❌ Failed to parse JSON: %v", err)
		http.Error(w, "Invalid JSON format", http.StatusBadRequest)
		return
	}

	log.Printf("📥 Received %d instruments from eClear", len(instruments))

	for i, inst := range instruments {
		// Validate required fields
		if inst.Code == "" || inst.Name == "" {
			log.Printf("⚠️  Skipping instrument with missing required fields: %+v", inst)
			continue
		}

		// Check if instrument already exists
		if _, exists := h.ledger.Instrument[inst.Code]; exists {
			log.Printf("⚠️  Instrument %s already exists, skipping", inst.Code)
			continue
		}

		instrument := ledger.Instrument{
			NID:    generateNID("instrument", i),
			Code:   inst.Code,
			Name:   inst.Name,
			Type:   "STOCK", // Default type
			Status: inst.Status,
		}

		// Commit to Kafka
		h.ledger.Commit <- instrument

		// Check eligibility status change
		if !inst.Status {
			log.Printf("⚠️  Instrument %s is now INELIGIBLE - matching will be blocked", inst.Code)
			// The OMS will handle blocking matching for this instrument
		}

		log.Printf("✅ Instrument committed: %s (%s) - Status: %v", inst.Code, inst.Name, inst.Status)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"message": fmt.Sprintf("Processed %d instruments", len(instruments)),
	})
}

// InsertParticipants handles POST /participant/insert
func (h *MasterDataHandler) InsertParticipants(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("❌ Failed to read request body: %v", err)
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var participants []ParticipantRequest
	if err := json.Unmarshal(body, &participants); err != nil {
		log.Printf("❌ Failed to parse JSON: %v", err)
		http.Error(w, "Invalid JSON format", http.StatusBadRequest)
		return
	}

	log.Printf("📥 Received %d participants from eClear", len(participants))

	for i, part := range participants {
		// Validate required fields
		if part.Code == "" || part.Name == "" {
			log.Printf("⚠️  Skipping participant with missing required fields: %+v", part)
			continue
		}

		// Check if participant already exists
		if _, exists := h.ledger.Participant[part.Code]; exists {
			log.Printf("⚠️  Participant %s already exists, skipping", part.Code)
			continue
		}

		participant := ledger.Participant{
			NID:             generateNID("participant", i),
			Code:            part.Code,
			Name:            part.Name,
			BorrEligibility: part.BorrEligibility,
			LendEligibility: part.LendEligibility,
		}

		// Commit to Kafka
		h.ledger.Commit <- participant
		log.Printf("✅ Participant committed: %s (%s) - Borr: %v, Lend: %v",
			part.Code, part.Name, part.BorrEligibility, part.LendEligibility)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"message": fmt.Sprintf("Processed %d participants", len(participants)),
	})
}

// UpdateAccountLimit handles POST /account/limit
func (h *MasterDataHandler) UpdateAccountLimit(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("❌ Failed to read request body: %v", err)
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var limits []AccountLimitRequest
	if err := json.Unmarshal(body, &limits); err != nil {
		log.Printf("❌ Failed to parse JSON: %v", err)
		http.Error(w, "Invalid JSON format", http.StatusBadRequest)
		return
	}

	log.Printf("📥 Received %d account limits from eClear", len(limits))

	for _, limit := range limits {
		// Validate required fields
		if limit.Code == "" {
			log.Printf("⚠️  Skipping limit with missing account code: %+v", limit)
			continue
		}

		// Check if account exists
		if _, exists := h.ledger.Account[limit.Code]; !exists {
			log.Printf("⚠️  Account %s not found for limit update", limit.Code)
			continue
		}

		accountEntity := h.ledger.Account[limit.Code]

		accountLimit := ledger.AccountLimit{
			NID:        accountEntity.NID,
			Code:       limit.Code,
			AccountNID: accountEntity.NID,
			TradeLimit: limit.BorrLimit, // BorrLimit is TradeLimit
			PoolLimit:  limit.PoolLimit,
		}

		// Commit to Kafka
		h.ledger.Commit <- accountLimit
		log.Printf("✅ Account limit updated: %s - TradeLimit: %.2f, PoolLimit: %.2f",
			limit.Code, limit.BorrLimit, limit.PoolLimit)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"message": fmt.Sprintf("Processed %d account limits", len(limits)),
	})
}

// generateNID generates a unique NID based on entity type and index
// In production, this should use a more robust ID generation strategy
func generateNID(entityType string, index int) int {
	// Simple strategy: use timestamp + index
	// For production, consider using a distributed ID generator
	return int(ledger.GetCurrentTimeMillis()) + index
}
