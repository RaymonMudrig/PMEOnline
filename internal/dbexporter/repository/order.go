package repository

import (
	"database/sql"
	"fmt"

	"pmeonline/pkg/ledger"
)

type OrderRepository struct {
	db *sql.DB
}

func NewOrderRepository(db *sql.DB) *OrderRepository {
	return &OrderRepository{db: db}
}

func (r *OrderRepository) Insert(o ledger.Order) error {
	query := `
		INSERT INTO orders (
			nid, prev_nid, reff_request_id, account_code, participant_code, instrument_code,
			side, quantity, done_quantity, settlement_date, reimbursement_date, periode,
			state, market_price, rate, instruction, aro, entry_at, last_update
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 0, $9, $10, $11, $12, $13, $14, $15, $16, CURRENT_TIMESTAMP, $17)
		ON CONFLICT (nid) DO UPDATE SET
			done_quantity = EXCLUDED.done_quantity,
			state = EXCLUDED.state,
			amend_at = CASE WHEN EXCLUDED.state = 'A' THEN CURRENT_TIMESTAMP ELSE orders.amend_at END,
			withdraw_at = CASE WHEN EXCLUDED.state = 'W' THEN CURRENT_TIMESTAMP ELSE orders.withdraw_at END,
			last_update = EXCLUDED.last_update
	`

	timestamp := ledger.GetCurrentTimeMillis()
	_, err := r.db.Exec(query,
		o.NID, o.PrevNID, o.ReffRequestID, o.AccountCode, o.ParticipantCode, o.InstrumentCode,
		o.Side, o.Quantity, o.SettlementDate, o.ReimbursementDate, o.Periode,
		o.State, o.MarketPrice, o.Rate, o.Instruction, o.ARO, timestamp,
	)
	if err != nil {
		return fmt.Errorf("failed to insert order: %w", err)
	}

	return nil
}

func (r *OrderRepository) UpdateState(nid int, state string, doneQuantity float64) error {
	query := `
		UPDATE orders
		SET state = $2, done_quantity = $3, last_update = $4
		WHERE nid = $1
	`

	timestamp := ledger.GetCurrentTimeMillis()
	_, err := r.db.Exec(query, nid, state, doneQuantity, timestamp)
	if err != nil {
		return fmt.Errorf("failed to update order state: %w", err)
	}

	return nil
}
