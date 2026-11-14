package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"github.com/Pprattyancha/geofencing-real-time-alert-system/backend/internal/config"
	"github.com/Pprattyancha/geofencing-real-time-alert-system/backend/internal/database"
	"github.com/Pprattyancha/geofencing-real-time-alert-system/backend/internal/handlers"
	ws "github.com/Pprattyancha/geofencing-real-time-alert-system/backend/internal/websocket"
)

func main() {
	// Load .env locally. Render uses environment variables.
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found. Using system environment variables.")
	}

	// Load application configuration.
	cfg := config.Load()

	if cfg.DatabaseURL == "" {
		log.Fatal("DATABASE_URL is missing. Configure it in Render or .env.")
	}

	// Render provides PORT automatically. Use 8080 locally if absent.
	if cfg.Port == "" {
		cfg.Port = "8080"
	}

	// Validate allowed frontend origins.
	if len(cfg.AllowedOrigins) == 0 {
		log.Fatal("ALLOWED_ORIGINS is missing. Configure your frontend origin.")
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

	// Normalize and validate frontend origins.
	var allowedOrigins []string

	for _, origin := range cfg.AllowedOrigins {
		origin = strings.TrimRight(strings.TrimSpace(origin), "/")

		if origin != "" {
			allowedOrigins = append(allowedOrigins, origin)
		}
	}

	if len(allowedOrigins) == 0 {
		log.Fatal("No valid frontend origins configured in ALLOWED_ORIGINS.")
	}

	// Initialize WebSocket Hub.
	hub := ws.NewHub(allowedOrigins...)
	go hub.Run()

	log.Println("WebSocket Hub started successfully.")

	// Initialize application handlers.
	app := &handlers.Server{
		DB:            db,
		Hub:           hub,
		AllowedOrigin: allowedOrigins[0],
	}

	// Configure HTTP server.
	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           app.Router(),
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      0,
		IdleTimeout:       60 * time.Second,
	}

	// Handle shutdown signals.
	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	serverErrors := make(chan error, 1)

	// Start HTTP server.
	go func() {
		log.Printf("API server listening on port %s", cfg.Port)

		if err := srv.ListenAndServe(); err != nil &&
			!errors.Is(err, http.ErrServerClosed) {
			serverErrors <- err
		}
	}()

	// Wait for shutdown signal or server error.
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
