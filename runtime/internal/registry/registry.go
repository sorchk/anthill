package registry

import (
    "encoding/json"
    "fmt"
    "sync"
    "time"

    "go.uber.org/zap"

    "anthill-runtime/internal/plugin"
)

type Manager struct {
    mu       sync.RWMutex
    plugins  map[string]*PluginEntry
    manager  *plugin.Manager
    logger   *zap.Logger
}

type PluginEntry struct {
    Info       *plugin.PluginInfo
    LoadedAt   time.Time
    Status     string
    ErrorMsg   string
}

func NewManager(pm *plugin.Manager, logger *zap.Logger) *Manager {
    return &Manager{
        plugins: make(map[string]*PluginEntry),
        manager: pm,
        logger:  logger,
    }
}

func (m *Manager) RegisterPlugin(info *plugin.PluginInfo, wasmData []byte) error {
    m.mu.Lock()
    defer m.mu.Unlock()

    entry := &PluginEntry{
        Info:     info,
        LoadedAt: time.Now(),
        Status:   "pending",
    }

    if err := m.manager.LoadPlugin(info.Name, wasmData); err != nil {
        entry.Status = "failed"
        entry.ErrorMsg = err.Error()
        m.plugins[info.Name] = entry
        return err
    }

    entry.Status = "loaded"
    m.plugins[info.Name] = entry
    m.logger.Info("Plugin registered", zap.String("name", info.Name))

    return nil
}

func (m *Manager) Invoke(name, method string, args json.RawMessage) (*plugin.Result, error) {
    m.mu.RLock()
    entry, exists := m.plugins[name]
    m.mu.RUnlock()

    if !exists || entry.Status != "loaded" {
        return nil, fmt.Errorf("plugin %s not available", name)
    }

    return m.manager.InvokePlugin(name, method, args), nil
}

func (m *Manager) ListPlugins() []*PluginEntry {
    m.mu.RLock()
    defer m.mu.RUnlock()

    entries := make([]*PluginEntry, 0, len(m.plugins))
    for _, entry := range m.plugins {
        entries = append(entries, entry)
    }
    return entries
}

func (m *Manager) Unregister(name string) error {
    m.mu.Lock()
    defer m.mu.Unlock()

    if err := m.manager.UnloadPlugin(name); err != nil {
        return err
    }

    delete(m.plugins, name)
    return nil
}