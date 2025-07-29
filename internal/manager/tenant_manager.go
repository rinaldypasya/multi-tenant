// internal/manager/tenant_manager.go
package manager

import (
	"fmt"
	"log"
	"sync"

	"github.com/google/uuid"
	"github.com/streadway/amqp"

	"multi-tenant/internal/consumer"
	"multi-tenant/internal/messaging"
	"multi-tenant/internal/model"
	"multi-tenant/internal/storage"
)

type TenantManager struct {
	rabbitConn *amqp.Connection
	rabbit     *messaging.RabbitClient
	storage    *storage.Storage

	mu        sync.RWMutex
	consumers map[uuid.UUID]*consumer.Consumer
}

func NewTenantManager(
	rabbitConn *amqp.Connection,
	rabbit *messaging.RabbitClient,
	storage *storage.Storage,
) *TenantManager {
	return &TenantManager{
		rabbitConn: rabbitConn,
		rabbit:     rabbit,
		storage:    storage,
		consumers:  make(map[uuid.UUID]*consumer.Consumer),
	}
}

// AddTenant creates a queue, a DB partition, and spawns the consumer
func (tm *TenantManager) AddTenant(tenantID uuid.UUID) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if _, exists := tm.consumers[tenantID]; exists {
		return nil // already exists
	}

	// Create DB partition
	if err := tm.storage.EnsurePartition(tenantID); err != nil {
		return err
	}

	// Declare RabbitMQ queue
	if err := tm.rabbit.DeclareQueue(tenantID.String()); err != nil {
		return err
	}

	// Start consumer
	c, err := consumer.StartConsumer(tm.rabbitConn, tenantID.String(), tm.handleMessage)
	if err != nil {
		return err
	}
	tm.consumers[tenantID] = c

	if err := tm.storage.CreateTenant(tenantID); err != nil {
		return fmt.Errorf("failed to save tenant: %w", err)
	}

	log.Printf("Tenant %s added and consumer started", tenantID)
	return nil
}

// RemoveTenant stops the consumer, deletes the queue, and removes from map
func (tm *TenantManager) RemoveTenant(tenantID uuid.UUID) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	c, exists := tm.consumers[tenantID]
	if !exists {
		return nil // nothing to remove
	}

	c.Stop()

	// Remove queue
	queueName := "tenant_" + tenantID.String() + "_queue"
	_, err := tm.rabbit.GetChannel().QueueDelete(queueName, false, false, false)
	if err != nil {
		log.Printf("Failed to delete queue %s: %v", queueName, err)
	}

	delete(tm.consumers, tenantID)

	if err := tm.storage.DeleteTenant(tenantID); err != nil {
		log.Printf("Failed to remove tenant record: %v", err)
	}

	log.Printf("Tenant %s removed and consumer stopped", tenantID)
	return nil
}

// Shutdown all tenants
func (tm *TenantManager) ShutdownAll() {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	for id, c := range tm.consumers {
		c.Stop()
		log.Printf("Stopped tenant %s", id)
	}
	tm.consumers = make(map[uuid.UUID]*consumer.Consumer)
}

// Handle incoming message (callback from consumer)
func (tm *TenantManager) handleMessage(tenantID string, msg amqp.Delivery) {
	tenantUUID, err := uuid.Parse(tenantID)
	if err != nil {
		log.Printf("Invalid tenant ID %s", tenantID)
		msg.Nack(false, false)
		return
	}

	m := &model.Message{
		ID:        uuid.New(),
		TenantID:  tenantUUID,
		Payload:   msg.Body,
		CreatedAt: msg.Timestamp,
	}
	if err := tm.storage.InsertMessage(m); err != nil {
		log.Printf("DB insert failed: %v", err)
		msg.Nack(false, false)
		return
	}

	msg.Ack(false)
}

// ListTenantIDs returns all currently registered tenant UUIDs
func (tm *TenantManager) ListTenantIDs() []string {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	ids := make([]string, 0, len(tm.consumers))
	for id := range tm.consumers {
		ids = append(ids, id.String())
	}
	return ids
}

func (tm *TenantManager) SetWorkerCount(tenantID string, n int) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	id, err := uuid.Parse(tenantID)
	if err != nil {
		return err
	}

	pool, ok := tm.consumers[id]
	if !ok {
		return fmt.Errorf("tenant not found: %s", tenantID)
	}

	// Update the worker pool
	pool.SetWorkerCount(n)

	// Persist concurrency level in DB
	if err := tm.storage.UpdateTenantConcurrency(tenantID, n); err != nil {
		return fmt.Errorf("failed to persist concurrency: %w", err)
	}
	return nil
}
