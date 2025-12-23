package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/afterdarksys/adsops-utils/internal/config"
	"github.com/afterdarksys/adsops-utils/internal/pkg/logger"
	"go.uber.org/zap"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize logger
	zapLogger, err := logger.New(cfg.LogLevel, cfg.Environment)
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer zapLogger.Sync()

	zapLogger.Info("Starting background worker",
		zap.String("environment", cfg.Environment),
	)

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// TODO: Initialize worker services
	// - Email notification processor
	// - Approval reminder scheduler
	// - Audit log exporter
	// - Cleanup jobs

	// Start workers
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(10 * time.Second):
				// TODO: Process notification queue
				zapLogger.Debug("Checking notification queue...")
			}
		}
	}()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(1 * time.Hour):
				// TODO: Send approval reminders
				zapLogger.Debug("Checking for pending approval reminders...")
			}
		}
	}()

	// Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	zapLogger.Info("Shutting down worker...")
	cancel()

	// Give workers time to finish
	time.Sleep(5 * time.Second)
	zapLogger.Info("Worker stopped")
}
