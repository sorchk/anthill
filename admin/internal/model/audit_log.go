package model

import "time"

type AuditLog struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	Username  string    `json:"username"`
	Action    string    `json:"action"`
	Resource  string    `json:"resource"`
	Method    string    `json:"method"`
	Path      string    `json:"path"`
	IP        string    `json:"ip"`
	Status    int       `json:"status"`
	Details   string    `json:"details,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}