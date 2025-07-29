Here's a complete `README.md` that includes:

✅ Project overview
✅ Requirements
✅ Setup instructions
✅ ✅ Migration steps using `golang-migrate`
✅ Running with Docker
✅ API overview

---

## 📘 `README.md`

````markdown
# Multi-Tenant Messaging System (Go + RabbitMQ + PostgreSQL)

This project is a scalable multi-tenant message processing system built in Go with:

- RabbitMQ for message queues
- PostgreSQL for metadata storage
- Dynamic worker pool per tenant
- JWT-based API authentication
- Cursor-based message listing
- Prometheus metrics
- Dead-letter queue support

---

## 🚀 Features

- Multi-tenant queue consumers
- Per-tenant concurrency control
- Graceful worker pool scaling
- JWT-authenticated API
- Swagger documentation
- PostgreSQL + RabbitMQ via Docker Compose
- Dead-letter retry queues
- Prometheus integration

---

## 📦 Requirements

- Go 1.22+
- PostgreSQL 13+
- RabbitMQ 3.8+
- [golang-migrate](https://github.com/golang-migrate/migrate) CLI
- Docker (optional for running locally)

---

## 🛠 Setup Instructions

### 1. Clone the Repo

```bash
git clone git@github.com:rinaldypasya/multi-tenant.git
cd multi-tenant
````

---

## 🗂 Project Structure

```
.
├── cmd/              # App entry point
├── config/           # Config loader
├── internal/
│   ├── api/          # REST API handlers
│   ├── auth/         # JWT logic
│   ├── consumer/     # Consumer wrapper
│   ├── messaging/    # RabbitMQ wrapper
|   ├── migration/    # SQL migrations
│   ├── model/        # Shared structs
│   ├── storage/      # DB access
│   ├── tenant/       # Tenant manager
│   ├── worker/       # Worker pool
├── docker-compose.yml
```

---

## 🔃 Database Migration Steps

We use [golang-migrate](https://github.com/golang-migrate/migrate) for database schema management.

### ✅ Install CLI

```bash
brew install golang-migrate
# or for Linux
curl -L https://github.com/golang-migrate/migrate/releases/latest/download/migrate.linux-amd64.tar.gz | tar xvz
sudo mv migrate /usr/local/bin
```

### ✅ Create a new migration

```bash
migrate create -ext sql -dir internal/migration -seq add_tenants_table
```

This will create:

```
db/migrations/
  000001_add_tenants_table.up.sql
  000001_add_tenants_table.down.sql
```

### ✅ Run migration

```bash
migrate -path internal/migration \
  -database "postgres://user:password@localhost:5432/yourdb?sslmode=disable" \
  up
```

### ✅ Rollback migration

```bash
migrate -path internal/migration \
  -database "postgres://user:password@localhost:5432/yourdb?sslmode=disable" \
  down 1
```

---

## 🐳 Running via Docker Compose

```bash
docker-compose up -d
```

This runs:

* PostgreSQL (port `5432`)
* RabbitMQ (port `5672`, dashboard on `15672`)
* App container (port `8080`)

---

## 🛡️ Authentication

Use `/auth/token` with a tenant UUID to get a JWT token:

```json
POST /auth/token
{
  "tenant_id": "your-tenant-uuid"
}
```

Use the token in `Authorization: Bearer` headers for all authenticated endpoints.

---
