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

	"gwi.com/jedi-team-challenge/internal/api"
	"gwi.com/jedi-team-challenge/internal/config"
	"gwi.com/jedi-team-challenge/internal/core"
	"gwi.com/jedi-team-challenge/internal/store"
)

func main() {
	// Load configuration
	config.LoadConfig()

	// Setup logging
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	if config.AppConfig.LogLevel == "DEBUG" {
		log.Println("Service starting in DEBUG mode")
	}

	// Command line flag for data ingestion
	ingestDataFlag := flag.Bool("ingest", false, "Run data ingestion from data.md and exit")
	flag.Parse()

	// Initialize database store
	dbStore, err := store.NewSQLiteStore(config.AppConfig.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer dbStore.Close()

	// Initialize LLM service
	llmService := core.NewLLMService()
	defer llmService.Close()

	// Handle data ingestion if flag is set
	if *ingestDataFlag {
		log.Println("Starting data ingestion process...")
		// Pass the GetEmbedding method from LLMService as the embedder function
		numIngested, err := dbStore.IngestDataFromFile("data.md", llmService.GetEmbedding)
		if err != nil {
			log.Fatalf("Data ingestion failed: %v", err)
		}
		log.Printf("Data ingestion complete. Ingested %d chunks. Exiting.", numIngested)
		// llmService.Close() and dbStore.Close() will be called by their defers on exit.
		os.Exit(0) // Exit after ingestion
	}

	// Initialize RAG service
	ragService, err := core.NewRAGService(dbStore, llmService)
	if err != nil {
		log.Fatalf("Failed to initialize RAG service: %v", err)
	}

	// Initialize Chat service
	chatService := core.NewChatService(dbStore, ragService, llmService)

	// Initialize API Handler and Router
	apiHandler := api.NewAPIHandler(chatService)
	router := api.NewRouter(apiHandler)

	// Start HTTP server
	serverAddr := fmt.Sprintf(":%s", config.AppConfig.HTTPPort)

	srv := &http.Server{
		Addr:         serverAddr,
		Handler:      router,
		ReadTimeout:  15 * time.Second, // Adjusted for potentially slower LLM handshakes
		WriteTimeout: 60 * time.Second, // LLM calls can take time
		IdleTimeout:  120 * time.Second,
	}

	// Graceful shutdown handling
	go func() {
		log.Printf("Starting server on %s. Press Ctrl+C to quit.", serverAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Could not listen on %s: %v\n", serverAddr, err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	// signal.Notify registers the given channel to receive notifications of the specified signals.
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit // Block until a signal is received
	log.Println("Shutting down server...")

	// Create a context with a timeout for the shutdown.
	// This gives active connections time to finish.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel() // Release resources if srv.Shutdown completes before timeout

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	// llmService.Close() and dbStore.Close() will be called by their defers.
	log.Println("Server exiting gracefully")
}
