package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/yourusername/geofencing-real-time-alert-system/backend/internal/config"
	"github.com/yourusername/geofencing-real-time-alert-system/backend/internal/database"
	"github.com/yourusername/geofencing-real-time-alert-system/backend/internal/handlers"
	ws "github.com/yourusername/geofencing-real-time-alert-system/backend/internal/websocket"
)

func main() {
	cfg := config.Load()
	ctx := context.Background()
	db, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("connect database: %v", err)
	}
	defer db.Close()
	if err = database.Migrate(ctx, db); err != nil {
		log.Fatalf("run migrations: %v", err)
	}
	app := &handlers.Server{DB: db, Hub: ws.NewHub(), AllowedOrigin: cfg.AllowedOrigin}
	srv := &http.Server{Addr: ":" + cfg.Port, Handler: app.Router(), ReadHeaderTimeout: 5 * time.Second}
	go func() {
		log.Printf("API listening on :%s", cfg.Port)
		if e := srv.ListenAndServe(); e != nil && e != http.ErrServerClosed {
			log.Fatalf("server: %v", e)
		}
	}()
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdown); err != nil {
		log.Printf("shutdown: %v", err)
	}
}
