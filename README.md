# PME Online - Securities Borrowing and Lending System

A microservices-based system for securities borrowing and lending (SBL) operations, built with Go and event-driven architecture using Kafka.

## Architecture Overview

```
┌─────────┐     ┌──────────────┐     ┌───────┐     ┌─────────────┐
│ eClear  │────>│  eClear API  │────>│ Kafka │────>│     OMS     │
│ System  │<────│   Service    │<────│       │<────│   Matching  │
└─────────┘     └──────────────┘     └───┬───┘     └─────────────┘
                                         │
                ┌────────────────────────┼────────────────────────┐
                │                        │                        │
                v                        v                        v
        ┌──────────────┐        ┌──────────────┐        ┌──────────────┐
        │  APME API    │        │ DB Exporter  │        │  Other       │
        │  Service     │        │   Service    │        │  Services    │
        └──────────────┘        └──────┬───────┘        └──────────────┘
                                       │
                                       v
                                ┌──────────────┐
                                │  PostgreSQL  │
                                └──────────────┘
```

## Components

### 1. eClear API Service ✅ (Implemented)
- Receives master data from eClear (Participants, Accounts, Instruments, Limits)
- Receives trade approvals and reimbursement instructions
- Sends trade submissions to eClear for approval
- **Port:** 8081
- **Status:** Ready for testing

### 2. OMS (Order Management System) ✅ (Implemented)
- Validates orders with risk management rules (pkg/risk)
- Matches borrowing and lending orders (pkg/oms)
- Generates trades and contracts
- Handles order lifecycle (New, Open, Matched, etc.)
- **Status:** Ready for testing

### 3. APME API Service ✅ (Implemented)
- REST API for order entry, amendment, and withdrawal
- WebSocket notifications for real-time updates
- Query APIs for order list, contract list, SBL display
- Web-based test dashboard with auto-populated forms
- **Port:** 8080
- **Status:** Ready for testing

### 4. Database Exporter ✅ (Implemented)
- Persists all events to PostgreSQL
- Maintains complete audit trail
- Repository pattern for clean data access
- Automatic database schema migration
- **Status:** Ready for testing

## Technology Stack

- **Language:** Go 1.23+
- **Message Broker:** Apache Kafka 4.1
- **Database:** PostgreSQL 18
- **Authentication:** Keycloak SSO (planned)
- **Containerization:** Docker & Docker Compose

## Getting Started

### Prerequisites

- Go 1.23 or higher
- Docker and Docker Compose
- Make (optional, for convenience)

### Quick Start

1. **Clone the repository**
   ```bash
   cd /Users/raymonmudrig/go/src/pmeonline
   ```

2. **Start infrastructure** (Kafka & PostgreSQL)
   ```bash
   make setup
   # Or manually:
   # docker-compose up -d
   # docker exec pme-kafka kafka-topics.sh --create --topic pme-ledger --bootstrap-server localhost:9092
   ```

3. **Run eClear API Service**
   ```bash
   make run-eclearapi
   # Or manually:
   # cd cmd/eclearapi && go run main.go
   ```

4. **Populate master data**
   ```bash
   make test-eclearapi
   # Or manually:
   # cd cmd/eclearapi && ./test.sh
   ```

### Manual Setup

If you prefer not to use Make:

```bash
# Start infrastructure
docker-compose up -d

# Wait for Kafka to be ready (about 10 seconds)
sleep 10

# Create Kafka topic
docker exec pme-kafka kafka-topics.sh \
  --create \
  --topic pme-ledger \
  --bootstrap-server localhost:9092 \
  --partitions 3 \
  --replication-factor 1

# Run eClear API
cd cmd/eclearapi
go run main.go
```

## Project Structure

