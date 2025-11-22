# eClear API Service

The eClear API service is responsible for:
1. Receiving master data from eClear (Participants, Accounts, Instruments, Account Limits)
2. Receiving trade approvals and reimbursement instructions from eClear
3. Sending trade submissions to eClear for approval

## Architecture

```
┌─────────┐                  ┌──────────────┐                ┌───────┐
│ eClear  │ ──Master Data──> │  eClear API  │ ───Events──>   │ Kafka │
│ System  │ <──Trades─────── │   Service    │ <──Events───   │       │
└─────────┘                  └──────────────┘                └───────┘
```

## Endpoints

### Inbound (from eClear)

#### Master Data
- `POST /account/insert` - Insert/update accounts
- `POST /instrument/insert` - Insert/update instruments
- `POST /participant/insert` - Insert/update participants
- `POST /account/limit` - Update account trading limits

#### Trade Approvals
- `POST /contract/matched` - Confirm trade approval from eClear
- `POST /contract/reimburse` - Process reimbursement instruction
- `POST /lender/recall` - Process lender recall instruction

### Outbound (to eClear)

The service automatically sends trades to eClear when they are matched by the OMS.

Endpoint: `POST {ECLEAR_BASE_URL}/contract/matched`

### Health Check
- `GET /health` - Service health check

## Configuration

Environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `KAFKA_URL` | `localhost:9092` | Kafka broker URL |
| `KAFKA_TOPIC` | `pme-ledger` | Kafka topic name |
| `API_PORT` | `8081` | HTTP server port |
| `ECLEAR_BASE_URL` | `http://localhost:9000` | eClear system base URL |

## Running

### Prerequisites

1. Kafka must be running at `KAFKA_URL`
2. Kafka topic `pme-ledger` must exist

### Start the service

```bash
# Using default configuration
go run main.go

# Using custom configuration
KAFKA_URL=localhost:9092 \
KAFKA_TOPIC=pme-ledger \
API_PORT=8081 \
ECLEAR_BASE_URL=http://eclear.example.com \
go run main.go
```

## Testing

### 1. Test Master Data Insertion

#### Insert Participants
```bash
curl -X POST http://localhost:8081/participant/insert \
  -H "Content-Type: application/json" \
  -d '[
    {
      "code": "YU",
      "name": "Yuanta Securities",
      "borr_eligibility": true,
      "lend_eligibility": true
    },
    {
      "code": "AA",
      "name": "AA Securities",
      "borr_eligibility": true,
      "lend_eligibility": true
    }
  ]'
```

#### Insert Instruments
```bash
curl -X POST http://localhost:8081/instrument/insert \
  -H "Content-Type: application/json" \
  -d '[
    {
      "code": "BBRI",
      "name": "Bank Rakyat Indonesia",
      "status": true
    },
    {
      "code": "BBCA",
      "name": "Bank Central Asia",
      "status": true
    }
  ]'
```

#### Insert Accounts
```bash
curl -X POST http://localhost:8081/account/insert \
  -H "Content-Type: application/json" \
  -d '[
    {
      "code": "YU-012345",
      "name": "John Doe",
      "sid": "1234567890ABCDEF",
      "email": "john@example.com",
      "address": "Jakarta",
      "participant": "YU"
    },
    {
      "code": "AA-067890",
      "name": "Jane Smith",
      "sid": "ABCDEF1234567890",
      "email": "jane@example.com",
      "address": "Surabaya",
      "participant": "AA"
    }
  ]'
```

#### Update Account Limits
```bash
curl -X POST http://localhost:8081/account/limit \
  -H "Content-Type: application/json" \
  -d '[
    {
      "code": "YU-012345",
      "borr_limit": 1000000000.00,
      "pool_limit": 500000000.00
    },
    {
      "code": "AA-067890",
      "borr_limit": 2000000000.00,
      "pool_limit": 1000000000.00
    }
  ]'
```

### 2. Test Trade Approval

#### Confirm Trade Match
```bash
curl -X POST http://localhost:8081/contract/matched \
  -H "Content-Type: application/json" \
  -d '{
    "pme_trade_reff": "TRADE-20251122-001",
    "state": "OK",
    "borr_contract_reff": "CONTRACT-BORR-001",
    "lend_contract_reff": "CONTRACT-LEND-001",
    "open_time": "2025-11-22T09:30:00Z"
  }'
```

