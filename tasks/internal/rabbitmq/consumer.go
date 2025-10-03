package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/moabdelazem/microservices/tasks/internal/database"
	"github.com/moabdelazem/microservices/tasks/internal/models"
	"github.com/streadway/amqp"
)

type Consumer struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	db      *database.DB
}

// NewConsumer creates a new RabbitMQ consumer
func NewConsumer(db *database.DB) (*Consumer, error) {
	conn, err := amqp.Dial(os.Getenv("RABBITMQ_URL"))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	channel, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	// Declare exchange
	exchange := os.Getenv("RABBITMQ_EXCHANGE")
	err = channel.ExchangeDeclare(
		exchange, // name
		"topic",  // type
		true,     // durable
		false,    // auto-deleted
		false,    // internal
		false,    // no-wait
		nil,      // arguments
	)
	if err != nil {
		channel.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to declare exchange: %w", err)
	}

	// Declare queue
	queueName := os.Getenv("RABBITMQ_QUEUE")
	queue, err := channel.QueueDeclare(
		queueName, // name
		true,      // durable
		false,     // delete when unused
		false,     // exclusive
		false,     // no-wait
		nil,       // arguments
	)
	if err != nil {
		channel.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to declare queue: %w", err)
	}

	// Bind queue to exchange
	err = channel.QueueBind(
		queue.Name,     // queue name
		"user.created", // routing key
		exchange,       // exchange
		false,
		nil,
	)
	if err != nil {
		channel.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to bind queue: %w", err)
	}

	// Also bind to user.updated
	err = channel.QueueBind(
		queue.Name,     // queue name
		"user.updated", // routing key
		exchange,       // exchange
		false,
		nil,
	)
	if err != nil {
		channel.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to bind queue to user.updated: %w", err)
	}

	log.Printf("✅ Connected to RabbitMQ, listening on queue: %s\n", queueName)

	return &Consumer{
		conn:    conn,
		channel: channel,
		db:      db,
	}, nil
}

// Start begins consuming messages
func (c *Consumer) Start(ctx context.Context) error {
	queueName := os.Getenv("RABBITMQ_QUEUE")
	msgs, err := c.channel.Consume(
		queueName, // queue
		"",        // consumer
		false,     // auto-ack
		false,     // exclusive
		false,     // no-local
		false,     // no-wait
		nil,       // args
	)
	if err != nil {
		return fmt.Errorf("failed to register consumer: %w", err)
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				log.Println("Stopping RabbitMQ consumer...")
				return
			case msg, ok := <-msgs:
				if !ok {
					log.Println("RabbitMQ channel closed")
					return
				}
				c.handleMessage(msg)
			}
		}
	}()

	return nil
}

// handleMessage processes incoming messages
func (c *Consumer) handleMessage(msg amqp.Delivery) {
	var event models.UserEvent
	if err := json.Unmarshal(msg.Body, &event); err != nil {
		log.Printf("❌ Failed to unmarshal message: %v\n", err)
		msg.Nack(false, false)
		return
	}

	log.Printf("📨 Received event: %s for user %s (%s)\n", msg.RoutingKey, event.Username, event.UserID)

	switch msg.RoutingKey {
	case "user.created", "user.updated":
		if err := c.cacheUser(event); err != nil {
			log.Printf("❌ Failed to cache user: %v\n", err)
			msg.Nack(false, true) // Requeue
			return
		}
	}

	msg.Ack(false)
}

// cacheUser inserts or updates user in local cache
func (c *Consumer) cacheUser(event models.UserEvent) error {
	query := `
		INSERT INTO tasks_users (user_id, username, email, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (user_id) 
		DO UPDATE SET 
			username = EXCLUDED.username,
			email = EXCLUDED.email,
			updated_at = EXCLUDED.updated_at
	`

	_, err := c.db.Exec(query, event.UserID, event.Username, event.Email, time.Now(), time.Now())
	if err != nil {
		return fmt.Errorf("failed to cache user: %w", err)
	}

	log.Printf("✅ User %s (%s) cached successfully\n", event.Username, event.UserID)
	return nil
}

// Close closes the RabbitMQ connection
func (c *Consumer) Close() error {
	if c.channel != nil {
		c.channel.Close()
	}
	if c.conn != nil {
		c.conn.Close()
	}
	log.Println("RabbitMQ connection closed")
	return nil
}
