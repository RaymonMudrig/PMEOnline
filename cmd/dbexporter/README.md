# Database Exporter Service

The Database Exporter service subscribes to all Kafka events from the PME Online system and persists them to a PostgreSQL database for historical queries, reporting, and auditing.

## Architecture

```
┌───────┐     ┌──────────────┐     ┌────────────┐
│ Kafka │────>│ DB Exporter  │────>│ PostgreSQL │
│       │     │   Service    │     │  Database  │
└───────┘     └──────────────┘     └────────────┘
```

## Features

- Event-sourced persistence of all system events
- Automatic database schema migration
- Repository pattern for clean data access
- Comprehensive event logging for audit trails
- Real-time synchronization with Kafka events
- Support for all entity types (orders, trades, contracts, accounts, etc.)

## Database Schema

The service automatically creates the following tables:

### Master Data Tables
- **participants** - Securities trading participants (APME/PEI)
- **instruments** - Trading instruments with eligibility status
- **accounts** - Client accounts with limits
- **holidays** - Non-trading days
- **parameters** - System parameters (fees, limits)
- **session_times** - Trading session schedules

### Transaction Tables
- **orders** - Borrowing and lending orders
- **trades** - Matched trades between parties
- **contracts** - Individual contracts (borrower and lender sides)

### Audit Tables
- **event_log** - Complete audit trail of all events in JSONB format
- **service_starts** - Service startup tracking

## Configuration

Environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `KAFKA_URL` | `localhost:9092` | Kafka broker URL |
| `KAFKA_TOPIC` | `pme-ledger` | Kafka topic name |
| `DB_HOST` | `localhost` | PostgreSQL host |
| `DB_PORT` | `5432` | PostgreSQL port |
| `DB_USER` | `pmeuser` | Database username |
| `DB_PASSWORD` | `pmepass` | Database password |
| `DB_NAME` | `pmedb` | Database name |
| `DB_SSLMODE` | `disable` | SSL mode (disable, require, verify-full) |

## Running

### Prerequisites

1. Kafka must be running
2. PostgreSQL must be running
3. Database and user must be created

### Create PostgreSQL Database

```bash
# Connect to PostgreSQL
psql -U postgres

# Create database and user
CREATE DATABASE pmedb;
CREATE USER pmeuser WITH PASSWORD 'pmepass';
GRANT ALL PRIVILEGES ON DATABASE pmedb TO pmeuser;
\q
```

### Start the service

```bash
# Using default configuration
cd cmd/dbexporter
go run main.go

# Using custom configuration
DB_HOST=localhost \
DB_PORT=5432 \
DB_USER=pmeuser \
DB_PASSWORD=pmepass \
DB_NAME=pmedb \
KAFKA_URL=localhost:9092 \
KAFKA_TOPIC=pme-ledger \
go run main.go

# Or use Make
make run-dbexporter
```

## Testing

### 1. Start Infrastructure

```bash
# Start Kafka and PostgreSQL
make setup
```

### 2. Create PostgreSQL Database

```bash
# Using docker-compose PostgreSQL
docker exec -it pme-postgres psql -U postgres -c "CREATE DATABASE pmedb;"
docker exec -it pme-postgres psql -U postgres -c "CREATE USER pmeuser WITH PASSWORD 'pmepass';"
docker exec -it pme-postgres psql -U postgres -c "GRANT ALL PRIVILEGES ON DATABASE pmedb TO pmeuser;"
```

### 3. Start Services in Order

```bash
# Terminal 1: DB Exporter
make run-dbexporter

# Terminal 2: eClear API
make run-eclearapi

# Terminal 3: Populate master data
make test-eclearapi

# Terminal 4: OMS
make run-pmeoms

# Terminal 5: APME API
make run-pmeapi
```

### 4. Verify Data Export

```bash
# Connect to PostgreSQL
docker exec -it pme-postgres psql -U pmeuser -d pmedb

# Check data
SELECT COUNT(*) FROM participants;
SELECT COUNT(*) FROM instruments;
SELECT COUNT(*) FROM accounts;
SELECT COUNT(*) FROM orders;
SELECT COUNT(*) FROM trades;
SELECT COUNT(*) FROM contracts;

# View recent events
SELECT event_type, COUNT(*) 
FROM event_log 
GROUP BY event_type 
ORDER BY COUNT(*) DESC;

# View recent events with data
SELECT id, event_type, timestamp, event_data 
FROM event_log 
ORDER BY timestamp DESC 
LIMIT 10;
```

## Repository Pattern

The service uses a repository pattern for clean separation of concerns:

### Repositories

- **ParticipantRepository** - CRUD operations for participants
- **InstrumentRepository** - CRUD operations for instruments
- **AccountRepository** - CRUD operations for accounts and limits
- **OrderRepository** - CRUD operations for orders
- **TradeRepository** - CRUD operations for trades
- **ContractRepository** - CRUD operations for contracts
- **OtherRepository** - Operations for parameters, holidays, events

### Example Usage

```go
// Create repository
repo := repository.NewParticipantRepository(db)

// Upsert participant
err := repo.Upsert(participant)

// Update order state
err := orderRepo.UpdateState(orderNID, "O", doneQuantity)
```

## Event Synchronization

The service implements the `LedgerPointInterface` to receive all events:

