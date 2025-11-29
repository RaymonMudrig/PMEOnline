package websocket

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer
	pongWait = 60 * time.Second

	// Send pings to peer with this period (must be less than pongWait)
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer
	maxMessageSize = 512
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Allow all origins for now
		// In production, implement proper origin checking
		return true
	},
}

// Client represents a WebSocket client
type Client struct {
	hub *Hub

	// The websocket connection
	conn *websocket.Conn

	// Buffered channel of outbound messages
	send chan []byte

	// Client ID for identification
	id string

	// Requested sequence number for recovery (0 = no recovery, just send buffer info)
	requestedSeq uint64
}

// ClientSubscribeMessage represents the initial message from client
type ClientSubscribeMessage struct {
	Type     string `json:"type"`      // "subscribe"
	FromSeq  uint64 `json:"from_seq"`  // Sequence number to start from (0 = from beginning)
}

// readPump pumps messages from the websocket connection to the hub
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("[WS-CLIENT] Unexpected close: %v", err)
			}
			break
		}

		log.Printf("[WS-CLIENT] Received message from %s: %s", c.id, message)

		// Parse subscribe message to get requested sequence
		var subMsg ClientSubscribeMessage
		if err := json.Unmarshal(message, &subMsg); err == nil && subMsg.Type == "subscribe" {
			c.requestedSeq = subMsg.FromSeq
			log.Printf("[WS-CLIENT] Client %s requesting from seq %d", c.id, c.requestedSeq)
			// Trigger recovery by re-registering
			c.hub.register <- c
		}
	}
}

// writePump pumps messages from the hub to the websocket connection
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued messages to the current websocket message
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// ServeWs handles websocket requests from the peer
func ServeWs(hub *Hub, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[WS] Failed to upgrade connection: %v", err)
		return
	}

	// Generate client ID (could use query params for account/participant filtering)
	clientID := r.RemoteAddr

	client := &Client{
		hub:  hub,
		conn: conn,
		send: make(chan []byte, 256),
		id:   clientID,
	}

	client.hub.register <- client

	// Start client goroutines
	go client.writePump()
	go client.readPump()

	log.Printf("[WS] Client connected: %s", clientID)
}
