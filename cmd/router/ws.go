package router

import (
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/saltmurai/drone-api-service/cmd/database"
	"go.uber.org/zap"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow any origin for WebSocket connections
	},
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	zap.L().Sugar().Info("WebSocket connection opened")
	// Upgrade HTTP request to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		zap.L().Sugar().Error(err)
		return
	}

	// Connect to RabbitMQ
	channel := database.GetChannel()
	queue, err := channel.QueueDeclare(
		"log", // Queue name
		false, // Durable
		false, // Auto-deleted
		false, // Exclusive
		false, // No-wait
		nil,   // Arguments
	)
	if err != nil {
		zap.L().Sugar().Error(err)
		return
	}

	// Consume messages from RabbitMQ
	messages, err := channel.Consume(
		queue.Name, // Queue name
		"",         // Consumer name
		true,       // Auto-acknowledge messages
		false,      // Exclusive
		false,      // No-local
		false,      // No-wait
		nil,        // Arguments
	)
	if err != nil {
		zap.L().Sugar().Error(err)
		return
	}

	// Forward RabbitMQ messages to WebSocket clients
	go func() {
		for message := range messages {
			err = conn.WriteMessage(websocket.TextMessage, message.Body)
			if err != nil {
				zap.L().Sugar().Error(err)
				break
			}
		}
	}()

	// Wait for WebSocket connection to close
	_, _, err = conn.ReadMessage()
	if err != nil {
		// Check if the error is "websocket: close 1001 (going away)"
		closeErr, ok := err.(*websocket.CloseError)
		if ok && closeErr.Code == websocket.CloseGoingAway {
			zap.L().Sugar().Info("WebSocket connection closed: going away")
			// Perform any necessary cleanup or handling for the "going away" scenario
		} else {
			zap.L().Sugar().Error(err)
		}

		// Close the WebSocket connection
		conn.Close()
		return
	}
}
