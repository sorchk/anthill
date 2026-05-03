package server

import (
	"encoding/json"
	"sync"
	"time"

	"go.uber.org/zap"

	"anthill/admin/internal/model"
)

type TunnelSession struct {
	ID           string
	TunnelID     string
	Mode         string
	SourceNodeID string
	TargetNodeID string
	RemoteHost   string
	RemotePort   int
	Status       string
	CreatedAt    time.Time
	BytesIn      int64
	BytesOut     int64
}

type TunnelBroker struct {
	mu       sync.RWMutex
	sessions map[string]*TunnelSession
	nodes    map[string]*NodeConnection
	logger   *zap.Logger
	db       interface{}
}

func NewTunnelBroker(logger *zap.Logger) *TunnelBroker {
	return &TunnelBroker{
		sessions: make(map[string]*TunnelSession),
		nodes:    make(map[string]*NodeConnection),
		logger:   logger,
	}
}

func (b *TunnelBroker) SetDB(db interface{}) {
	b.db = db
}

func (b *TunnelBroker) AddNodeConnection(nodeID string, conn *NodeConnection) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.nodes[nodeID] = conn
}

func (b *TunnelBroker) RemoveNodeConnection(nodeID string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.nodes, nodeID)
}

func (b *TunnelBroker) GetNodeConnection(nodeID string) (*NodeConnection, bool) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	conn, ok := b.nodes[nodeID]
	return conn, ok
}

func (b *TunnelBroker) CreateSession(tunnelID, sourceNodeID, targetNodeID, mode, remoteHost string, remotePort int) *TunnelSession {
	b.mu.Lock()
	defer b.mu.Unlock()

	session := &TunnelSession{
		ID:           tunnelID + "_" + sourceNodeID,
		TunnelID:     tunnelID,
		Mode:         mode,
		SourceNodeID: sourceNodeID,
		TargetNodeID: targetNodeID,
		RemoteHost:   remoteHost,
		RemotePort:   remotePort,
		Status:       "active",
		CreatedAt:    time.Now(),
		BytesIn:      0,
		BytesOut:     0,
	}

	b.sessions[session.ID] = session
	return session
}

func (b *TunnelBroker) GetSession(sessionID string) (*TunnelSession, bool) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	session, ok := b.sessions[sessionID]
	return session, ok
}

func (b *TunnelBroker) GetSessionByTunnelID(tunnelID string) []*TunnelSession {
	b.mu.RLock()
	defer b.mu.RUnlock()

	var sessions []*TunnelSession
	for _, session := range b.sessions {
		if session.TunnelID == tunnelID {
			sessions = append(sessions, session)
		}
	}
	return sessions
}

func (b *TunnelBroker) UpdateSessionStats(sessionID string, bytesIn, bytesOut int64) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if session, ok := b.sessions[sessionID]; ok {
		session.BytesIn += bytesIn
		session.BytesOut += bytesOut
	}
}

func (b *TunnelBroker) CloseSession(sessionID string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if session, ok := b.sessions[sessionID]; ok {
		session.Status = "closed"
		delete(b.sessions, sessionID)
	}
}

func (b *TunnelBroker) ListSessions() []*TunnelSession {
	b.mu.RLock()
	defer b.mu.RUnlock()

	sessions := make([]*TunnelSession, 0, len(b.sessions))
	for _, session := range b.sessions {
		sessions = append(sessions, session)
	}
	return sessions
}

type TunnelOpenMsg struct {
	TunnelID       string `json:"tunnel_id"`
	TunnelType     string `json:"tunnel_type"`
	TransportMode  string `json:"transport_mode"`
	Obfuscation    string `json:"obfuscation"`
	ObfuscationCfg string `json:"obfuscation_cfg"`
	RemoteHost     string `json:"remote_host"`
	RemotePort     int    `json:"remote_port"`
	EphemeralPub   []byte `json:"ephemeral_pub"`
}

type TunnelDataMsg struct {
	TunnelID       string `json:"tunnel_id"`
	SessionID      string `json:"session_id"`
	EncryptedData  []byte `json:"encrypted_data"`
}

