# EClearAPI - eClear Integration Service

## Overview

EClearAPI is the bi-directional integration service between the PME platform and the external eClear clearing house system. It handles:
- **Outbound**: Sending matched trades to eClear for approval
- **Inbound**: Receiving master data, trade approvals, and settlement notifications from eClear

## Architecture

```
┌──────────────────────────────────────────────────────────────┐
│                        EClearAPI                             │
│                                                              │
│  ┌─────────────────┐           ┌──────────────────────────┐  │
│  │  HTTP Handlers  │           │    EClearClient          │  │
│  │   (Inbound)     │           │    (Outbound)            │  │
│  │                 │           │                          │  │
│  │  • MasterData   │           │  • SyncHandler           │  │
│  │  • Trade        │           │  • SendTrade()           │  │
│  │  • Query        │           │  • CheckPendingTrades()  │  │
│  │  • Settings     │           │                          │  │
│  └────────┬────────┘           └────────┬─────────────────┘  │
│           │                             │                    │
│           │                             │                    │
│           └──────────┬──────────────────┘                    │
│                      │                                       │
│               ┌──────▼──────┐                                │
│               │ LedgerPoint │                                │
│               └──────┬──────┘                                │
│                      │                                       │
└──────────────────────┼───────────────────────────────────────┘
                       │
                       ▼
                ┌─────────────┐           ┌──────────────┐
                │    Kafka    │◄─────────►│    eClear    │
                │ "pme-ledger"│           │   System     │
                └─────────────┘           └──────────────┘
```

## Components

### 1. EClearClient (`internal/eclearapi/handler/eclear_client.go`)

Manages outbound communication to eClear system.

**Responsibilities:**
- Subscribe to Trade events from LedgerPoint
- Send matched trades to eClear for approval
- Handle trade approval/rejection responses
- Check for trades pending approval at EOD

**Key Methods:**
- `SendTrade(trade)` - POST trade to eClear endpoint
- `CheckPendingTrades()` - NAK trades not approved by EOD
- `GetSyncHandler()` - Return subscriber for LedgerPoint

**Trade Submission Flow:**
```
1. Receive Trade event (state: M)
   │
2. Extract borrower & lender contracts
   │
3. Build TradeMatchedPayload with:
   │  - Trade details (instrument, quantity, period)
   │  - Borrower info (account, SID, fees)
   │  - Lender info (account, SID, fees)
   │  - Timestamps (matched_at, reimburse_at)
   │
4. POST to eClear: /contract/matched
   │
   ├─► Success (200 OK) ──────► TradeWait event
   └─► Failure (non-200) ─────► TradeWait event (retry)
```

**Payload Format:**
```json
{
  "pme_trade_reff": "KPEI-20251129-0001",
  "instrument_code": "BBRI",
  "quantity": 1000,
  "periode": 7,
  "aro_status": false,
  "fee_flat_rate": 0.001,
  "fee_borr_rate": 0.0005,
  "fee_lend_rate": 0.0004,
  "matched_at": "2025-11-29 10:00:00",
  "reimburse_at": "2025-12-06 10:00:00",
  "lender": {
    "pme_contract_reff": "CONTRACT-L-001",
    "account_code": "ACC002",
    "sid": "SID002",
    "participant_code": "PART02",
    "fee_lender": 12600.00
  },
  "borrower": {
    "pme_contract_reff": "CONTRACT-B-001",
    "account_code": "ACC001",
    "sid": "SID001",
    "participant_code": "PART01",
    "fee_flat": 4500.00,
    "fee_borrower": 15750.00
  }
}
```

### 2. MasterDataHandler (`internal/eclearapi/handler/masterdata.go`)

Receives master data from eClear and publishes to Kafka.

**Endpoints:**

#### Insert Accounts
```http
POST /account/insert
Content-Type: application/json

{
  "accounts": [
    {
      "code": "ACC001",
      "participant_code": "PART01",
      "sid": "SID001"
    }
  ]
}
```

Publishes `Account` events to Kafka.

#### Insert Instruments
```http
POST /instrument/insert
Content-Type: application/json

{
  "instruments": [
    {
      "code": "BBRI",
      "name": "Bank BRI",
      "status": true  // eligible
    }
  ]
}
```

Publishes `Instrument` events to Kafka.

#### Insert Participants
```http
POST /participant/insert
Content-Type: application/json

{
  "participants": [
    {
      "code": "PART01",
      "name": "Participant 1"
    }
  ]
}
```

