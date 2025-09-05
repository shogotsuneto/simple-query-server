package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/shogotsuneto/simple-query-server/internal/config"
	"github.com/shogotsuneto/simple-query-server/internal/server"
)

func main() {
	var (
		dbConfigPath      = flag.String("db-config", "", "Path to database configuration YAML file")
		queriesConfigPath = flag.String("queries-config", "", "Path to queries configuration YAML file")
		serverConfigPath  = flag.String("server-config", "", "Path to server configuration YAML file (optional)")
		port              = flag.String("port", "8080", "Port to run the server on")
		openapiEnabled    = flag.Bool("openapi-enabled", false, "Enable OpenAPI spec generation and hosting")
		help              = flag.Bool("help", false, "Show help message")
	)
	flag.Parse()

	if *help {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		os.Exit(0)
	}

	if *dbConfigPath == "" || *queriesConfigPath == "" {
		fmt.Fprintf(os.Stderr, "Error: Both --db-config and --queries-config are required\n\n")
		fmt.Fprintf(os.Stderr, "Usage: %s --db-config ./example/database.yaml --queries-config ./example/queries.yaml\n", os.Args[0])
		os.Exit(1)
	}

	log.Printf("Starting simple-query-server...")
	log.Printf("Database config: %s", *dbConfigPath)
	log.Printf("Queries config: %s", *queriesConfigPath)
	if *serverConfigPath != "" {
		log.Printf("Server config: %s", *serverConfigPath)
	}
	log.Printf("Port: %s", *port)

	// Load configurations
	dbConfig, err := config.LoadDatabaseConfig(*dbConfigPath)
	if err != nil {
		log.Fatalf("Failed to load database config: %v", err)
	}

	queriesConfig, err := config.LoadQueriesConfig(*queriesConfigPath)
	if err != nil {
		log.Fatalf("Failed to load queries config: %v", err)
	}

	log.Printf("Loaded %d queries from configuration", len(queriesConfig.Queries))

	// Load server configuration if provided
	var serverConfig *config.ServerConfig
	if *serverConfigPath != "" {
		serverConfig, err = config.LoadServerConfig(*serverConfigPath)
		if err != nil {
			log.Fatalf("Failed to load server config: %v", err)
		}
		log.Printf("Loaded %d middleware configurations", len(serverConfig.Middleware))
	}

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up signal handling for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Start HTTP server
	srv, err := server.New(dbConfig, queriesConfig, serverConfig, *openapiEnabled)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	// Start server in a goroutine
	serverErrors := make(chan error, 1)
	go func() {
		if err := srv.Start(ctx, *port); err != nil {
			serverErrors <- err
		}
	}()

	// Wait for either a signal or server error
	select {
	case err := <-serverErrors:
		log.Fatalf("Server failed to start: %v", err)
	case sig := <-sigCh:
		log.Printf("Received signal %s, initiating graceful shutdown...", sig)
		cancel()

		// Give the server a moment to shut down gracefully
		shutdownTimeout := time.NewTimer(10 * time.Second)
		defer shutdownTimeout.Stop()

		select {
		case <-srv.Done():
			log.Printf("Server shutdown completed")
		case <-shutdownTimeout.C:
			log.Printf("Server shutdown timeout, forcing exit")
		}
	}
}
