// internal/model/message.go
package model

import (
	"time"

	"github.com/google/uuid"
)

type Message struct {
	ID        uuid.UUID `db:"id"`
	TenantID  uuid.UUID `db:"tenant_id"`
	Payload   []byte    `db:"payload"`
	CreatedAt time.Time `db:"created_at"`
}
