package server

import (
	"database/sql"
	"net"
	"sync"
	"time"
)

type NodeConnection struct {
	NodeID        string
	Conn          net.Conn
	Protocol      string
	LastHeartbeat time.Time
	Mode          string
}

type ConnManagerInterface interface {
	AddConnection(nodeID string, conn net.Conn, protocol, mode string) error
	RemoveConnection(nodeID string) error
	GetConnection(nodeID string) (*NodeConnection, bool)
	BroadcastToNode(nodeID string, msg []byte) error
	Close() error
}

type ConnManager struct {
	mu    sync.RWMutex
	nodes map[string]*NodeConnection
	db    *sql.DB
}

func NewConnManager(db *sql.DB) *ConnManager {
	return &ConnManager{
		nodes: make(map[string]*NodeConnection),
		db:    db,
	}
}

func (cm *ConnManager) AddConnection(nodeID string, conn net.Conn, protocol, mode string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	nodeConn := &NodeConnection{
		NodeID:        nodeID,
		Conn:          conn,
		Protocol:      protocol,
		LastHeartbeat: time.Now(),
		Mode:          mode,
	}
	cm.nodes[nodeID] = nodeConn

	if cm.db != nil {
		cm.db.Exec(`UPDATE nodes SET last_seen = ? WHERE id = ?`, time.Now(), nodeID)
	}

	return nil
}

func (cm *ConnManager) RemoveConnection(nodeID string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if nodeConn, ok := cm.nodes[nodeID]; ok {
		if nodeConn.Conn != nil {
			nodeConn.Conn.Close()
		}
		delete(cm.nodes, nodeID)
	}

	return nil
}

func (cm *ConnManager) GetConnection(nodeID string) (*NodeConnection, bool) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	nodeConn, ok := cm.nodes[nodeID]
	return nodeConn, ok
}

func (cm *ConnManager) BroadcastToNode(nodeID string, msg []byte) error {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	nodeConn, ok := cm.nodes[nodeID]
	if !ok {
		return nil
	}

	_, err := nodeConn.Conn.Write(msg)
	return err
}

func (cm *ConnManager) Close() error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	for _, nodeConn := range cm.nodes {
		if nodeConn.Conn != nil {
			nodeConn.Conn.Close()
		}
	}
	cm.nodes = make(map[string]*NodeConnection)

	return nil
}