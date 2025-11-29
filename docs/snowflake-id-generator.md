# Snowflake ID Generator

## Overview

The PME Online system uses a **Snowflake-like ID generation algorithm** to ensure globally unique order NIDs across multiple load-balanced pmeapi instances in a high-concurrency environment.

## ID Structure (64 bits)

```
┌─────────────────────────────────────────────────────────────┐
│ 1 bit │    41 bits      │   10 bits    │     12 bits       │
│ (0)   │   Timestamp     │ Instance ID  │    Sequence       │
└─────────────────────────────────────────────────────────────┘
  Sign     Milliseconds      0-1023         0-4095
          since epoch
```

### Bit Breakdown

- **Bit 63**: Sign bit (always 0, ensures positive numbers)
- **Bits 62-22**: 41-bit timestamp (milliseconds since custom epoch)
  - Epoch: 2024-01-01 00:00:00 UTC
  - Range: ~69 years from epoch
- **Bits 21-12**: 10-bit instance ID
  - Range: 0-1023 (supports 1024 instances)
- **Bits 11-0**: 12-bit sequence number
  - Range: 0-4095 (4096 IDs per millisecond per instance)

## Capacity

- **Per Instance**: 4,096 unique IDs per millisecond = **4.096 million IDs/second**
- **All Instances**: 1024 instances × 4.096M = **4.2 billion IDs/second**
- **Total IDs**: 2^63 ≈ 9.2 quintillion unique IDs

## Deployment Configuration

### Environment Variables

Each pmeapi instance **MUST** have a unique `INSTANCE_ID`:

```bash
# Instance 1
export INSTANCE_ID=0

# Instance 2
export INSTANCE_ID=1

# Instance 3
export INSTANCE_ID=2
```

**Valid range**: 0-1023

### Docker Compose Example

```yaml
version: '3.8'

services:
  pmeapi-1:
    image: pmeapi:latest
    environment:
      - INSTANCE_ID=0
      - KAFKA_URL=kafka:9092
      - API_PORT=8080
    ports:
      - "8080:8080"

  pmeapi-2:
    image: pmeapi:latest
    environment:
      - INSTANCE_ID=1
      - KAFKA_URL=kafka:9092
      - API_PORT=8080
    ports:
      - "8081:8080"

  pmeapi-3:
    image: pmeapi:latest
    environment:
      - INSTANCE_ID=2
      - KAFKA_URL=kafka:9092
      - API_PORT=8080
    ports:
      - "8082:8080"
```

### Kubernetes Example

```yaml
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: pmeapi
spec:
  serviceName: pmeapi
  replicas: 3
  selector:
    matchLabels:
      app: pmeapi
  template:
    metadata:
      labels:
        app: pmeapi
    spec:
      containers:
      - name: pmeapi
        image: pmeapi:latest
        env:
        - name: KAFKA_URL
          value: "kafka:9092"
        - name: API_PORT
          value: "8080"
        - name: INSTANCE_ID
          valueFrom:
            fieldRef:
              fieldPath: metadata.labels['statefulset.kubernetes.io/pod-name']
        # Extract pod ordinal (0, 1, 2...) as INSTANCE_ID
        command: ["/bin/sh"]
        args:
        - -c
        - |
          export INSTANCE_ID=$(echo $HOSTNAME | sed 's/.*-//')
          ./pmeapi
```

## Performance Characteristics

Benchmarked on Apple M1 Max:

```
BenchmarkNextID-10          4,400,253 ops/sec    244 ns/op    0 allocs/op
BenchmarkNextIDParallel-10  4,922,896 ops/sec    247 ns/op    0 allocs/op
```

- **Thread-safe**: Uses mutex for concurrent access
- **Zero allocations**: No heap allocations per ID generation
- **High throughput**: ~4.4M IDs/second sustained

## Features

### 1. Clock Backward Handling

When system clock moves backwards:
- Automatically waits until clock catches up
- Logs all clock backward events
- Never generates duplicate IDs

```go
// Check for clock backward events
logs := idGenerator.GetClockBackwardLog()
for _, log := range logs {
    fmt.Println(log)
}
```

### 2. Sequence Overflow Protection

When sequence reaches 4095 in same millisecond:
- Automatically waits for next millisecond
- Resets sequence to 0
- Maintains uniqueness guarantee

### 3. ID Parsing

