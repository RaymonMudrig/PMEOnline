package repository

import (
	"database/sql"
	"fmt"

	"pmeonline/pkg/ledger"
)

type AccountRepository struct {
	db *sql.DB
}

func NewAccountRepository(db *sql.DB) *AccountRepository {
	return &AccountRepository{db: db}
}

func (r *AccountRepository) Upsert(a ledger.Account) error {
	query := `
		INSERT INTO accounts (nid, code, sid, name, participant_code, trade_limit, pool_limit, last_update)
		VALUES ($1, $2, $3, $4, $5, 0, 0, $6)
		ON CONFLICT (nid) DO UPDATE SET
			name = EXCLUDED.name,
			last_update = EXCLUDED.last_update
	`

	timestamp := ledger.GetCurrentTimeMillis()
	_, err := r.db.Exec(query, a.NID, a.Code, a.SID, a.Name, a.ParticipantCode, timestamp)
	if err != nil {
		return fmt.Errorf("failed to upsert account: %w", err)
	}

	return nil
}

func (r *AccountRepository) UpdateLimit(a ledger.AccountLimit) error {
	query := `
		UPDATE accounts
		SET trade_limit = $2, pool_limit = $3, last_update = $4
		WHERE code = $1
	`

	timestamp := ledger.GetCurrentTimeMillis()
	result, err := r.db.Exec(query, a.Code, a.TradeLimit, a.PoolLimit, timestamp)
	if err != nil {
		return fmt.Errorf("failed to update account limit: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("account not found: %s", a.Code)
	}

	return nil
}
