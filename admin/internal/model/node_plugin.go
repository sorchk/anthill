package model

import "time"

type NodePlugin struct {
    ID         int64     `json:"id"`
    NodeID     int64     `json:"node_id"`
    PluginName string    `json:"plugin_name"`
    Version    string    `json:"version"`
    PluginType string    `json:"plugin_type"`
    Status     string    `json:"status"`
    Enabled    bool      `json:"enabled"`
    InstalledAt time.Time `json:"installed_at"`
}

type NodePluginInstall struct {
    PluginName string `json:"plugin_name" binding:"required"`
    Version    string `json:"version" binding:"required"`
}