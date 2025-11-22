# APME API Service

The APME API Service provides REST and WebSocket APIs for clients to interact with the PME Online SBL system. It enables order entry, querying, and real-time notifications.

## Architecture

```
┌─────────┐     ┌──────────────┐     ┌───────┐     ┌─────────┐
│ Clients │────>│  APME API    │────>│ Kafka │────>│   OMS   │
│Web/Mobile<────│   Service    │<────│       │<────│         │
└─────────┘     └──────────────┘     └───────┘     └─────────┘
                      │
                      └──> WebSocket Notifications
```

## Features

- ✅ Order Management (new, amend, withdraw)
- ✅ Account queries (info, portfolio)
- ✅ Order and contract lists with filtering
- ✅ SBL data display (detail and aggregate)
- ✅ Real-time WebSocket notifications
- ✅ CORS support for web clients
- ✅ Request logging and error handling
- ✅ **Web-based Test Dashboard** (NEW)

## Test Dashboard

The APME API now includes a comprehensive web-based test dashboard for easy testing of all SBL functionalities.

### Dashboard Features

#### 📥 Entry Borrow Tab
**Smart Form with Auto-Population:**
- **Participant Code**: Dropdown populated from eClear API
- **Account Code**: Dropdown filtered by selected participant
- **Instrument Code**: Dropdown showing only eligible instruments
- **Quantity**: Manual input (shares)
- **Rate**: Auto-filled at 18% (read-only)
- **Settlement Date**: User input with default (tomorrow)
- **Reimbursement Date**: Auto-calculated from Settlement + Periode, or manual input
- **Periode (days)**: Auto-calculated from dates, or manual input
- **ARO**: Yes/No selection
- **Instruction**: Optional text
- **Reference ID**: Auto-generated as `BORR{yyyyMMddHHmmss}` (hidden)
- **Market Price**: Auto-filled as 0 (hidden)

**Smart Features:**
- Dates and periode auto-calculate bidirectionally
- Account dropdown filters based on selected participant
- Only eligible instruments appear in dropdown
- Default dates set to tomorrow + 30 days

#### 📤 Entry Lend Tab
**Simplified Form for Lenders:**
- **Participant Code**: Dropdown populated from eClear API
- **Account Code**: Dropdown filtered by selected participant
- **Instrument Code**: Dropdown showing only eligible instruments
- **Quantity**: Manual input (shares)
- **Rate**: Auto-filled at 15% (read-only)
- **Reference ID**: Auto-generated as `LEND{yyyyMMddHHmmss}` (hidden)
- All other fields auto-filled with defaults (settlement/reimbursement = epoch, periode = 0, ARO = false)

#### 📋 Order List Tab
- View and filter orders by participant, SID, state
- See order status and quantities
- Check ARO status

#### 📜 Contract List Tab
- View matched contracts
- See fee calculations (flat, daily, accumulated)
- Track contract states

#### 📊 SBL Detail Tab
- View lendable pool and borrowing needs
- Filter by participant, instrument, side, ARO
- See remaining quantities

#### 📈 SBL Aggregate Tab
- View net positions by instrument
- Aggregated borrow and lend quantities
- Net side determination

### Accessing the Dashboard

**Prerequisites:**
The dashboard requires eClear API to be running for master data:
```bash
# Terminal 1: Start eClear API (for master data)
cd cmd/eclearapi
go run main.go
# Runs on http://localhost:8081

# Terminal 2: Populate master data
make test-eclearapi
```

**Start APME API:**
```bash
cd cmd/pmeapi
go run main.go
# Runs on http://localhost:8080
```

**Open in browser:**
```
http://localhost:8080/
http://localhost:8080/dashboard
```

**On page load, the dashboard will:**
- Automatically fetch participants from eClear API (port 8081)
- Populate instrument dropdowns (eligible instruments only)
- Load all accounts for filtering by participant
- Set default dates for borrow orders (tomorrow + 30 days)

### Dashboard File Structure