```go
type Exporter struct {
    participantRepo *repository.ParticipantRepository
    instrumentRepo  *repository.InstrumentRepository
    accountRepo     *repository.AccountRepository
    orderRepo       *repository.OrderRepository
    tradeRepo       *repository.TradeRepository
    contractRepo    *repository.ContractRepository
    otherRepo       *repository.OtherRepository
}

// Implement sync methods
func (e *Exporter) SyncOrder(o ledger.Order) { ... }
func (e *Exporter) SyncTrade(t ledger.Trade) { ... }
func (e *Exporter) SyncContract(c ledger.Contract) { ... }
// ... and more
```

## Database Queries

### Common Queries

```sql
-- Get all open orders
SELECT * FROM orders WHERE state = 'O' ORDER BY entry_at DESC;

-- Get orders by participant
SELECT * FROM orders WHERE participant_code = 'YU' ORDER BY entry_at DESC;

-- Get contracts with fees
SELECT 
    c.*, 
    t.kpei_reff, 
    t.matched_at
FROM contracts c
JOIN trades t ON c.trade_nid = t.nid
WHERE c.state = 'O'
ORDER BY c.matched_at DESC;

-- Calculate total fees by participant
SELECT 
    account_participant_code,
    side,
    SUM(fee_val_accumulated) as total_fees,
    COUNT(*) as contract_count
FROM contracts
WHERE state = 'O'
GROUP BY account_participant_code, side;

-- Get SBL aggregate by instrument
SELECT 
    instrument_code,
    SUM(CASE WHEN side = 'BORR' THEN quantity ELSE 0 END) as borrow_quantity,
    SUM(CASE WHEN side = 'LEND' THEN quantity ELSE 0 END) as lend_quantity
FROM orders
WHERE state IN ('O', 'P')
GROUP BY instrument_code;

-- Event log statistics
SELECT 
    event_type,
    COUNT(*) as count,
    MIN(timestamp) as first_event,
    MAX(timestamp) as last_event
FROM event_log
GROUP BY event_type
ORDER BY count DESC;
```

## Migration Management

The service automatically runs migrations on startup. Migration files are located in `migrations/`:

- `001_create_tables.sql` - Initial schema creation

To add new migrations:

1. Create a new SQL file in `migrations/` (e.g., `002_add_indexes.sql`)
2. Update `main.go` to run the new migration
3. The service will apply it on next startup

## Performance Considerations

### Indexes

The schema includes indexes on:
- Primary keys (nid columns)
- Foreign keys (participant_code, instrument_code, account_code, etc.)
- Frequently queried fields (state, side, dates)

### Connection Pooling

The service uses connection pooling for optimal performance:
- Max open connections: 25
- Max idle connections: 5
- Connection max lifetime: 5 minutes

### Event Logging

All events are logged to the `event_log` table in JSONB format for:
- Complete audit trail
- Forensic analysis
- Debugging
- Compliance

## Troubleshooting

### Service won't start

**Check PostgreSQL connection:**
```bash
# Test connection
psql -h localhost -U pmeuser -d pmedb -c "SELECT 1;"

# Check environment variables
echo $DB_HOST
echo $DB_USER
echo $DB_NAME
```

### Migration fails

**Check permissions:**
```sql
-- Grant schema permissions
GRANT ALL ON SCHEMA public TO pmeuser;
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO pmeuser;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO pmeuser;
```

### No events being written

**Check Kafka connection:**
```bash
# Verify events in Kafka
docker exec pme-kafka kafka-console-consumer.sh \
    --bootstrap-server localhost:9092 \
    --topic pme-ledger \
    --from-beginning \
    --max-messages 10
```

**Check service logs:**
- Look for "[DB-EXPORTER]" prefix in logs
- Check for connection errors
- Verify repository operations

### Data inconsistencies

**Check event log:**
```sql
-- Find failed operations in logs
SELECT * FROM event_log 
WHERE event_data::text LIKE '%error%' 
ORDER BY timestamp DESC;

-- Compare event counts
SELECT 
    (SELECT COUNT(*) FROM orders) as orders_count,
    (SELECT COUNT(*) FROM event_log WHERE event_type LIKE 'Order%') as order_events_count;
```

## Production Considerations

1. **Backup Strategy**
   - Regular PostgreSQL backups
   - Point-in-time recovery enabled
   - Transaction log archiving

2. **Monitoring**
   - Database connection health
   - Event processing lag
   - Disk space usage
   - Query performance

3. **Scaling**
   - Read replicas for reporting queries
   - Partitioning for large tables (orders, trades, contracts)
   - Archive old data to cold storage

4. **Security**
   - Use SSL/TLS for database connections
   - Encrypt sensitive data at rest
   - Implement proper access controls
   - Regular security audits

5. **Maintenance**
   - Regular VACUUM and ANALYZE
   - Index maintenance
   - Partition management
   - Log rotation

## Architecture Notes

### Event Sourcing Pattern

The service follows event sourcing principles:
- All state changes come from Kafka events
- Events are immutable once logged
- Database represents current state + history
- Event log provides complete audit trail

### CQRS Pattern

The service acts as the "Write" side of CQRS:
- Commands (events) are written to database
- Queries can be optimized separately
- Read models can be materialized views
- Reporting queries don't impact write performance

### Idempotency

All repository operations are designed to be idempotent:
- Upserts use `ON CONFLICT` clauses
- Timestamps track last update
- Duplicate events are handled gracefully

## Build Information

**Binary Size:** ~9.5 MB  
**Go Version:** 1.23+  
**Dependencies:**
- github.com/lib/pq - PostgreSQL driver
- github.com/segmentio/kafka-go - Kafka client
- pmeonline/pkg/ledger - Event sourcing framework
