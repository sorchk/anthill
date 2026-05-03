package server

import (
	"crypto/tls"
	"crypto/x509"
	"net"
	"sync"
	"time"

	"gorm.io/gorm"

	"anthill/admin/internal/model"
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
	ConnectActiveNode(nodeID, addr string, port int, protocol string, cert *tls.Certificate, caCert *x509.Certificate) error
}

type ConnManager struct {
	mu    sync.RWMutex
	nodes map[string]*NodeConnection
	db    *gorm.DB

	activeClients   map[string]*ActiveClient
	caCert         *tls.Certificate
	autoModeRetries map[string]int
	maxAutoRetries  int
}

func NewConnManager(db *gorm.DB) *ConnManager {
	return &ConnManager{
		nodes:          make(map[string]*NodeConnection),
		db:             db,
		activeClients:   make(map[string]*ActiveClient),
		autoModeRetries: make(map[string]int),
		maxAutoRetries:  10,
	}
}

func (cm *ConnManager) SetCACert(caCert *tls.Certificate) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.caCert = caCert
}

func (cm *ConnManager) GetCACert() *tls.Certificate {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.caCert
}

func (cm *ConnManager) AddActiveClient(nodeID string, client *ActiveClient) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.activeClients[nodeID] = client
}

func (cm *ConnManager) RemoveActiveClient(nodeID string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	if client, ok := cm.activeClients[nodeID]; ok && client != nil {
		client.Stop()
	}
	delete(cm.activeClients, nodeID)
}

func (cm *ConnManager) GetActiveClient(nodeID string) (*ActiveClient, bool) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	client, ok := cm.activeClients[nodeID]
	return client, ok
}

func (cm *ConnManager) IncrementAutoModeRetry(nodeID string) int {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.autoModeRetries[nodeID]++
	return cm.autoModeRetries[nodeID]
}

func (cm *ConnManager) GetAutoModeRetryCount(nodeID string) int {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.autoModeRetries[nodeID]
}

func (cm *ConnManager) ResetAutoModeRetry(nodeID string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	delete(cm.autoModeRetries, nodeID)
}

func (cm *ConnManager) ShouldSwitchToPassive(nodeID string) bool {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	retryCount, ok := cm.autoModeRetries[nodeID]
	return ok && retryCount >= cm.maxAutoRetries
}

func (cm *ConnManager) ConnectActiveNode(nodeID, addr string, port int, protocol string, cert *tls.Certificate, caCert *x509.Certificate) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if _, ok := cm.nodes[nodeID]; ok {
		return nil
	}

	client := NewActiveClient(nodeID, addr, port, protocol, cert, caCert, cm)
	if err := client.TryConnect(); err != nil {
		return err
	}

	cm.activeClients[nodeID] = client
	return nil
}

func (cm *ConnManager) AddConnection(nodeID string, conn net.Conn, protocol, mode string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if mode == "passive" || mode == string(model.ConnectModePassiveTLS) || mode == string(model.ConnectModePassiveWSS) {
		delete(cm.autoModeRetries, nodeID)
	}

	nodeConn := &NodeConnection{
		NodeID:        nodeID,
		Conn:          conn,
		Protocol:      protocol,
		LastHeartbeat: time.Now(),
		Mode:          mode,
	}
	cm.nodes[nodeID] = nodeConn

	if cm.db != nil {
		cm.db.Model(&model.Node{}).Where("id = ?", nodeID).Update("last_seen", time.Now())
		cm.db.Model(&model.Node{}).Where("id = ?", nodeID).Updates(map[string]interface{}{
			"last_conn_mode": mode,
			"updated_at":     time.Now(),
		})
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

	for _, client := range cm.activeClients {
		client.Stop()
	}
	cm.activeClients = make(map[string]*ActiveClient)

	return nil
}