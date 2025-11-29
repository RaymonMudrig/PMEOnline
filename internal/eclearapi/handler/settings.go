package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"pmeonline/pkg/ledger"
)

type SettingsHandler struct {
	ledger *ledger.LedgerPoint
}

func NewSettingsHandler(l *ledger.LedgerPoint) *SettingsHandler {
	return &SettingsHandler{ledger: l}
}

// GetParameter handles GET /parameter
func (h *SettingsHandler) GetParameter(w http.ResponseWriter, r *http.Request) {
	param := h.ledger.GetParameter()

	respondSuccess(w, "Parameter retrieved", map[string]interface{}{
		"parameter": map[string]interface{}{
			"nid":                 param.NID,
			"description":         param.Description,
			"flat_fee":            param.FlatFee,
			"lending_fee":         param.LendingFee,
			"borrowing_fee":       param.BorrowingFee,
			"max_quantity":        param.MaxQuantity,
			"borrow_max_open_day": param.BorrowMaxOpenDay,
			"denomination_limit":  param.DenominationLimit,
			"update":              param.Update.Format("2006-01-02 15:04:05"),
		},
	})
}

// UpdateParameter handles POST /parameter/update
func (h *SettingsHandler) UpdateParameter(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Description       string  `json:"description"`
		FlatFee           float64 `json:"flat_fee"`
		LendingFee        float64 `json:"lending_fee"`
		BorrowingFee      float64 `json:"borrowing_fee"`
		MaxQuantity       float64 `json:"max_quantity"`
		BorrowMaxOpenDay  int     `json:"borrow_max_open_day"`
		DenominationLimit int     `json:"denomination_limit"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	// Validate inputs
	if req.FlatFee < 0 || req.LendingFee < 0 || req.BorrowingFee < 0 {
		respondError(w, http.StatusBadRequest, "Fees cannot be negative", nil)
		return
	}

	if req.MaxQuantity <= 0 {
		respondError(w, http.StatusBadRequest, "Max quantity must be positive", nil)
		return
	}

	if req.BorrowMaxOpenDay <= 0 {
		respondError(w, http.StatusBadRequest, "Borrow max open day must be positive", nil)
		return
	}

	if req.DenominationLimit <= 0 {
		respondError(w, http.StatusBadRequest, "Denomination limit must be positive", nil)
		return
	}

	// Create parameter entry
	param := ledger.Parameter{
		NID:               int(ledger.GetCurrentTimeMillis()),
		Update:            time.Now(),
		Description:       req.Description,
		FlatFee:           req.FlatFee,
		LendingFee:        req.LendingFee,
		BorrowingFee:      req.BorrowingFee,
		MaxQuantity:       req.MaxQuantity,
		BorrowMaxOpenDay:  req.BorrowMaxOpenDay,
		DenominationLimit: req.DenominationLimit,
	}

	// Commit to ledger
	h.ledger.Commit <- param

	respondSuccess(w, "Parameter updated successfully", map[string]interface{}{
		"parameter": param,
	})
}

// GetHolidays handles GET /holiday/list
func (h *SettingsHandler) GetHolidays(w http.ResponseWriter, r *http.Request) {
	holidays := make([]map[string]interface{}, 0)

	h.ledger.ForEachHoliday(func(hol ledger.HolidayEntity) bool {
		holidays = append(holidays, map[string]interface{}{
			"nid":         hol.NID,
			"tahun":       hol.Tahun,
			"date":        hol.Date.Format("2006-01-02"),
			"description": hol.Description,
		})
		return true
	})

	respondSuccess(w, "Holidays retrieved", map[string]interface{}{
		"count":    len(holidays),
		"holidays": holidays,
	})
}

// AddHoliday handles POST /holiday/add
func (h *SettingsHandler) AddHoliday(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Date        string `json:"date"`
		Description string `json:"description"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	// Parse date
	date, err := time.Parse("2006-01-02", req.Date)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid date format, use YYYY-MM-DD", err)
		return
	}

	// Create holiday entry
	holiday := ledger.Holiday{
		NID:         int(ledger.GetCurrentTimeMillis()),
		Tahun:       date.Year(),
		Date:        date,
		Description: req.Description,
	}

	// Commit to ledger
	h.ledger.Commit <- holiday

	respondSuccess(w, "Holiday added successfully", map[string]interface{}{
		"holiday": map[string]interface{}{
			"nid":         holiday.NID,
			"tahun":       holiday.Tahun,
			"date":        holiday.Date.Format("2006-01-02"),
			"description": holiday.Description,
		},
	})
}

// GetSessionTime handles GET /sessiontime
func (h *SettingsHandler) GetSessionTime(w http.ResponseWriter, r *http.Request) {
	st := h.ledger.GetSessionTime()

	respondSuccess(w, "Session time retrieved", map[string]interface{}{
		"sessiontime": map[string]interface{}{
			"nid":             st.NID,
			"description":     st.Description,
			"session1_start":  st.Session1Start.Format("15:04:05"),
			"session1_end":    st.Session1End.Format("15:04:05"),
			"session2_start":  st.Session2Start.Format("15:04:05"),
			"session2_end":    st.Session2End.Format("15:04:05"),
			"update":          st.Update.Format("2006-01-02 15:04:05"),
		},
	})
}

// UpdateSessionTime handles POST /sessiontime/update
func (h *SettingsHandler) UpdateSessionTime(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Description   string `json:"description"`
		Session1Start string `json:"session1_start"`
		Session1End   string `json:"session1_end"`
		Session2Start string `json:"session2_start"`
		Session2End   string `json:"session2_end"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	// Parse times (format: HH:MM:SS or HH:MM)
	parseTime := func(timeStr string) (time.Time, error) {
		// Try HH:MM:SS first
		t, err := time.Parse("15:04:05", timeStr)
		if err != nil {
			// Try HH:MM
			t, err = time.Parse("15:04", timeStr)
			if err != nil {
				return time.Time{}, err
			}
		}
		return t, nil
	}

	session1Start, err := parseTime(req.Session1Start)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid session1_start format, use HH:MM:SS or HH:MM", err)
		return
	}

	session1End, err := parseTime(req.Session1End)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid session1_end format, use HH:MM:SS or HH:MM", err)
		return
	}

	session2Start, err := parseTime(req.Session2Start)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid session2_start format, use HH:MM:SS or HH:MM", err)
		return
	}

	session2End, err := parseTime(req.Session2End)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid session2_end format, use HH:MM:SS or HH:MM", err)
		return
	}

	// Create session time entry
	sessionTime := ledger.SessionTime{
		NID:           int(ledger.GetCurrentTimeMillis()),
		Description:   req.Description,
		Update:        time.Now(),
		Session1Start: session1Start,
		Session1End:   session1End,
		Session2Start: session2Start,
		Session2End:   session2End,
	}

	// Commit to ledger
	h.ledger.Commit <- sessionTime

	respondSuccess(w, "Session time updated successfully", map[string]interface{}{
		"sessiontime": map[string]interface{}{
			"nid":             sessionTime.NID,
			"description":     sessionTime.Description,
			"session1_start":  sessionTime.Session1Start.Format("15:04:05"),
			"session1_end":    sessionTime.Session1End.Format("15:04:05"),
			"session2_start":  sessionTime.Session2Start.Format("15:04:05"),
			"session2_end":    sessionTime.Session2End.Format("15:04:05"),
			"update":          sessionTime.Update.Format("2006-01-02 15:04:05"),
		},
	})
}
