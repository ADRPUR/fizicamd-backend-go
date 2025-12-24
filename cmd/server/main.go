package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"fizicamd-backend-go/internal/config"
	"fizicamd-backend-go/internal/db"
	httpapi "fizicamd-backend-go/internal/http"
	"fizicamd-backend-go/internal/migrations"
	"fizicamd-backend-go/internal/services"

	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()
	cfg := config.Load()

	database, err := db.Open(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	if err := migrations.Apply(database, "migrations"); err != nil {
		log.Fatalf("migrations: %v", err)
	}
	if err := services.EnsureRoleGroups(database); err != nil {
		log.Fatalf("role groups: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	hub := services.NewMetricsHub()
	go hub.Run(ctx)

	server := httpapi.NewServer(database, cfg, hub)
	go metricsLoop(ctx, server)

	addr := ":8080"
	if value := os.Getenv("PORT"); value != "" {
		addr = ":" + value
	}
	httpServer := &http.Server{
		Addr:    addr,
		Handler: server.Router(ctx),
	}

	go func() {
		log.Printf("listening on %s", addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT)
	<-stop
	cancel()
	ctxShutdown, cancelShutdown := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelShutdown()
	_ = httpServer.Shutdown(ctxShutdown)
}

func metricsLoop(ctx context.Context, server *httpapi.Server) {
	ticker := time.NewTicker(time.Duration(server.Config.MetricsSampleSeconds) * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			sample, err := services.CaptureMetrics(server.DB, server.Config.MetricsDiskPath)
			if err == nil {
				server.MetricsHub.Broadcast(sample)
			}
		case <-ctx.Done():
			return
		}
	}
}
