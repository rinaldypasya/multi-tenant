package main

import (
	"context"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"multi-tenant/internal/api"
	"multi-tenant/internal/auth"
	"multi-tenant/internal/config"
	"multi-tenant/internal/manager"
	"multi-tenant/internal/messaging"
	"multi-tenant/internal/metrics"
	"multi-tenant/internal/storage"
	"multi-tenant/internal/worker"
)

// @title Multi-Tenant Messaging API
// @version 1.0
// @description API for multi-tenant messaging system with per-tenant JWT
// @host localhost:8080
// @BasePath /
// @schemes http

// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name Authorization
func main() {
	// Init Metrics
	metrics.Init()

	// Load Configuration
	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	log.Println("Configuration loaded")

	// Setup JWT Secret
	auth.SetSecret(cfg.Auth.JWTSecret)

	// Init PostgreSQL
	db, err := storage.NewStorage(cfg.Database.URL)
	if err != nil {
		log.Fatalf("Failed to init DB: %v", err)
	}
	defer db.DB.Close()
	log.Println("PostgreSQL connected")

	// Init RabbitMQ
	rabbitClient, err := messaging.NewRabbitClient(cfg.RabbitMQ.URL)
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	defer rabbitClient.Close()
	log.Println("RabbitMQ connected")

	// Init TenantManager
	rabbitConn := rabbitClient.GetConnection() // use connection from exposed channel
	tm := manager.NewTenantManager(rabbitConn, rabbitClient, db)

	// Start background loop for updating queue depth metrics
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				for _, tenantID := range tm.ListTenantIDs() {
					rabbitClient.UpdateQueueDepth(tenantID)
				}
			}
		}
	}()

	// Recover Existing Tenants
	tenants, err := db.ListTenants()
	if err != nil {
		log.Fatalf("Failed to load tenants: %v", err)
	}

	for _, tenant := range tenants {
		if err := tm.AddTenant(tenant.ID); err != nil {
			log.Printf("⚠️ Failed to recover tenant %s: %v", tenant.ID, err)
			continue
		}

		// Optional: Start default worker pool
		pool := worker.NewWorkerPool(tenant.ID.String(), rabbitClient, cfg.Workers)
		pool.Start()
		log.Printf("🔁 Recovered tenant %s", tenant.ID)
	}

	// Init API
	apiHandler := api.NewAPI(tm, db, cfg)
	server := &http.Server{
		Addr:    ":8080",
		Handler: apiHandler.Router(),
	}

	// Graceful Shutdown Setup
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Println("🚀 Starting API server on port 8080")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	<-ctx.Done() // Wait for interrupt signal
	log.Println("Shutdown initiated...")

	// Shutdown sequence
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Stop HTTP server
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP shutdown error: %v", err)
	}

	// Stop all tenant consumers
	tm.ShutdownAll()

	log.Println("Graceful shutdown complete")
}
