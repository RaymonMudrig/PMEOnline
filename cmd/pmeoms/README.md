## OMS (Order Management System) Service

The OMS is the core matching engine for the PME Online system. It validates orders, performs risk management checks, matches borrowing and lending orders, and generates trades.

## Architecture

```
┌─────────┐     ┌──────────────┐     ┌───────┐
│ Order   │────>│     OMS      │────>│ Trade │
│ Events  │     │   Service    │     │Events │
└─────────┘     └──────────────┘     └───────┘
                       │
                       ├─ Risk Validator
                       ├─ Matching Engine
                       ├─ Order Books
                       └─ Trade Generator
```

## Components

### 1. Risk Management (`pkg/risk/`)

#### Validator
- Pre-trade validation for all orders
- Account existence and eligibility checks
- Instrument eligibility verification
- Participant eligibility verification
- Date and quantity validation
- **Borrowing-specific:** Trading limit validation
- **Lending-specific:** Basic validation only (no limit check)

#### Calculator
- Fee calculations (flat fee, borrowing fee, lending fee)
- Borrowing value calculation
- Daily and accumulated fee calculations
- Complete fee breakdown generation

#### Checker
- Monitors instrument eligibility changes
- Monitors participant eligibility changes
- Handles BlockProcess state for ineligible instruments
- Identifies orders that should be blocked

### 2. OMS Engine (`pkg/oms/`)

#### Order Book
- Maintains separate queues for borrow and lend orders per instrument
- Separates same-participant and cross-participant orders
- Implements priority sorting for matching

#### Matcher
- Implements matching algorithm from design spec (F.2)
- **For Borrowing orders:** Match with Lend orders
  - Priority: Same participant first
  - Sort: Quantity DESC (prefer larger lenders)
- **For Lending orders:** Match with Borrow orders
  - Priority: Same participant first
  - Sort: Time ASC (FIFO)
- Supports partial fills

#### Trade Generator
- Creates Trade and Contract entities from matches
- Calculates fees using risk calculator
- Generates unique trade references (KpeiReff)
- Creates borrower and lender contracts

## Matching Rules (F.2)

1. **Instrument:** Must match exactly
2. **Participant Priority:** In-house matching first (same participant)
3. **For Lending:** Prefer larger lenders (Quantity DESC)
4. **For Borrowing:** FIFO (Time ASC)

## Validation Rules (F.1)

### Borrowing Orders

**Formula:**
```
BorrVal = MarketPrice × Quantity
TotalFee = BorrVal × FeeBorr × Period / 365 + FeeFlat
TradingLimit >= TotalFee + BorrVal
```

**Checks:**
- Account exists and belongs to correct participant
- Instrument exists and is eligible
- Participant has borrowing eligibility
- Settlement date is in the future
- Reimbursement date > Settlement date
- Periode matches date range
- Quantity is multiple of denomination limit
- Quantity <= maximum quantity
- **Trading limit is sufficient**

### Lending Orders

**Checks:**
- Account exists and belongs to correct participant
- Instrument exists and is eligible
- Participant has lending eligibility
- Basic field validation
- **No pool limit check** (per design F.1.2)

## Fee Calculation (F.3)

### Static Rates
- Flat Fee: 0.05% (one-time, borrower only)
- Borrowing Fee: 18% annual
- Lending Fee: 15% annual

### Formulas

**Borrower:**
```
FeeFlat = MarketPrice × Quantity × 0.0005
FeeBorrDaily = MarketPrice × Quantity × 0.18 / 365
FeeBorrAccum = FeeBorrDaily × DaysPassed
```

**Lender:**
```
FeeLendDaily = MarketPrice × Quantity × 0.15 / 365
FeeLendAccum = FeeLendDaily × DaysPassed
```

## Order States

```
S (Saved) → O (Open) → P (Partial) → M (Matched)
S → R (Rejected)
O/P → W (Withdrawn)
O/P → B (BlockProcess) → O/P
```

## Trade States

```
S (Submitted) → E (Approval/Wait) → O (Open) → C (Closed)
S/E → R (Rejected)
```

## Configuration

Environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `KAFKA_URL` | `localhost:9092` | Kafka broker URL |
| `KAFKA_TOPIC` | `pme-ledger` | Kafka topic name |

## Running

### Prerequisites

1. Kafka must be running
2. Kafka topic `pme-ledger` must exist
3. eClear API service should be running (for master data)

### Start the service

```bash
# Using default configuration
cd cmd/pmeoms
go run main.go

# Using custom configuration
KAFKA_URL=localhost:9092 \
KAFKA_TOPIC=pme-ledger \
go run main.go

# Or use Make
make run-pmeoms
```

