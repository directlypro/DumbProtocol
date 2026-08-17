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

	"DumbProtocol/internal/config"
	"DumbProtocol/internal/database"
	"DumbProtocol/internal/handler"
	"DumbProtocol/internal/repository"
	"DumbProtocol/internal/service"
)

const ServiceName = "DumbProtocol"

func main() {
	fmt.Printf("Starting %v Service...\n", ServiceName)

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	fmt.Printf("Connecting to %s database at %s...\n", cfg.DatabaseDriver, cfg.DatabaseURL)
	db, err := database.NewConnection(cfg.DatabaseDriver, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	totpRepo := repository.NewTOTPRepository(db)

	totpService := service.NewTOTPService(totpRepo, cfg.TOTPIssuer)

	srv, err := handler.NewServer(cfg, totpService)
	if err != nil {
		log.Fatalf("Failed to create HTTP server: %v", err)
	}

	serverErrors := make(chan error, 1)
	go func() {
		log.Printf("HTTP Server listening on %s", cfg.ServerAddress())
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
