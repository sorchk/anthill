package plugin

import (
    "sync"
)

type Registry struct {
    mu       sync.RWMutex
    plugins  map[string]*PluginInfo
}

type PluginInfo struct {
    Name        string `json:"name"`
    Version     string `json:"version"`
    Type        string `json:"plugin_type"`
    Description string `json:"description"`
    Author      string `json:"author,omitempty"`
    Capabilities []string `json:"capabilities,omitempty"`
}

var (
    defaultRegistry = &Registry{
        plugins: make(map[string]*PluginInfo),
    }
)

func GetRegistry() *Registry {
    return defaultRegistry
}

func (r *Registry) Register(info *PluginInfo) {
    r.mu.Lock()
    defer r.mu.Unlock()
    r.plugins[info.Name] = info
}

func (r *Registry) Get(name string) *PluginInfo {
    r.mu.RLock()
    defer r.mu.RUnlock()
    return r.plugins[name]
}

func (r *Registry) List() []*PluginInfo {
    r.mu.RLock()
    defer r.mu.RUnlock()

    infos := make([]*PluginInfo, 0, len(r.plugins))
    for _, info := range r.plugins {
        infos = append(infos, info)
    }
    return infos
}