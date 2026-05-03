package tunnel

import (
    "context"
    "fmt"
    "sync"
    "sync/atomic"

    "go.uber.org/zap"
)

type Manager struct {
    mu      sync.RWMutex
    tunnels map[string]*Tunnel
    sessions map[uint64]*TunnelSession
    nextConnID uint64
    logger    *zap.Logger
    ctx       context.Context
    cancel    context.CancelFunc
}

func NewManager(logger *zap.Logger) *Manager {
    ctx, cancel := context.WithCancel(context.Background())
    return &Manager{
        tunnels:  make(map[string]*Tunnel),
        sessions: make(map[uint64]*TunnelSession),
        logger:   logger,
        ctx:      ctx,
        cancel:   cancel,
    }
}

func (m *Manager) CreateTunnel(cfg *TunnelConfig) error {
    m.mu.Lock()
    defer m.mu.Unlock()

    if _, exists := m.tunnels[cfg.ID]; exists {
        return fmt.Errorf("tunnel %s already exists", cfg.ID)
    }

    tunnel := &Tunnel{
        Config: cfg,
        manager: m,
        sessions: make(map[uint64]*TunnelSession),
        status: "active",
    }

    m.tunnels[cfg.ID] = tunnel
    m.logger.Info("Tunnel created", zap.String("id", cfg.ID), zap.String("type", string(cfg.Type)))

    return nil
}

func (m *Manager) GetTunnel(id string) (*Tunnel, bool) {
    m.mu.RLock()
    defer m.mu.RUnlock()
    t, ok := m.tunnels[id]
    return t, ok
}

func (m *Manager) ListTunnels() []*TunnelConfig {
    m.mu.RLock()
    defer m.mu.RUnlock()

    configs := make([]*TunnelConfig, 0, len(m.tunnels))
    for _, t := range m.tunnels {
        configs = append(configs, t.Config)
    }
    return configs
}

func (m *Manager) DeleteTunnel(id string) error {
    m.mu.Lock()
    defer m.mu.Unlock()

    tunnel, ok := m.tunnels[id]
    if !ok {
        return fmt.Errorf("tunnel not found")
    }

    tunnel.Close()
    delete(m.tunnels, id)
    m.logger.Info("Tunnel deleted", zap.String("id", id))

    return nil
}

func (m *Manager) NextConnID() uint64 {
    return atomic.AddUint64(&m.nextConnID, 1)
}

func (m *Manager) Close() error {
    m.cancel()
    m.mu.Lock()
    defer m.mu.Unlock()

    for id, tunnel := range m.tunnels {
        tunnel.Close()
        delete(m.tunnels, id)
    }
    return nil
}

type Tunnel struct {
    Config  *TunnelConfig
    manager *Manager
    sessions map[uint64]*TunnelSession
    status  string
    mu      sync.RWMutex
}

func (t *Tunnel) Close() {
    t.mu.Lock()
    defer t.mu.Unlock()
    t.status = "closed"
    for _, session := range t.sessions {
        session.Status = "closed"
    }
}

func (t *Tunnel) Status() string {
    t.mu.RLock()
    defer t.mu.RUnlock()
    return t.status
}