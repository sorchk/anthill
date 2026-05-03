package config

import (
	"os"
)

type Config struct {
	DBPath        string
	ListenAddr    string
	PassiveAddr   string
	CACertFile    string
	CAKeyFile     string
	JWTKey        string
	AdminEmail    string
	AdminPassword string
}

func Load() *Config {
	return &Config{
		DBPath:        getEnv("DB_PATH", "./data/admin.db"),
		ListenAddr:    getEnv("LISTEN_ADDR", ":8080"),
		PassiveAddr:   getEnv("PASSIVE_ADDR", ":18888"),
		CACertFile:    getEnv("CA_CERT_FILE", ""),
		CAKeyFile:     getEnv("CA_KEY_FILE", ""),
		JWTKey:        getEnv("JWT_SECRET", "change-me-in-production"),
		AdminEmail:    getEnv("ADMIN_EMAIL", "admin@localhost"),
		AdminPassword: getEnv("ADMIN_PASSWORD", "admin123"),
	}
}

func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}