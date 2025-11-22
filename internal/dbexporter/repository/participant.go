package repository

import (
	"database/sql"
	"fmt"

	"pmeonline/pkg/ledger"
)

type ParticipantRepository struct {
	db *sql.DB
}

func NewParticipantRepository(db *sql.DB) *ParticipantRepository {
	return &ParticipantRepository{db: db}
}

func (r *ParticipantRepository) Upsert(p ledger.Participant) error {
	query := `
		INSERT INTO participants (nid, code, name, borr_eligibility, lend_eligibility, last_update)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (nid) DO UPDATE SET
			name = EXCLUDED.name,
			borr_eligibility = EXCLUDED.borr_eligibility,
			lend_eligibility = EXCLUDED.lend_eligibility,
			last_update = EXCLUDED.last_update
	`

	timestamp := ledger.GetCurrentTimeMillis()
	_, err := r.db.Exec(query, p.NID, p.Code, p.Name, p.BorrEligibility, p.LendEligibility, timestamp)
	if err != nil {
		return fmt.Errorf("failed to upsert participant: %w", err)
	}

	return nil
}
