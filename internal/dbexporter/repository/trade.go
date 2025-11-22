package repository

import (
	"database/sql"
	"fmt"

	"pmeonline/pkg/ledger"
)

type TradeRepository struct {
	db *sql.DB
}

func NewTradeRepository(db *sql.DB) *TradeRepository {
	return &TradeRepository{db: db}
}

func (r *TradeRepository) Insert(t ledger.Trade) error {
	query := `
		INSERT INTO trades (
			nid, kpei_reff, instrument_code, quantity, periode, state,
			fee_flat_rate, fee_borr_rate, fee_lend_rate, matched_at, reimburse_at, last_update
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (nid) DO UPDATE SET
			state = EXCLUDED.state,
			last_update = EXCLUDED.last_update
	`

	timestamp := ledger.GetCurrentTimeMillis()
	_, err := r.db.Exec(query,
		t.NID, t.KpeiReff, t.InstrumentCode, t.Quantity, t.Periode, t.State,
		t.FeeFlatRate, t.FeeBorrRate, t.FeeLendRate, t.MatchedAt, t.ReimburseAt, timestamp,
	)
	if err != nil {
		return fmt.Errorf("failed to insert trade: %w", err)
	}

	return nil
}

func (r *TradeRepository) UpdateState(nid int, state string) error {
	query := `
		UPDATE trades
		SET state = $2, last_update = $3
		WHERE nid = $1
	`

	timestamp := ledger.GetCurrentTimeMillis()
	_, err := r.db.Exec(query, nid, state, timestamp)
	if err != nil {
		return fmt.Errorf("failed to update trade state: %w", err)
	}

	return nil
}
