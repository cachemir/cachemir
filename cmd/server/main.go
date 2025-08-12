package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/cachemir/cachemir/internal/server"
	"github.com/cachemir/cachemir/pkg/config"
)

func main() {
	cfg := config.LoadServerConfig()

	if err := cfg.Validate(); err != nil {
		log.Fatalf("Invalid configuration: %v", err)
	}

	log.Printf("Starting CacheMir server with config: %+v", cfg)

	srv := server.New(cfg.Port)

	go func() {
		if err := srv.Start(); err != nil {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	<-sigChan
	log.Println("Shutting down server...")

	if err := srv.Stop(); err != nil {
		log.Printf("Error stopping server: %v", err)
	}

	log.Println("Server stopped")
}
