package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"

	"anthill-runtime/internal/bootstrap"
	"anthill-runtime/internal/cert"
	"anthill-runtime/internal/client"
	"anthill-runtime/internal/config"
	"anthill-runtime/internal/plugin"
	"anthill-runtime/internal/tunnel"
	"anthill-runtime/internal/transport"
)

var version = "1.0.0"

func main() {
	showVersion := flag.Bool("version", false, "Show version")
	adminURL := flag.String("admin", "", "Admin panel URL (WS)")
	nodeID := flag.String("node-id", "", "Node ID")
	bootstrapToken := flag.String("bootstrap-token", "", "Bootstrap token")
	flag.Parse()

	if *showVersion {
		fmt.Printf("TCP Runtime Node %s\n", version)
		os.Exit(0)
	}

	logger, _ := zap.NewProduction()
	defer logger.Sync()

	cfg := config.Load()

	if *adminURL != "" {
		cfg.AdminURL = *adminURL
	}
	if *nodeID != "" {
		cfg.NodeID = *nodeID
	}
	if *bootstrapToken != "" {
		cfg.BootstrapToken = *bootstrapToken
	}
	if cfg.NodeID == "" {
		cfg.NodeID = generateNodeID()
	}

	logger.Info("Starting TCP Runtime Node",
		zap.String("node_id", cfg.NodeID),
		zap.String("admin_url", cfg.AdminURL),
		zap.String("listen", cfg.ListenAddr),
	)

	nodeCert := &cert.NodeCert{
		CertFile: cfg.TLSCertFile,
		KeyFile:  cfg.TLSKeyFile,
	}

	bootstrapper := bootstrap.NewBootstrapper(cfg.AdminURL, cfg.NodeID, cfg.BootstrapToken, nodeCert)
	if !bootstrapper.IsBootstrapped() {
		if cfg.BootstrapToken == "" {
			logger.Fatal("No certificate found and no bootstrap token configured")
		}
		logger.Info("Certificate not found or invalid, starting bootstrap...")
		if err := bootstrapper.Bootstrap(context.Background()); err != nil {
			logger.Fatal("Bootstrap failed", zap.Error(err))
		}
		logger.Info("Bootstrap completed successfully")
	}

	var err error
	nodeCert, err = cert.LoadFromFiles(cfg.TLSCertFile, cfg.TLSKeyFile, cfg.TLSCACert)
	if err != nil {
		logger.Fatal("Failed to load certificate", zap.Error(err))
	}

	tlsCert, err := cfg.LoadTLSCert()
	if err != nil {
		logger.Fatal("Failed to load TLS certificate", zap.Error(err))
	}

	pluginMgr, err := plugin.NewManager(logger, cfg.PluginDir)
	if err != nil {
		logger.Fatal("Failed to init plugin manager", zap.Error(err))
	}
	defer pluginMgr.Close()

	pluginMgr.LoadPluginFromDir(cfg.PluginDir)

	tunnelMgr := tunnel.NewManager(logger)
	defer tunnelMgr.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	server := transport.NewServer(cfg.ListenAddr, tlsCert, logger)

	adminClient := client.NewClient(cfg, logger)
	if err := adminClient.Connect(ctx); err != nil {
		logger.Warn("Failed to connect to admin", zap.Error(err))
	}

	go startHeartbeat(ctx, adminClient, logger)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		logger.Info("Shutting down...")
		cancel()
	}()

	if err := server.Start(ctx); err != nil {
		logger.Fatal("Server error", zap.Error(err))
	}

	logger.Info("Node stopped")
}

func startHeartbeat(ctx context.Context, adminClient *client.Client, logger *zap.Logger) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if adminClient.Status() == "connected" {
				if err := adminClient.SendHeartbeat(); err != nil {
					logger.Warn("Heartbeat failed", zap.Error(err))
				}
			}
		}
	}
}

func generateNodeID() string {
	hostname, _ := os.Hostname()
	return fmt.Sprintf("node-%s-%d", hostname, os.Getpid())
}
