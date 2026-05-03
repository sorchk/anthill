package model

import (
	"encoding/json"
	"time"
)

type Node struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Host        string    `json:"host"`
	Port        int       `json:"port"`
	SSHHost     string    `json:"ssh_host,omitempty"`
	SSHPort     int       `json:"ssh_port,omitempty"`
	SSHUsername string    `json:"ssh_username,omitempty"`
	SSHPassword string    `json:"-"`
	SSHKeyPath  string    `json:"ssh_key_path,omitempty"`
	TLSCertPath string    `json:"tls_cert_path,omitempty"`
	TLSCertCN   string    `json:"tls_cert_cn,omitempty"`
	Status      string    `json:"status"`
	LastSeen    time.Time `json:"last_seen,omitempty"`
	NodeGroup   string    `json:"node_group,omitempty"`
	IsPrivate   bool      `json:"is_private"`
	OwnerID     int64     `json:"owner_id"`
	VisibleTo   []int64   `json:"visible_to,omitempty"`
	HiddenFrom  []int64   `json:"hidden_from,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	ConnectMode    string    `json:"connect_mode"`     // 连接模式: passive, active_tls, active_wss, auto
	NodePort       int       `json:"node_port"`        // 节点监听端口，默认 18888
	NodeHost       string    `json:"node_host"`        // 节点公网地址（主动模式用）
	BootstrapToken string    `json:"-"` // bootstrap 认证 token
	NodeCert       string    `json:"-"` // 节点证书（PEM 格式）
	CertSerial     string    `json:"cert_serial,omitempty"`    // 证书序列号
	CertExpires    time.Time `json:"cert_expires,omitempty"`   // 证书过期时间
	LastConnMode   string    `json:"last_conn_mode,omitempty"` // 当前连接模式
}

func (n *Node) CanView(userID int64) bool {
	if n.OwnerID == userID {
		return true
	}
	if n.IsPrivate {
		return false
	}
	hasVisibleRestriction := len(n.VisibleTo) > 0
	hasHiddenRestriction := len(n.HiddenFrom) > 0

	if hasVisibleRestriction && hasHiddenRestriction {
		for _, id := range n.HiddenFrom {
			if id == userID {
				return false
			}
		}
		for _, id := range n.VisibleTo {
			if id == userID {
				return true
			}
		}
		return false
	}
	if hasVisibleRestriction {
		for _, id := range n.VisibleTo {
			if id == userID {
				return true
			}
		}
		return false
	}
	if hasHiddenRestriction {
		for _, id := range n.HiddenFrom {
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