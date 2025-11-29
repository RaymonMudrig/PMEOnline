package websocket

import (
	"sync"
)

// SequencedNotification represents a notification with a sequence number
type SequencedNotification struct {
	Sequence uint64 `json:"seq"`
	Type     string `json:"event_type"`
	Data     map[string]interface{} `json:"data"`
}

// NotificationBuffer maintains a ring buffer of recent notifications
type NotificationBuffer struct {
	mu           sync.RWMutex
	buffer       []SequencedNotification
	capacity     int
	nextSequence uint64
	oldestSeq    uint64 // Sequence number of oldest message in buffer
}

// NewNotificationBuffer creates a new notification buffer
func NewNotificationBuffer(capacity int) *NotificationBuffer {
	return &NotificationBuffer{
		buffer:       make([]SequencedNotification, 0, capacity),
		capacity:     capacity,
		nextSequence: 1, // Start from 1 (0 means "from beginning")
		oldestSeq:    1,
	}
}

// Add adds a notification to the buffer and returns the sequence number
func (nb *NotificationBuffer) Add(eventType string, data map[string]interface{}) uint64 {
	nb.mu.Lock()
	defer nb.mu.Unlock()

	seq := nb.nextSequence
	nb.nextSequence++

	notif := SequencedNotification{
		Sequence: seq,
		Type:     eventType,
		Data:     data,
	}

	// Add to buffer
	if len(nb.buffer) < nb.capacity {
		// Buffer not full yet
		nb.buffer = append(nb.buffer, notif)
	} else {
		// Buffer full, remove oldest and append new
		// This creates a ring buffer behavior
		nb.buffer = append(nb.buffer[1:], notif)
		nb.oldestSeq++
	}

	return seq
}

// GetFrom returns all notifications from a given sequence number (inclusive)
// Returns notifications and a boolean indicating if all requested messages are available
func (nb *NotificationBuffer) GetFrom(fromSeq uint64) ([]SequencedNotification, bool) {
	nb.mu.RLock()
	defer nb.mu.RUnlock()

	// If fromSeq is 0, client wants all available messages
	if fromSeq == 0 {
		fromSeq = nb.oldestSeq
	}

	// Check if requested sequence is too old (already evicted from buffer)
	if fromSeq < nb.oldestSeq {
		// Some messages are missing, return what we have
		result := make([]SequencedNotification, len(nb.buffer))
		copy(result, nb.buffer)
		return result, false
	}

	// Check if requested sequence is in the future
	if fromSeq >= nb.nextSequence {
		// No messages to send
		return []SequencedNotification{}, true
	}

	// Calculate the index in the buffer
	startIndex := int(fromSeq - nb.oldestSeq)

	// Return all messages from the start index
	result := make([]SequencedNotification, len(nb.buffer)-startIndex)
	copy(result, nb.buffer[startIndex:])

	return result, true
}

// GetLatestSequence returns the latest sequence number
func (nb *NotificationBuffer) GetLatestSequence() uint64 {
	nb.mu.RLock()
	defer nb.mu.RUnlock()
	return nb.nextSequence - 1
}

// GetOldestSequence returns the oldest sequence number in buffer
func (nb *NotificationBuffer) GetOldestSequence() uint64 {
	nb.mu.RLock()
	defer nb.mu.RUnlock()
	return nb.oldestSeq
}

// GetBufferInfo returns buffer statistics
func (nb *NotificationBuffer) GetBufferInfo() (size int, capacity int, oldest uint64, latest uint64) {
	nb.mu.RLock()
	defer nb.mu.RUnlock()
	return len(nb.buffer), nb.capacity, nb.oldestSeq, nb.nextSequence - 1
}
