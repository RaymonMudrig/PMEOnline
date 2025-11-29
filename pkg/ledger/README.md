# Ledger System

## Overview

The Ledger system is an event-sourced, Kafka-based state management system for the PME (Pinjam Meminjam Efek) securities borrowing & lending platform. It provides a centralized, append-only log of all business events (orders, trades, contracts, etc.) that flows through Kafka and is consumed by multiple services to maintain synchronized state.

## Architecture

```
┌─────────────┐
│   pmeoms    │ ──┐
└─────────────┘   │
                  │ Publish Events
┌─────────────┐   │
│   pmeapi    │ ──┼──────────► ┌──────────────┐
└─────────────┘   │            │ Kafka Topic  │
                  │            │ "pme-ledger" │
┌─────────────┐   │            └──────────────┘
│ eclearapi   │ ──┘                    │
└─────────────┘                        │
                                       │ Consume Events
                  ┌────────────────────┼────────────────────┐
                  │                    │                    │
                  ▼                    ▼                    ▼
           ┌──────────────┐    ┌──────────────┐    ┌──────────────┐
           │ LedgerPoint  │    │ LedgerPoint  │    │ LedgerPoint  │
           │  (pmeoms)    │    │  (pmeapi)    │    │ (eclearapi)  │
           └──────────────┘    └──────────────┘    └──────────────┘
                  │                    │                    │
                  │                    │                    │
                  ▼                    ▼                    ▼
           ┌──────────────┐    ┌──────────────┐    ┌──────────────┐
           │ Subscribers  │    │ Subscribers  │    │ Subscribers  │
           │ (handlers)   │    │ (notifier)   │    │ (eclear)     │
           └──────────────┘    └──────────────┘    └──────────────┘
```

## Core Components

### LedgerPoint

`LedgerPoint` is the main component that:
- Consumes events from Kafka topic `pme-ledger`
- Maintains in-memory state of all entities (orders, trades, contracts, accounts, etc.)
- Notifies subscribers when events occur
- Provides thread-safe read access to entities
- Provides a `Commit` channel for publishing new events to Kafka

**Key Fields:**
- `Commit chan any` - Channel for publishing events to Kafka
- `IsReady bool` - Indicates when initial Kafka replay is complete
- Entity maps: `orders`, `trades`, `contracts`, `accounts`, `participants`, `instruments`, etc.
- Mutexes for thread-safe access to each entity collection

### LedgerPointInterface

All event subscribers must implement this interface with handler methods for each event type:

```go
type LedgerPointInterface interface {
    SyncServiceStart(a ServiceStart)
    SyncParameter(a Parameter)
    SyncSessionTime(a SessionTime)
    SyncHoliday(a Holiday)
    SyncAccount(a Account)
    SyncAccountLimit(a AccountLimit)
    SyncParticipant(a Participant)
    SyncInstrument(a Instrument)
    SyncOrder(a Order)
    SyncOrderAck(a OrderAck)
    SyncOrderNak(a OrderNak)
    SyncOrderPending(a OrderPending)
    SyncOrderWithdraw(a OrderWithdraw)
    SyncOrderWithdrawAck(a OrderWithdrawAck)
    SyncOrderWithdrawNak(a OrderWithdrawNak)
    SyncTrade(a Trade)
    SyncTradeWait(a TradeWait)
    SyncTradeAck(a TradeAck)
    SyncTradeNak(a TradeNak)
    SyncTradeReimburse(a TradeReimburse)
    SyncContract(a Contract)
    SyncSod(a Sod)
    SyncEod(a Eod)
}
```

## Kafka Integration

### Topic Configuration
- **Topic Name**: `pme-ledger`
- **Partition**: 0 (single partition for strict ordering)
- **Consumer Behavior**: Reads from `FirstOffset` (beginning) on startup

### Message Format
Each Kafka message contains:
- **Key**: Event type (e.g., "Order", "Trade", "OrderAck")
- **Value**: JSON-serialized event data
- **Headers**: Additional metadata

### Consumer Group
Each service uses a unique consumer ID (e.g., "pmeoms", "pmeapi", "eclearapi") but does NOT use consumer groups, ensuring each service reads from the beginning and maintains its own complete state.

## Event Flow

### Publishing Events

1. Service commits an event to the `Commit` channel:
   ```go
   ledgerPoint.Commit <- ledger.Order{
       NID:            orderNID,
       AccountCode:    "ACC001",
       InstrumentCode: "BBRI",
       Side:           "BORR",
       Quantity:       1000,
       // ... other fields
   }
   ```

2. LedgerPoint serializes the event to JSON
3. Event is published to Kafka topic `pme-ledger`
4. Kafka message key is set to event type (e.g., "Order")

### Consuming Events

