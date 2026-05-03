package tunnel

import (
    "fmt"
    "io"
    "net"
    "sync"
    "time"

    "go.uber.org/zap"
)

type PortForward struct {
    tunnel    *Tunnel
    localAddr string
    remoteAddr string
    relayMode bool
    e2e       *E2EEncryptor
    logger    *zap.Logger
}

func NewPortForward(t *Tunnel, local, remote string, relay bool, logger *zap.Logger) *PortForward {
    return &PortForward{
        tunnel:    t,
        localAddr: local,
        remoteAddr: remote,
        relayMode: relay,
        logger:    logger,
    }
}

func (pf *PortForward) Start() error {
    listener, err := net.Listen("tcp", pf.localAddr)
    if err != nil {
        return err
    }

    pf.logger.Info("Port forward started",
        zap.String("local", pf.localAddr),
        zap.String("remote", pf.remoteAddr),
        zap.Bool("relay", pf.relayMode))

    go pf.acceptLoop(listener)
    return nil
}

func (pf *PortForward) acceptLoop(listener net.Listener) {
    for {
        conn, err := listener.Accept()
        if err != nil {
            return
        }
        go pf.handleConn(conn)
    }
}

func (pf *PortForward) handleConn(localConn net.Conn) {
    defer localConn.Close()

    connID := pf.tunnel.manager.NextConnID()
    session := &TunnelSession{
        ID:          fmt.Sprintf("%s-%d", pf.tunnel.Config.ID, connID),
        TunnelID:    pf.tunnel.Config.ID,
        LocalConnID: connID,
        Status:      "active",
        StartedAt:   time.Now(),
    }

    pf.tunnel.mu.Lock()
    pf.tunnel.sessions[connID] = session
    pf.tunnel.manager.sessions[connID] = session
    pf.tunnel.mu.Unlock()

    defer func() {
        pf.tunnel.mu.Lock()
        delete(pf.tunnel.sessions, connID)
        delete(pf.tunnel.manager.sessions, connID)
        pf.tunnel.mu.Unlock()
        session.Status = "closed"
    }()

    var remoteConn net.Conn
    var err error

    if pf.relayMode {
        remoteConn, err = pf.dialRelay()
    } else {
        remoteConn, err = net.DialTimeout("tcp", pf.remoteAddr, 10*time.Second)
    }

    if err != nil {
        pf.logger.Error("Failed to connect to remote", zap.Error(err))
        return
    }
    defer remoteConn.Close()

    session.RemoteConnID = connID + 1

    var wg sync.WaitGroup
    wg.Add(2)

    go func() {
        defer wg.Done()
        io.Copy(localConn, remoteConn)
        localConn.Close()
    }()

    go func() {
        defer wg.Done()
        io.Copy(remoteConn, localConn)
        remoteConn.Close()
    }()

    wg.Wait()

    pf.logger.Debug("Port forward session closed",
        zap.String("session", session.ID),
        zap.Int64("bytes_in", session.BytesIn),
        zap.Int64("bytes_out", session.BytesOut))
}

func (pf *PortForward) dialRelay() (net.Conn, error) {
    return net.DialTimeout("tcp", pf.remoteAddr, 10*time.Second)
}