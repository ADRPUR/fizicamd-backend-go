package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
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

	cleanupLogs, err := setupLogger()
	if err != nil {
		log.Printf("logger setup failed: %v", err)
	} else {
		defer cleanupLogs()
	}

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
	log.Printf("shutdown complete")
}

func setupLogger() (func(), error) {
	logDir := os.Getenv("LOG_DIR")
	if logDir == "" {
		logDir = "storage/logs"
	}
	retentionDays := 7
	if value := os.Getenv("LOG_RETENTION_DAYS"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err == nil && parsed > 0 {
			if parsed > 7 {
				parsed = 7
			}
			retentionDays = parsed
		}
	}
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return nil, err
	}

	var mu sync.Mutex
	currentDate := time.Now().Format("2006-01-02")
	file, err := openLogFile(logDir, currentDate)
	if err != nil {
		return nil, err
	}
	log.SetOutput(io.MultiWriter(os.Stdout, file))
	cleanupOldLogs(logDir, retentionDays)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				date := time.Now().Format("2006-01-02")
				mu.Lock()
				if date != currentDate {
					newFile, err := openLogFile(logDir, date)
					if err == nil {
						log.SetOutput(io.MultiWriter(os.Stdout, newFile))
						_ = file.Close()
						file = newFile
						currentDate = date
						cleanupOldLogs(logDir, retentionDays)
					}
				}
				mu.Unlock()
			case <-ctx.Done():
				return
			}
		}
	}()

	return func() {
		cancel()
		mu.Lock()
		_ = file.Close()
		mu.Unlock()
	}, nil
}

func openLogFile(logDir, date string) (*os.File, error) {
	filename := filepath.Join(logDir, fmt.Sprintf("app-%s.log", date))
	return os.OpenFile(filename, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
}

func cleanupOldLogs(logDir string, retentionDays int) {
	entries, err := os.ReadDir(logDir)
	if err != nil {
		return
	}
	cutoff := time.Now().AddDate(0, 0, -(retentionDays - 1))
	for _, entry := range entries {
		name := entry.Name()
		if !entry.Type().IsRegular() {
			continue
		}
		if !strings.HasPrefix(name, "app-") || !strings.HasSuffix(name, ".log") {
			continue
		}
		datePart := strings.TrimSuffix(strings.TrimPrefix(name, "app-"), ".log")
		logDate, err := time.Parse("2006-01-02", datePart)
		if err != nil {
			continue
		}
		if logDate.Before(cutoff) {
			_ = os.Remove(filepath.Join(logDir, name))
		}
	}
}

func metricsLoop(ctx context.Context, server *httpapi.Server) {
	ticker := time.NewTicker(time.Duration(server.Config.MetricsSampleSeconds) * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			sample, err := services.CaptureMetrics(server.DB, server.Config.MetricsDiskPath)
			if err != nil {
				log.Printf("metrics capture: %v", err)
				continue
			}
			server.MetricsHub.Broadcast(sample)
		case <-ctx.Done():
			return
		}
	}
}
