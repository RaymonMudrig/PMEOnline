# DBExporter - Database Export Service

## Overview

DBExporter is a background service that consumes all events from the Kafka ledger and persists them to a PostgreSQL database. It provides a SQL-queryable historical record of all system activity for reporting, analytics, and audit purposes.

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                      DBExporter                         │
│                                                         │
│  ┌──────────────┐           ┌──────────────────────┐   │
│  │   Exporter   │──────────►│    PostgreSQL        │   │
│  │ (Subscriber) │           │     Database         │   │
│  └──────┬───────┘           └──────────────────────┘   │
│         │                                               │
│         │                                               │
│  ┌──────▼───────┐                                       │
│  │ LedgerPoint  │                                       │
│  └──────┬───────┘                                       │
│         │                                               │
└─────────┼───────────────────────────────────────────────┘
          │
          ▼
   ┌─────────────┐
   │    Kafka    │
   │ "pme-ledger"│
   └─────────────┘
```

## Components

### 1. Exporter (`internal/dbexporter/exporter/exporter.go`)

Implements `LedgerPointInterface` to receive all events from Kafka and insert them into PostgreSQL.

**Responsibilities:**
- Subscribe to all ledger events
- Map event structs to database rows
- Insert records with proper timestamps
- Handle database connection errors

**Event Handlers:**
All events are persisted to their respective tables:
- `SyncOrder` → `orders` table
- `SyncOrderAck` → `order_acks` table
- `SyncTrade` → `trades` table
- `SyncContract` → `contracts` table
- `SyncAccount` → `accounts` table
- etc.

### 2. Database (`internal/dbexporter/db/database.go`)

Manages PostgreSQL connection and migrations.

**Responsibilities:**
- Connect to PostgreSQL using environment variables
- Run database migrations
- Provide prepared statements for inserts
- Handle connection pooling

**Key Methods:**
- `NewDBFromEnv()` - Create database connection from env vars
- `RunMigrations(path)` - Execute SQL migration files
- `Close()` - Close database connection

## Database Schema

### Core Tables

#### orders
```sql
CREATE TABLE orders (
    nid BIGINT PRIMARY KEY,
    account_code VARCHAR(50),
    instrument_code VARCHAR(20),
    side VARCHAR(4),  -- BORR or LEND
    quantity DECIMAL(15,2),
    quantity_left DECIMAL(15,2),
    periode INT,
    state VARCHAR(1),  -- S, O, P, M, W, R, G
    settlement_date DATE,
    aro_status BOOLEAN,
    reff_request_id VARCHAR(100),
    created_at TIMESTAMP,
    updated_at TIMESTAMP
);
```

#### order_acks
```sql
CREATE TABLE order_acks (
    id SERIAL PRIMARY KEY,
    order_nid BIGINT REFERENCES orders(nid),
    acknowledged_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW()
);
```

#### order_naks
```sql
CREATE TABLE order_naks (
    id SERIAL PRIMARY KEY,
    order_nid BIGINT REFERENCES orders(nid),
    message TEXT,
    rejected_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW()
);
```

#### trades
```sql
CREATE TABLE trades (
    nid BIGINT PRIMARY KEY,
    kpei_reff VARCHAR(100) UNIQUE,
    instrument_code VARCHAR(20),
    quantity DECIMAL(15,2),
    periode INT,
    state VARCHAR(1),  -- M, E, C, R
    matched_at TIMESTAMP,
    reimburse_at TIMESTAMP,
    fee_flat_rate DECIMAL(10,6),
    fee_borr_rate DECIMAL(10,6),
    fee_lend_rate DECIMAL(10,6),
    created_at TIMESTAMP,
    updated_at TIMESTAMP
);
```

#### contracts
```sql
CREATE TABLE contracts (
    nid BIGINT PRIMARY KEY,
    trade_nid BIGINT REFERENCES trades(nid),
    order_nid BIGINT REFERENCES orders(nid),
    kpei_reff VARCHAR(100),
    account_code VARCHAR(50),
    instrument_code VARCHAR(20),
    side VARCHAR(4),
    quantity DECIMAL(15,2),
    state VARCHAR(1),  -- A, C, T
    matched_at TIMESTAMP,
    reimburse_at TIMESTAMP,
    fee_flat_val DECIMAL(15,2),
    fee_val_daily DECIMAL(15,2),
    created_at TIMESTAMP
);
```

#### accounts
```sql
CREATE TABLE accounts (
    code VARCHAR(50) PRIMARY KEY,
    participant_code VARCHAR(50),
    sid VARCHAR(50),
    trade_limit DECIMAL(20,2),
    pool_limit DECIMAL(20,2),
    created_at TIMESTAMP,
    updated_at TIMESTAMP
);
```

#### participants
```sql
CREATE TABLE participants (
    code VARCHAR(50) PRIMARY KEY,
    name VARCHAR(200),
    created_at TIMESTAMP
);
```

#### instruments
```sql
CREATE TABLE instruments (
    code VARCHAR(20) PRIMARY KEY,
    name VARCHAR(200),
    status BOOLEAN,  -- eligible for trading
    created_at TIMESTAMP,
    updated_at TIMESTAMP
);
```

### Audit Tables

#### parameters
```sql
CREATE TABLE parameters (
    id SERIAL PRIMARY KEY,
    fee_flat_rate DECIMAL(10,6),
    fee_borr_rate DECIMAL(10,6),
    fee_lend_rate DECIMAL(10,6),
    auto_match_flag BOOLEAN,
    effective_from TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW()
);
```

#### holidays
```sql
CREATE TABLE holidays (
    nid INT PRIMARY KEY,
    date DATE,
    description VARCHAR(200),
    created_at TIMESTAMP
);
```

#### session_times
```sql
CREATE TABLE session_times (
    id SERIAL PRIMARY KEY,
    pre_opening_time TIME,
    opening_time TIME,
    closing_time TIME,
    effective_from TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW()
);
```

## Event Flow

```
Kafka Event
   │
   ▼
