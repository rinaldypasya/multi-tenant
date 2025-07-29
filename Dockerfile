FROM golang:1.23-alpine AS builder

# Add Git (optional if using private repos)
RUN apk add --no-cache git

# Create app directory
WORKDIR /app

# Copy go mod files and download deps
COPY go.mod go.sum ./
RUN go mod download

# Copy app source and build statically
COPY . .
RUN CGO_ENABLED=0 go build -o app ./cmd/main.go

FROM alpine:3.20

# Add optional ca-certificates if your app calls HTTPS APIs
RUN apk add --no-cache ca-certificates

WORKDIR /app
COPY --from=builder /app/app /app/app
COPY config.yaml /app/config.yaml
COPY docs /app/docs

EXPOSE 8080
ENTRYPOINT ["/app/app"]
