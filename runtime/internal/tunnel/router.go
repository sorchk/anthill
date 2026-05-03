package tunnel

import (
    "fmt"
    "strings"

    "go.uber.org/zap"
)

type Router struct {
    manager *Manager
    logger  *zap.Logger
}

func NewRouter(m *Manager, logger *zap.Logger) *Router {
    return &Router{manager: m, logger: logger}
}

func (r *Router) HandleTunnelOpen(addr string) (*Tunnel, error) {
    parts := strings.Split(addr, "->")
    if len(parts) != 2 {
        return nil, fmt.Errorf("invalid tunnel address format: %s", addr)
    }

    localAddr := parts[0]
    remoteAddr := parts[1]

    tunnelType := TunnelTypePortForward
    relayMode := TransportDirect

    if strings.HasPrefix(localAddr, "socks5://") {
        tunnelType = TunnelTypeSOCKS5
        localAddr = strings.TrimPrefix(localAddr, "socks5://")
    } else if strings.HasPrefix(localAddr, "http://") {
        tunnelType = TunnelTypeHTTP
        localAddr = strings.TrimPrefix(localAddr, "http://")
    } else if strings.HasPrefix(remoteAddr, "relay://") {
        relayMode = TransportRelay
        remoteAddr = strings.TrimPrefix(remoteAddr, "relay://")
    }

    cfg := &TunnelConfig{
        ID:            fmt.Sprintf("tunnel-%d", r.manager.NextConnID()),
        Type:          tunnelType,
        TransportMode: relayMode,
        LocalAddr:     localAddr,
        RemoteAddr:    remoteAddr,
        E2EEnabled:    true,
        Status:        "pending",
    }

    if err := r.manager.CreateTunnel(cfg); err != nil {
        return nil, err
    }

    tunnel, _ := r.manager.GetTunnel(cfg.ID)

    var err error
    switch tunnelType {
    case TunnelTypePortForward:
        pf := NewPortForward(tunnel, localAddr, remoteAddr, relayMode == TransportRelay, r.logger)
        err = pf.Start()
    case TunnelTypeSOCKS5:
        proxy := NewSOCKS5Proxy(tunnel, localAddr, r.logger)
        err = proxy.Start()
    default:
        err = fmt.Errorf("unsupported tunnel type: %s", tunnelType)
    }

    if err != nil {
        r.manager.DeleteTunnel(cfg.ID)
        return nil, err
    }

    cfg.Status = "active"
    r.logger.Info("Tunnel opened", zap.String("id", cfg.ID), zap.String("type", string(tunnelType)))

    return tunnel, nil
}