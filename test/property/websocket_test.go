// +build property

package property

import (
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
	"github.com/mosiko1234/heimdal/sensor/internal/desktop/visualizer"
)

// Feature: monorepo-architecture, Property 5: Real-time Update Propagation
// Validates: Requirements 4.4
//
// For any new traffic event, the LocalVisualizer should send WebSocket updates
// to all connected clients within a reasonable time window.
func TestProperty_RealtimeUpdatePropagation(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("WebSocket hub broadcasts updates to all connected clients", prop.ForAll(
		func(updateType string, numClients int) bool {
			// Limit number of clients for testing
			if numClients < 1 || numClients > 10 {
				return true // Skip invalid inputs
			}

			// Create mock storage
			storage := &MockStorageForDevices{}

			// Create visualizer with mock storage
			vis, err := visualizer.NewVisualizer(&visualizer.Config{
				Port:        18080, // Use different port to avoid conflicts
				Storage:     storage,
				FeatureGate: nil,
			})
			if err != nil {
				t.Logf("Failed to create visualizer: %v", err)
				return false
			}

			// Start the visualizer (which starts the WebSocket hub and HTTP server)
			if err := vis.Start(); err != nil {
				t.Logf("Failed to start visualizer: %v", err)
				return false
			}
			defer vis.Stop()

			// Give the server time to start
			time.Sleep(100 * time.Millisecond)

			// WebSocket URL
			wsURL := "ws://localhost:18080/ws"

			// Connect multiple WebSocket clients
			clients := make([]*websocket.Conn, numClients)
			receivedChans := make([]chan *visualizer.UpdateMessage, numClients)

			for i := 0; i < numClients; i++ {
				conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
				if err != nil {
					t.Logf("Failed to connect client %d: %v", i, err)
					// Clean up already connected clients
					for j := 0; j < i; j++ {
						clients[j].Close()
					}
					return false
				}
				clients[i] = conn
				receivedChans[i] = make(chan *visualizer.UpdateMessage, 1)

				// Start goroutine to receive messages
				go func(idx int, c *websocket.Conn, ch chan *visualizer.UpdateMessage) {
					var msg visualizer.UpdateMessage
					err := c.ReadJSON(&msg)
					if err == nil {
						ch <- &msg
					}
				}(i, conn, receivedChans[i])
			}

			// Give clients time to connect and register with hub
			time.Sleep(100 * time.Millisecond)

			// Broadcast an update
			testPayload := map[string]interface{}{
				"test": "data",
				"id":   123,
			}
			vis.BroadcastUpdate(updateType, testPayload)

			// Wait for messages with timeout
			timeout := time.After(2 * time.Second)
			receivedCount := 0

			for i := 0; i < numClients; i++ {
				select {
				case msg := <-receivedChans[i]:
					// Verify message type matches
					if msg.Type != updateType {
						t.Logf("Client %d received wrong type: expected %s, got %s", i, updateType, msg.Type)
						// Clean up
						for _, c := range clients {
							c.Close()
						}
						return false
					}

					// Verify timestamp is present
					if msg.Timestamp == "" {
						t.Logf("Client %d received message without timestamp", i)
						// Clean up
						for _, c := range clients {
							c.Close()
						}
						return false
					}

					// Verify payload is present
					if msg.Payload == nil {
						t.Logf("Client %d received message without payload", i)
						// Clean up
						for _, c := range clients {
							c.Close()
						}
						return false
					}

					receivedCount++

				case <-timeout:
					t.Logf("Timeout waiting for message on client %d (received %d/%d)", i, receivedCount, numClients)
					// Clean up
					for _, c := range clients {
						c.Close()
					}
					return false
				}
			}

			// Clean up connections
			for _, c := range clients {
				c.Close()
			}

			// Verify all clients received the message
			if receivedCount != numClients {
				t.Logf("Not all clients received message: %d/%d", receivedCount, numClients)
				return false
			}

			return true
		},
		genUpdateType(),
		gen.IntRange(1, 5), // Test with 1-5 clients
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// genUpdateType generates a random update type
func genUpdateType() gopter.Gen {
	updateTypes := []string{"device", "traffic", "anomaly", "profile"}
	return gen.OneConstOf(
		updateTypes[0], updateTypes[1], updateTypes[2], updateTypes[3],
	)
}
