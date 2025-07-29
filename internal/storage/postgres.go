// internal/storage/postgres.go
package storage

import (
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	_ "github.com/lib/pq"

	"multi-tenant/internal/model"
)

type Storage struct {
	DB *sql.DB
}

func NewStorage(dsn string) (*Storage, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open db: %w", err)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to connect to db: %w", err)
	}
	return &Storage{DB: db}, nil
}

// EnsurePartition creates a tenant partition if not exists
func (s *Storage) EnsurePartition(tenantID uuid.UUID) error {
	partitionName := fmt.Sprintf("messages_%s", tenantID.String())
	query := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s PARTITION OF messages
		FOR VALUES IN ('%s')`, partitionName, tenantID.String())

	_, err := s.DB.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to create partition: %w", err)
	}
	return nil
}

// InsertMessage inserts a message into the tenant's partition
func (s *Storage) InsertMessage(m *model.Message) error {
	query := `
		INSERT INTO messages (id, tenant_id, payload, created_at)
		VALUES ($1, $2, $3, $4)
	`
	_, err := s.DB.Exec(query, m.ID, m.TenantID, m.Payload, m.CreatedAt)
	return err
}

// ListMessagesPaginated retrieves messages using cursor-based pagination
func (s *Storage) ListMessagesPaginated(tenantID uuid.UUID, cursor string, limit int) ([]model.Message, string, error) {
	query := `
		SELECT id, tenant_id, payload, created_at
		FROM messages
		WHERE tenant_id = $1
		  AND ($2::uuid IS NULL OR id > $2::uuid)
		ORDER BY id
		LIMIT $3
	`

	var rows *sql.Rows
	var err error
	if cursor == "" {
		rows, err = s.DB.Query(query, tenantID, nil, limit)
	} else {
		rows, err = s.DB.Query(query, tenantID, cursor, limit)
	}
	if err != nil {
		return nil, "", fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	var messages []model.Message
	var lastID uuid.UUID
	for rows.Next() {
		var m model.Message
		if err := rows.Scan(&m.ID, &m.TenantID, &m.Payload, &m.CreatedAt); err != nil {
			return nil, "", fmt.Errorf("scan failed: %w", err)
		}
		lastID = m.ID
		messages = append(messages, m)
	}

	nextCursor := ""
	if len(messages) == limit {
		nextCursor = lastID.String()
	}

	return messages, nextCursor, nil
}

func (s *Storage) CreateTenant(id uuid.UUID) error {
	_, err := s.DB.Exec(`INSERT INTO tenants (id) VALUES ($1) ON CONFLICT DO NOTHING`, id)
	return err
}

func (s *Storage) DeleteTenant(id uuid.UUID) error {
	_, err := s.DB.Exec(`DELETE FROM tenants WHERE id = $1`, id)
	return err
}

func (s *Storage) ListTenants() ([]model.Tenant, error) {
	rows, err := s.DB.Query(`SELECT id, name, concurrency FROM tenants`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tenants []model.Tenant
	for rows.Next() {
		var t model.Tenant
		if err := rows.Scan(&t.ID, &t.Name, &t.Concurrency); err != nil {
			return nil, err
		}
		tenants = append(tenants, t)
	}
	return tenants, nil
}

func (s *Storage) UpdateTenantConcurrency(tenantID string, workers int) error {
	_, err := s.DB.Exec(`
		UPDATE tenants
		SET concurrency = $1
		WHERE id = $2
	`, workers, tenantID)
	return err
}
