package config

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"sync"
)

type ConnectionMode int

const (
	ModeActiveTLS   ConnectionMode = 1
	ModeActiveWSS   ConnectionMode = 2
	ModePassiveTLS  ConnectionMode = 3
	ModePassiveWSS  ConnectionMode = 4
	ModeAuto        ConnectionMode = 5
)

func (m ConnectionMode) String() string {
	switch m {
	case ModeActiveTLS:
		return "active_tls"
	case ModeActiveWSS:
		return "active_wss"
	case ModePassiveTLS:
		return "passive_tls"
	case ModePassiveWSS:
		return "passive_wss"
	case ModeAuto:
		return "auto"
	default:
		return "unknown"
	}
}

func ParseConnectionMode(s string) ConnectionMode {
	switch s {
	case "active_tls", "1":
		return ModeActiveTLS
	case "active_wss", "2":
		return ModeActiveWSS
	case "passive_tls", "3":
		return ModePassiveTLS
	case "passive_wss", "4":
		return ModePassiveWSS
	case "auto", "5":
		return ModeAuto
	default:
		return ModePassiveTLS
	}
}

type Config struct {
	NodeID          string
	AdminURL        string
	TLSCertFile     string
	TLSKeyFile      string
	TLSCACert       string
	InsecureMode    bool
	ListenAddr      string
	NodeGroup       string
	PluginDir       string
	BootstrapToken  string
	BootstrapURL    string
	ConnectionMode  ConnectionMode
}

var (
	cfg  *Config
	once sync.Once
)

func Load() *Config {
	once.Do(func() {
		cfg = &Config{
			NodeID:         getEnv("NODE_ID", ""),
			AdminURL:       getEnv("ADMIN_URL", "wss://localhost:8080/runtime/conn"),
			TLSCertFile:    getEnv("TLS_CERT_FILE", ""),
			TLSKeyFile:     getEnv("TLS_KEY_FILE", ""),
			TLSCACert:      getEnv("TLS_CA_CERT", ""),
			InsecureMode:   getEnv("INSECURE_MODE", "false") == "true",
			ListenAddr:     getEnv("LISTEN_ADDR", ":18888"),
			NodeGroup:      getEnv("NODE_GROUP", "default"),
			PluginDir:      getEnv("PLUGIN_DIR", "./plugins"),
			BootstrapToken: getEnv("BOOTSTRAP_TOKEN", ""),
			BootstrapURL:   getEnv("BOOTSTRAP_URL", ""),
			ConnectionMode: ParseConnectionMode(getEnv("CONNECT_MODE", "passive_tls")),
		}
	})
	return cfg
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func (c *Config) LoadTLSCert() (*tls.Certificate, error) {
	if c.TLSCertFile == "" || c.TLSKeyFile == "" {
		return nil, nil
	}

	cert, err := tls.LoadX509KeyPair(c.TLSCertFile, c.TLSKeyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load TLS cert: %w", err)
	}

	return &cert, nil
}

func (c *Config) LoadCACertPool() (*x509.CertPool, error) {
	if c.TLSCACert == "" {
		return nil, nil
	}

	caCert, err := os.ReadFile(c.TLSCACert)
	if err != nil {
		return nil, fmt.Errorf("failed to read CA cert: %w", err)
	}

	certPool := x509.NewCertPool()
	if !certPool.AppendCertsFromPEM(caCert) {
		return nil, fmt.Errorf("failed to parse CA cert")
	}

	return certPool, nil
}