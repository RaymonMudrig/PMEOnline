package websocket

import (
	"context"
	"encoding/json"
	"log"
)

// Hub maintains active WebSocket clients and broadcasts messages
type Hub struct {
	// Registered clients
	clients map[*Client]bool

	// Inbound messages from clients
	broadcast chan SequencedNotification

	// Register requests from clients
	register chan *Client

	// Unregister requests from clients
	unregister chan *Client

	// Notification buffer for DropCopy functionality
	buffer *NotificationBuffer
}

// NewHub creates a new Hub
func NewHub() *Hub {
	return &Hub{
		broadcast:  make(chan SequencedNotification, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
		buffer:     NewNotificationBuffer(10000), // Store last 10,000 notifications
	}
}

// Run starts the hub event loop
func (h *Hub) Run(ctx context.Context) {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
			log.Printf("[WS-HUB] Client registered: %s (total: %d)", client.id, len(h.clients))

			// Send recovery messages only if client has sent subscribe message
			if client.hasSubscribed {
				go h.sendRecoveryMessages(client)
			}

		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
				log.Printf("[WS-HUB] Client unregistered: %s (total: %d)", client.id, len(h.clients))
			}

		case notification := <-h.broadcast:
			// Marshal notification to JSON
			jsonData, err := json.Marshal(notification)
			if err != nil {
				log.Printf("[WS-HUB] Failed to marshal notification: %v", err)
				continue
			}

			// Send message to all connected clients
			for client := range h.clients {
				select {
				case client.send <- jsonData:
					// Message sent successfully
				default:
					// Client's send buffer is full, disconnect
					close(client.send)
					delete(h.clients, client)
					log.Printf("[WS-HUB] Client disconnected (buffer full): %s", client.id)
				}
			}

		case <-ctx.Done():
			log.Println("[WS-HUB] Shutting down hub...")
			// Close all client connections
			for client := range h.clients {
				close(client.send)
			}
			return
		}
	}
}

// sendRecoveryMessages sends buffered messages to a newly connected client
func (h *Hub) sendRecoveryMessages(client *Client) {
	// Log buffer state
	size, capacity, oldest, latest := h.buffer.GetBufferInfo()
	log.Printf("[WS-HUB] Buffer state: size=%d/%d, oldest_seq=%d, latest_seq=%d",
		size, capacity, oldest, latest)

	// Get notifications from requested sequence (0 means from oldest available)
	notifications, allAvailable := h.buffer.GetFrom(client.requestedSeq)

	if !allAvailable {
		log.Printf("[WS-HUB] Client %s requested seq %d, but oldest available is %d",
			client.id, client.requestedSeq, h.buffer.GetOldestSequence())
	}

	log.Printf("[WS-HUB] Sending %d recovery messages to %s (from seq %d)",
		len(notifications), client.id, client.requestedSeq)

	// Send recovery header
	header := map[string]interface{}{
		"type":           "recovery_start",
		"requested_seq":  client.requestedSeq,
		"oldest_seq":     h.buffer.GetOldestSequence(),
		"latest_seq":     h.buffer.GetLatestSequence(),
		"count":          len(notifications),
		"all_available":  allAvailable,
	}
	headerJSON, _ := json.Marshal(header)
	client.send <- headerJSON

	// Send all recovery messages
	for _, notif := range notifications {
		jsonData, err := json.Marshal(notif)
		if err != nil {
			log.Printf("[WS-HUB] Failed to marshal recovery notification: %v", err)
			continue
		}
		client.send <- jsonData
	}

	// Send recovery complete message
	complete := map[string]interface{}{
		"type": "recovery_complete",
		"count": len(notifications),
		"latest_seq": h.buffer.GetLatestSequence(),
	}
	completeJSON, _ := json.Marshal(complete)
	client.send <- completeJSON

	log.Printf("[WS-HUB] Recovery complete for %s", client.id)
}

// sendBufferInfo sends buffer statistics to client
func (h *Hub) sendBufferInfo(client *Client) {
	size, capacity, oldest, latest := h.buffer.GetBufferInfo()
	info := map[string]interface{}{
		"type":     "buffer_info",
		"size":     size,
		"capacity": capacity,
		"oldest_seq": oldest,
		"latest_seq": latest,
	}
	infoJSON, _ := json.Marshal(info)
	client.send <- infoJSON
}

// BroadcastNotification adds a notification to buffer and broadcasts it
func (h *Hub) BroadcastNotification(eventType string, data map[string]interface{}) uint64 {
	// Add to buffer and get sequence number
	seq := h.buffer.Add(eventType, data)

	// Create sequenced notification
	notif := SequencedNotification{
		Sequence: seq,
		Type:     eventType,
		Data:     data,
	}

	// Broadcast to all clients
	h.broadcast <- notif

	return seq
}

// ClientCount returns the number of connected clients
func (h *Hub) ClientCount() int {
	return len(h.clients)
}

// GetBufferInfo returns buffer statistics
func (h *Hub) GetBufferInfo() (size int, capacity int, oldest uint64, latest uint64) {
	return h.buffer.GetBufferInfo()
}
