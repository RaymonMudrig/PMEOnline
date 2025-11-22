package repository

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"pmeonline/pkg/ledger"
)

type OtherRepository struct {
	db *sql.DB
}

func NewOtherRepository(db *sql.DB) *OtherRepository {
	return &OtherRepository{db: db}
}

// Parameter operations
func (r *OtherRepository) UpsertParameter(p ledger.Parameter) error {
	// Delete old parameters (we only keep the latest)
	if _, err := r.db.Exec("DELETE FROM parameters"); err != nil {
		return fmt.Errorf("failed to delete old parameters: %w", err)
	}

	query := `
		INSERT INTO parameters (flat_fee, lending_fee, borrowing_fee, max_quantity, borrow_max_open_day, denomination_limit, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	timestamp := ledger.GetCurrentTimeMillis()
	_, err := r.db.Exec(query, p.FlatFee, p.LendingFee, p.BorrowingFee, p.MaxQuantity, p.BorrowMaxOpenDay, p.DenominationLimit, timestamp)
	if err != nil {
		return fmt.Errorf("failed to upsert parameter: %w", err)
	}

	return nil
}

// SessionTime operations
func (r *OtherRepository) UpsertSessionTime(s ledger.SessionTime) error {
	// Delete old session times (we only keep the latest)
	if _, err := r.db.Exec("DELETE FROM session_times"); err != nil {
		return fmt.Errorf("failed to delete old session times: %w", err)
	}

	query := `
		INSERT INTO session_times (session1_start, session1_end, session2_start, session2_end, updated_at)
		VALUES ($1, $2, $3, $4, $5)
	`

	timestamp := ledger.GetCurrentTimeMillis()
	_, err := r.db.Exec(query, s.Session1Start, s.Session1End, s.Session2Start, s.Session2End, timestamp)
	if err != nil {
		return fmt.Errorf("failed to upsert session time: %w", err)
	}

	return nil
}

// Holiday operations
func (r *OtherRepository) UpsertHoliday(h ledger.Holiday) error {
	query := `
		INSERT INTO holidays (nid, year, date, description, last_update)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (nid) DO UPDATE SET
			year = EXCLUDED.year,
			date = EXCLUDED.date,
			description = EXCLUDED.description,
			last_update = EXCLUDED.last_update
	`

	timestamp := ledger.GetCurrentTimeMillis()
	_, err := r.db.Exec(query, h.NID, h.Tahun, h.Date, h.Description, timestamp)
	if err != nil {
		return fmt.Errorf("failed to upsert holiday: %w", err)
	}

	return nil
}

// ServiceStart operations
func (r *OtherRepository) InsertServiceStart(s ledger.ServiceStart) error {
	query := `
		INSERT INTO service_starts (service_name, started_at)
		VALUES ($1, $2)
	`

	timestamp := ledger.GetCurrentTimeMillis()
	_, err := r.db.Exec(query, s.ID, timestamp)
	if err != nil {
		return fmt.Errorf("failed to insert service start: %w", err)
	}

	return nil
}

// Event log operations
func (r *OtherRepository) LogEvent(eventType string, eventData interface{}, timestamp int64) error {
	dataJSON, err := json.Marshal(eventData)
	if err != nil {
		return fmt.Errorf("failed to marshal event data: %w", err)
	}

	query := `
		INSERT INTO event_log (event_type, event_data, timestamp)
		VALUES ($1, $2, $3)
	`

	_, err = r.db.Exec(query, eventType, dataJSON, timestamp)
	if err != nil {
		return fmt.Errorf("failed to log event: %w", err)
	}

	return nil
}
