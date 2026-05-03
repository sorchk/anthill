package model

import (
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

type Node struct {
	ID          int64          `gorm:"primaryKey;autoIncrement" json:"id"`
	Name        string         `gorm:"size:255;not null" json:"name"`
	Host        string         `gorm:"size:255;not null" json:"host"`
	Port        int            `gorm:"default:18888" json:"port"`
	SSHHost     string         `gorm:"size:255" json:"ssh_host,omitempty"`
	SSHPort     int            `gorm:"default:22" json:"ssh_port,omitempty"`
	SSHUsername string         `gorm:"size:255" json:"ssh_username,omitempty"`
	SSHPassword string         `gorm:"size:255" json:"-" json:"-"`
	SSHKeyPath  string         `gorm:"size:512" json:"ssh_key_path,omitempty"`
	TLSCertPath string         `gorm:"size:512" json:"tls_cert_path,omitempty"`
	TLSCertCN   string         `gorm:"size:255" json:"tls_cert_cn,omitempty"`
	Status      string         `gorm:"size:50;default:unknown" json:"status"`
	LastSeen    *time.Time     `json:"last_seen,omitempty"`
	NodeGroup   string         `gorm:"size:255" json:"node_group,omitempty"`
	IsPrivate   bool           `gorm:"default:false" json:"is_private"`
	OwnerID     int64          `gorm:"not null" json:"owner_id"`
	VisibleTo   string         `gorm:"type:text;default:'[]'" json:"visible_to,omitempty"`
	HiddenFrom  string         `gorm:"type:text;default:'[]'" json:"hidden_from,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
	ConnectMode    string    `gorm:"size:50;default:passive" json:"connect_mode"`
	NodePort       int       `gorm:"default:18888" json:"node_port"`
	NodeHost       string    `gorm:"size:255" json:"node_host,omitempty"`
	BootstrapToken string    `gorm:"size:255" json:"-"`
	NodeCert       string    `gorm:"type:text" json:"-"`
	CertSerial     string    `gorm:"size:255" json:"cert_serial,omitempty"`
	CertExpires    *time.Time `json:"cert_expires,omitempty"`
	LastConnMode   string    `gorm:"size:50" json:"last_conn_mode,omitempty"`
}

func (n *Node) CanView(userID int64) bool {
	if n.OwnerID == userID {
		return true
	}
	if n.IsPrivate {
		return false
	}
	visibleTo := n.GetVisibleTo()
	hiddenFrom := n.GetHiddenFrom()
	hasVisibleRestriction := len(visibleTo) > 0
	hasHiddenRestriction := len(hiddenFrom) > 0

	if hasVisibleRestriction && hasHiddenRestriction {
		for _, id := range hiddenFrom {
			if id == userID {
				return false
			}
		}
		for _, id := range visibleTo {
			if id == userID {
				return true
			}
		}
		return false
	}
	if hasVisibleRestriction {
		for _, id := range visibleTo {
			if id == userID {
				return true
			}
		}
		return false
	}
	if hasHiddenRestriction {
		for _, id := range hiddenFrom {
			if id == userID {
				return false
			}
		}
		return true
	}
	return true
}

func (n *Node) CanEdit(userID int64, userRole string) bool {
	if n.OwnerID == userID {
		return true
	}
	return userRole == "admin"
}

func (n *Node) IsOwner(userID int64) bool {
	return n.OwnerID == userID
}

func (n *Node) GetVisibleTo() []int64 {
	if n.VisibleTo == "" || n.VisibleTo == "[]" {
		return []int64{}
	}
	var ids []int64
	json.Unmarshal([]byte(n.VisibleTo), &ids)
	return ids
}

func (n *Node) SetVisibleTo(ids []int64) {
	b, _ := json.Marshal(ids)
	n.VisibleTo = string(b)
}

func (n *Node) GetHiddenFrom() []int64 {
	if n.HiddenFrom == "" || n.HiddenFrom == "[]" {
		return []int64{}
	}
	var ids []int64
	json.Unmarshal([]byte(n.HiddenFrom), &ids)
	return ids
}

func (n *Node) SetHiddenFrom(ids []int64) {
	b, _ := json.Marshal(ids)
	n.HiddenFrom = string(b)
}

func ParseVisibleTo(jsonStr string) ([]int64, error) {
	if jsonStr == "" || jsonStr == "[]" {
		return []int64{}, nil
	}
	var ids []int64
	err := json.Unmarshal([]byte(jsonStr), &ids)
	return ids, err
}

func ParseHiddenFrom(jsonStr string) ([]int64, error) {
	if jsonStr == "" || jsonStr == "[]" {
		return []int64{}, nil
	}
	var ids []int64
	err := json.Unmarshal([]byte(jsonStr), &ids)
	return ids, err
}