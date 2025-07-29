// internal/model/tenant.go
package model

import (
	"time"

	"github.com/google/uuid"
)

type Tenant struct {
	ID          uuid.UUID `db:"id"`
	Name        string    `json:"name"`
	Concurrency int       `json:"concurrency"`
	CreatedAt   time.Time `db:"created_at"`
}
