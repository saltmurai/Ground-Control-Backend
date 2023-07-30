package router

import (
	"log"
	"net/http"
	"os"

	"github.com/gorilla/websocket"
	"github.com/streadway/amqp"
	"go.uber.org/zap"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow any origin for WebSocket connections
	},
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Upgrade HTTP request to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		zap.L().Sugar().Error(err)
		return
	}

	// Connect to RabbitMQ
	connRabbitMQ, err := amqp.Dial(os.Getenv("AMQP_URL"))
	if err != nil {
		zap.L().Sugar().Error(err)
		return
	}
	defer connRabbitMQ.Close()

	// Create a channel and declare a queue
	channel, err := connRabbitMQ.Channel()
	if err != nil {
		zap.L().Sugar().Error(err)
		return
	}
	defer channel.Close()

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
		log.Println("RabbitMQ consume error:", err)
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
		zap.L().Sugar().Error(err)
		//close websocket
		conn.Close()
		return
	}
}