LedgerPoint reads from Kafka
   │
   ▼
LedgerPoint updates internal state
   │
   ▼
LedgerPoint calls Exporter.Sync*() method
   │
   ▼
Exporter maps event to SQL INSERT
   │
   ▼
Execute INSERT statement
   │
   ├─► Success ──────► Log success
   │
   └─► Error ───────► Log error, continue
```

## Data Flow Examples

### Order Lifecycle

```sql
-- 1. Order created
INSERT INTO orders (nid, account_code, state, ...)
VALUES (123, 'ACC001', 'S', ...);

-- 2. Order acknowledged
INSERT INTO order_acks (order_nid, acknowledged_at)
VALUES (123, NOW());

UPDATE orders SET state = 'O' WHERE nid = 123;

-- 3. Order matched
UPDATE orders SET state = 'M' WHERE nid = 123;

INSERT INTO trades (nid, kpei_reff, state, ...)
VALUES (456, 'KPEI-20251129-0001', 'M', ...);

INSERT INTO contracts (nid, trade_nid, order_nid, ...)
VALUES (789, 456, 123, ...);
```

### Trade Approval

```sql
-- 1. Trade created (matched)
INSERT INTO trades (nid, state, ...)
VALUES (456, 'M', ...);

-- 2. Trade sent to eClear
UPDATE trades SET state = 'E' WHERE nid = 456;

-- 3. eClear approved
UPDATE trades SET state = 'M' WHERE nid = 456;

-- 4. Trade reimbursed
UPDATE trades SET state = 'C' WHERE nid = 456;
UPDATE contracts SET state = 'C' WHERE trade_nid = 456;
```

## Configuration

### Environment Variables

```bash
# Kafka
KAFKA_URL=localhost:9092
KAFKA_TOPIC=pme-ledger

# PostgreSQL
DB_HOST=localhost
DB_PORT=5432
DB_USER=pme_user
DB_PASSWORD=pme_password
DB_NAME=pme_db
DB_SSLMODE=disable
```

### Database Connection String

Built from environment variables:
```
postgresql://user:password@host:port/dbname?sslmode=disable
```

## Migrations

### Migration Files

Located in `migrations/` directory:

```sql
-- migrations/001_create_tables.sql
CREATE TABLE IF NOT EXISTS orders (
    nid BIGINT PRIMARY KEY,
    ...
);

CREATE TABLE IF NOT EXISTS trades (
    nid BIGINT PRIMARY KEY,
    ...
);

