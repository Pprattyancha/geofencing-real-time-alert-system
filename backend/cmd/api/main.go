package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"github.com/yourusername/geofencing-real-time-alert-system/backend/internal/config"
	"github.com/yourusername/geofencing-real-time-alert-system/backend/internal/database"
	"github.com/yourusername/geofencing-real-time-alert-system/backend/internal/handlers"
	ws "github.com/yourusername/geofencing-real-time-alert-system/backend/internal/websocket"
)

func main() {
	// Load .env file locally.
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found. Using system environment variables.")
	}

	// Load application configuration.
	cfg := config.Load()

	if cfg.DatabaseURL == "" {
		log.Fatal("DATABASE_URL is missing. Configure it in Render or .env.")
	}

	if cfg.Port == "" {
		log.Fatal("PORT is missing.")
	}

	if len(cfg.AllowedOrigins) == 0 {
		log.Fatal("ALLOWED_ORIGIN is missing. Configure your frontend origin.")
	}

	log.Println("Starting Geofencing Real-Time Alert System...")

	// Connect to PostgreSQL.
	dbCtx, dbCancel := context.WithTimeout(
		context.Background(),
		30*time.Second,
	)

	db, err := database.Connect(dbCtx, cfg.DatabaseURL)
	dbCancel()

	if err != nil {
		log.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}

	defer db.Close()

	log.Println("Connected to PostgreSQL successfully.")

	// Run database migrations.
	migrationCtx, migrationCancel := context.WithTimeout(
		context.Background(),
		60*time.Second,
	)

	err = database.Migrate(migrationCtx, db)
	migrationCancel()

	if err != nil {
		log.Fatalf("Failed to run database migrations: %v", err)
	}

	log.Println("Database migrations completed successfully.")

	// Initialize WebSocket Hub with allowed frontend origins.
	hub := ws.NewHub(cfg.AllowedOrigins...)
	go hub.Run()

	app := &handlers.Server{
    DB:            db,
    Hub:           hub,
    AllowedOrigin: cfg.AllowedOrigins[0],
	}
	log.Println("WebSocket Hub started successfully.")

	// Configure HTTP server.
	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           app.Router(),
		ReadHeaderTimeout: 5 * time.Second,

		// Do not set WriteTimeout for long-lived WebSocket connections.
		WriteTimeout: 0,

		IdleTimeout: 60 * time.Second,
	}

	// Listen for shutdown signals.
	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	// Start HTTP server.
	serverErrors := make(chan error, 1)

	go func() {
		log.Printf("API server listening on port %s", cfg.Port)

		if err := srv.ListenAndServe(); err != nil &&
			!errors.Is(err, http.ErrServerClosed) {
			serverErrors <- err
		}
	}()

	// Wait for shutdown signal or server failure.
	select {
	case <-ctx.Done():
		log.Println("Shutdown signal received.")

	case err := <-serverErrors:
		log.Printf("HTTP server failed: %v", err)
	}

	// Graceful shutdown.
	shutdownCtx, shutdownCancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("Graceful shutdown failed: %v", err)

		if closeErr := srv.Close(); closeErr != nil {
			log.Printf("Force close failed: %v", closeErr)
		}
	}

	log.Println("Server shutdown completed.")
}
