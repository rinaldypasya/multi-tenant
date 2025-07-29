# 📨 Multi-Tenant Messaging System (Go + RabbitMQ + PostgreSQL)

This project is a scalable multi-tenant message processing platform built in Go, designed for concurrent processing of messages using RabbitMQ and PostgreSQL. It supports dynamic per-tenant concurrency, JWT-secured APIs, cursor-based pagination, and Prometheus monitoring.

---

## ✨ Features

- ✅ Multi-tenant architecture
- ✅ Dynamic worker pool per tenant
- ✅ RabbitMQ + PostgreSQL integration
- ✅ JWT authentication
- ✅ Swagger docs
- ✅ Dead-letter queue retry logic
- ✅ Prometheus metrics and queue depth monitoring

---

## 📦 Requirements

- Go 1.22+
- PostgreSQL 13+
- RabbitMQ 3.8+
- [`golang-migrate`](https://github.com/golang-migrate/migrate) CLI
- Docker (optional for local development)

---

## 🛠 Setup

### 1. Clone the Repo

```bash
git clone git@github.com:rinaldypasya/multi-tenant.git
cd multi-tenant
````

### 2. Copy Environment Config

```bash
cp config/config.example.yaml config/config.yaml
```

---

## 🗂 Project Structure

```
.
├── cmd/              # App entry point
├── internal/
│   ├── api/          # REST API
│   ├── auth/         # JWT logic
│   ├── config/       # Config loader
│   ├── consumer/     # Tenant worker consumer
│   ├── messaging/    # RabbitMQ wrapper
│   ├── migration/    # SQL migrations
│   ├── model/        # Shared models
│   ├── storage/      # PostgreSQL interaction
│   ├── tenant/       # Tenant manager
│   ├── worker/       # Worker pool
├── docs/             # Swagger-generated files
├── docker-compose.yml
```

---

## 🔃 Database Migrations

We use [`golang-migrate`](https://github.com/golang-migrate/migrate) to manage schema.

### 1. Install Migrate CLI

```bash
brew install golang-migrate
# or
curl -L https://github.com/golang-migrate/migrate/releases/latest/download/migrate.linux-amd64.tar.gz | tar xvz
sudo mv migrate /usr/local/bin
```

### 2. Create Migration

```bash
migrate create -ext sql -dir db/migrations -seq add_tenants_table
```

### 3. Run Migration

```bash
migrate -path db/migrations \
  -database "postgres://user:password@localhost:5432/yourdb?sslmode=disable" \
  up
```

---

## 📘 Swagger API Docs

This project uses [Swaggo](https://github.com/swaggo/swag) to generate OpenAPI 3 docs.

### 1. Install `swag` CLI

```bash
go install github.com/swaggo/swag/cmd/swag@latest
export PATH="$PATH:$(go env GOPATH)/bin"
```

### 2. Generate Swagger Files

```bash
swag init -g cmd/main.go -o ./docs
```

### 3. Access Swagger UI

Start the app and visit:

```
http://localhost:8080/swagger/index.html
```

---

## 🧪 Testing with Postman

You can import the generated Swagger file into Postman:

```bash
# after swag init
docs/swagger.json
```

Postman → `Import` → `Upload Files` → choose `swagger.json`.

---

## 🐳 Running with Docker Compose

```bash
docker-compose up --build
```

* PostgreSQL: `localhost:5432`
* RabbitMQ: `localhost:5672`, dashboard on `localhost:15672` (user/pass: guest/guest)
* App: `localhost:8080`
