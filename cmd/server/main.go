package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/shogotsuneto/simple-query-server/internal/config"
	"github.com/shogotsuneto/simple-query-server/internal/server"
)

func main() {
	var (
		dbConfigPath      = flag.String("db-config", "", "Path to database configuration YAML file")
		queriesConfigPath = flag.String("queries-config", "", "Path to queries configuration YAML file")
		serverConfigPath  = flag.String("server-config", "", "Path to server configuration YAML file (optional)")
		port              = flag.String("port", "8080", "Port to run the server on")
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

	// Start HTTP server
	srv, err := server.New(dbConfig, queriesConfig, serverConfig)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}
	if err := srv.Start(*port); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
