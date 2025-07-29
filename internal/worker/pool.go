package worker

import (
	"encoding/json"
	"log"

	"github.com/streadway/amqp"

	"multi-tenant/internal/messaging"
	"multi-tenant/internal/metrics"
)

type WorkerPool struct {
	tenantID string
	conn     *amqp.Connection
	ch       *amqp.Channel
	stopCh   chan struct{}
	workers  int
}

func NewWorkerPool(tenantID string, rabbit *messaging.RabbitClient, workerCount int) *WorkerPool {
	return &WorkerPool{
		tenantID: tenantID,
		conn:     rabbit.GetConnection(),
		ch:       rabbit.GetChannel(),
		stopCh:   make(chan struct{}),
		workers:  workerCount,
	}
}

func (wp *WorkerPool) Start() {
	log.Printf("[Worker] Starting pool for tenant %s", wp.tenantID)

	msgs, err := wp.ch.Consume(
		"tenant_"+wp.tenantID+"_queue",
		"",
		false, // manual ack
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Fatalf("Failed to register consumer: %v", err)
	}

	go func() {
		metrics.WorkerActive.WithLabelValues(wp.tenantID).Add(1)
		defer metrics.WorkerActive.WithLabelValues(wp.tenantID).Sub(1)

		for {
			select {
			case <-wp.stopCh:
				log.Printf("[Worker] Stopping pool for tenant %s", wp.tenantID)
				return
			case msg, ok := <-msgs:
				if !ok {
					return
				}

				// Simulate processing
				err := wp.handleMessage(msg)
				if err != nil {
					log.Printf("Failed to process message: %v", err)
					_ = msg.Reject(false) // send to DLQ
					continue
				}

				_ = msg.Ack(false)
				metrics.WorkerProcessed.WithLabelValues(wp.tenantID).Inc()
			}
		}
	}()
}

func (wp *WorkerPool) Stop() {
	close(wp.stopCh)
}

func (wp *WorkerPool) handleMessage(msg amqp.Delivery) error {
	// Simulate decoding JSON payload
	var payload map[string]interface{}
	if err := json.Unmarshal(msg.Body, &payload); err != nil {
		log.Printf("[Worker] Failed to parse message: %v", err)
		return err // will trigger DLQ if reject(false)
	}
	log.Printf("[Worker] Tenant %s processed message: %v", wp.tenantID, payload)
	return nil
}

// SetWorkerCount updates the worker pool to use a new concurrency level
func (wp *WorkerPool) SetWorkerCount(n int) {
	if n <= 0 || n == wp.workers {
		return
	}

	log.Printf("[Worker][%s] Rescaling worker pool: %d → %d", wp.tenantID, wp.workers, n)

	// Stop existing workers
	wp.Stop()

	// Update count and restart
	wp.workers = n
	wp.stopCh = make(chan struct{}) // recreate stop channel
	wp.Start()
}
