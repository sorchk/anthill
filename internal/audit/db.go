package audit

import (
	"database/sql"
	"fmt"
	"sync"

	_ "github.com/mattn/go-sqlite3"
)

var (
	db   *sql.DB
	once sync.Once
)

func InitDB(dbPath string) error {
	var err error
	once.Do(func() {
		db, err = sql.Open("sqlite3", dbPath)
		if err != nil {
			return
		}
		err = createTables()
	})
	return err
}

func createTables() error {
	schema := `
	CREATE TABLE IF NOT EXISTS audit_logs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
		event TEXT NOT NULL,
		client_id TEXT,
		role TEXT,
		result TEXT NOT NULL,
		details TEXT
	);
	CREATE INDEX IF NOT EXISTS idx_audit_logs_timestamp ON audit_logs(timestamp);
	CREATE INDEX IF NOT EXISTS idx_audit_logs_event ON audit_logs(event);
	CREATE INDEX IF NOT EXISTS idx_audit_logs_client_id ON audit_logs(client_id);
	`
	_, err := db.Exec(schema)
	return err
}

func GetDB() *sql.DB {
	return db
}

func CloseDB() error {
	if db != nil {
		return db.Close()
	}
	return nil
}

func InsertLog(event, clientID, role, result, details string) error {
	if db == nil {
		return fmt.Errorf("database not initialized")
	}
	_, err := db.Exec(
		"INSERT INTO audit_logs (event, client_id, role, result, details) VALUES (?, ?, ?, ?, ?)",
		event, clientID, role, result, details,
	)
	return err
}