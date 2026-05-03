package model

import "time"

type DeployTask struct {
	ID          int64     `json:"id"`
	NodeID      int64     `json:"node_id,omitempty"`
	SSHHost     string    `json:"ssh_host"`
	SSHPort     int       `json:"ssh_port"`
	SSHUser     string    `json:"ssh_user"`
	SSHKey      string    `json:"ssh_key,omitempty"`
	Status      string    `json:"status"`
	Log         string    `json:"log,omitempty"`
	CreatedBy   int64     `json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
	CompletedAt time.Time `json:"completed_at,omitempty"`
}

type DeployRequest struct {
	SSHHost          string   `json:"ssh_host" binding:"required"`
	SSHPort          int      `json:"ssh_port"`
	SSHUser          string   `json:"ssh_user" binding:"required"`
	SSHPassword      string   `json:"ssh_password,omitempty"`
	SSHKey           string   `json:"ssh_key,omitempty"`
	NodeVersion      string   `json:"node_version" binding:"required"`
	PreinstallPlugins []string `json:"preinstall_plugins,omitempty"`
}