#### Process Reimbursement
```bash
curl -X POST http://localhost:8081/contract/reimburse \
  -H "Content-Type: application/json" \
  -d '{
    "pme_trade_reff": "TRADE-20251122-001",
    "state": "REIM",
    "borr_contract_reff": "CONTRACT-BORR-001",
    "lend_contract_reff": "CONTRACT-LEND-001",
    "close_time": "2025-12-22T09:30:00Z"
  }'
```

#### Process Reimbursement with ARO
```bash
curl -X POST http://localhost:8081/contract/reimburse \
  -H "Content-Type: application/json" \
  -d '{
    "pme_trade_reff": "TRADE-20251122-001",
    "state": "ARO",
    "borr_contract_reff": "CONTRACT-BORR-001",
    "lend_contract_reff": "CONTRACT-LEND-001",
    "close_time": "2025-12-22T09:30:00Z"
  }'
```

### 3. Test Lender Recall
```bash
curl -X POST http://localhost:8081/lender/recall \
  -H "Content-Type: application/json" \
  -d '{
    "contract_reff": "CONTRACT-LEND-001",
    "kpei_reff": "KPEI-12345"
  }'
```

### 4. Health Check
```bash
curl http://localhost:8081/health
```

## Event Flow

### Master Data Initialization

```
eClear → POST /participant/insert → Kafka (Participant events)
      → POST /instrument/insert  → Kafka (Instrument events)
      → POST /account/insert     → Kafka (Account events)
      → POST /account/limit      → Kafka (AccountLimit events)
```

### Trade Approval Flow

```
OMS → Trade Matched → Kafka (Trade event)
                   → EClearClient listens
                   → POST to eClear /contract/matched
                   → Kafka (TradeWait event)

eClear → POST /contract/matched → Kafka (TradeAck event)
                                → Trade state: E → O (Open)
```

### Reimbursement Flow

```
eClear → POST /contract/reimburse → Kafka (TradeReimburse event)
                                  → Trade state: O → C (Closed)
                                  → If ARO: Create new Order
```

### Lender Recall Flow

```
eClear → POST /lender/recall → Kafka (Order event for re-matching)
                             → OMS will match with new lender
                             → Old contract terminated
                             → New contract created
```

## Monitoring

The service logs important events:

- `📥` Incoming requests from eClear
- `✅` Successful operations
- `❌` Errors and failures
- `⚠️` Warnings (e.g., missing data, ineligible instruments)
- `📤` Outbound requests to eClear

## Error Handling

The service handles various error scenarios:

1. **Invalid JSON**: Returns 400 Bad Request
2. **Missing required fields**: Logs warning and skips record
3. **Entity not found**: Returns 404 Not Found
4. **eClear timeout**: Commits TradeWait event, marks for retry
5. **EOD pending trades**: Automatically drops trades not approved by EOD

## Integration with Other Services

### OMS (Order Management System)
- OMS creates Trade events when orders are matched
- EClearClient listens to Trade events and sends to eClear
- OMS processes TradeAck/TradeNak events from eClear

### APME API
- APME API creates Order events
- Uses master data from Kafka (via LedgerPoint sync)
- Displays trade/contract status to users

### Database Exporter
- Subscribes to all events and persists to PostgreSQL
- Maintains audit trail of all master data changes

## Production Considerations

1. **Retry Logic**: Implement exponential backoff for failed eClear requests
2. **Queue Management**: Use persistent queue for pending trade submissions
3. **Monitoring**: Set up alerts for failed trade submissions
4. **Rate Limiting**: Implement rate limiting for eClear API calls
5. **Security**: Add authentication/authorization for inbound endpoints
6. **High Availability**: Run multiple instances behind load balancer
7. **Circuit Breaker**: Implement circuit breaker pattern for eClear calls

## Troubleshooting

### Service won't start
- Check Kafka is running and accessible
- Verify Kafka topic exists
- Check port 8081 is not in use

### Master data not appearing
- Check Kafka consumer logs
- Verify JSON format matches expected structure
- Check for validation errors in logs

### Trade not sent to eClear
- Check eClear base URL is correct
- Verify network connectivity to eClear
- Check for missing account/instrument data
- Review EClearClient logs

### Trade stuck in Wait state
- Check eClear returned success response
- Verify eClear sent confirmation callback
- Check EOD job is running to drop expired trades