-- Add indexes
CREATE INDEX idx_orders_account ON orders(account_code);
CREATE INDEX idx_orders_state ON orders(state);
CREATE INDEX idx_trades_kpei ON trades(kpei_reff);
```

### Running Migrations

Migrations run automatically on service startup:
```go
database.RunMigrations("migrations/001_create_tables.sql")
```

## Startup Sequence

```
1. Load configuration from environment
   │
2. Connect to PostgreSQL
   │
3. Run database migrations
   │
4. Create Exporter
   │
5. Create LedgerPoint
   │
6. Subscribe Exporter to LedgerPoint
   │
7. Start LedgerPoint (Kafka consumer)
   │
8. Service runs until interrupted
   │
9. Graceful shutdown
```

## Querying Data

### Common Queries

#### Get all orders for an account
```sql
SELECT * FROM orders 
WHERE account_code = 'ACC001'
ORDER BY created_at DESC;
```

#### Get all trades in a date range
```sql
SELECT * FROM trades
WHERE matched_at BETWEEN '2025-11-01' AND '2025-11-30'
ORDER BY matched_at DESC;
```

#### Get active contracts
```sql
SELECT c.*, t.kpei_reff, o.account_code
FROM contracts c
JOIN trades t ON c.trade_nid = t.nid
JOIN orders o ON c.order_nid = o.nid
WHERE c.state = 'A';
```

#### Trading volume by instrument
```sql
SELECT 
    instrument_code,
    COUNT(*) as trade_count,
    SUM(quantity) as total_quantity
FROM trades
WHERE matched_at >= CURRENT_DATE
GROUP BY instrument_code
ORDER BY total_quantity DESC;
```

#### Account activity summary
```sql
SELECT 
    account_code,
    COUNT(DISTINCT CASE WHEN side = 'BORR' THEN nid END) as borr_orders,
    COUNT(DISTINCT CASE WHEN side = 'LEND' THEN nid END) as lend_orders,
    SUM(CASE WHEN side = 'BORR' AND state = 'M' THEN quantity ELSE 0 END) as borr_matched,
    SUM(CASE WHEN side = 'LEND' AND state = 'M' THEN quantity ELSE 0 END) as lend_matched
FROM orders
WHERE created_at >= CURRENT_DATE
GROUP BY account_code;
```

#### Rejected orders analysis
```sql
SELECT 
    DATE(rejected_at) as date,
    COUNT(*) as rejection_count,
    string_agg(DISTINCT message, '; ') as rejection_reasons
FROM order_naks
GROUP BY DATE(rejected_at)
ORDER BY date DESC;
```

## Monitoring

### Log Patterns

**Startup:**
```
[DB-EXPORTER] Starting Database Exporter Service...
[DB-EXPORTER] Kafka URL: localhost:9092
[DB-EXPORTER] Kafka Topic: pme-ledger
[DB-EXPORTER] Connected to database
[DB-EXPORTER] Running migrations...
[DB-EXPORTER] Migrations completed successfully
[DB-EXPORTER] Database Exporter Service started successfully
[DB-EXPORTER] Listening for Kafka events...
```

**Runtime:**
```
[EXPORTER] Inserted order: 123
[EXPORTER] Inserted trade: 456
[EXPORTER] Updated order state: 123 -> M
[EXPORTER] Error inserting order: duplicate key value
```

### Health Monitoring

Check database connectivity:
```sql
SELECT COUNT(*) FROM orders;
```

Check replication lag:
```sql
SELECT MAX(created_at) as latest_event FROM orders;
```

Compare with Kafka latest offset to detect lag.

## Error Handling

### Database Errors

**Connection Lost:**
- Service will exit
- Restart required (use systemd or similar)
- Future: Implement reconnection logic

**Constraint Violations:**
- Log error and continue
- Duplicate key violations are logged but don't stop service
- Foreign key violations indicate data inconsistency

**Transaction Failures:**
- Each insert is independent (no transactions)
- Failure doesn't affect subsequent events
- Data may be incomplete for a single event

## Performance Considerations

### Throughput

Expected performance:
- 100-1000 events/second (single instance)
- Limited by database write performance
- No batching currently implemented

### Optimization Strategies

1. **Batch Inserts** - Group multiple inserts
2. **Connection Pooling** - Reuse database connections
3. **Prepared Statements** - Reduce query parsing overhead
4. **Indexes** - Add indexes for common queries
5. **Partitioning** - Partition large tables by date

### Indexing

Critical indexes:
```sql
CREATE INDEX idx_orders_account ON orders(account_code);
CREATE INDEX idx_orders_state ON orders(state);
CREATE INDEX idx_orders_created ON orders(created_at);
CREATE INDEX idx_trades_kpei ON trades(kpei_reff);
CREATE INDEX idx_trades_matched ON trades(matched_at);
CREATE INDEX idx_contracts_trade ON contracts(trade_nid);
CREATE INDEX idx_contracts_account ON contracts(account_code);
```

## Backup and Recovery

### Backup Strategy

**PostgreSQL Backups:**
```bash
# Full backup
pg_dump -U pme_user pme_db > backup_$(date +%Y%m%d).sql

