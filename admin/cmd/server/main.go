package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"anthill/admin/internal/config"
	"anthill/admin/internal/database"
	"anthill/admin/internal/handler"
	"anthill/admin/internal/router"
	"anthill/admin/internal/server"
)

func main() {
	cfg := config.Load()

	db, err := database.InitDB(cfg.DBPath)
	if err != nil {
		log.Fatalf("Failed to init DB: %v", err)
	}
	defer db.Close()

	database.SetDB(db)

	ca, err := handler.NewCertCA(cfg.CACertFile, cfg.CAKeyFile)
	if err != nil {
		log.Fatalf("Failed to init CertCA: %v", err)
	}
	fmt.Printf("CertCA initialized (CA cert file: %s)\n", cfg.CACertFile)

	connMgr := server.NewConnManager(db)
	fmt.Printf("ConnManager initialized\n")

	passiveServer := server.NewPassiveServer(18888, ca, connMgr)
	fmt.Printf("PassiveServer created on port %d\n", 18888)

	bootstrapHandler := handler.NewBootstrapHandler(db, ca)

	r := router.Setup(db, cfg, bootstrapHandler)

	go func() {
		if err := passiveServer.Start(); err != nil {
			log.Fatalf("Failed to start PassiveServer: %v", err)
		}
	}()

	fmt.Printf("Admin server starting on %s\n", cfg.ListenAddr)
	fmt.Printf("Database: %s\n", cfg.DBPath)

	go func() {
		if err := r.Run(cfg.ListenAddr); err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	fmt.Println("\nShutting down gracefully...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*1000000000)
	defer cancel()

	if err := passiveServer.Stop(); err != nil {
		fmt.Printf("Error stopping PassiveServer: %v\n", err)
	}

	if err := connMgr.Close(); err != nil {
		fmt.Printf("Error closing ConnManager: %v\n", err)
	}

	select {
	case <-ctx.Done():
		fmt.Println("Shutdown completed")
	}
}