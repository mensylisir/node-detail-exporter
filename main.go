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

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"node-prober/internal/collector"
	"node-prober/internal/config"
)

var (
	// Version and BuildDate are set at build time
	Version   string
	BuildDate string
)

func main() {
	configFile := flag.String("config.file", "configs/config.example.yaml", "Path to the configuration file.")
	versionFlag := flag.Bool("version", false, "Print version information and exit.")
	flag.Parse()

	if *versionFlag {
		fmt.Printf("node-prober version %s, built on %s\n", Version, BuildDate)
		os.Exit(0)
	}

	cfg, err := config.Load(*configFile)
	if err != nil {
		log.Fatalf("Error loading configuration: %v", err)
	}

	registry := collector.NewRegistry()
	prometheus.MustRegister(registry)

	registerCollectors(registry, cfg)

	// Create a ticker for periodic updates
	ticker := time.NewTicker(cfg.ScrapeInterval)
	defer ticker.Stop()

	// Create a cancellable context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start the collector update loop in a goroutine
	go func(ctx context.Context) {
		for {
			select {
			case <-ticker.C:
				registry.UpdateAll(cfg.ScrapeInterval)
			case <-ctx.Done():
				return
			}
		}
	}(ctx)

	// Set up the HTTP server
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	// Start the server in a goroutine
	go func() {
		log.Printf("Starting node-prober %s", Version)
		log.Println("Beginning to serve on port 8080")
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatalf("HTTP server ListenAndServe: %v", err)
		}
	}()

	// Listen for shutdown signals
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	// Create a context with a timeout for the shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server Shutdown Failed:%+v", err)
	}
	log.Println("Server gracefully stopped")
}


func registerCollectors(registry *collector.Registry, cfg *config.Config) {
	type collectorFactory func() (collector.Collector, error)

	factories := make(map[bool]collectorFactory)

	factories[cfg.Collectors.Pidstat] = func() (collector.Collector, error) { return collector.NewPidstatCollector() }
	factories[cfg.Collectors.Iostat] = func() (collector.Collector, error) { return collector.NewIostatCollector() }
	factories[cfg.Collectors.Vmstat] = func() (collector.Collector, error) { return collector.NewVmstatCollector() }
	factories[cfg.Collectors.DiskUsage] = func() (collector.Collector, error) { return collector.NewDiskUsageCollector() }
	factories[cfg.Collectors.Benchmark] = func() (collector.Collector, error) { return collector.NewBenchmarkCollector() }
	factories[cfg.Collectors.Ping.Enabled] = func() (collector.Collector, error) { return collector.NewPingCollector(cfg.Collectors.Ping.Targets) }
	factories[cfg.Collectors.Mtr.Enabled] = func() (collector.Collector, error) { return collector.NewMtrCollector(cfg.Collectors.Mtr.Targets) }
	factories[cfg.Collectors.Audit.Enabled] = func() (collector.Collector, error) { return collector.NewAuditCollector(cfg.Collectors.Audit.LogPath) }

	for enabled, factory := range factories {
		if enabled {
			c, err := factory()
			if err != nil {
				log.Printf("Error creating collector: %v", err)
				continue
			}
			registry.Register(c.Name(), c)
		}
	}
}
