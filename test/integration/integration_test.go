// test/integration/integration_test.go
package integration

import (
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/ory/dockertest/v3"
	"github.com/streadway/amqp"
	"github.com/stretchr/testify/require"

	"multi-tenant/internal/manager"
	"multi-tenant/internal/messaging"
	"multi-tenant/internal/storage"
)

var (
	db         *storage.Storage
	rabbit     *messaging.RabbitClient
	rabbitConn *amqp.Connection
	tenantMgr  *manager.TenantManager
	dsn        string
	rabbitURL  string
)

func TestMain(m *testing.M) {
	// Create Docker pool
	pool, err := dockertest.NewPool("")
	if err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}

	// PostgreSQL
	dbResource, err := pool.Run("postgres", "13", []string{
		"POSTGRES_USER=test",
		"POSTGRES_PASSWORD=test",
		"POSTGRES_DB=testdb",
	})
	require.NoError(nil, err)

	// RabbitMQ
	rmqResource, err := pool.Run("rabbitmq", "3-management", []string{})
	require.NoError(nil, err)

	// Wait for DB
	dsn = fmt.Sprintf("postgres://test:test@localhost:%s/testdb?sslmode=disable", dbResource.GetPort("5432/tcp"))
	err = pool.Retry(func() error {
		db, err = storage.NewStorage(dsn)
		if err != nil {
			return err
		}
		return db.DB.Ping()
	})
	require.NoError(nil, err)

	// Create tables
	_, _ = db.DB.Exec(`CREATE TABLE IF NOT EXISTS tenants (
		id UUID PRIMARY KEY,
		created_at TIMESTAMPTZ DEFAULT NOW()
	);
	CREATE TABLE IF NOT EXISTS messages (
		id UUID PRIMARY KEY,
		tenant_id UUID NOT NULL,
		payload JSONB,
		created_at TIMESTAMPTZ DEFAULT NOW()
	) PARTITION BY LIST (tenant_id);`)

	// Wait for RabbitMQ
	rabbitURL = fmt.Sprintf("amqp://guest:guest@localhost:%s/", rmqResource.GetPort("5672/tcp"))
	err = pool.Retry(func() error {
		rabbit, err = messaging.NewRabbitClient(rabbitURL)
		if err != nil {
			return err
		}
		rabbitConn = rabbit.GetConnection()
		return nil
	})
	require.NoError(nil, err)

	// Init TenantManager
	tenantMgr = manager.NewTenantManager(rabbitConn, rabbit, db)

	// Run tests
	code := m.Run()

	// Cleanup
	_ = pool.Purge(dbResource)
	_ = pool.Purge(rmqResource)
	os.Exit(code)
}

func TestTenantLifecycle(t *testing.T) {
	tenantID := uuid.New()

	// Add tenant
	err := tenantMgr.AddTenant(tenantID)
	require.NoError(t, err)

	// Declare queue and publish message
	err = rabbit.Publish(tenantID.String(), []byte(`{"event":"created"}`))
	require.NoError(t, err)

	// Wait and verify message in DB
	time.Sleep(500 * time.Millisecond)

	rows, err := db.DB.Query(`SELECT payload FROM messages WHERE tenant_id = $1`, tenantID)
	require.NoError(t, err)

	var count int
	for rows.Next() {
		count++
	}
	require.Equal(t, 1, count)

	// Remove tenant
	err = tenantMgr.RemoveTenant(tenantID)
	require.NoError(t, err)
}
