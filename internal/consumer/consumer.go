// internal/consumer/consumer.go
package consumer

import (
	"fmt"
	"log"
	"multi-tenant/internal/worker"

	"github.com/streadway/amqp"
)

type MessageHandlerFunc func(tenantID string, delivery amqp.Delivery)

// Consumer holds control channels and metadata for a running tenant consumer
type Consumer struct {
	TenantID    string
	QueueName   string
	Channel     *amqp.Channel
	StopChan    chan struct{}
	DoneChan    chan struct{}
	Handler     MessageHandlerFunc
	ConsumerTag string
	Pool        *worker.WorkerPool
}

// StartConsumer starts a goroutine that consumes messages for a tenant
func StartConsumer(conn *amqp.Connection, tenantID string, handler MessageHandlerFunc) (*Consumer, error) {
	ch, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("tenant %s: failed to open channel: %w", tenantID, err)
	}

	queueName := fmt.Sprintf("tenant_%s_queue", tenantID)
	consumerTag := fmt.Sprintf("consumer-%s", tenantID)

	msgs, err := ch.Consume(
		queueName,
		consumerTag,
		false, // autoAck: false to handle manually
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("tenant %s: failed to start consuming: %w", tenantID, err)
	}

	c := &Consumer{
		TenantID:    tenantID,
		QueueName:   queueName,
		Channel:     ch,
		StopChan:    make(chan struct{}),
		DoneChan:    make(chan struct{}),
		Handler:     handler,
		ConsumerTag: consumerTag,
	}

	go c.consumeLoop(msgs)

	log.Printf("Started consumer for tenant %s", tenantID)
	return c, nil
}

// consumeLoop processes messages until StopChan is closed
func (c *Consumer) consumeLoop(msgs <-chan amqp.Delivery) {
	defer func() {
		close(c.DoneChan)
	}()

	for {
		select {
		case msg, ok := <-msgs:
			if !ok {
				log.Printf("Tenant %s: delivery channel closed", c.TenantID)
				return
			}
			c.Handler(c.TenantID, msg)

		case <-c.StopChan:
			log.Printf("Stopping consumer for tenant %s...", c.TenantID)
			_ = c.Channel.Cancel(c.ConsumerTag, false)
			return
		}
	}
}

// Stop signals the consumer to stop and waits for cleanup
func (c *Consumer) Stop() {
	close(c.StopChan)
	<-c.DoneChan
	_ = c.Channel.Close()
	log.Printf("Stopped consumer for tenant %s", c.TenantID)
}

func (c *Consumer) SetWorkerCount(n int) {
	c.Pool.SetWorkerCount(n)
}
