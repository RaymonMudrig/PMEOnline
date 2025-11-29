package handler

import (
	"encoding/json"
	"net/http"

	"pmeonline/pkg/ledger"
)

type QueryHandler struct {
	ledger *ledger.LedgerPoint
}

func NewQueryHandler(l *ledger.LedgerPoint) *QueryHandler {
	return &QueryHandler{ledger: l}
}

// GetParticipants handles GET /participant/list
func (h *QueryHandler) GetParticipants(w http.ResponseWriter, r *http.Request) {
	participants := make([]map[string]interface{}, 0)

	h.ledger.ForEachParticipant(func(p ledger.ParticipantEntity) bool {
		participants = append(participants, map[string]interface{}{
			"code":             p.Code,
			"name":             p.Name,
			"borr_eligibility": p.BorrEligibility,
			"lend_eligibility": p.LendEligibility,
		})
		return true
	})

	respondSuccess(w, "Participants retrieved", map[string]interface{}{
		"count":        len(participants),
		"participants": participants,
	})
}

// GetInstruments handles GET /instrument/list
func (h *QueryHandler) GetInstruments(w http.ResponseWriter, r *http.Request) {
	instruments := make([]map[string]interface{}, 0)

	h.ledger.ForEachInstrument(func(i ledger.InstrumentEntity) bool {
		instruments = append(instruments, map[string]interface{}{
			"code":   i.Code,
			"name":   i.Name,
			"type":   i.Type,
			"status": i.Status,
		})
		return true
	})

	respondSuccess(w, "Instruments retrieved", map[string]interface{}{
		"count":       len(instruments),
		"instruments": instruments,
	})
}

// GetAccounts handles GET /account/list
func (h *QueryHandler) GetAccounts(w http.ResponseWriter, r *http.Request) {
	accounts := make([]map[string]interface{}, 0)

	h.ledger.ForEachAccount(func(a ledger.AccountEntity) bool {
		accounts = append(accounts, map[string]interface{}{
			"code":             a.Code,
			"sid":              a.SID,
			"name":             a.Name,
			"participant_code": a.ParticipantCode,
			"trade_limit":      a.TradeLimit,
			"pool_limit":       a.PoolLimit,
		})
		return true
	})

	respondSuccess(w, "Accounts retrieved", map[string]interface{}{
		"count":    len(accounts),
		"accounts": accounts,
	})
}

// Helper function to send success response
func respondSuccess(w http.ResponseWriter, message string, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "success",
		"message": message,
		"data":    data,
	})
}

// Helper function to send error response
func respondError(w http.ResponseWriter, statusCode int, message string, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	response := map[string]interface{}{
		"status":  "error",
		"message": message,
	}
	if err != nil {
		response["error"] = err.Error()
	}
	json.NewEncoder(w).Encode(response)
}