Following [golang-standards/project-layout](https://github.com/golang-standards/project-layout):

```
pmeonline/
├── cmd/                         # Application entry points (thin wrappers)
│   ├── eclearapi/
│   │   ├── main.go              # ✅ ONLY main.go entry point
│   │   ├── Dockerfile           # Container image
│   │   └── readme.md
│   ├── pmeoms/
│   │   ├── main.go              # ✅ ONLY main.go entry point
│   │   ├── Dockerfile           # Container image
│   │   └── readme.md
│   ├── pmeapi/
│   │   ├── main.go              # ✅ ONLY main.go entry point
│   │   ├── Dockerfile           # Container image
│   │   └── readme.md
│   └── dbexporter/
│       ├── main.go              # ✅ ONLY main.go entry point
│       ├── migrations/          # SQL migrations (deployment asset)
│       ├── Dockerfile           # Container image
│       └── readme.md
│
├── internal/                    # Private application code (unexportable)
│   ├── eclearapi/
│   │   └── handler/             # HTTP handlers, eClear client
│   ├── pmeapi/
│   │   ├── handler/             # REST API handlers
│   │   ├── middleware/          # HTTP middleware
│   │   └── websocket/           # WebSocket hub & notifier
│   └── dbexporter/
│       ├── db/                  # Database connection
│       ├── exporter/            # Event exporter logic
│       └── repository/          # Data access layer
│
├── pkg/                         # Public reusable libraries (exportable)
│   ├── ledger/                  # Event-sourcing framework
│   │   ├── entities.go          # Domain entities
│   │   ├── entries.go           # Event types
│   │   └── ledgerpoint.go       # Event bus & state sync
│   ├── risk/                    # Risk management
│   │   ├── validator.go         # Pre-trade validation
│   │   ├── calculator.go        # Fee calculations
│   │   └── checker.go           # Eligibility monitoring
│   └── oms/                     # Order matching engine
│       ├── oms.go               # OMS orchestrator
│       ├── orderbook.go         # Order book per instrument
│       ├── matcher.go           # Matching algorithm
│       └── tradegen.go          # Trade & contract generator
│
├── web/                         # Web assets (static files)
│   └── static/
│       ├── eclearapi/           # eClear API dashboard
│       │   ├── index.html
│       │   ├── css/
│       │   └── js/
│       └── pmeapi/              # APME API dashboard
│           ├── index.html
│           ├── css/
│           └── js/
│
├── tests/                       # Integration & E2E tests
│   ├── eclearapi/
│   │   ├── testdata/            # Sample test data (JSON fixtures)
│   │   └── test.sh              # Integration test script
│   └── pmeapi/
│
├── docs/                        # Documentation
│   ├── design.md                # System design specification
│   ├── deployment.md            # Production deployment guide
│   ├── eclearapi/               # eClear API specific documentation
│   │   ├── DASHBOARD.md         # Web dashboard usage guide
│   │   └── SETTINGS_API.md      # Settings API (Parameters, Holidays, SessionTime)
│   └── pmeoms/                  # OMS specific documentation
│       └── RISK_AND_OMS_IMPLEMENTATION.md  # Risk management & matching engine details
│
├── configs/                     # Configuration files
├── scripts/                     # Build & utility scripts
│
├── docker-compose.yml           # Local development setup
├── Makefile                     # Build commands
├── .dockerignore                # Docker build exclusions
├── go.mod                       # Go module definition
└── README.md                    # This file
```

**Key Principles:**
- `cmd/` contains **only** `main.go` files (thin entry points)
- `internal/` contains private implementation (Go enforces privacy)
- `pkg/` contains public reusable libraries
- `web/` contains static assets (HTML, CSS, JS)
- Service logic is in `internal/{service}/`, not in `cmd/`

## Testing

### Full System Test

#### 1. Start Infrastructure
```bash
# Start Kafka and PostgreSQL
make setup

# Wait for services to be ready (~10 seconds)
sleep 10
```

#### 2. Start All Services

Open separate terminals for each service:

```bash
# Terminal 1: Database Exporter (start first to capture all events)
make run-dbexporter

# Terminal 2: eClear API
make run-eclearapi

# Terminal 3: OMS
make run-pmeoms

# Terminal 4: APME API
make run-pmeapi
```

#### 3. Load Master Data
```bash
# Use the automated test script
make test-eclearapi

# Or manually:
cd tests/eclearapi && ./test.sh
```

#### 4. Test Order Entry

**Using Web Dashboard (Recommended):**
```
Open browser: http://localhost:8080
- Go to "Entry Borrow" or "Entry Lend" tab
- Fill in form (dropdowns auto-populated from master data)
- Submit order
- View results in "Order List" and "SBL Detail" tabs
```

**Using API:**
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
    "aro": false
  }'
