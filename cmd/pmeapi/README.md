# PMEAPI - API Service

## Overview

PMEAPI is the REST API and WebSocket gateway for the PME securities borrowing & lending platform. It provides HTTP endpoints for order submission, queries, and real-time WebSocket notifications for all system events.

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                          PMEAPI                             │
│                                                             │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐ │
│  │    HTTP      │    │  WebSocket   │    │  Notifier    │ │
│  │  Handlers    │    │     Hub      │◄───│  (Subscriber)│ │
│  └──────┬───────┘    └──────┬───────┘    └──────▲───────┘ │
│         │                   │                    │         │
│         │                   │                    │         │
│         └───────┬───────────┘                    │         │
│                 │                                │         │
│          ┌──────▼──────┐                        │         │
│          │ LedgerPoint │────────────────────────┘         │
│          └──────┬──────┘                                   │
│                 │                                          │
└─────────────────┼──────────────────────────────────────────┘
                  │
                  ▼
           ┌─────────────┐
           │    Kafka    │
           │ "pme-ledger"│
           └─────────────┘
```

## Components

### 1. HTTP Handlers

#### OrderHandler (`internal/pmeapi/handler/order.go`)

Handles order operations.

**Endpoints:**
- `POST /api/order/new` - Submit new order
- `POST /api/order/amend` - Amend existing order  
- `POST /api/order/withdraw` - Withdraw order

**Features:**
- Generates Snowflake IDs for orders
- Validates request payloads
- Publishes events to Kafka via LedgerPoint
- Returns order NID to caller

#### QueryHandler (`internal/pmeapi/handler/query.go`)

Provides read-only access to ledger state.

**Endpoints:**
- `GET /api/account/info` - Get account details with balance summary
- `GET /api/order/list` - List orders (with filters)
- `GET /api/contract/list` - List contracts (with filters)

**Query Parameters:**
- `account_code` - Filter by account
- `instrument_code` - Filter by instrument
- `side` - Filter by BORR/LEND
- `state` - Filter by state (S, O, M, W, R, etc.)

#### SBLHandler (`internal/pmeapi/handler/sbl.go`)

Securities Borrowing & Lending position queries.

**Endpoints:**
- `GET /api/sbl/detail` - Detailed SBL positions by account/instrument
- `GET /api/sbl/aggregate` - Aggregated SBL statistics

**Response:**
- Current borrowing/lending positions
- Available quantities
- Utilized amounts
- Settlement date breakdowns

### 2. WebSocket System

#### Hub (`internal/pmeapi/websocket/hub.go`)

Manages WebSocket connections and message broadcasting.

**Responsibilities:**
- Maintain active client connections
- Broadcast notifications to all clients
- Handle client registration/unregistration
- Manage notification buffer (DropCopy)

**Features:**
- **DropCopy Protocol** - Sequenced notifications with recovery
- **Buffer Size** - 10,000 notifications
- **Auto-Recovery** - Clients can request historical notifications

#### Client (`internal/pmeapi/websocket/client.go`)

Represents individual WebSocket connection.

**Protocol:**
```json
// Client subscribes
{
  "type": "subscribe",
  "from_seq": 0  // 0 = from oldest available
}

// Server sends recovery start
{
  "type": "recovery_start",
  "requested_seq": 0,
  "oldest_seq": 1,
  "latest_seq": 1000,
  "count": 1000
}

// Server sends notifications
{
  "seq": 1,
  "event_type": "order_created",
  "data": {
    "order_nid": 123,
    "account_code": "ACC001",
    "instrument": "BBRI",
    "side": "BORR",
    "quantity": 1000,
    "state": "S"
  }
}

// Server sends recovery complete
{
  "type": "recovery_complete",
  "count": 1000,
  "latest_seq": 1000
}
```

#### Notifier (`internal/pmeapi/websocket/notifier.go`)

Subscribes to ledger events and broadcasts to WebSocket clients.

**Event Subscriptions:**
- Order events (created, acknowledged, rejected, withdrawn, etc.)
- Trade events (matched, approved, rejected, reimbursed)
- Contract events (created)
- Account limit updates
- Instrument status changes
- Session events (SOD, EOD)

#### Buffer (`internal/pmeapi/websocket/buffer.go`)

Ring buffer storing last 10,000 notifications for recovery.

**Features:**
- Sequence numbering (1, 2, 3, ...)
- FIFO eviction when full
- Thread-safe access
- Gap detection

### 3. Middleware (`internal/pmeapi/middleware/middleware.go`)

HTTP middleware for cross-cutting concerns.

**Middleware:**
- **CORS** - Allow cross-origin requests
- **Logging** - Log all HTTP requests with timing

### 4. ID Generator (`pkg/idgen`)

Snowflake ID generator for distributed unique IDs.

**Features:**
- 64-bit IDs
- Time-based ordering
- Instance ID support (0-1023)
- Thread-safe

## API Endpoints

### Order Operations

#### Submit New Order
```http
POST /api/order/new
Content-Type: application/json

