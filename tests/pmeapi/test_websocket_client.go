package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/websocket"
)

// Notification represents a WebSocket notification message
type Notification struct {
	Type      string                 `json:"type"`
	Timestamp int64                  `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
}

func main() {
	// WebSocket URL
	wsURL := "ws://localhost:8080/ws/notifications"

	log.Printf("Connecting to WebSocket server at %s...", wsURL)

	// Create interrupt signal handler
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	// Connect to WebSocket
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		log.Fatal("Failed to connect to WebSocket:", err)
	}
	defer conn.Close()

	log.Println("✓ Connected to WebSocket server")
	log.Println("Listening for notifications... (Press Ctrl+C to exit)")
	log.Println("========================================")

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Channel for received messages
	done := make(chan struct{})

	// Read messages in a goroutine
	go func() {
		defer close(done)
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Printf("WebSocket error: %v", err)
				}
				return
			}

			// Parse notification
			var notif Notification
			if err := json.Unmarshal(message, &notif); err != nil {
				log.Printf("Failed to parse notification: %v", err)
				log.Printf("Raw message: %s", string(message))
				continue
			}

			// Format timestamp
			timestamp := time.UnixMilli(notif.Timestamp).Format("2006-01-02 15:04:05")

			// Print notification
			log.Println("----------------------------------------")
			log.Printf("Type: %s", notif.Type)
			log.Printf("Time: %s", timestamp)
			log.Println("Data:")

			// Pretty print data
			dataJSON, _ := json.MarshalIndent(notif.Data, "  ", "  ")
			log.Printf("  %s", string(dataJSON))
		}
	}()

	// Handle graceful shutdown
	select {
	case <-done:
		log.Println("Connection closed by server")
	case <-interrupt:
		log.Println("\nInterrupt signal received, closing connection...")

		// Send close message
		err := conn.WriteMessage(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		if err != nil {
			log.Println("Error sending close message:", err)
			return
		}

		// Wait for server to close connection or timeout
		select {
		case <-done:
		case <-time.After(time.Second):
		}
	case <-ctx.Done():
		log.Println("Context cancelled, closing connection...")
	}

	log.Println("WebSocket client stopped")
}
