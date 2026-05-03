package config

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"sync"
)

type Config struct {
	NodeID         string
	AdminURL       string
	TLSCertFile    string
	TLSKeyFile     string
	TLSCACert      string
	InsecureMode   bool
	ListenAddr     string
	NodeGroup      string
	PluginDir      string
	BootstrapToken string
	BootstrapURL   string
}

var (
	cfg  *Config
	once sync.Once
)

func Load() *Config {
	once.Do(func() {
		cfg = &Config{
			NodeID:         getEnv("NODE_ID", ""),
			AdminURL:       getEnv("ADMIN_URL", "wss://localhost:8080/ws"),
			TLSCertFile:    getEnv("TLS_CERT_FILE", ""),
			TLSKeyFile:     getEnv("TLS_KEY_FILE", ""),
			TLSCACert:      getEnv("TLS_CA_CERT", ""),
			InsecureMode:   getEnv("INSECURE_MODE", "false") == "true",
			ListenAddr:     getEnv("LISTEN_ADDR", ":18888"),
			NodeGroup:      getEnv("NODE_GROUP", "default"),
			PluginDir:      getEnv("PLUGIN_DIR", "./plugins"),
			BootstrapToken: getEnv("BOOTSTRAP_TOKEN", ""),
			BootstrapURL:   getEnv("BOOTSTRAP_URL", ""),
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