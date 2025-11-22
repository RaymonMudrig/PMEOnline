package repository

import (
	"database/sql"
	"fmt"

	"pmeonline/pkg/ledger"
)

type InstrumentRepository struct {
	db *sql.DB
}

func NewInstrumentRepository(db *sql.DB) *InstrumentRepository {
	return &InstrumentRepository{db: db}
}

func (r *InstrumentRepository) Upsert(i ledger.Instrument) error {
	query := `
		INSERT INTO instruments (nid, code, name, type, status, last_update)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (nid) DO UPDATE SET
			name = EXCLUDED.name,
			type = EXCLUDED.type,
			status = EXCLUDED.status,
			last_update = EXCLUDED.last_update
	`

	timestamp := ledger.GetCurrentTimeMillis()
	_, err := r.db.Exec(query, i.NID, i.Code, i.Name, i.Type, i.Status, timestamp)
	if err != nil {
		return fmt.Errorf("failed to upsert instrument: %w", err)
	}

	return nil
}
