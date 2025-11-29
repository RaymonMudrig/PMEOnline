# PMEOMS - Order Management System

## Overview

PMEOMS (Pinjam Meminjam Efek Order Management System) is the core matching engine for the securities borrowing & lending platform. It processes orders, validates risk limits, matches borrowers with lenders, and generates trades.

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                         PMEOMS                              │
│                                                             │
│  ┌──────────────┐     ┌──────────────┐    ┌─────────────┐ │
│  │ SyncHandler  │────►│     OMS      │───►│   Matcher   │ │
│  └──────────────┘     │   Engine     │    └─────────────┘ │
│         │             └──────────────┘            │        │
│         │                    │                    │        │
│         │             ┌──────┴──────┐            │        │
│         │             │             │            │        │
│         │      ┌──────▼─────┐ ┌────▼────────┐   │        │
│         │      │ Validator  │ │   Checker   │   │        │
│         │      └────────────┘ └─────────────┘   │        │
│         │                                        │        │
│         │             ┌──────────────┐           │        │
│         └─────────────│ LedgerPoint  │◄──────────┘        │
│                       └──────────────┘                     │
│                              │                             │
└──────────────────────────────┼─────────────────────────────┘
                               │
                               ▼
                        ┌─────────────┐
                        │    Kafka    │
                        │ "pme-ledger"│
                        └─────────────┘
```

## Components

### 1. OMS Engine (`internal/pmeoms/oms.go`)

The main orchestrator that coordinates all order processing activities.

**Responsibilities:**
- Process new orders (validation → acknowledgment → matching)
- Initialize existing orders on startup
- Track instrument eligibility
- Coordinate validator, checker, matcher, and trade generator

**Key Methods:**
- `ProcessOrder(orderNID)` - Process new order through validation pipeline
- `MatchOrder(orderNID)` - Attempt to match an acknowledged order
- `InitOrders()` - Process all saved/open orders on startup

### 2. SyncHandler (`internal/pmeoms/sync_handler.go`)

Implements `LedgerPointInterface` to receive events from Kafka.

**Event Handlers:**
- `SyncOrder` - New order submitted, triggers `ProcessOrder()`
- `SyncOrderAck` - Order acknowledged, triggers `MatchOrder()`
- `SyncInstrument` - Instrument eligibility changed, triggers risk check
- Other events handled by risk checker

### 3. Validator (`pkg/ledger/risk/validator.go`)

Validates orders against business rules.

**Validations:**
- Account exists and is active
- Instrument exists and is eligible
- Participant exists
- Quantity > 0
- Settlement date is valid
- Order date/time constraints

**Key Methods:**
- `ValidateOrder(order)` - Full validation
- `IsPendingNew(order)` - Check if settlement date is in future
- `IsPendingReopen(order)` - Check if eligible to reopen from pending

### 4. Checker (`pkg/ledger/risk/checker.go`)

Performs risk and limit checks.

**Checks:**
- Trading limits (account-level)
- Pool limits (participant-level)
- Future commitment calculations
- Session time validation
- Holiday calendar

**Key Methods:**
- `CheckOrderRisk(order)` - Validate order against limits
- `CheckPendingOrders()` - Reopen pending orders when session time allows

### 5. Matcher (`internal/pmeoms/matcher.go`)

Matches borrower orders with lender orders using FIFO algorithm.

**Matching Rules:**
- Same instrument code
- Opposite sides (BORR ↔ LEND)
- Same settlement date
- Same period
- FIFO (First In, First Out)
- Supports partial fills

**Key Methods:**
- `AddOrder(order)` - Add order to book
- `FindMatch(order)` - Find matching counterparty
- `RemoveOrder(orderNID)` - Remove order from book

### 6. OrderBook (`internal/pmeoms/orderbook.go`)

Maintains lists of open orders for matching.

**Data Structure:**
- Map of instrument code → order list
- Separate books for BORR and LEND sides
- Orders stored in FIFO order

### 7. TradeGenerator (`internal/pmeoms/tradegen.go`)

Creates trade and contract events from matched orders.

**Responsibilities:**
- Calculate trade fees (flat fee, borrower fee, lender fee)
- Generate unique KPEI reference
- Create trade event
- Create contract events for each participant
- Handle partial fills

**Fee Calculation:**
- Uses risk.Calculator for fee computation
- Supports ARO (Automatic Roll-Over) fee adjustment
- Different fees for borrower vs lender

## Event Flow

### New Order Flow

```
1. Order Event (Kafka)
   │
   ▼