{
  "account_code": "ACC001",
  "instrument_code": "BBRI",
  "side": "BORR",           // or "LEND"
  "quantity": 1000,
  "periode": 7,             // days
  "settlement_date": "2025-11-29",
  "aro_status": false,
  "reff_request_id": "REQ-001"
}

Response:
{
  "order_nid": 123456789,
  "message": "Order submitted successfully"
}
```

#### Amend Order
```http
POST /api/order/amend
Content-Type: application/json

{
  "order_nid": 123456789,
  "new_quantity": 1500
}

Response:
{
  "message": "Order amended successfully"
}
```

#### Withdraw Order
```http
POST /api/order/withdraw
Content-Type: application/json

{
  "order_nid": 123456789
}

Response:
{
  "message": "Order withdrawal requested"
}
```

### Query Operations

#### Get Account Info
```http
GET /api/account/info?account_code=ACC001

Response:
{
  "account": {
    "code": "ACC001",
    "participant_code": "PART01",
    "sid": "SID001",
    "trade_limit": 1000000000,
    "pool_limit": 5000000000
  },
  "summary": {
    "open_borr_orders": 5,
    "open_lend_orders": 3,
    "active_borr_contracts": 10,
    "active_lend_contracts": 8,
    "total_borr_value": 50000000,
    "total_lend_value": 40000000
  }
}
```

#### List Orders
```http
GET /api/order/list?account_code=ACC001&state=O

Response:
{
  "orders": [
    {
      "nid": 123456789,
      "account_code": "ACC001",
      "instrument_code": "BBRI",
      "side": "BORR",
      "quantity": 1000,
      "quantity_left": 1000,
      "periode": 7,
      "state": "O",
      "settlement_date": "2025-11-29",
      "created_at": "2025-11-29T10:00:00Z"
    }
  ]
}
```

#### List Contracts
```http
GET /api/contract/list?account_code=ACC001

Response:
{
  "contracts": [
    {
      "nid": 987654321,
      "trade_nid": 555555,
      "kpei_reff": "KPEI-20251129-0001",
      "account_code": "ACC001",
      "instrument_code": "BBRI",
      "side": "BORR",
      "quantity": 1000,
      "matched_at": "2025-11-29T10:00:00Z",
      "reimburse_at": "2025-12-06T10:00:00Z",
      "state": "A"
    }
  ]
}
```

### SBL Queries

#### Get SBL Detail
```http
GET /api/sbl/detail?account_code=ACC001&instrument_code=BBRI

Response:
{
  "account_code": "ACC001",
  "instrument_code": "BBRI",
  "borrowing": {
    "total_quantity": 5000,
    "active_contracts": 3,
    "positions": [
      {
        "settlement_date": "2025-11-29",
        "quantity": 2000
      }
    ]
  },
  "lending": {
    "total_quantity": 3000,
    "active_contracts": 2,
    "positions": [
      {
        "settlement_date": "2025-11-29",
        "quantity": 3000
      }
    ]
  }
}
```

#### Get SBL Aggregate
```http
GET /api/sbl/aggregate?instrument_code=BBRI

Response:
{
  "instrument_code": "BBRI",
  "total_borr_quantity": 50000,
  "total_lend_quantity": 45000,
  "total_accounts": 20,
  "settlement_dates": [
    {
      "date": "2025-11-29",
      "borr_quantity": 30000,
      "lend_quantity": 28000
    }
  ]
}
```

### WebSocket

#### Connect
```javascript
const ws = new WebSocket('ws://localhost:8080/ws/notifications');

ws.onopen = () => {
  // Subscribe to receive all buffered notifications
  ws.send(JSON.stringify({
    type: 'subscribe',
    from_seq: 0  // 0 = from beginning
  }));
};

ws.onmessage = (event) => {
  const messages = event.data.split('\n');
  messages.forEach(msg => {
    const notification = JSON.parse(msg);
    console.log(notification);
  });
};
```

### Health Check
```http
GET /health

Response:
{
  "status": "ok",
  "service": "pmeapi"
}
```

### Dashboard
```http
GET /
GET /dashboard

Returns: HTML dashboard with tabs for:
- Orders
- Contracts  
- Accounts
- Instruments
- Notifications (WebSocket)
```

## Event Types (WebSocket)

### Order Events
- `order_created` - New order submitted
- `order_acknowledged` - Order accepted
- `order_rejected` - Order rejected
- `order_pending` - Order pending (future settlement)
- `order_withdrawn` - Order cancelled
- `order_withdrawal_rejected` - Withdrawal rejected

### Trade Events
- `trade_matched` - Trade matched
- `trade_pending_approval` - Waiting for eClear
- `trade_approved` - eClear approved
- `trade_rejected` - eClear rejected
- `trade_reimbursed` - Trade closed

### Contract Events
- `contract_created` - Contract created from trade

### Master Data Events
- `account_limit_updated` - Trading limits changed
- `instrument_status_changed` - Instrument eligibility changed

### Session Events
- `sod` - Start of Day
- `eod` - End of Day

## Configuration

### Environment Variables

```bash
KAFKA_URL=localhost:9092      # Kafka broker
KAFKA_TOPIC=pme-ledger        # Kafka topic
API_PORT=8080                 # HTTP port
INSTANCE_ID=0                 # Snowflake instance ID (0-1023)
```

### Static Files

Web dashboard served from:
```
web/static/pmeapi/
├── index.html
├── css/
│   └── style.css
└── js/
    ├── app.js
    └── notifications.js