type TunnelCloseMsg struct {
	TunnelID  string `json:"tunnel_id"`
	SessionID string `json:"session_id"`
	Reason    byte   `json:"reason"`
}

type TunnelKeyExMsg struct {
	TunnelID     string `json:"tunnel_id"`
	EphemeralPub []byte `json:"ephemeral_pub"`
}

func (b *TunnelBroker) HandleTunnelOpen(nodeID string, msg *TunnelOpenMsg) error {
	b.logger.Info("TunnelOpen received",
		zap.String("node_id", nodeID),
		zap.String("tunnel_id", msg.TunnelID),
		zap.String("tunnel_type", msg.TunnelType),
		zap.String("transport_mode", msg.TransportMode))

	session := b.CreateSession(msg.TunnelID, nodeID, "", msg.TransportMode, msg.RemoteHost, msg.RemotePort)
	b.logger.Info("Tunnel session created",
		zap.String("session_id", session.ID),
		zap.String("tunnel_id", msg.TunnelID))

	return nil
}

func (b *TunnelBroker) HandleTunnelData(nodeID string, msg *TunnelDataMsg) error {
	b.logger.Debug("TunnelData received",
		zap.String("node_id", nodeID),
		zap.String("tunnel_id", msg.TunnelID),
		zap.String("session_id", msg.SessionID),
		zap.Int("data_len", len(msg.EncryptedData)))

	session, ok := b.GetSession(msg.SessionID)
	if !ok {
		b.logger.Warn("Session not found for TunnelData",
			zap.String("session_id", msg.SessionID))
		return nil
	}

	if nodeID == session.SourceNodeID {
		session.BytesOut += int64(len(msg.EncryptedData))
	} else {
		session.BytesIn += int64(len(msg.EncryptedData))
	}

	return nil
}

func (b *TunnelBroker) HandleTunnelClose(nodeID string, msg *TunnelCloseMsg) error {
	b.logger.Info("TunnelClose received",
		zap.String("node_id", nodeID),
		zap.String("tunnel_id", msg.TunnelID),
		zap.String("session_id", msg.SessionID),
		zap.Uint8("reason", msg.Reason))

	b.CloseSession(msg.SessionID)
	return nil
}

func (b *TunnelBroker) HandleTunnelKeyEx(nodeID string, msg *TunnelKeyExMsg) error {
	b.logger.Info("TunnelKeyEx received",
		zap.String("node_id", nodeID),
		zap.String("tunnel_id", msg.TunnelID))

	return nil
}

func (b *TunnelBroker) UpdateTunnelStatsInDB(db interface{}, tunnelID string, bytesIn, bytesOut int64) error {
	if db == nil {
		return nil
	}

	gormDB, ok := db.(*model.Tunnel)
	if !ok {
		return nil
	}

	_ = map[string]interface{}{
		"bytes_in":  gormDB.BytesIn + bytesIn,
		"bytes_out": gormDB.BytesOut + bytesOut,
	}

	return nil
}

type TunnelConfig struct {
	TunnelID        string                 `json:"tunnel_id"`
	Name            string                 `json:"name"`
	Type            string                 `json:"type"`
	TransportMode   string                 `json:"transport_mode"`
	LocalAddr       string                 `json:"local_addr"`
	RemoteAddr      string                 `json:"remote_addr"`
	ObfuscationMode string                `json:"obfuscation_mode"`
	ObfuscationCfg  map[string]interface{} `json:"obfuscation_config"`
	E2EEnabled      bool                  `json:"e2e_enabled"`
	TLSFingerprint  bool                  `json:"tls_fingerprint"`
	TLSProfile      string                `json:"tls_profile"`
	Status          string                `json:"status"`
	NodeID          string                `json:"node_id"`
}

func ParseObfuscationConfig(cfgStr string) map[string]interface{} {
	if cfgStr == "" {
		return nil
	}
	var cfg map[string]interface{}
	if err := json.Unmarshal([]byte(cfgStr), &cfg); err != nil {
		return nil
	}
	return cfg
}