Publishes `Participant` events to Kafka.

#### Update Account Limit
```http
POST /account/limit
Content-Type: application/json

{
  "code": "ACC001",
  "trade_limit": 1000000000,
  "pool_limit": 5000000000
}
```

Publishes `AccountLimit` event to Kafka.

### 3. TradeHandler (`internal/eclearapi/handler/trade.go`)

Receives trade lifecycle events from eClear.

**Endpoints:**

#### Trade Approval/Rejection
```http
POST /contract/matched
Content-Type: application/json

{
  "pme_trade_reff": "KPEI-20251129-0001",
  "status": "approved"  // or "rejected"
  "message": "Approved by eClear"  // optional
}
```

Publishes:
- `TradeAck` if status = "approved"
- `TradeNak` if status = "rejected"

#### Trade Reimbursement
```http
POST /contract/reimburse
Content-Type: application/json

{
  "pme_trade_reff": "KPEI-20251129-0001"
}
```

Publishes `TradeReimburse` event (contract settlement).

#### Lender Recall
```http
POST /lender/recall
Content-Type: application/json

{
  "pme_trade_reff": "KPEI-20251129-0001",
  "recall_date": "2025-12-01"
}
```

Early termination requested by lender.

### 4. QueryHandler (`internal/eclearapi/handler/query.go`)

Provides read-only access for eClear dashboard.

**Endpoints:**
- `GET /participant/list` - List all participants
- `GET /instrument/list` - List all instruments
- `GET /account/list` - List all accounts

### 5. SettingsHandler (`internal/eclearapi/handler/settings.go`)

Manages system parameters and configuration.

**Endpoints:**

#### Get/Update Parameters
```http
GET /parameter

Response:
{
  "fee_flat_rate": 0.001,
  "fee_borr_rate": 0.0005,
  "fee_lend_rate": 0.0004,
  "auto_match_flag": true
}

POST /parameter/update
Content-Type: application/json

{
  "fee_flat_rate": 0.0015
}
```

#### Holiday Management
```http
GET /holiday/list

POST /holiday/add
{
  "date": "2025-12-25",
  "description": "Christmas"
}
```

#### Session Time
```http
GET /sessiontime

POST /sessiontime/update
{
  "pre_opening_time": "08:00:00",
  "opening_time": "09:00:00",
  "closing_time": "16:00:00"
}
```

## Event Flow

### Outbound: Trade Submission

```
Trade Matched (PME)
   │
   ▼
EClearClient.SyncTrade()
   │
   ▼
Build TradeMatchedPayload
   │
   ├─► Lookup borrower contract
   ├─► Lookup lender contract
   ├─► Lookup account SIDs
   └─► Calculate fees
   │
   ▼
POST /contract/matched (eClear)
   │
   ├─► 200 OK ──────────► TradeWait (state: M → E)
   │
   └─► Error ───────────► TradeWait (state: M → E) + Log error
```

### Inbound: Trade Approval

```
eClear Decision
   │
   ▼
POST /contract/matched (EClearAPI)
   │
   ├─► status = "approved" ───► TradeAck (state: E → M)
   │
   └─► status = "rejected" ───► TradeNak (state: E → R)
   │
   ▼
Kafka Event Published
   │
   ▼
All Services Updated
```

### EOD Cleanup

```
End of Day
   │
   ▼
EClearClient.CheckPendingTrades()
   │
   ▼
Find all trades in state "E" (Approval/Wait)
   │
   ▼
For each trade:
   │
   ├─► matched_at > 24 hours ago?
   │
   └─► YES ──────► TradeNak (timeout)
```

## Trade States

### State Flow

```
M (Matched)
   │
   ├─► Sent to eClear ──────────► E (Approval/Wait)
   │                                    │
   │                                    ├─► Approved ──► M (Matched)
   │                                    │
   │                                    ├─► Rejected ──► R (Rejected)
   │                                    │
   │                                    └─► Timeout ───► R (Rejected)
   │
   └─► Reimbursed ──────────────► C (Closed)
```

### State Meanings

- **M (Matched)** - Trade created, active
- **E (Approval)** - Waiting for eClear approval
- **R (Rejected)** - eClear rejected or timeout
- **C (Closed)** - Trade settled/reimbursed

## Configuration

### Environment Variables

```bash
KAFKA_URL=localhost:9092      # Kafka broker
KAFKA_TOPIC=pme-ledger        # Kafka topic
API_PORT=8081                 # HTTP port
ECLEAR_BASE_URL=http://localhost:9000  # eClear system URL
```