```

## Startup Sequence

```
1. Create LedgerPoint
   │
2. Create WebSocket Hub
   │
3. Create Notifier (subscribes to LedgerPoint)
   │
4. Start LedgerPoint with Notifier
   │
5. Wait for IsReady
   │
6. Create HTTP handlers (OrderHandler, QueryHandler, SBLHandler)
   │
7. Setup HTTP routes
   │
8. Start HTTP server
   │
9. Service ready
```

## DropCopy Protocol

The WebSocket system implements a DropCopy-style protocol for reliable notification delivery.

### Features

1. **Sequence Numbers** - Every notification has a unique sequence number
2. **Ring Buffer** - Last 10,000 notifications kept in memory
3. **Recovery** - Clients can request missed notifications
4. **Gap Detection** - Server detects if requested notifications are lost

### Client Flow

```
1. Connect to WebSocket
   │
2. Send subscribe message with from_seq
   │
3. Receive recovery_start (if notifications available)
   │
4. Receive buffered notifications (seq: 1, 2, 3, ...)
   │
5. Receive recovery_complete
   │
6. Receive live notifications as they occur
```

### Handling Disconnections

```javascript
// Save last received sequence
let lastSeq = 0;

ws.onmessage = (event) => {
  const notification = JSON.parse(event.data);
  if (notification.seq) {
    lastSeq = notification.seq;
    localStorage.setItem('last_seq', lastSeq);
  }
};

// On reconnect, request from last sequence
ws.onopen = () => {
  const savedSeq = localStorage.getItem('last_seq') || 0;
  ws.send(JSON.stringify({
    type: 'subscribe',
    from_seq: savedSeq + 1  // Next expected sequence
  }));
};
```

## Monitoring

### Log Patterns

**HTTP Requests:**
```
📨 POST /api/order/new
✅ POST /api/order/new completed in 5.2ms
```

**WebSocket:**
```
[WS] Client connected: 127.0.0.1:56488
[WS-HUB] Client registered: 127.0.0.1:56488 (total: 1)
[WS-HUB] Buffer state: size=83/10000, oldest_seq=1, latest_seq=83
[WS-HUB] Sending 83 recovery messages to 127.0.0.1:56488 (from seq 0)
[WS-HUB] Recovery complete for 127.0.0.1:56488
```

**Notifications:**
```
[NOTIFIER] Sent order_created notification (seq: 1) to 5 clients
[NOTIFIER] Sent trade_matched notification (seq: 2) to 5 clients
```

## Error Handling

### HTTP Errors

```json
// 400 Bad Request
{
  "error": "invalid request: account_code is required"
}

// 404 Not Found
{
  "error": "order not found"
}

// 500 Internal Server Error
{
  "error": "failed to process order"
}
```

### WebSocket Errors

Clients are disconnected if:
- Send buffer is full (client too slow)
- Invalid message format
- Connection idle timeout

## Security Considerations

**Current Implementation:**
- No authentication required
- No authorization checks
- CORS allows all origins
- No rate limiting

**Production Requirements:**
- Add JWT authentication
- Implement role-based access control
- Restrict CORS origins
- Add rate limiting per client
- Enable HTTPS/WSS
- Add request validation
- Implement audit logging

## Performance

### Throughput
- HTTP: ~1000 requests/sec (single instance)
- WebSocket: Broadcasts to unlimited clients
- Notification buffer: 10,000 messages

### Scalability
- Stateless HTTP handlers (horizontal scaling)
- WebSocket requires sticky sessions
- Notification buffer per instance (not shared)

## Testing

### Manual Testing

```bash
# Start service
./bin/pmeapi

# Submit order
curl -X POST http://localhost:8080/api/order/new \
  -H "Content-Type: application/json" \
  -d '{
    "account_code": "ACC001",
    "instrument_code": "BBRI",
    "side": "BORR",
    "quantity": 1000,
    "periode": 7,
    "settlement_date": "2025-11-29",
    "aro_status": false
  }'

# Query orders
curl "http://localhost:8080/api/order/list?account_code=ACC001"

# Test WebSocket
websocat ws://localhost:8080/ws/notifications
```

### Load Testing

```bash
# HTTP load test
ab -n 1000 -c 10 -p order.json \
  -T application/json \
  http://localhost:8080/api/order/new

# WebSocket load test
# Connect multiple clients and monitor broadcast performance
```

## Future Enhancements

- GraphQL API
- gRPC endpoints
- Kafka consumer groups for horizontal scaling
- Redis for shared notification buffer
- Prometheus metrics
- Distributed tracing
- API versioning
- Webhook notifications
- Batch operations
