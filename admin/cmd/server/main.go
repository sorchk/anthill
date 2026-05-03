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

	db, err := database.InitDB(cfg.DBURL)
	if err != nil {
		log.Fatalf("Failed to init DB: %v", err)
	}

	database.SetDB(db)

	ca, err := handler.NewCertCA(cfg.CACertFile, cfg.CAKeyFile)
	if err != nil {
		log.Fatalf("Failed to init CertCA: %v", err)
	}
	fmt.Printf("CertCA initialized (CA cert file: %s)\n", cfg.CACertFile)

	connMgr := server.NewConnManager(db)
	fmt.Printf("ConnManager initialized\n")

	bootstrapHandler := handler.NewBootstrapHandler(db, ca)

	r := router.Setup(db, cfg, bootstrapHandler, connMgr)

	fmt.Printf("Admin server starting on %s\n", cfg.ListenAddr)

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

	if err := connMgr.Close(); err != nil {
		fmt.Printf("Error closing ConnManager: %v\n", err)
	}

	select {
	case <-ctx.Done():
		fmt.Println("Shutdown completed")
	}
}