## Event Flow

### Order Processing

```
1. Order event received from Kafka
   ↓
2. Risk validation
   ├─ Valid → OrderAck
   └─ Invalid → OrderNak
   ↓
3. Check settlement date
   ├─ Future → Pending-New state
   └─ Today → Continue
   ↓
4. Attempt matching
   ├─ Fully matched → Generate Trade(s)
   ├─ Partially matched → Generate Trade(s) + Queue remainder
   └─ No match → Queue order
   ↓
5. Trade(s) committed to Kafka
   ↓
6. eClear API sends to eClear for approval
   ↓
7. TradeAck/TradeNak received
```

### Withdrawal Processing

```
1. OrderWithdraw event received
   ↓
2. Check order state (must be O or P)
   ├─ Valid → Remove from order book
   └─ Invalid → OrderWithdrawNak
   ↓
3. OrderWithdrawAck committed
```

### Instrument Eligibility Change

```
1. Instrument event received (Status = false)
   ↓
2. Mark instrument as ineligible
   ↓
3. Block matching for this instrument
   ↓
4. Find all open orders for instrument
   ↓
5. Mark orders as BlockProcess state
   ↓
6. When Status = true again
   ↓
7. Restore orders to previous state
   ↓
8. Resume matching
```

## Testing

### 1. Start Infrastructure

```bash
# Start Kafka and PostgreSQL
make setup

# Start eClear API (for master data)
make run-eclearapi

# In another terminal, populate master data
cd cmd/eclearapi
./test.sh
```

### 2. Start OMS

```bash
# In another terminal
make run-pmeoms
```

### 3. Submit Test Orders

You can submit orders by creating Order events. For testing, you can use the Kafka console producer:

```bash
docker exec -it pme-kafka kafka-console-producer.sh \
  --bootstrap-server localhost:9092 \
  --topic pme-ledger \
  --property "parse.key=true" \
  --property "key.separator=:"
```

Then send JSON like:
```json
ledgerpoint:{"nid":1001,"reff_request_id":"TEST-001","account_nid":1,"account_code":"YU-012345","participant_nid":1,"participant_code":"YU","instrument_nid":1,"instrument_code":"BBRI","side":"BORR","quantity":1000,"settlement_date":"2025-11-23T00:00:00Z","reimbursement_date":"2025-12-23T00:00:00Z","periode":30,"market_price":5000,"rate":0.18,"aro":false}
```

### 4. Monitor Logs

Watch the OMS logs for:
- Order validation
- Matching attempts
- Trade generation
- Statistics updates

## Integration with Other Services

### eClear API
- Receives master data (Participants, Accounts, Instruments, Limits)
- Sends trades to eClear for approval
- Receives trade approvals (TradeAck/TradeNak)

### APME API (when implemented)
- Receives orders from clients
- Displays order status
- Shows SBL data from order books
- Notifies clients of matches

### Database Exporter (when implemented)
- Persists all orders, trades, and contracts
- Maintains audit trail

## Monitoring

The OMS logs important events with emoji indicators:

- `📥` Incoming events
- `✅` Successful operations
- `❌` Errors and rejections
- `⚠️` Warnings (e.g., eligibility changes)
- `🔄` Matching operations
- `📝` Trade generation
- `📊` Statistics
- `🎯` Full matches
- `⚡` Partial matches
- `📋` Orders queued

Statistics are logged every 30 seconds showing:
- Total instruments with orders
- Order counts per instrument (borrow/lend)

## Troubleshooting

### Orders are rejected

Check logs for validation errors:
- Account exists?
- Instrument eligible?
- Participant eligible?
- Trading limit sufficient? (for borrowing)
- Quantity valid?
- Dates valid?

### Orders not matching

- Check if instrument is eligible
- Verify there are orders on opposite side
- Check settlement dates match
- Review matching rules (same participant priority)

### Trades not being sent to eClear

- Ensure eClear API service is running
- Check eClear API logs for errors
- Verify network connectivity

## Performance Considerations

- Order books use read-write locks for concurrent access
- In-memory data structures for fast matching
- Event-driven architecture for scalability
- Stateless matching (can run multiple instances)

## Future Enhancements

1. Pending order scheduler (for future settlement dates)
2. BlockProcess state implementation
3. ARO (Auto Roll-Over) order processing
4. Lender recall matching
5. Settlement date triggers
6. Market price updates
7. EOD processing
8. Performance metrics and monitoring
