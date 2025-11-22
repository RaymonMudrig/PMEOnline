package websocket

import (
	"context"
	"log"
)

// Hub maintains active WebSocket clients and broadcasts messages
type Hub struct {
	// Registered clients
	clients map[*Client]bool

	// Inbound messages from clients
	broadcast chan []byte

	// Register requests from clients
	register chan *Client

	// Unregister requests from clients
	unregister chan *Client
}

// NewHub creates a new Hub
func NewHub() *Hub {
	return &Hub{
		broadcast:  make(chan []byte, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
	}
}

// Run starts the hub event loop
func (h *Hub) Run(ctx context.Context) {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
			log.Printf("[WS-HUB] Client registered: %s (total: %d)", client.id, len(h.clients))

		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
				log.Printf("[WS-HUB] Client unregistered: %s (total: %d)", client.id, len(h.clients))
			}

		case message := <-h.broadcast:
			// Send message to all connected clients
			for client := range h.clients {
				select {
				case client.send <- message:
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

// Broadcast sends a message to all connected clients
func (h *Hub) Broadcast(message []byte) {
	h.broadcast <- message
}

// ClientCount returns the number of connected clients
func (h *Hub) ClientCount() int {
	return len(h.clients)
}
