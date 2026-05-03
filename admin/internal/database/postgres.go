package database

import (
	"fmt"
	"os"
	"strings"
	"time"

	"anthill/admin/internal/model"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func SetDB(db *gorm.DB) {
	DB = db
}

func GetDB() (*gorm.DB, error) {
	if DB == nil {
		return nil, gorm.ErrInvalidDB
	}
	return DB, nil
}

type DBConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

func parseDBPath(dbPath string) (string, error) {
	if strings.HasPrefix(dbPath, "postgres://") || strings.HasPrefix(dbPath, "postgresql://") {
		return dbPath, nil
	}
	if strings.Contains(dbPath, "@") {
		return dbPath, nil
	}
	if _, err := os.Stat(dbPath); err == nil {
		return dbPath, nil
	}
	return "", fmt.Errorf("invalid database path: %s", dbPath)
}

func InitDB(dbPath string) (*gorm.DB, error) {
	dsn, err := parseDBPath(dbPath)
	if err != nil {
		return nil, err
	}

	var dialector gorm.Dialector
	if strings.HasPrefix(dsn, "postgres://") || strings.HasPrefix(dsn, "postgresql://") || strings.Contains(dsn, "@") {
		if !strings.Contains(dsn, "sslmode=") && !strings.Contains(dsn, "?") {
			if strings.Contains(dsn, "?") {
				dsn += "&sslmode=disable"
			} else {
				dsn += "?sslmode=disable"
			}
		}
		dialector = postgres.Open(dsn)
	} else {
		return nil, fmt.Errorf("unsupported database type for path: %s", dbPath)
	}

	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	if err := AutoMigrate(db); err != nil {
		return nil, fmt.Errorf("failed to auto migrate: %w", err)
	}

	DB = db
	return db, nil
}

func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&model.User{},
		&model.Node{},
		&model.NodeGroup{},
		&model.Plugin{},
		&model.AuditLog{},
		&model.Deployment{},
		&model.Session{},
		&model.DeployTask{},
		&model.NodePlugin{},
	)
}

func InitAdminUser(db *gorm.DB, username, password string) error {
	var count int64
	db.Model(&model.User{}).Where("username = ?", username).Count(&count)
	if count > 0 {
		return nil
	}

	user := &model.User{
		Username: username,
		Role:     "admin",
	}
	if err := user.SetPassword(password); err != nil {
		return err
	}
	return db.Create(user).Error
}

func LogAudit(db *gorm.DB, userID int64, username, action, path, method, ip string, status int, details string) {
	log := &model.AuditLog{
		UserID:   userID,
		Username: username,
		Action:   action,
		Resource: path,
		Method:   method,
		Path:     path,
		IP:       ip,
		Status:   status,
		Details:  details,
	}
	db.Create(log)
}

func directoryExists(dir string) bool {
	info, err := os.Stat(dir)
	return err == nil && info.IsDir()
}