1. LedgerPoint reads messages from Kafka (from beginning)
2. Message key determines event type
3. JSON is deserialized into appropriate event struct
4. Event is processed:
   - State is updated in LedgerPoint's entity maps
   - All subscribers are notified via their Sync methods
5. Process continues until `IsReady` is set (when ServiceStart with matching ID is received)

## Usage Patterns

### Creating and Starting LedgerPoint

**IMPORTANT**: Subscribers must be registered BEFORE calling `Start()`, so they receive all events from the beginning.

```go
// 1. Create LedgerPoint
ledgerPoint := ledger.CreateLedgerPoint(kafkaURL, kafkaTopic, "service-name")

// 2. Create subscribers BEFORE starting
notifier := websocket.NewNotifier(hub, ledgerPoint)
handler := eclear.NewEClearClient(eclearURL, ledgerPoint)

// 3. Collect all subscribers
subscribers := []ledger.LedgerPointInterface{
    notifier,
    handler.GetSyncHandler(),
}

// 4. Start LedgerPoint with subscribers
ledgerPoint.Start(subscribers, ctx)

// 5. Wait for IsReady (initial Kafka replay complete)
for !ledgerPoint.IsReady {
    time.Sleep(100 * time.Millisecond)
}

// 6. Now safe to start processing that requires complete state
go handler.RunProcessing(ctx)
```

### Implementing a Subscriber

```go
type MyHandler struct {
    ledger *ledger.LedgerPoint
}

// Implement all required Sync methods
func (h *MyHandler) SyncOrder(a ledger.Order) {
    log.Printf("New order: %d for account %s", a.NID, a.AccountCode)
    // Handle order creation
}

func (h *MyHandler) SyncTrade(a ledger.Trade) {
    log.Printf("New trade: %s", a.KpeiReff)
    // Handle trade matching
}

// Implement remaining methods (can be no-ops if not needed)
func (h *MyHandler) SyncServiceStart(a ledger.ServiceStart) {}
func (h *MyHandler) SyncParameter(a ledger.Parameter) {}
// ... etc
```

### Reading Entity State

LedgerPoint provides thread-safe getter methods:

```go
// Get an order by NID
order, exists := ledgerPoint.GetOrder(orderNID)
if exists {
    log.Printf("Order state: %s", order.State)
}

// Get an account
account, exists := ledgerPoint.GetAccount("ACC001")
if exists {
    log.Printf("Account SID: %s", account.SID)
}

// Iterate over all trades
ledgerPoint.ForEachTrade(func(trade ledger.TradeEntity) bool {
    log.Printf("Trade: %s, State: %s", trade.KpeiReff, trade.State)
    return true // Continue iteration
})
```

## Event Types

### Configuration Events
- **ServiceStart** - Service initialization marker, triggers `IsReady`
- **Parameter** - System parameters (fees, dates, etc.)
- **SessionTime** - Trading session schedule
- **Holiday** - Holiday calendar

### Master Data Events
- **Account** - Account registration
- **AccountLimit** - Trading limits
- **Participant** - Participant registration
- **Instrument** - Instrument eligibility

### Order Events
- **Order** - New order submitted (state: S)
- **OrderAck** - Order accepted by matching engine (state: S → O)
- **OrderNak** - Order rejected (state: S → R)
- **OrderPending** - Order pending validation (state: O → G)
- **OrderWithdraw** - Withdrawal request
- **OrderWithdrawAck** - Withdrawal accepted (state: O/P → W)
- **OrderWithdrawNak** - Withdrawal rejected

### Trade Events
- **Trade** - Trade matched (state: M)
- **TradeWait** - Waiting for eClear approval (state: M → E)
- **TradeAck** - eClear approved (state: E → M)
- **TradeNak** - eClear rejected (state: E → R)
- **TradeReimburse** - Contract reimbursed (state: M → C)

### Contract Events
- **Contract** - Contract created from trade

### Session Events
- **Sod** - Start of Day
- **Eod** - End of Day

## State Machine

### Order States
- **S** (Submitted) - Order created, not yet acknowledged
- **O** (Open) - Order acknowledged, active in book
- **P** (Partial) - Partially filled
- **M** (Matched) - Fully matched
- **W** (Withdrawn) - Cancelled
- **R** (Rejected) - Rejected by system
- **G** (Pending) - Pending validation

### Trade States
- **M** (Matched) - Trade created
- **E** (Approval) - Waiting for eClear approval
- **C** (Closed) - Reimbursed/closed
- **R** (Rejected) - Rejected by eClear

### Contract States
- **A** (Active) - Contract active
- **C** (Closed) - Contract closed/reimbursed
- **T** (Terminated) - Contract terminated

## Thread Safety

All entity collections use individual RWMutex locks:
- Read operations use `RLock()` / `RUnlock()`
- Write operations use `Lock()` / `Unlock()`
- Getter methods return copies of entities, not references
- Iterator functions (ForEach*) hold read locks during iteration