### eClear Endpoints (External)

EClearAPI calls these eClear endpoints:
- `POST /contract/matched` - Submit trade for approval

## Startup Sequence

```
1. Create LedgerPoint
   │
2. Create EClearClient
   │
3. Get SyncHandler from EClearClient
   │
4. Subscribe to LedgerPoint events
   │
5. Start LedgerPoint
   │
6. Wait for IsReady
   │
7. Start EClearClient processing
   │
8. Create HTTP handlers
   │
9. Setup HTTP routes
   │
10. Start HTTP server
   │
11. Service ready
```

## Monitoring

### Log Patterns

**Outbound (to eClear):**
```
📤 Sending trade to eClear: KPEI-20251129-0001
✅ Trade sent to eClear successfully: KPEI-20251129-0001
❌ Failed to send trade to eClear: connection refused
```

**Inbound (from eClear):**
```
📨 POST /contract/matched
✅ Trade approved: KPEI-20251129-0001
❌ Trade rejected: KPEI-20251129-0001
📨 POST /account/insert
✅ Inserted 10 accounts
```

**EOD Cleanup:**
```
🔍 Checking for pending trades at EOD...
⚠️  Trade KPEI-20251129-0001 not approved by EOD, dropping trade
✅ Pending trades check completed
```

## Error Handling

### Retry Logic

Currently NO automatic retry:
- Failed submissions log error and publish TradeWait
- Manual intervention required for failed submissions

**Future Enhancement:**
- Implement retry queue
- Exponential backoff
- Dead letter queue
- Alert on persistent failures

### Timeout Handling

Trades waiting for approval > 24 hours:
- Automatically NAK'd at EOD
- Prevents indefinite pending state
- Message: "Trade not approved by eClear by EOD"

## Security

**Current Implementation:**
- No authentication on inbound endpoints
- No authorization checks
- HTTP (not HTTPS)

**Production Requirements:**
- API key authentication from eClear
- TLS/HTTPS for all communication
- IP whitelist for eClear endpoints
- Request signature verification
- Audit logging

## Dashboard

Static HTML dashboard served at `/` and `/dashboard`:

**Features:**
- View participants, instruments, accounts
- Update system parameters
- Manage holidays
- Update session times
- Real-time statistics

**Static Files:**
```
web/static/eclearapi/
├── index.html
├── css/
│   └── style.css
└── js/
    └── app.js
```

## Testing

### Test eClear Integration

```bash
# Start EClearAPI
./bin/eclearapi

# Simulate eClear sending account data
curl -X POST http://localhost:8081/account/insert \
  -H "Content-Type: application/json" \
  -d '{
    "accounts": [{
      "code": "ACC001",
      "participant_code": "PART01",
      "sid": "SID001"
    }]
  }'

# Simulate eClear approving a trade
curl -X POST http://localhost:8081/contract/matched \
  -H "Content-Type: application/json" \
  -d '{
    "pme_trade_reff": "KPEI-20251129-0001",
    "status": "approved"
  }'
```

### Mock eClear Server

For testing outbound calls, run a mock eClear server:

```go
// mock_eclear.go
http.HandleFunc("/contract/matched", func(w http.ResponseWriter, r *http.Request) {
    log.Println("Received trade from PME")
    w.WriteHeader(http.StatusOK)
})
http.ListenAndServe(":9000", nil)
```

## Integration Patterns

### Pattern 1: Real-time Approval

```
Trade Matched → Send to eClear → Immediate Response → Publish Ack/Nak
```

Fastest path, requires eClear synchronous API.

### Pattern 2: Async Approval (Current)

```
Trade Matched → Send to eClear → TradeWait
                                     │
eClear processes async ──────────────┘
                                     │
eClear calls back ──────────► TradeAck/Nak
```

Allows eClear to process asynchronously.

### Pattern 3: Polling

```
Trade Matched → Send to eClear → TradeWait
                                     │
Poll eClear status every 30s ────────┤
                                     │
Status change ──────────────► TradeAck/Nak
```

Alternative when eClear doesn't support callbacks.

## Future Enhancements

- Automatic retry with exponential backoff
- Circuit breaker for eClear connectivity
- Message queue for reliable delivery
- Idempotency keys for duplicate prevention
- Webhook support for callbacks
- Batch operations for master data
- Real-time metrics dashboard
- Integration tests with mock eClear
- Support multiple eClear endpoints (failover)
