package database

import (
	"database/sql"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"

	"golang.org/x/crypto/bcrypt"
)

var db *sql.DB

func SetDB(database *sql.DB) {
	db = database
}

func getDB() (*sql.DB, error) {
	if db == nil {
		return nil, sql.ErrConnDone
	}
	return db, nil
}

func InitDB(dbPath string) (*sql.DB, error) {
	dir := filepath.Dir(dbPath)
	os.MkdirAll(dir, 0755)

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}

	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT UNIQUE NOT NULL,
		password_hash TEXT NOT NULL,
		role TEXT NOT NULL DEFAULT 'viewer',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS nodes (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		host TEXT NOT NULL,
		port INTEGER DEFAULT 18888,
		ssh_host TEXT,
		ssh_port INTEGER DEFAULT 22,
		ssh_username TEXT,
		ssh_password TEXT,
		ssh_key_path TEXT,
		tls_cert_path TEXT,
		tls_cert_cn TEXT,
		status TEXT DEFAULT 'unknown',
		last_seen DATETIME,
		node_group TEXT,
		is_private INTEGER DEFAULT 0,
		owner_id INTEGER NOT NULL,
		visible_to_users TEXT DEFAULT '[]',
		hidden_from_users TEXT DEFAULT '[]',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		connect_mode TEXT DEFAULT 'passive',
		node_port INTEGER DEFAULT 18888,
		node_host TEXT,
		bootstrap_token TEXT,
		node_cert TEXT,
		cert_serial TEXT,
		cert_expires DATETIME,
		last_conn_mode TEXT,
		FOREIGN KEY (owner_id) REFERENCES users(id)
	);

	CREATE TABLE IF NOT EXISTS node_groups (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT UNIQUE NOT NULL,
		description TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS plugins (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		version TEXT NOT NULL,
		description TEXT,
		file_path TEXT NOT NULL,
		file_size INTEGER,
		plugin_type TEXT NOT NULL,
		checksum TEXT,
		uploaded_by INTEGER,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS audit_logs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		username TEXT NOT NULL,
		action TEXT NOT NULL,
		resource TEXT NOT NULL,
		method TEXT NOT NULL,
		path TEXT NOT NULL,
		ip TEXT,
		status INTEGER,
		details TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS deployments (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		node_id INTEGER NOT NULL,
		plugin_id INTEGER NOT NULL,
		version TEXT NOT NULL,
		status TEXT DEFAULT 'pending',
		result TEXT,
		deployed_by INTEGER NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (node_id) REFERENCES nodes(id),
		FOREIGN KEY (plugin_id) REFERENCES plugins(id)
	);

	CREATE TABLE IF NOT EXISTS sessions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		token TEXT NOT NULL UNIQUE,
		ip TEXT,
		user_agent TEXT,
		expires_at TIMESTAMP NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (user_id) REFERENCES users(id)
	);

	CREATE TABLE IF NOT EXISTS deploy_tasks (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		node_id INTEGER,
		ssh_host TEXT NOT NULL,
		ssh_port INTEGER DEFAULT 22,
		ssh_user TEXT NOT NULL,
		status TEXT DEFAULT 'pending',
		log TEXT,
		created_by INTEGER,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		completed_at DATETIME,
		FOREIGN KEY (node_id) REFERENCES nodes(id),
		FOREIGN KEY (created_by) REFERENCES users(id)
	);

	CREATE TABLE IF NOT EXISTS node_plugins (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		node_id INTEGER NOT NULL,
		plugin_name TEXT NOT NULL,
		version TEXT NOT NULL,
		plugin_type TEXT NOT NULL,
		status TEXT DEFAULT 'installing',
		enabled BOOLEAN DEFAULT true,
		installed_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(node_id, plugin_name),
		FOREIGN KEY (node_id) REFERENCES nodes(id) ON DELETE CASCADE
	);
	`

	_, err = db.Exec(schema)
	if err != nil {
		return nil, err
	}

	return db, nil
}

func InitAdminUser(db *sql.DB, username, password string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	_, err = db.Exec(`
		INSERT INTO users (username, password_hash, role)
		VALUES (?, ?, 'admin')
		ON CONFLICT(username) DO NOTHING
	`, username, string(hash))

	return err
}

func LogAudit(userID int64, username, action, path, method, ip string, status int, details string) {
	db, err := getDB()
	if err != nil {
		return
	}
	db.Exec(`
		INSERT INTO audit_logs (user_id, username, action, resource, method, path, ip, status, details)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, userID, username, action, path, method, ip, status, details)
}