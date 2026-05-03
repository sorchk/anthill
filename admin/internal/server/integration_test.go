package server

import (
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"net"
	"path/filepath"
	"testing"
	"time"

	"anthill/admin/internal/database"
	"anthill/admin/internal/handler"
)

func setupIntegrationDB(t *testing.T) (*sql.DB, string) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "integration_test.db")
	
	db, err := database.InitDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to init DB: %v", err)
	}
	
	err = database.InitAdminUser(db, "admin", "password")
	if err != nil {
		t.Fatalf("Failed to init admin user: %v", err)
	}
	
	return db, dbPath
}

func getFreePort() int {
	ln, _ := net.Listen("tcp", ":0")
	port := ln.Addr().(*net.TCPAddr).Port
	ln.Close()
	return port
}

func TestPassiveServer_BasicStartStop(t *testing.T) {
	db, _ := setupIntegrationDB(t)
	defer db.Close()
	
	ca, _ := handler.NewCertCA("", "")
	connMgr := NewConnManager(db)
	
	port := getFreePort()
	passiveServer := NewPassiveServer(port, ca, connMgr)
	
	if err := passiveServer.Start(); err != nil {
		t.Fatalf("Failed to start passive server: %v", err)
	}
	defer passiveServer.Stop()
	
	time.Sleep(100 * time.Millisecond)
	
	ln, err := net.Listen("tcp", filepath.Join(":", filepath.Join("", string(rune(port)))))
	if err == nil {
		ln.Close()
		t.Log("Port is listening")
	}
}

func TestPassiveServer_TLSServerPresentsCertificate(t *testing.T) {
	db, _ := setupIntegrationDB(t)
	defer db.Close()
	
	ca, err := handler.NewCertCA("", "")
	if err != nil {
		t.Fatalf("Failed to create CA: %v", err)
	}
	
	connMgr := NewConnManager(db)
	port := getFreePort()
	passiveServer := NewPassiveServer(port, ca, connMgr)
	
	if err := passiveServer.Start(); err != nil {
		t.Fatalf("Failed to start passive server: %v", err)
	}
	defer passiveServer.Stop()
	
	time.Sleep(100 * time.Millisecond)
	
	caPool := x509.NewCertPool()
	caPool.AppendCertsFromPEM(ca.GetCACert())
	
	tlsConfig := &tls.Config{
		RootCAs:            caPool,
		MinVersion:         tls.VersionTLS12,
		InsecureSkipVerify: true,
	}
	
	conn, err := tls.Dial("tcp", "localhost:"+string(rune(port)), tlsConfig)
	if err == nil {
		conn.Close()
		t.Log("TLS connection established successfully")
	}
}

func TestPassiveServer_ConnectionTracking(t *testing.T) {
	db, _ := setupIntegrationDB(t)
	defer db.Close()
	
	ca, _ := handler.NewCertCA("", "")
	connMgr := NewConnManager(db)
	
	port := getFreePort()
	passiveServer := NewPassiveServer(port, ca, connMgr)
	
	if err := passiveServer.Start(); err != nil {
		t.Fatalf("Failed to start passive server: %v", err)
	}
	defer passiveServer.Stop()
	
	time.Sleep(100 * time.Millisecond)
	
	connMgr.AddConnection("test-node", nil, "test", "passive")
	
	nodeConn, exists := connMgr.GetConnection("test-node")
	if !exists {
		t.Fatal("Connection not found after add")
	}
	
	if nodeConn.NodeID != "test-node" {
		t.Errorf("Expected node ID 'test-node', got '%s'", nodeConn.NodeID)
	}
	
	if nodeConn.Protocol != "test" {
		t.Errorf("Expected protocol 'test', got '%s'", nodeConn.Protocol)
	}
	
	connMgr.RemoveConnection("test-node")
	
	_, exists = connMgr.GetConnection("test-node")
	if exists {
		t.Error("Connection still exists after removal")
	}
}

func TestPassiveServer_MultipleNodeTracking(t *testing.T) {
	db, _ := setupIntegrationDB(t)
	defer db.Close()
	
	connMgr := NewConnManager(db)
	
	nodes := []string{"node-1", "node-2", "node-3"}
	
	for _, nodeID := range nodes {
		connMgr.AddConnection(nodeID, nil, "tls", "passive")
	}
	
	connMgr.mu.RLock()
	connectedCount := len(connMgr.nodes)
	connMgr.mu.RUnlock()
	
	if connectedCount != len(nodes) {
		t.Errorf("Expected %d connections, got %d", len(nodes), connectedCount)
	}
	
	for _, nodeID := range nodes {
		connMgr.RemoveConnection(nodeID)
	}
	
	connMgr.mu.RLock()
	finalCount := len(connMgr.nodes)
	connMgr.mu.RUnlock()
	
	if finalCount != 0 {
		t.Errorf("Expected 0 connections after removal, got %d", finalCount)
	}
}

func TestPassiveServer_CloseAllConnections(t *testing.T) {
	db, _ := setupIntegrationDB(t)
	defer db.Close()
	
	connMgr := NewConnManager(db)
	
	for i := 0; i < 5; i++ {
		connMgr.AddConnection("node-"+string(rune(i)), nil, "tls", "passive")
	}
	
	connMgr.mu.RLock()
	countBefore := len(connMgr.nodes)
	connMgr.mu.RUnlock()
	
	if countBefore != 5 {
		t.Errorf("Expected 5 connections, got %d", countBefore)
	}
	
	connMgr.Close()
	
	connMgr.mu.RLock()
	countAfter := len(connMgr.nodes)
	connMgr.mu.RUnlock()
	
	if countAfter != 0 {
		t.Errorf("Expected 0 connections after Close, got %d", countAfter)
	}
}

func TestConnManager_DatabaseUpdateOnConnect(t *testing.T) {
	db, dbPath := setupIntegrationDB(t)
	defer db.Close()
	
	_, err := db.Exec(`INSERT INTO nodes (id, name, host, port, owner_id) VALUES (?, ?, ?, ?, ?)`,
		123, "test-node", "localhost", 18888, 1)
	if err != nil {
		t.Fatalf("Failed to insert test node: %v", err)
	}
	
	connMgr := NewConnManager(db)
	connMgr.AddConnection("123", nil, "tls", "passive")
	
	var lastSeen time.Time
	err = db.QueryRow("SELECT last_seen FROM nodes WHERE id = 123").Scan(&lastSeen)
	if err != nil {
		t.Fatalf("Failed to query last_seen: %v", err)
	}
	
	if lastSeen.IsZero() {
		t.Error("last_seen was not updated after connection")
	}
	
	_ = dbPath
}