Extract components from any Snowflake ID:

```go
timestamp, instanceID, sequence := idgen.ParseID(orderNID)

// Get specific components
ts := idgen.GetTimestamp(orderNID)    // Returns time.Time
inst := idgen.GetInstanceIDFromID(orderNID)
seq := idgen.GetSequence(orderNID)
```

## Usage Example

```go
import "pmeonline/pkg/idgen"

// Initialize generator (usually in main.go)
instanceID := int64(0) // From environment variable
generator, err := idgen.NewGenerator(instanceID)
if err != nil {
    log.Fatal(err)
}

// Generate IDs
orderNID, err := generator.NextID()
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Generated Order NID: %d\n", orderNID)

// Parse the ID
timestamp, instance, seq := idgen.ParseID(orderNID)
fmt.Printf("Timestamp: %d, Instance: %d, Sequence: %d\n",
    timestamp, instance, seq)
```

## Advantages Over Alternative Approaches

### vs. Timestamp-only (Current)
- ❌ Timestamp: High collision risk in load-balanced setup
- ✅ Snowflake: Guaranteed unique across all instances

### vs. Database Sequence
- ❌ Database: Extra roundtrip, single point of contention
- ✅ Snowflake: Local generation, zero coordination

### vs. UUID/GUID
- ❌ UUID: 128 bits, not sequential, harder to debug
- ✅ Snowflake: 64 bits, time-ordered, human-debuggable

### vs. Redis INCR
- ❌ Redis: Additional dependency, network latency
- ✅ Snowflake: No external dependencies, instant

## Monitoring and Debugging

### Check Instance ID from Order NID

```go
orderNID := 123456789
instanceID := idgen.GetInstanceIDFromID(orderNID)
fmt.Printf("Order %d was created by instance %d\n", orderNID, instanceID)
```

### Check Order Creation Time

```go
orderNID := 123456789
createdAt := idgen.GetTimestamp(orderNID)
fmt.Printf("Order %d created at %s\n", orderNID, createdAt.Format(time.RFC3339))
```

### Monitor Clock Backward Events

```go
// Periodically check for clock issues
logs := generator.GetClockBackwardLog()
if len(logs) > 0 {
    log.Printf("WARNING: Clock moved backwards %d times", len(logs))
    for _, event := range logs {
        log.Printf("  - %s", event)
    }
}
```

## Best Practices

1. **Unique Instance IDs**: Ensure each pmeapi instance has a unique INSTANCE_ID (0-1023)
2. **StatefulSet/Pod Naming**: Use StatefulSets in Kubernetes to get predictable pod names
3. **NTP Sync**: Keep server clocks synchronized with NTP to minimize clock drift
4. **Instance ID Management**: Document instance ID assignments in your infrastructure
5. **Testing**: Test with multiple instances to verify uniqueness guarantee

## Troubleshooting

### Error: "instance ID must be between 0 and 1023"

**Cause**: Invalid INSTANCE_ID environment variable

**Solution**:
```bash
export INSTANCE_ID=0  # Valid: 0-1023
```

### Clock Backward Warnings

**Cause**: System clock was adjusted backwards (NTP sync, manual change, VM migration)

**Impact**: ID generation pauses briefly until clock catches up

**Solution**:
- Use NTP to prevent large clock jumps
- Monitor clock backward logs
- Consider using monotonic clock in VM environments

### Duplicate Order NIDs

**Cause**: Two instances using the same INSTANCE_ID

**Solution**: Verify each instance has unique INSTANCE_ID:
```bash
# On each server
echo $INSTANCE_ID

# Should be different on each pmeapi instance
```

## Migration from Timestamp-based IDs

The old system used `time.Now().UnixNano() / int64(time.Millisecond)` which:
- Had collision risk with multiple instances
- Had no sequence counter
- Could not identify which instance created the order

New Snowflake IDs are backward compatible (same int64 type) but provide:
- Guaranteed uniqueness
- Instance identification
- Higher capacity (4096 IDs/ms vs 1 ID/ms)

## References

- [Twitter Snowflake](https://github.com/twitter-archive/snowflake/tree/snowflake-2010)
- [Discord Snowflake](https://discord.com/developers/docs/reference#snowflakes)
- [Instagram ID System](https://instagram-engineering.com/sharding-ids-at-instagram-1cf5a71e5a5c)
