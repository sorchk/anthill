package model

import "time"

type Session struct {
	ID        int64     `json:"id"`
    UserID    int64     `json:"user_id"`
    Token     string    `json:"token"`
    IP        string    `json:"ip"`
    UserAgent string    `json:"user_agent,omitempty"`
    ExpiresAt time.Time `json:"expires_at"`
    CreatedAt time.Time `json:"created_at"`
}