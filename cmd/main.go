package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"cn-rail-monitor/internal/api"
	"cn-rail-monitor/internal/config"
	"cn-rail-monitor/internal/metrics"
	"cn-rail-monitor/internal/output"
	"cn-rail-monitor/internal/scheduler"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	configPath = flag.String("config", "config.yaml", "Path to configuration file")
)

func main() {
	flag.Parse()

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Set custom station cache path if configured
	if cfg.Station.CachePath != "" {
		api.SetStationCachePath(cfg.Station.CachePath)
	}

	// Initialize logger
	initLogger(cfg.Log)

	// Create API client
	client := api.NewClient(cfg.Query.EnablePrice)

	// Initialize metrics collector
	metricsCollector := metrics.NewCollector(cfg)

	// Initialize Telegraf output
	telegrafOutput, err := output.NewTelegrafOutput(&cfg.Telegraf)
	if err != nil {
		log.Printf("Warning: Failed to initialize Telegraf output: %v", err)
	}

	// Create scheduler
	sched := scheduler.NewScheduler(&cfg.Query, client, metricsCollector, telegrafOutput)

	// Start scheduler
	ctx, cancel := context.WithCancel(context.Background())
	sched.Start(ctx)

	// Setup HTTP server with Prometheus metrics
	mux := http.NewServeMux()

	// Prometheus metrics endpoint
	if cfg.Prometheus.Enabled {
		mux.Handle(cfg.Prometheus.Path, promhttp.Handler())
	}

	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Custom metrics endpoint for debugging
	mux.HandleFunc("/debug/metrics", func(w http.ResponseWriter, r *http.Request) {
		metricsCollector.DebugPrint(w)
	})

	addr := fmt.Sprintf("%s:%d", cfg.App.Host, cfg.App.Port)
	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	// Start HTTP server in goroutine
	go func() {
		log.Printf("Starting HTTP server on %s", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("HTTP server error: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down...")

	// Stop scheduler
	cancel()

	// Graceful shutdown of HTTP server
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}

	log.Println("Server stopped")
}

func initLogger(cfg config.LogConfig) {
	if cfg.File != "" {
		f, err := os.OpenFile(cfg.File, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			log.Printf("Failed to open log file: %v", err)
			return
		}
		log.SetOutput(f)
	}

	switch cfg.Level {
	case "debug":
		log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	case "info", "warn", "error":
		log.SetFlags(log.Ldate | log.Ltime)
	default:
		log.SetFlags(log.Ldate | log.Ltime)
	}
}