```

#### 5. Verify Results

**Check OMS logs** for matching:
- Order validation
- Matching process
- Trade generation

**Check APME API dashboard:**
- Order List tab
- Contract List tab
- SBL Detail and Aggregate tabs

**Query PostgreSQL:**
```bash
docker exec -it pme-postgres psql -U pmeuser -d pmedb

# View orders
SELECT * FROM orders;

# View trades
SELECT * FROM trades;

# View contracts
SELECT * FROM contracts;
```

### Health Checks
```bash
# eClear API
curl http://localhost:8081/health

# APME API
curl http://localhost:8080/health
```

## Event-Driven Architecture

The system uses an event-sourcing pattern where all state changes flow through Kafka:

1. **Commands** → Services publish events to Kafka
2. **Events** → All services consume events to update their state
3. **Queries** → Services read from their in-memory state (CQRS pattern)

### Event Flow Example

```
eClear sends account data
    ↓
POST /account/insert
    ↓
eClear API creates Account event
    ↓
Commit to Kafka
    ↓
LedgerPoint syncs (all services)
    ↓
Services update their in-memory state
    ↓
DB Exporter persists to PostgreSQL
```

## Configuration

Environment variables for eClear API Service:

| Variable | Default | Description |
|----------|---------|-------------|
| `KAFKA_URL` | `localhost:9092` | Kafka broker address |
| `KAFKA_TOPIC` | `pme-ledger` | Kafka topic name |
| `API_PORT` | `8081` | HTTP server port |
| `ECLEAR_BASE_URL` | `http://localhost:9000` | eClear system URL |

## Development

### Running Individual Services

```bash
# eClear API (Port 8081)
make run-eclearapi

# OMS (Order Management System)
make run-pmeoms

# APME API (Port 8080)
make run-pmeapi

# DB Exporter
make run-dbexporter
```

### Building

```bash
# Build all services
make build

# Binaries will be in ./bin/
```

### Monitoring Kafka Messages

```bash
# View all messages in the topic
docker exec pme-kafka kafka-console-consumer.sh \
  --bootstrap-server localhost:9092 \
  --topic pme-ledger \
  --from-beginning \
  --property print.headers=true
```

## System Status

All core components have been implemented and are ready for testing:

- ✅ **eClear API Service** - Master data and trade approval integration
- ✅ **OMS Matching Engine** - Order validation, risk management, and matching
- ✅ **APME API Service** - Client-facing REST APIs and WebSocket notifications
- ✅ **Database Exporter** - Event persistence and audit trail

The system is now in the **testing and integration phase**. See individual service README files for testing instructions.

## Contributing

1. Follow Go best practices and conventions
2. Write tests for new functionality
3. Update documentation when adding features
4. Use meaningful commit messages

## Documentation

- [System Design](docs/design.md) - Complete system specification
- [Production Deployment](docs/deployment.md) - Deployment guide for distributed systems
- [eClear API Documentation](cmd/eclearapi/readme.md) - eClear API service guide
- [OMS Documentation](cmd/pmeoms/readme.md) - Order Management System guide
- [APME API Documentation](cmd/pmeapi/readme.md) - APME API service guide
- [DB Exporter Documentation](cmd/dbexporter/readme.md) - Database Exporter guide

## License

Proprietary - KPEI (Indonesia Clearing and Guarantee Corporation)

## Support

For questions or issues, please refer to:
- System design: `CLAUDE.md`
- Implementation plan: `IMPLEMENTATION_PLAN.md`
- Service-specific docs: `cmd/*/README.md`
