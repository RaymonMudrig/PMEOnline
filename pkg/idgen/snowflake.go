package idgen

import (
	"fmt"
	"sync"
	"time"
)

// Snowflake ID structure (64 bits):
// - 1 bit: unused (always 0)
// - 41 bits: millisecond timestamp (can represent ~69 years)
// - 10 bits: instance ID (supports 1024 instances)
// - 12 bits: sequence number (4096 IDs per millisecond per instance)

const (
	// Epoch is the custom epoch (January 1, 2024 00:00:00 UTC)
	// Using a recent epoch maximizes the timestamp range
	Epoch int64 = 1704067200000 // 2024-01-01 00:00:00 UTC in milliseconds

	// Bit lengths
	InstanceBits = 10
	SequenceBits = 12

	// Maximum values
	MaxInstance = -1 ^ (-1 << InstanceBits) // 1023
	MaxSequence = -1 ^ (-1 << SequenceBits) // 4095

	// Bit shifts
	InstanceShift = SequenceBits
	TimestampShift = SequenceBits + InstanceBits
)

// Generator generates unique Snowflake IDs
type Generator struct {
	mu           sync.Mutex
	instanceID   int64
	lastTime     int64
	sequence     int64
	clockBackLog []string // Log clock backward events
}

// NewGenerator creates a new Snowflake ID generator
// instanceID must be between 0 and 1023
func NewGenerator(instanceID int64) (*Generator, error) {
	if instanceID < 0 || instanceID > MaxInstance {
		return nil, fmt.Errorf("instance ID must be between 0 and %d, got %d", MaxInstance, instanceID)
	}

	return &Generator{
		instanceID:   instanceID,
		lastTime:     0,
		sequence:     0,
		clockBackLog: make([]string, 0),
	}, nil
}

// NextID generates the next unique ID
func (g *Generator) NextID() (int64, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	now := g.getCurrentMillis()

	// Clock moved backwards
	if now < g.lastTime {
		// Log the event
		backwardDiff := g.lastTime - now
		logMsg := fmt.Sprintf("Clock moved backwards by %d ms (was: %d, now: %d)",
			backwardDiff, g.lastTime, now)
		g.clockBackLog = append(g.clockBackLog, logMsg)

		// Wait until clock catches up
		// This is safer than returning error in production
		for now <= g.lastTime {
			time.Sleep(time.Duration(g.lastTime-now+1) * time.Millisecond)
			now = g.getCurrentMillis()
		}
	}

	// Same millisecond - increment sequence
	if now == g.lastTime {
		g.sequence = (g.sequence + 1) & MaxSequence

		// Sequence overflow - wait for next millisecond
		if g.sequence == 0 {
			now = g.waitNextMillis(g.lastTime)
		}
	} else {
		// New millisecond - reset sequence
		g.sequence = 0
	}

	g.lastTime = now

	// Generate ID
	id := ((now - Epoch) << TimestampShift) |
		(g.instanceID << InstanceShift) |
		g.sequence

	return id, nil
}

// getCurrentMillis returns current time in milliseconds
func (g *Generator) getCurrentMillis() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}

// waitNextMillis waits until next millisecond
func (g *Generator) waitNextMillis(lastTime int64) int64 {
	now := g.getCurrentMillis()
	for now <= lastTime {
		time.Sleep(100 * time.Microsecond)
		now = g.getCurrentMillis()
	}
	return now
}

// GetInstanceID returns the instance ID
func (g *Generator) GetInstanceID() int64 {
	return g.instanceID
}

// GetClockBackwardLog returns all clock backward events
func (g *Generator) GetClockBackwardLog() []string {
	g.mu.Lock()
	defer g.mu.Unlock()
	return append([]string{}, g.clockBackLog...)
}

// ParseID extracts components from a Snowflake ID
func ParseID(id int64) (timestamp int64, instanceID int64, sequence int64) {
	timestamp = (id >> TimestampShift) + Epoch
	instanceID = (id >> InstanceShift) & MaxInstance
	sequence = id & MaxSequence
	return
}

// GetTimestamp extracts the timestamp from a Snowflake ID
func GetTimestamp(id int64) time.Time {
	timestamp, _, _ := ParseID(id)
	return time.Unix(timestamp/1000, (timestamp%1000)*int64(time.Millisecond))
}

// GetInstanceIDFromID extracts the instance ID from a Snowflake ID
func GetInstanceIDFromID(id int64) int64 {
	_, instanceID, _ := ParseID(id)
	return instanceID
}

// GetSequence extracts the sequence from a Snowflake ID
func GetSequence(id int64) int64 {
	_, _, sequence := ParseID(id)
	return sequence
}
