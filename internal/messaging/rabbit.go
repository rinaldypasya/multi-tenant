// internal/messaging/rabbit.go
package messaging

import (
	"fmt"
	"log"
	"multi-tenant/internal/metrics"

	"github.com/streadway/amqp"
)

type RabbitClient struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	URL     string
}

func NewRabbitClient(url string) (*RabbitClient, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}
	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to create channel: %w", err)
	}

	return &RabbitClient{
		conn:    conn,
		channel: ch,
		URL:     url,
	}, nil
}

func (r *RabbitClient) GetChannel() *amqp.Channel {
	return r.channel
}

func (r *RabbitClient) GetConnection() *amqp.Connection {
	return r.conn
}

// DeclareQueue creates a tenant-specific durable queue
func (r *RabbitClient) DeclareQueue(tenantID string) error {
	queueName := fmt.Sprintf("tenant_%s_queue", tenantID)
	dlqName := fmt.Sprintf("tenant_%s_dlq", tenantID)

	// 1. DLQ
	_, err := r.channel.QueueDeclare(
		dlqName,
		true, false, false, false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("declare DLQ: %w", err)
	}

	// 2. Main Queue with DLQ binding
	args := amqp.Table{
		"x-dead-letter-exchange":    "",
		"x-dead-letter-routing-key": dlqName,
	}
	_, err = r.channel.QueueDeclare(
		queueName,
		true, false, false, false,
		args,
	)
	if err != nil {
		return fmt.Errorf("declare main queue: %w", err)
	}

	log.Printf("[Rabbit] Queues declared for tenant %s", tenantID)
	return nil
}

// Publish sends a message to the specified tenant queue
func (r *RabbitClient) Publish(tenantID string, body []byte) error {
	queueName := fmt.Sprintf("tenant_%s_queue", tenantID)
	err := r.channel.Publish(
		"",        // default exchange
		queueName, // routing key (queue name)
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to publish to queue %s: %w", queueName, err)
	}
	return nil
}

// Close cleans up connection and channel
func (r *RabbitClient) Close() error {
	if err := r.channel.Close(); err != nil {
		return err
	}
	if err := r.conn.Close(); err != nil {
		return err
	}
	return nil
}

func (r *RabbitClient) UpdateQueueDepth(tenantID string) {
	queueName := fmt.Sprintf("tenant_%s_queue", tenantID)

	q, err := r.channel.QueueInspect(queueName)
	if err != nil {
		log.Printf("[Rabbit] Failed to inspect queue for %s: %v", tenantID, err)
		return
	}

	metrics.QueueDepth.WithLabelValues(tenantID).Set(float64(q.Messages))
}
