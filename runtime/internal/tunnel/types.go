package tunnel

import "time"

type TunnelType string

const (
    TunnelTypePortForward TunnelType = "port_forward"
    TunnelTypeSOCKS5     TunnelType = "socks5"
    TunnelTypeHTTP       TunnelType = "http"
)

type TransportMode string

const (
    TransportDirect TransportMode = "direct"
    TransportRelay TransportMode = "relay"
    TransportAuto  TransportMode = "auto"
)

type TunnelConfig struct {
    ID              string        `json:"tunnel_id"`
    Type            TunnelType   `json:"type"`
    TransportMode   TransportMode `json:"transport_mode"`
    LocalAddr       string        `json:"local_addr"`
    RemoteAddr      string        `json:"remote_addr"`
    ObfuscationMode string        `json:"obfuscation_mode"`
    ObfuscationCfg  map[string]interface{} `json:"obfuscation_config,omitempty"`
    E2EEnabled      bool          `json:"e2e_enabled"`
    Status          string        `json:"status"`
    CreatedAt       time.Time     `json:"created_at"`
}

type TunnelSession struct {
    ID           string    `json:"id"`
    TunnelID     string    `json:"tunnel_id"`
    LocalConnID  uint64    `json:"local_conn_id"`
    RemoteConnID uint64    `json:"remote_conn_id"`
    BytesIn      int64     `json:"bytes_in"`
    BytesOut     int64     `json:"bytes_out"`
    Status       string    `json:"status"`
    StartedAt    time.Time `json:"started_at"`
}