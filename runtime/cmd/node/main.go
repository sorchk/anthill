package main

import (
	"context"
	"crypto/tls"
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
	showHelp := flag.Bool("help", false, "Show this help message")
	adminURL := flag.String("admin", "", "Admin panel URL")
	connectMode := flag.String("connect-mode", "", "Connection mode (active_tls, active_wss, passive_tls, passive_wss, auto)")
	listenAddr := flag.String("listen", "", "Listen address")
	nodeID := flag.String("node-id", "", "Node ID")
	bootstrapToken := flag.String("bootstrap-token", "", "Bootstrap token")
	flag.Parse()

	if *showVersion {
		fmt.Printf("Anthill Runtime Node %s\n", version)
		os.Exit(0)
	}

	if *showHelp {
		printHelp()
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
	if *connectMode != "" {
		cfg.ConnectionMode = config.ParseConnectionMode(*connectMode)
	}
	if *listenAddr != "" {
		cfg.ListenAddr = *listenAddr
	}
	if cfg.NodeID == "" {
		cfg.NodeID = generateNodeID()
	}

	logger.Info("Starting TCP Runtime Node",
		zap.String("node_id", cfg.NodeID),
		zap.String("admin_url", cfg.AdminURL),
		zap.String("listen", cfg.ListenAddr),
		zap.String("connect_mode", cfg.ConnectionMode.String()),
	)

	var nodeCert *cert.NodeCert
	var tlsCert *tls.Certificate
	var err error

	if cfg.ConnectionMode == config.ModePassiveTLS || cfg.ConnectionMode == config.ModePassiveWSS || cfg.ConnectionMode == config.ModeAuto {
		nodeCert = &cert.NodeCert{
			CertFile: cfg.TLSCertFile,
			KeyFile:  cfg.TLSKeyFile,
		}

		nodeCert, err = cert.LoadFromFiles(cfg.TLSCertFile, cfg.TLSKeyFile, cfg.TLSCACert)
		if err != nil {
			logger.Fatal("Failed to load certificate", zap.Error(err))
		}

		tlsCert, err = cfg.LoadTLSCert()
		if err != nil {
			logger.Fatal("Failed to load TLS certificate", zap.Error(err))
		}
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

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		logger.Info("Shutting down...")
		cancel()
	}()

	runMode(ctx, cfg, nodeCert, tlsCert, logger)

	logger.Info("Node stopped")
}

func runMode(ctx context.Context, cfg *config.Config, nodeCert *cert.NodeCert, tlsCert *tls.Certificate, logger *zap.Logger) {
	switch cfg.ConnectionMode {
	case config.ModePassiveTLS, config.ModePassiveWSS:
		runPassiveMode(ctx, cfg, nodeCert, logger)
	case config.ModeActiveTLS, config.ModeActiveWSS:
		runActiveMode(ctx, cfg, tlsCert, logger)
	case config.ModeAuto:
		runAutoMode(ctx, cfg, nodeCert, tlsCert, logger)
	default:
		runPassiveMode(ctx, cfg, nodeCert, logger)
	}
}

func runPassiveMode(ctx context.Context, cfg *config.Config, nodeCert *cert.NodeCert, logger *zap.Logger) {
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

	adminClient := client.NewClient(cfg, logger)

	if err := adminClient.Connect(ctx); err != nil {
		logger.Warn("Failed to connect to admin (passive mode)", zap.Error(err))
	}

	go startHeartbeat(ctx, adminClient, logger)

	<-ctx.Done()
	adminClient.Close()
}

func runActiveMode(ctx context.Context, cfg *config.Config, tlsCert *tls.Certificate, logger *zap.Logger) {
	server := transport.NewServer(cfg.ListenAddr, tlsCert, logger)

	if err := server.Start(ctx); err != nil {
		logger.Fatal("Server error", zap.Error(err))
	}
}

func runAutoMode(ctx context.Context, cfg *config.Config, nodeCert *cert.NodeCert, tlsCert *tls.Certificate, logger *zap.Logger) {
	bootstrapper := bootstrap.NewBootstrapper(cfg.AdminURL, cfg.NodeID, cfg.BootstrapToken, nodeCert)
	maxRetries := 10
	retryCount := 0
	retryInterval := time.Second * 5

	for retryCount < maxRetries {
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

		adminClient := client.NewClient(cfg, logger)

		logger.Info("Attempting passive connection", zap.Int("attempt", retryCount+1), zap.Int("max", maxRetries))

		if err := adminClient.Connect(ctx); err != nil {
			retryCount++
			logger.Warn("Passive connection failed, retrying...",
				zap.Error(err),
				zap.Int("retry_count", retryCount),
				zap.Duration("next_retry_in", retryInterval))

			select {
			case <-ctx.Done():
				return
			case <-time.After(retryInterval):
				retryInterval = retryInterval * 2
				if retryInterval > time.Minute*5 {
					retryInterval = time.Minute * 5
				}
				continue
			}
		}

		logger.Info("Passive connection successful")
		go startHeartbeat(ctx, adminClient, logger)
		<-ctx.Done()
		adminClient.Close()
		return
	}

	logger.Warn("Passive mode failed after max retries, switching to active mode")
	runActiveMode(ctx, cfg, tlsCert, logger)
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

func printHelp() {
	fmt.Printf(`Anthill Runtime Node %s

Usage: anthill-runtime [options]

Options:
  -version           Show version information
  -help              Show this help message
  -admin string      Admin panel URL (default: wss://localhost:8080/runtime/conn)
  -connect-mode string Connection mode (active_tls, active_wss, passive_tls, passive_wss, auto) (default: passive_tls)
  -node-id string    Node ID (default: auto-generated)
  -bootstrap-token string Bootstrap token for initial certificate

Environment Variables:
  ADMIN_URL       Admin panel URL (default: wss://localhost:8080/runtime/conn)
  NODE_ID         Node ID
  TLS_CERT_FILE   TLS certificate file path
  TLS_KEY_FILE    TLS key file path
  TLS_CA_CERT     CA certificate file path
  LISTEN_ADDR     Listen address (default: :18888)
  NODE_GROUP      Node group name (default: default)
  PLUGIN_DIR      Plugin directory (default: ./plugins)
  BOOTSTRAP_TOKEN Bootstrap token
  BOOTSTRAP_URL   Bootstrap URL
  CONNECT_MODE     Connection mode (active_tls, active_wss, passive_tls, passive_wss, auto) (default: passive_tls)

Connection Modes:
  - passive_tls  Node connects to admin via TLS (default)
  - passive_wss  Node connects to admin via WSS
  - active_tls   Admin connects to node via TLS
  - active_wss   Admin connects to node via WSS
  - auto         Try passive first, fallback to active (10 retries with exponential backoff)

Examples:
  # Start with bootstrap (passive mode - connects to admin)
  anthill-runtime -admin wss://admin.example.com:8080/runtime/conn -bootstrap-token TOKEN

  # Start in active mode (admin connects to node)
  anthill-runtime -connect-mode active_wss -listen :18888

  # Start with auto mode (tries passive, falls back to active)
  anthill-runtime -connect-mode auto -admin wss://admin.example.com:8080/runtime/conn

  # Use environment variables
  export ADMIN_URL=wss://admin.example.com:8080/runtime/conn
  export BOOTSTRAP_TOKEN=your-token
  export CONNECT_MODE=passive_tls
  anthill-runtime
`, version)
}