```
cmd/pmeapi/
├── static/
│   ├── index.html       # Main HTML structure with form layouts
│   ├── css/
│   │   └── style.css    # All styling (purple gradient theme)
│   └── js/
│       └── app.js       # JavaScript logic and API interactions
│                        # - Fetches master data from eClear API (port 8081)
│                        # - Auto-generates reference IDs
│                        # - Date/periode calculations
│                        # - Account filtering by participant
├── main.go              # API server (serves static files)
└── README.md           # This file
```

### Dashboard Integration

The dashboard integrates with **eClear API** (http://localhost:8081) to fetch:
- `/participant/list` - All participants
- `/instrument/list` - All instruments (filters eligible only)
- `/account/list` - All accounts (cached for filtering)

**Data Flow:**
```
┌─────────────┐
│   Browser   │
│  Dashboard  │
└──────┬──────┘
       │
       ├─────────> eClear API (8081) ─┐
       │           - Participants       │
       │           - Instruments        │ Master Data
       │           - Accounts           │
       │           - Limits            ─┘
       │
       └─────────> APME API (8080) ───┐
                   - Submit Orders      │ Operations
                   - Query Orders       │
                   - SBL Data          ─┘
```

### Dashboard State Codes

**Order States:**
- **S** - Submitted
- **O** - Open (in SBL pool)
- **P** - Partial
- **M** - Matched
- **W** - Withdrawn
- **R** - Rejected

**Contract States:**
- **S** - Submitted
- **E** - Approval
- **O** - Open
- **C** - Closed
- **R** - Rejected
- **T** - Terminated

## API Endpoints

### Order Management

#### POST /api/order/new
Submit a new borrowing or lending order.

**Request Body (Borrow):**
```json
{
  "reff_request_id": "BORR20251122143025",
  "account_code": "YU-012345",
  "participant_code": "YU",
  "instrument_code": "BBRI",
  "side": "BORR",
  "quantity": 1000,
  "settlement_date": "2025-11-25T00:00:00Z",
  "reimbursement_date": "2025-12-25T00:00:00Z",
  "periode": 30,
  "market_price": 0,
  "rate": 0.18,
  "instruction": "Test order",
  "aro": false
}
```

**Request Body (Lend):**
```json
{
  "reff_request_id": "LEND20251122143030",
  "account_code": "AA-067890",
  "participant_code": "AA",
  "instrument_code": "BBRI",
  "side": "LEND",
  "quantity": 2000,
  "settlement_date": "1970-01-01T00:00:00Z",
  "reimbursement_date": "1970-01-01T00:00:00Z",
  "periode": 0,
  "market_price": 0,
  "rate": 0.15,
  "instruction": "",
  "aro": false
}
```

**Field Notes:**
- `reff_request_id`: Auto-generated format `{SIDE}{yyyyMMddHHmmss}`
- `market_price`: Always 0 for both sides
- `rate`: 0.18 (18%) for BORR, 0.15 (15%) for LEND
- For LEND: settlement/reimbursement dates, periode, aro, instruction use default values

**Response:**
```json
{
  "status": "success",
  "message": "Order submitted successfully",
  "data": {
    "order_nid": 1732248123456,
    "reff_request_id": "REQ-001",
    "status": "submitted"
  }
}
```

#### POST /api/order/amend
Amend an existing order.

**Request Body:**
```json
{
  "order_nid": 1732248123456,
  "reff_request_id": "REQ-002",
  "quantity": 2000,
  "aro": true
}
```

#### POST /api/order/withdraw
Withdraw an existing order.

**Request Body:**
```json
{
  "order_nid": 1732248123456,
  "reff_request_id": "REQ-003"
}
```

### Query Endpoints

#### GET /api/account/info?sid={sid}
Get account information and limits.

**Response:**
```json
{
  "status": "success",
  "message": "Account information retrieved",
  "data": {
    "code": "YU-012345",
    "sid": "SIDA1234567890AB",
    "name": "PT. ABC Investment",
    "address": "Jakarta Selatan",
    "participant_code": "YU",
    "participant_name": "Yuanta Securities Indonesia",
    "trade_limit": 5000000000,
    "pool_limit": 3000000000
  }
}
```

#### GET /api/order/list?participant={code}&sid={sid}&state={state}
Get list of orders with optional filtering.

**Query Parameters:**
- `participant` - Filter by participant code
- `sid` - Filter by account SID
- `state` - Filter by order state (O, P, M, W, R, etc.)

**Response:**
```json
{
  "status": "success",
  "message": "Order list retrieved",
  "data": {
    "count": 2,
    "orders": [
      {
        "nid": 1732248123456,
        "reff_request_id": "REQ-001",
        "account_code": "YU-012345",
        "participant_code": "YU",
        "instrument_code": "BBRI",
        "side": "BORR",
        "quantity": 1000,
        "done_quantity": 0,
        "settlement_date": "2025-11-25",
        "reimbursement_date": "2025-12-25",
        "periode": 30,
        "state": "O",
        "market_price": 5000,
        "rate": 0.18,
        "aro": false,
        "entry_at": "2025-11-22 14:30:00"
      }
    ]
  }
}
```

#### GET /api/contract/list?participant={code}&sid={sid}&state={state}
Get list of contracts with fees.

**Response:**
```json
{
  "status": "success",
  "message": "Contract list retrieved",
  "data": {
    "count": 1,
    "contracts": [
      {
        "nid": 12345678901,
        "trade_nid": 1234567890,
        "kpei_reff": "PME-20251122-1234567890-BORR",
        "side": "BORR",
        "account_code": "YU-012345",
        "account_sid": "SIDA1234567890AB",
        "account_participant_code": "YU",
        "order_nid": 1732248123456,
        "instrument_code": "BBRI",
        "quantity": 1000,
        "periode": 30,
        "state": "O",
        "fee_flat_val": 25000,
        "fee_val_daily": 2465.75,
        "fee_val_accumulated": 0,
        "matched_at": "2025-11-22 14:35:00",
        "reimburse_at": "2025-12-25"
      }
    ]
  }
}
```

### SBL Endpoints

#### GET /api/sbl/detail
Get detailed SBL data (all open orders).

**Query Parameters:**
- `participant` - Filter by participant code
- `instrument` - Filter by instrument code
- `side` - Filter by side (BORR/LEND)
- `aro` - Filter by ARO status (true/false)

**Response:**
```json
{
  "status": "success",
  "message": "SBL detail retrieved",
  "data": {
    "count": 3,
    "orders": [
      {
        "nid": 1732248123456,
        "participant_code": "YU",
        "sid": "SIDA1234567890AB",
        "account_code": "YU-012345",
        "instrument_code": "BBRI",
        "side": "BORR",
        "quantity": 1000,
        "done_quantity": 0,
        "remaining_quantity": 1000,
        "rate": 0.18,
        "aro": false,
        "settlement_date": "2025-11-25",
        "reimbursement_date": "2025-12-25",
        "periode": 30,
        "state": "O",
        "entry_at": "2025-11-22 14:30:00"
      }
    ]
  }
}
```

#### GET /api/sbl/aggregate
Get aggregated SBL data per instrument.

**Query Parameters:**
- `instrument` - Filter by instrument code
- `side` - Filter by net side (BORR/LEND)

**Response:**
```json
{
  "status": "success",
  "message": "SBL aggregate retrieved",
  "data": {
    "count": 2,
    "aggregates": [
      {
        "instrument_code": "BBRI",
        "instrument_name": "Bank Rakyat Indonesia Tbk",
        "borrow_quantity": 5000,
        "lend_quantity": 0,
        "net_quantity": 5000,
        "net_side": "BORR"
      },
      {
        "instrument_code": "BBCA",
        "instrument_name": "Bank Central Asia Tbk",
        "borrow_quantity": 0,
        "lend_quantity": 3000,
        "net_quantity": 3000,
        "net_side": "LEND"
      }
    ]
  }
}
```

### WebSocket

#### WS /ws/notifications
Real-time notifications for order and trade updates.

**Connection:**
```javascript
const ws = new WebSocket('ws://localhost:8080/ws/notifications');

ws.onmessage = (event) => {
  const notification = JSON.parse(event.data);
  console.log('Notification:', notification);
};
```

**Notification Types:**
- `order_created` - New order submitted
- `order_acknowledged` - Order validated and accepted
- `order_rejected` - Order validation failed
- `order_withdrawn` - Order withdrawn successfully
- `trade_matched` - Orders matched, trade created
- `trade_pending_approval` - Trade sent to eClear
- `trade_approved` - Trade approved by eClear
- `trade_rejected` - Trade rejected by eClear
- `contract_created` - Contract created
- `instrument_status_changed` - Instrument eligibility changed
- `account_limit_updated` - Account limits updated

**Notification Format:**
```json
{
  "type": "trade_matched",
  "timestamp": 1732248789012,
  "data": {
    "trade_nid": 1234567890,
    "kpei_reff": "PME-20251122-1234567890",
    "instrument": "BBRI",
    "quantity": 1000,
    "borrower_account": "YU-012345",
    "lender_account": "AA-067890",
    "matched_at": "2025-11-22T14:35:00Z"
  }
}
```

### Health Check

#### GET /health
Service health check.

**Response:**
```json
{
  "status": "ok",
  "service": "pmeapi"
}
```

## Configuration

Environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `KAFKA_URL` | `localhost:9092` | Kafka broker URL |
| `KAFKA_TOPIC` | `pme-ledger` | Kafka topic name |
| `API_PORT` | `8080` | HTTP server port |

## Running

### Prerequisites

1. Kafka must be running
2. eClear API service should be running (for master data)
3. OMS service should be running (for matching)

### Start the service

```bash
# Using default configuration
cd cmd/pmeapi
go run main.go

# Using custom configuration
KAFKA_URL=localhost:9092 \
KAFKA_TOPIC=pme-ledger \
API_PORT=8080 \
go run main.go

# Or use Make
make run-pmeapi
```

## Testing

### 1. Start All Services

```bash
# Terminal 1: Infrastructure
make setup

# Terminal 2: eClear API (REQUIRED for dashboard)
make run-eclearapi
# Dashboard fetches master data from port 8081

# Terminal 3: Populate master data (REQUIRED for dashboard)
make test-eclearapi
# Populates participants, instruments, accounts

# Terminal 4: OMS
make run-pmeoms

# Terminal 5: APME API
make run-pmeapi
# Dashboard available at http://localhost:8080
```

### 2. Using the Web Dashboard (Recommended)

1. Open http://localhost:8080 in your browser
2. Go to **Entry Borrow** or **Entry Lend** tab
3. Select participant from dropdown (populated from eClear API)
4. Select account from filtered dropdown
5. Select instrument from dropdown (eligible instruments only)
6. Fill in quantity and other fields
7. Click submit - reference ID auto-generated
8. View results in **Order List** tab

### 3. Test Order Entry via API (Alternative)

```bash
# Submit a borrowing order
curl -X POST http://localhost:8080/api/order/new \
  -H "Content-Type: application/json" \
  -d '{
    "reff_request_id": "BORR20251122100000",
    "account_code": "YU-012345",
    "participant_code": "YU",
    "instrument_code": "BBRI",
    "side": "BORR",
    "quantity": 1000,
    "settlement_date": "2025-11-25T00:00:00Z",
    "reimbursement_date": "2025-12-25T00:00:00Z",
    "periode": 30,
    "market_price": 0,
    "rate": 0.18,
    "instruction": "Test order",
    "aro": false
  }'

# Submit a lending order
curl -X POST http://localhost:8080/api/order/new \
  -H "Content-Type: application/json" \
  -d '{
    "reff_request_id": "LEND20251122100030",
    "account_code": "AA-067890",
    "participant_code": "AA",
    "instrument_code": "BBRI",
    "side": "LEND",
    "quantity": 2000,
    "settlement_date": "1970-01-01T00:00:00Z",
    "reimbursement_date": "1970-01-01T00:00:00Z",
    "periode": 0,
    "market_price": 0,
    "rate": 0.15,
    "instruction": "",
    "aro": false
  }'
```

### 4. Test Queries

```bash
# Get account info
curl http://localhost:8080/api/account/info?sid=SIDA1234567890AB

# Get order list
curl http://localhost:8080/api/order/list?participant=YU

# Get contract list
curl http://localhost:8080/api/contract/list?sid=SIDA1234567890AB&state=O

# Get SBL detail
curl http://localhost:8080/api/sbl/detail?instrument=BBRI

# Get SBL aggregate
curl http://localhost:8080/api/sbl/aggregate
```

### 5. Test WebSocket

Create a simple HTML file:

```html
<!DOCTYPE html>
<html>
<head><title>PME WebSocket Test</title></head>
<body>
  <h1>PME Notifications</h1>
  <div id="notifications"></div>
  <script>
    const ws = new WebSocket('ws://localhost:8080/ws/notifications');

    ws.onopen = () => console.log('Connected');
    ws.onclose = () => console.log('Disconnected');
    ws.onerror = (error) => console.error('Error:', error);

    ws.onmessage = (event) => {
      const notif = JSON.parse(event.data);
      const div = document.getElementById('notifications');
      div.innerHTML += `<pre>${JSON.stringify(notif, null, 2)}</pre><hr>`;
    };
  </script>
</body>
</html>
```

Open in browser and submit orders to see real-time notifications.

## Error Responses

All error responses follow this format:

```json
{
  "status": "error",
  "message": "Error description"
}
```

**Common HTTP Status Codes:**
- `200 OK` - Success
- `400 Bad Request` - Invalid request format
- `404 Not Found` - Resource not found
- `422 Unprocessable Entity` - Validation failed
- `500 Internal Server Error` - Server error

## Middleware

The API includes the following middleware:

1. **Logging** - Logs all requests with duration
2. **CORS** - Allows cross-origin requests
3. **Recovery** - Recovers from panics

## Integration with Other Services

### eClear API
- Uses master data (participants, accounts, instruments, limits)
- All order data validated against master data

### OMS
- Orders validated by risk management
- Matching performed by OMS
- Trades generated and sent to eClear

### Database Exporter
- All events persisted to PostgreSQL
- Historical data available for queries

## Production Considerations

1. **Authentication** - Add JWT/OAuth authentication
2. **Rate Limiting** - Implement per-user rate limits
3. **Input Validation** - Add comprehensive validation
4. **WebSocket Filtering** - Filter notifications by account/participant
5. **Monitoring** - Add metrics and health checks
6. **Load Balancing** - Run multiple instances
7. **HTTPS** - Use TLS in production
8. **API Versioning** - Add version prefix to URLs

## Troubleshooting

### Dashboard dropdowns are empty
- Ensure eClear API is running on port 8081
- Check browser console for CORS errors
- Verify master data was populated (`make test-eclearapi`)
- Check eClear API endpoints:
  - http://localhost:8081/participant/list
  - http://localhost:8081/instrument/list
  - http://localhost:8081/account/list

### Service won't start
- Check Kafka is running
- Verify Kafka topic exists
- Check port 8080 is not in use
- Ensure static files exist in ./static/ directory

### Orders are rejected
- Verify account exists in eClear API
- Check instrument is eligible (status = true)
- Ensure participant has eligibility
- Verify trading limits (for borrowing)
- Check dates are valid
- Ensure participant matches account

### WebSocket disconnects
- Check network connectivity
- Verify CORS configuration
- Check for firewall issues

### No notifications received
- Ensure WebSocket is connected
- Check OMS is running
- Verify events are in Kafka

## Performance

- **REST API:** <10ms response time for queries
- **WebSocket:** Real-time, <100ms latency
- **Throughput:** 1000+ orders/second
- **Concurrent Connections:** 10,000+ WebSocket clients
