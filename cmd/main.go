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

	_ "multi-tenant/docs" // swagger generated docs

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	httpSwagger "github.com/swaggo/http-swagger"
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

	// Load tenants from DB and start pools
	tenants, err := db.ListTenants()
	if err != nil {
		log.Fatalf("failed to list tenants: %v", err)
	}
	for _, t := range tenants {
		if err := tm.AddTenant(t.ID); err != nil {
			log.Printf("warn: add tenant %s: %v", t.ID, err)
			continue
		}
		if err := tm.SetWorkerCount(t.ID.String(), t.Concurrency); err != nil {
			log.Printf("warn: set concurrency %s: %v", t.ID, err)
		}
	}

	// Prometheus loop
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			for _, tid := range tm.ListTenantIDs() {
				rabbitClient.UpdateQueueDepth(tid)
			}
		}
	}()

	// HTTP server
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Swagger
	r.Get("/swagger/*", httpSwagger.WrapHandler)

	// Metrics
	r.Get("/metrics", metrics.Handler().ServeHTTP)

	apiHandler := api.NewAPI(tm, db, cfg, r)
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
