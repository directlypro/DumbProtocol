package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"DumbProtocol/internal/http_server"

	"github.com/joho/godotenv"
)

const ServiceName = "DumbProtocol"

func main() {
	// Service initialization
	fmt.Printf("Starting the %v Service\n", ServiceName)

	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found or error loading variables, using default environment settings")
	}

	dbURL := os.Getenv("DATABASE")
	if dbURL == "" {
		fmt.Println("DATABASE environment variable not set")
	} else {
		fmt.Printf("Connecting to the %v Database\n", dbURL)
	}

	url := os.Getenv("URL")
	if url == "" {
		url = "127.0.0.1:3306"
	}

	srv, err := http_server.NewServer(url, 15*time.Second)
	if err != nil {
		log.Fatalf("Failed to create HTTP server: %v", err)
	}

	serverErrors := make(chan error, 1)
	go func() {
		log.Printf("HTTP Server listening on %s", url)
		if err := srv.Start(); err != nil && err != http.ErrServerClosed {
			serverErrors <- err
		}
	}()

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		log.Fatalf("Server startup error: %v", err)
	case sig := <-shutdown:
		log.Printf("Received signal %v, initiating shutdown...\n", sig)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			log.Fatalf("Graceful shutdown failed: %v", err)
		}
		log.Println("Server stopped cleanly")
	}
}
