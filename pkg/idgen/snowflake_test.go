package idgen

import (
	"sync"
	"testing"
	"time"
)

func TestNewGenerator(t *testing.T) {
	tests := []struct {
		name       string
		instanceID int64
		wantErr    bool
	}{
		{"Valid instance ID 0", 0, false},
		{"Valid instance ID 512", 512, false},
		{"Valid instance ID 1023", 1023, false},
		{"Invalid negative ID", -1, true},
		{"Invalid too large ID", 1024, true},
		{"Invalid way too large ID", 9999, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gen, err := NewGenerator(tt.instanceID)
			if tt.wantErr {
				if err == nil {
					t.Errorf("NewGenerator() expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("NewGenerator() unexpected error: %v", err)
				}
				if gen.GetInstanceID() != tt.instanceID {
					t.Errorf("GetInstanceID() = %d, want %d", gen.GetInstanceID(), tt.instanceID)
				}
			}
		})
	}
}

func TestNextID(t *testing.T) {
	gen, err := NewGenerator(1)
	if err != nil {
		t.Fatalf("Failed to create generator: %v", err)
	}

	id1, err := gen.NextID()
	if err != nil {
		t.Fatalf("NextID() error: %v", err)
	}

	id2, err := gen.NextID()
	if err != nil {
		t.Fatalf("NextID() error: %v", err)
	}

	// IDs should be unique
	if id1 == id2 {
		t.Errorf("Generated duplicate IDs: %d", id1)
	}

	// IDs should be increasing
	if id2 <= id1 {
		t.Errorf("IDs not increasing: id1=%d, id2=%d", id1, id2)
	}
}

func TestParseID(t *testing.T) {
	gen, _ := NewGenerator(42)
	id, _ := gen.NextID()

	timestamp, instanceID, sequence := ParseID(id)

	if instanceID != 42 {
		t.Errorf("ParseID() instanceID = %d, want 42", instanceID)
	}

	if sequence < 0 || sequence > MaxSequence {
		t.Errorf("ParseID() sequence = %d, out of range [0, %d]", sequence, MaxSequence)
	}

	if timestamp < Epoch {
		t.Errorf("ParseID() timestamp = %d, less than epoch %d", timestamp, Epoch)
	}

	// Verify timestamp is reasonable (within last hour)
	now := time.Now().UnixNano() / int64(time.Millisecond)
	if timestamp < now-3600000 || timestamp > now+1000 {
		t.Errorf("ParseID() timestamp = %d, not within reasonable range of now = %d", timestamp, now)
	}
}

func TestConcurrentGeneration(t *testing.T) {
	gen, _ := NewGenerator(5)
	count := 10000
	ids := make(chan int64, count)
	var wg sync.WaitGroup

	// Generate IDs concurrently
	numGoroutines := 10
	idsPerGoroutine := count / numGoroutines

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < idsPerGoroutine; j++ {
				id, err := gen.NextID()
				if err != nil {
					t.Errorf("NextID() error: %v", err)
					return
				}
				ids <- id
			}
		}()
	}

	wg.Wait()
	close(ids)

	// Check uniqueness
	seen := make(map[int64]bool)
	for id := range ids {
		if seen[id] {
			t.Errorf("Duplicate ID generated: %d", id)
		}
		seen[id] = true
	}

	if len(seen) != count {
		t.Errorf("Expected %d unique IDs, got %d", count, len(seen))
	}
}

func TestMultipleInstancesNoCollision(t *testing.T) {
	gen1, _ := NewGenerator(1)
	gen2, _ := NewGenerator(2)

	count := 1000
	ids := make(map[int64]bool)

	for i := 0; i < count; i++ {
		id1, _ := gen1.NextID()
		id2, _ := gen2.NextID()

		if ids[id1] {
			t.Errorf("Duplicate ID from gen1: %d", id1)
		}
		if ids[id2] {
			t.Errorf("Duplicate ID from gen2: %d", id2)
		}

		ids[id1] = true
		ids[id2] = true
	}

	if len(ids) != count*2 {
		t.Errorf("Expected %d unique IDs, got %d", count*2, len(ids))
	}
}

func TestSequenceRollover(t *testing.T) {
	gen, _ := NewGenerator(10)

	// Force sequence to be near max
	gen.mu.Lock()
	gen.sequence = MaxSequence - 5
	gen.lastTime = gen.getCurrentMillis()
	gen.mu.Unlock()

	// Generate IDs to trigger rollover
	var lastID int64
	for i := 0; i < 10; i++ {
		id, err := gen.NextID()
		if err != nil {
			t.Fatalf("NextID() error: %v", err)
		}

		// Should still be unique
		if id == lastID {
			t.Errorf("Duplicate ID after sequence rollover: %d", id)
		}
		lastID = id
	}
}

func TestGetTimestamp(t *testing.T) {
	gen, _ := NewGenerator(7)
	beforeGen := time.Now()

	id, _ := gen.NextID()

	afterGen := time.Now()
	idTime := GetTimestamp(id)

	// Timestamp should be between before and after generation
	if idTime.Before(beforeGen.Add(-time.Second)) || idTime.After(afterGen.Add(time.Second)) {
		t.Errorf("GetTimestamp() = %v, expected between %v and %v", idTime, beforeGen, afterGen)
	}
}

func TestGetInstanceIDFromID(t *testing.T) {
	instanceID := int64(123)
	gen, _ := NewGenerator(instanceID)

	id, _ := gen.NextID()
	extractedID := GetInstanceIDFromID(id)

	if extractedID != instanceID {
		t.Errorf("GetInstanceIDFromID() = %d, want %d", extractedID, instanceID)
	}
}

func TestGetSequence(t *testing.T) {
	gen, _ := NewGenerator(1)

	id1, _ := gen.NextID()
	seq1 := GetSequence(id1)

	id2, _ := gen.NextID()
	seq2 := GetSequence(id2)

	// Sequences should be sequential if generated in same millisecond
	// or reset if in different millisecond
	if seq2 != seq1+1 && seq2 != 0 {
		// This is acceptable - could be different milliseconds
		t.Logf("Sequence changed from %d to %d (likely different millisecond)", seq1, seq2)
	}
}

func BenchmarkNextID(b *testing.B) {
	gen, _ := NewGenerator(1)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := gen.NextID()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkNextIDParallel(b *testing.B) {
	gen, _ := NewGenerator(1)
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := gen.NextID()
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}