2. SyncHandler.SyncOrder()
   │
   ▼
3. OMS.ProcessOrder()
   │
   ├──► Validator.ValidateOrder()
   │    │
   │    ├─► VALID ──────────┐
   │    └─► INVALID ────────┼──► OrderNak (Rejected)
   │                         │
   ├──► Checker.IsPending()  │
   │    │                    │
   │    ├─► FUTURE DATE ─────┼──► OrderPending
   │    └─► READY ───────────┘
   │
   ├──► Checker.CheckOrderRisk()
   │    │
   │    ├─► EXCEEDS LIMITS ──────► OrderNak (Rejected)
   │    └─► WITHIN LIMITS
   │
   ▼
4. OrderAck (Acknowledged)
   │
   ▼
5. SyncHandler.SyncOrderAck()
   │
   ▼
6. OMS.MatchOrder()
   │
   ▼
7. Matcher.FindMatch()
   │
   ├─► NO MATCH ────────► Order remains in book
   │
   └─► MATCH FOUND ─────► TradeGenerator.GenerateTrade()
                          │
                          ├──► Trade Event
                          └──► Contract Events
```

### Pending Order Reopening

```
Session Time Change (SOD event)
   │
   ▼
Checker.CheckPendingOrders()
   │
   ▼
For each pending order:
   │
   ├──► Is settlement date valid now?
   │    │
   │    ├─► YES ──► OrderAck (Reopen)
   │    └─► NO ───► Remains pending
   │
   ▼
SyncHandler.SyncOrderAck()
   │
   ▼
OMS.MatchOrder()
```

## Order States

### State Transitions

```
S (Submitted)
  │
  ├──► Validation Failed ──────► R (Rejected) [OrderNak]
  │
  ├──► Future Settlement ───────► G (Pending) [OrderPending]
  │                                    │
  │                                    └──► SOD ──► O (Open) [OrderAck]
  │
  ├──► Exceeds Limits ──────────► R (Rejected) [OrderNak]
  │
  └──► Valid ───────────────────► O (Open) [OrderAck]
                                       │
                                       ├──► Matched ──► M (Matched) [Trade]
                                       │
                                       └──► Withdrawn ─► W (Withdrawn) [OrderWithdrawAck]
```

### State Meanings

- **S (Submitted)** - Order received, awaiting validation
- **O (Open)** - Order validated and active in matching book
- **P (Partial)** - Order partially matched
- **M (Matched)** - Order fully matched
- **G (Pending)** - Order waiting for future settlement date
- **W (Withdrawn)** - Order cancelled by user
- **R (Rejected)** - Order failed validation

## Configuration

### Environment Variables

```bash
KAFKA_URL=localhost:9092      # Kafka broker address
KAFKA_TOPIC=pme-ledger        # Kafka topic name
```

## Startup Sequence

```
1. Create LedgerPoint
   │
2. Create OMS Engine
   │
3. Create SyncHandler
   │
4. Subscribe to LedgerPoint events
   │
5. Start LedgerPoint (Kafka consumer)
   │
6. Wait for IsReady
   │
7. InitOrders() - Process existing orders
   │
8. Start statistics reporter (every 30 seconds)
   │
9. Service ready for new orders
```

## Monitoring

### Log Patterns

**Order Processing:**
```
📥 Processing order: 123 (BORR BBRI 1000 shares)
✅ Order 123 acknowledged
🔄 Attempting to match order 123
✅ Trade matched: KPEI-20251129-0001
```

**Validation Failures:**
```
❌ Order 123 validation failed: account not found
❌ Order 124 rejected: exceeds trading limit
⚠️  Order 125 pending: settlement date in future
```
