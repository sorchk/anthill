package database

import (
	"database/sql"
	"os"

	_ "github.com/lib/pq"

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
	db, err := sql.Open("postgres", dbPath)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id BIGSERIAL PRIMARY KEY,
		username TEXT UNIQUE NOT NULL,
		password_hash TEXT NOT NULL,
		role TEXT NOT NULL DEFAULT 'viewer',
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS nodes (
		id BIGSERIAL PRIMARY KEY,
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
		last_seen TIMESTAMP,
		node_group TEXT,
		is_private INTEGER DEFAULT 0,
		owner_id BIGINT NOT NULL,
		visible_to_users TEXT DEFAULT '[]',
		hidden_from_users TEXT DEFAULT '[]',
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		connect_mode TEXT DEFAULT 'passive',
		node_port INTEGER DEFAULT 18888,
		node_host TEXT,
		bootstrap_token TEXT,
		node_cert TEXT,
		cert_serial TEXT,
		cert_expires TIMESTAMP,
		last_conn_mode TEXT,
		FOREIGN KEY (owner_id) REFERENCES users(id)
	);

	CREATE TABLE IF NOT EXISTS node_groups (
		id BIGSERIAL PRIMARY KEY,
		name TEXT UNIQUE NOT NULL,
		description TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS plugins (
		id BIGSERIAL PRIMARY KEY,
		name TEXT NOT NULL,
		version TEXT NOT NULL,
		description TEXT,
		file_path TEXT NOT NULL,
		file_size BIGINT,
		plugin_type TEXT NOT NULL,
		checksum TEXT,
		uploaded_by BIGINT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS audit_logs (
		id BIGSERIAL PRIMARY KEY,
		user_id BIGINT NOT NULL,
		username TEXT NOT NULL,
		action TEXT NOT NULL,
		resource TEXT NOT NULL,
		method TEXT NOT NULL,
		path TEXT NOT NULL,
		ip TEXT,
		status INTEGER,
		details TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS deployments (
		id BIGSERIAL PRIMARY KEY,
		node_id BIGINT NOT NULL,
		plugin_id BIGINT NOT NULL,
		version TEXT NOT NULL,
		status TEXT DEFAULT 'pending',
		result TEXT,
		deployed_by BIGINT NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (node_id) REFERENCES nodes(id),
		FOREIGN KEY (plugin_id) REFERENCES plugins(id)
	);

	CREATE TABLE IF NOT EXISTS sessions (
		id BIGSERIAL PRIMARY KEY,
		user_id BIGINT NOT NULL,
		token TEXT NOT NULL UNIQUE,
		ip TEXT,
		user_agent TEXT,
		expires_at TIMESTAMP NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (user_id) REFERENCES users(id)
	);

	CREATE TABLE IF NOT EXISTS deploy_tasks (
		id BIGSERIAL PRIMARY KEY,
		node_id BIGINT,
		ssh_host TEXT NOT NULL,
		ssh_port INTEGER DEFAULT 22,
		ssh_user TEXT NOT NULL,
		status TEXT DEFAULT 'pending',
		log TEXT,
		created_by BIGINT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		completed_at TIMESTAMP,
		FOREIGN KEY (node_id) REFERENCES nodes(id),
		FOREIGN KEY (created_by) REFERENCES users(id)
	);

	CREATE TABLE IF NOT EXISTS node_plugins (
		id BIGSERIAL PRIMARY KEY,
		node_id BIGINT NOT NULL,
		plugin_name TEXT NOT NULL,
		version TEXT NOT NULL,
		plugin_type TEXT NOT NULL,
		status TEXT DEFAULT 'installing',
		enabled BOOLEAN DEFAULT true,
		installed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
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
		VALUES ($1, $2, 'admin')
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
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, userID, username, action, path, method, ip, status, details)
}

func directoryExists(dir string) bool {
	info, err := os.Stat(dir)
	return err == nil && info.IsDir()
}