## IsReady Mechanism

`IsReady` becomes `true` when:
1. Upon starting a LedgerPoint will create a unique `startid` and send with ServiceStart event.
2. LedgerPoint receives a `ServiceStart` event
3. The `StartID` in the event matches this instance's `startid`
4. This indicates the service has processed all historical events up to its own start

**Why this matters:**
- Services start reading from Kafka beginning (FirstOffset)
- Historical events from previous runs are replayed
- `IsReady` signals when replay is complete and service can begin processing
- Some operations (like matching new orders) should wait for `IsReady`
- Subscribers receive ALL events, both historical and new

## Common Patterns

### Pattern 1: Query Handler (Read-Only)

```go
type QueryHandler struct {
    ledger *ledger.LedgerPoint
}

func (h *QueryHandler) GetOrderList(accountCode string) []OrderInfo {
    var orders []OrderInfo
    h.ledger.ForEachOrder(func(order ledger.OrderEntity) bool {
        if order.AccountCode == accountCode {
            orders = append(orders, convertOrder(order))
        }
        return true
    })
    return orders
}
```

### Pattern 2: Event Processor (Read-Write)

```go
type OrderHandler struct {
    ledger   *ledger.LedgerPoint
    idGen    *idgen.Generator
}

func (h *OrderHandler) NewOrder(req OrderRequest) error {
    // Wait for IsReady before processing new orders
    if !h.ledger.IsReady {
        return errors.New("system not ready")
    }

    // Validate using current state
    account, exists := h.ledger.GetAccount(req.AccountCode)
    if !exists {
        return errors.New("account not found")
    }

    // Commit event
    h.ledger.Commit <- ledger.Order{
        NID:            h.idGen.NextID(),
        AccountCode:    req.AccountCode,
        InstrumentCode: req.InstrumentCode,
        // ...
    }
    return nil
}
```

### Pattern 3: Outbound Integration (Event Forwarder)

```go
type EClearSyncHandler struct {
    client *EClearClient
}

func (h *EClearSyncHandler) SyncTrade(a ledger.Trade) {
    // Forward trade to external system
    go h.client.SendTrade(a)
}

// No-op for events we don't care about
func (h *EClearSyncHandler) SyncOrder(a ledger.Order) {}
func (h *EClearSyncHandler) SyncServiceStart(a ledger.ServiceStart) {}
// ...
```

### Pattern 4: Notification System (Event Broadcaster)

```go
type Notifier struct {
    hub    *websocket.Hub
    ledger *ledger.LedgerPoint
}

func (n *Notifier) SyncOrder(a ledger.Order) {
    n.hub.BroadcastNotification("order_created", map[string]interface{}{
        "order_nid":    a.NID,
        "account_code": a.AccountCode,
        "instrument":   a.InstrumentCode,
        "state":        "S",
    })
}
```

## Best Practices

1. **Subscribe Before Start**: Always register subscribers before calling `Start()` to ensure they receive all events from the beginning

2. **Wait for IsReady**: For operations requiring complete state (like order matching), wait for `IsReady` before processing

3. **Use Getters**: Always use provided getter methods (`GetOrder()`, `GetAccount()`, etc.) instead of accessing maps directly

4. **Copy, Don't Reference**: Getter methods return copies - don't store references to internal entities

5. **Don't Block in Sync Methods**: Keep Sync methods fast - use goroutines for slow operations

6. **Handle Missing Entities**: Always check the `exists` return value from getter methods

7. **Avoid Circular Dependencies**: Don't commit events from within Sync methods of the same event type

## Debugging

### Enable Debug Logging
Uncomment the logging line in `go_receive()`:
```go
// log.Printf("🔔 <-- key=%s value=%s offset=%d\n",
//     string(m.Key), string(m.Value), m.Offset)
```

### Check Kafka Messages
```bash
kafka-console-consumer --bootstrap-server localhost:9092 \
    --topic pme-ledger \
    --from-beginning \
    --property print.key=true \
    --property print.timestamp=true
```

### Monitor Event Flow
Look for these log patterns:
- `🚀 Starting LedgerPoint processing...` - Service started
- `✅ LedgerPoint is ready` - Initial replay complete
- `⚠️ Received message without headers, skipping` - Invalid Kafka message

## Performance Considerations

- **Single Partition**: Ensures total ordering but limits horizontal scaling
- **In-Memory State**: All entities kept in RAM for fast access
- **Read-Optimized**: RWMutex allows concurrent reads
- **Kafka Replay**: Services replay entire topic on startup - may be slow with large history
- **No Snapshots**: No state snapshots - full replay every time

## Future Improvements

- Add state snapshots to reduce startup time
- Implement compaction for old events
- Add event replay from specific offset
- Support multiple partitions with partition key
- Add metrics and monitoring
- Implement event versioning
