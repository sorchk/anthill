package main

import (
	"context"
	"crypto/x509"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"gorm.io/gorm"

	"anthill/admin/internal/config"
	"anthill/admin/internal/database"
	"anthill/admin/internal/handler"
	"anthill/admin/internal/model"
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
	connMgr.SetCACert(ca.GetCACertTLS())
	fmt.Printf("ConnManager initialized\n")

	bootstrapHandler := handler.NewBootstrapHandler(db, ca)
	certHandler := handler.NewCertHandler(db, ca)

	r := router.Setup(db, cfg, bootstrapHandler, certHandler, connMgr)

	passiveServer := server.NewPassiveServer(8080, ca, connMgr)

	fmt.Printf("Admin server starting on %s\n", cfg.ListenAddr)

	if err := passiveServer.StartWithGin(r); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}

	startActiveModeNodes(db, ca, connMgr)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	fmt.Println("\nShutting down gracefully...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*1000000000)
	defer cancel()

	if err := passiveServer.Stop(); err != nil {
		fmt.Printf("Error closing PassiveServer: %v\n", err)
	}

	if err := connMgr.Close(); err != nil {
		fmt.Printf("Error closing ConnManager: %v\n", err)
	}

	select {
	case <-ctx.Done():
		fmt.Println("Shutdown completed")
	}
}

func startActiveModeNodes(db *gorm.DB, ca *handler.CertCA, connMgr *server.ConnManager) {
	var nodes []model.Node
	if err := db.Find(&nodes).Error; err != nil {
		fmt.Printf("Failed to load nodes: %v\n", err)
		return
	}

	for _, node := range nodes {
		if node.ConnectMode == model.ConnectModeActiveTLS || node.ConnectMode == model.ConnectModeActiveWSS {
			protocol := "tls"
			if node.ConnectMode == model.ConnectModeActiveWSS {
				protocol = "wss"
			}

			addr := node.NodeHost
			if addr == "" {
				addr = node.Host
			}
			port := node.NodePort
			if port == 0 {
				port = 18888
			}

			nodeIDStr := fmt.Sprintf("%d", node.ID)

			caCert, err := x509.ParseCertificate(ca.GetCACertTLS().Certificate[0])
			if err != nil {
				fmt.Printf("Failed to parse CA cert for node %s: %v\n", node.Name, err)
				continue
			}

			activeClient := server.NewActiveClient(
				nodeIDStr,
				addr,
				port,
				protocol,
				ca.GetCACertTLS(),
				caCert,
				connMgr,
			)

			connMgr.AddActiveClient(nodeIDStr, activeClient)
			if err := activeClient.Start(); err != nil {
				fmt.Printf("Failed to start ActiveClient for node %s: %v\n", node.Name, err)
			} else {
				fmt.Printf("Started ActiveClient for node %s (%s:%d, %s)\n", node.Name, addr, port, protocol)
			}
		}
	}
}