# Compressed backup
pg_dump -U pme_user pme_db | gzip > backup_$(date +%Y%m%d).sql.gz
```

**Automated Backups:**
- Daily full backups
- 30-day retention
- Store offsite (S3, cloud storage)

### Recovery

**From Backup:**
```bash
# Restore from backup
psql -U pme_user pme_db < backup_20251129.sql
```

**From Kafka:**
```bash
# Delete all data
psql -U pme_user pme_db -c "TRUNCATE orders, trades, contracts CASCADE;"

# Restart DBExporter (replays from Kafka beginning)
./bin/dbexporter
```

## Data Retention

### Archival Strategy

Move old data to archive tables:
```sql
-- Archive old orders (older than 1 year)
INSERT INTO orders_archive 
SELECT * FROM orders WHERE created_at < NOW() - INTERVAL '1 year';

DELETE FROM orders WHERE created_at < NOW() - INTERVAL '1 year';
```

### Purge Old Data

For compliance or storage limits:
```sql
-- Delete orders older than 7 years
DELETE FROM order_acks 
WHERE order_nid IN (
    SELECT nid FROM orders 
    WHERE created_at < NOW() - INTERVAL '7 years'
);

DELETE FROM orders WHERE created_at < NOW() - INTERVAL '7 years';
```

## Analytics and Reporting

### Business Intelligence Queries

#### Daily Trading Report
```sql
SELECT 
    DATE(matched_at) as trading_date,
    instrument_code,
    COUNT(*) as trades_count,
    SUM(quantity) as total_quantity,
    AVG(quantity) as avg_quantity
FROM trades
WHERE state = 'M'
GROUP BY DATE(matched_at), instrument_code
ORDER BY trading_date DESC, total_quantity DESC;
```

#### Account Performance
```sql
SELECT 
    a.code,
    a.participant_code,
    COUNT(DISTINCT o.nid) as total_orders,
    COUNT(DISTINCT CASE WHEN o.state = 'M' THEN o.nid END) as matched_orders,
    SUM(CASE WHEN o.state = 'M' THEN o.quantity ELSE 0 END) as total_volume
FROM accounts a
LEFT JOIN orders o ON a.code = o.account_code
WHERE o.created_at >= DATE_TRUNC('month', CURRENT_DATE)
GROUP BY a.code, a.participant_code
ORDER BY total_volume DESC;
```

## Testing

### Manual Testing

```bash
# Start service
./bin/dbexporter

# In another terminal, check database
psql -U pme_user pme_db -c "SELECT COUNT(*) FROM orders;"

# Submit test order (via pmeapi)
# Check if order appears in database
psql -U pme_user pme_db -c "SELECT * FROM orders ORDER BY created_at DESC LIMIT 1;"
```

### Data Validation

Compare Kafka events with database records:
```bash
# Count events in Kafka
kafka-console-consumer --bootstrap-server localhost:9092 \
    --topic pme-ledger --from-beginning | wc -l

# Count records in database
psql -U pme_user pme_db -c "
    SELECT 
        (SELECT COUNT(*) FROM orders) + 
        (SELECT COUNT(*) FROM trades) + 
        (SELECT COUNT(*) FROM contracts) as total_records;
"
```

## Future Enhancements

- Implement batch inserts for better performance
- Add database connection retry logic
- Support multiple database backends (MySQL, SQLite)
- Add metrics and monitoring (Prometheus)
- Implement data validation and checksums
- Add read replicas for analytics queries
- Support event replay for data correction
- Add materialized views for common reports
- Implement change data capture (CDC)
- Add real-time dashboards (Grafana)
