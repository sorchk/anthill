package model

import (
	"time"

	"gorm.io/gorm"
)

type Tunnel struct {
	ID              uint           `gorm:"primarykey" json:"id"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"-"`

	TunnelID        string         `gorm:"uniqueIndex;size:64;not null" json:"tunnel_id"`
	Name            string         `gorm:"size:255" json:"name"`
	Type            string         `gorm:"size:32;not null" json:"type"`
	TransportMode   string         `gorm:"size:16;not null" json:"transport_mode"`

	LocalAddr       string         `gorm:"size:255" json:"local_addr"`
	RemoteAddr      string         `gorm:"size:255" json:"remote_addr"`
	ObfuscationMode string         `gorm:"size:32;default:none" json:"obfuscation_mode"`
	ObfuscationCfg  string         `gorm:"type:text" json:"obfuscation_config"`

	E2EEnabled      bool           `gorm:"default:true" json:"e2e_enabled"`
	TLSFingerprint  bool           `gorm:"default:false" json:"tls_fingerprint"`
	TLSProfile      string         `gorm:"size:32" json:"tls_profile"`

	Status          string         `gorm:"size:16;default:active" json:"status"`
	NodeID          string         `gorm:"size:64;index" json:"node_id"`

	BytesIn         int64          `gorm:"default:0" json:"bytes_in"`
	BytesOut        int64          `gorm:"default:0" json:"bytes_out"`
}

func (Tunnel) TableName() string {
	return "tunnels"
}