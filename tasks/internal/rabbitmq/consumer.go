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

// NewConsumer creates a new RabbitMQ consumer with retry logic
func NewConsumer(db *database.DB) (*Consumer, error) {
	var conn *amqp.Connection
	var err error

	// Retry connection up to 10 times with exponential backoff
	// Get maxRetries from environment variable or use default
	const defaultMaxRetries = 10
	maxRetries := defaultMaxRetries
	if val := os.Getenv("RABBITMQ_MAX_RETRIES"); val != "" {
		if parsed, err := fmt.Sscanf(val, "%d", &maxRetries); err != nil || parsed != 1 || maxRetries < 1 {
			maxRetries = defaultMaxRetries
		}
	}

	for i := 0; i < maxRetries; i++ {
		conn, err = amqp.Dial(os.Getenv("RABBITMQ_URL"))
		if err == nil {
			break
		}

		if i < maxRetries-1 {
			waitTime := time.Duration(i+1) * time.Second
			log.Printf("⚠️  Failed to connect to RabbitMQ (attempt %d/%d), retrying in %v...", i+1, maxRetries, waitTime)
			time.Sleep(waitTime)
		}
	}

	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ after %d attempts: %w", maxRetries, err)
	}

	log.Println("✅ Connected to RabbitMQ")

	channel, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	// Get exchange name with default
	exchange := os.Getenv("RABBITMQ_EXCHANGE")
	if exchange == "" {
		exchange = "auth_events" // Default exchange name
	}

	// Declare exchange
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

	log.Printf("✅ Declared exchange: %s", exchange)

	// Get queue name with default
	queueName := os.Getenv("RABBITMQ_QUEUE")
	if queueName == "" {
		queueName = "tasks-service-queue" // Default queue name
	}

	// Declare queue
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

	log.Printf("✅ Declared queue: %s", queue.Name)

	// Bind queue to exchange for user.created events
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

	log.Printf("✅ Bound queue to exchange with routing key: user.created")

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
