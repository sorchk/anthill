package model

import "time"

type Deployment struct {
    ID         int64     `json:"id"`
    NodeID     int64     `json:"node_id"`
    PluginID   int64     `json:"plugin_id"`
    Version    string    `json:"version"`
    Status     string    `json:"status"`
    Result     string    `json:"result,omitempty"`
    DeployedBy int64     `json:"deployed_by"`
    CreatedAt  time.Time `json:"created_at"`
    UpdatedAt  time.Time `json:"updated_at"`
}

type DeploymentRequest struct {
    NodeIDs  []int64 `json:"node_ids" binding:"required"`
    PluginID int64   `json:"plugin_id" binding:"required"`
}