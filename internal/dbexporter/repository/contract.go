package repository

import (
	"database/sql"
	"fmt"

	"pmeonline/pkg/ledger"
)

type ContractRepository struct {
	db *sql.DB
}

func NewContractRepository(db *sql.DB) *ContractRepository {
	return &ContractRepository{db: db}
}

func (r *ContractRepository) Insert(c ledger.Contract) error {
	query := `
		INSERT INTO contracts (
			nid, trade_nid, kpei_reff, side, account_code, account_sid, account_participant_code,
			order_nid, instrument_code, quantity, periode, state, fee_flat_val, fee_val_daily,
			fee_val_accumulated, matched_at, reimburse_at, last_update
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
		ON CONFLICT (nid) DO UPDATE SET
			state = EXCLUDED.state,
			fee_flat_val = EXCLUDED.fee_flat_val,
			fee_val_daily = EXCLUDED.fee_val_daily,
			fee_val_accumulated = EXCLUDED.fee_val_accumulated,
			last_update = EXCLUDED.last_update
	`

	timestamp := ledger.GetCurrentTimeMillis()
	_, err := r.db.Exec(query,
		c.NID, c.TradeNID, c.KpeiReff, c.Side, c.AccountCode, c.AccountSID, c.AccountParticipantCode,
		c.OrderNID, c.InstrumentCode, c.Quantity, c.Periode, c.State, c.FeeFlatVal, c.FeeValDaily,
		c.FeeValAccumulated, c.MatchedAt, c.ReimburseAt, timestamp,
	)
	if err != nil {
		return fmt.Errorf("failed to insert contract: %w", err)
	}

	return nil
}

func (r *ContractRepository) UpdateState(nid int, state string) error {
	query := `
		UPDATE contracts
		SET state = $2, last_update = $3
		WHERE nid = $1
	`

	timestamp := ledger.GetCurrentTimeMillis()
	_, err := r.db.Exec(query, nid, state, timestamp)
	if err != nil {
		return fmt.Errorf("failed to update contract state: %w", err)
	}

	return nil
}

func (r *ContractRepository) UpdateFees(nid int, feeFlatVal, feeValDaily, feeValAccumulated float64) error {
	query := `
		UPDATE contracts
		SET fee_flat_val = $2, fee_val_daily = $3, fee_val_accumulated = $4, last_update = $5
		WHERE nid = $1
	`

	timestamp := ledger.GetCurrentTimeMillis()
	_, err := r.db.Exec(query, nid, feeFlatVal, feeValDaily, feeValAccumulated, timestamp)
	if err != nil {
		return fmt.Errorf("failed to update contract fees: %w", err)
	}

	return nil
}
