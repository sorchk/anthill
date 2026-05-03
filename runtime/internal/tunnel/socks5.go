package tunnel

import (
    "fmt"
    "io"
    "net"
    "sync"
    "time"

    "go.uber.org/zap"
)

type SOCKS5Proxy struct {
    tunnel    *Tunnel
    listenAddr string
    logger    *zap.Logger
}

func NewSOCKS5Proxy(t *Tunnel, listen string, logger *zap.Logger) *SOCKS5Proxy {
    return &SOCKS5Proxy{
        tunnel:    t,
        listenAddr: listen,
        logger:    logger,
    }
}

const (
    socks5Version = 0x05
    socks5CmdConnect = 0x01
    socks5AuthNone = 0x00
    socks5AuthPassword = 0x02
    socks5AddrIPv4 = 0x01
    socks5AddrDomain = 0x03
    socks5AddrIPv6 = 0x04
    socks5Success = 0x00
    socks5Failure = 0x01
)

func (p *SOCKS5Proxy) Start() error {
    listener, err := net.Listen("tcp", p.listenAddr)
    if err != nil {
        return err
    }

    p.logger.Info("SOCKS5 proxy started", zap.String("addr", p.listenAddr))

    go p.acceptLoop(listener)
    return nil
}

func (p *SOCKS5Proxy) acceptLoop(listener net.Listener) {
    for {
        conn, err := listener.Accept()
        if err != nil {
            return
        }
        go p.handleConn(conn)
    }
}

func (p *SOCKS5Proxy) handleConn(conn net.Conn) {
    defer conn.Close()

    deadline := time.Now().Add(30 * time.Second)
    conn.SetDeadline(deadline)

    if err := p.authenticate(conn); err != nil {
        p.logger.Warn("SOCKS5 auth failed", zap.Error(err))
        return
    }

    targetAddr, err := p.readRequest(conn)
    if err != nil {
        p.logger.Warn("SOCKS5 read request failed", zap.Error(err))
        return
    }

    remoteConn, err := net.DialTimeout("tcp", targetAddr, 10*time.Second)
    if err != nil {
        p.reply(conn, 0x01)
        return
    }
    defer remoteConn.Close()

    p.reply(conn, 0x00)
    conn.SetDeadline(time.Time{})

    var wg sync.WaitGroup
    wg.Add(2)

    go func() {
        defer wg.Done()
        io.Copy(remoteConn, conn)
    }()

    go func() {
        defer wg.Done()
        io.Copy(conn, remoteConn)
    }()

    wg.Wait()
}

func (p *SOCKS5Proxy) authenticate(conn net.Conn) error {
    var buf [2]byte
    if _, err := io.ReadFull(conn, buf[:]); err != nil {
        return err
    }

    if buf[0] != socks5Version {
        return fmt.Errorf("unsupported SOCKS version: %d", buf[0])
    }

    nAuth := int(buf[1])
    if nAuth > 255 {
        return fmt.Errorf("invalid auth count")
    }

    authMethods := make([]byte, nAuth)
    if _, err := io.ReadFull(conn, authMethods); err != nil {
        return err
    }

    hasNone := false
    for _, m := range authMethods {
        if m == socks5AuthNone {
            hasNone = true
            break
        }
    }

    if !hasNone {
        conn.Write([]byte{socks5Version, 0xFF})
        return fmt.Errorf("no acceptable auth method")
    }

    conn.Write([]byte{socks5Version, socks5AuthNone})
    return nil
}

func (p *SOCKS5Proxy) readRequest(conn net.Conn) (string, error) {
    var buf [4]byte
    if _, err := io.ReadFull(conn, buf[:]); err != nil {
        return "", err
    }

    if buf[0] != socks5Version || buf[1] != socks5CmdConnect {
        return "", fmt.Errorf("unsupported command or version")
    }

    addr := ""

    switch buf[3] {
    case socks5AddrIPv4:
        var ip [4]byte
        if _, err := io.ReadFull(conn, ip[:]); err != nil {
            return "", err
        }
        addr = net.IP(ip[:]).String()

    case socks5AddrDomain:
        var domainLen [1]byte
        if _, err := io.ReadFull(conn, domainLen[:]); err != nil {
            return "", err
        }
        domain := make([]byte, domainLen[0])
        if _, err := io.ReadFull(conn, domain); err != nil {
            return "", err
        }
        addr = string(domain)

    case socks5AddrIPv6:
        var ip [16]byte
        if _, err := io.ReadFull(conn, ip[:]); err != nil {
            return "", err
        }
        addr = net.IP(ip[:]).String()
    }

    var port [2]byte
    if _, err := io.ReadFull(conn, port[:]); err != nil {
        return "", err
    }

    portNum := int(port[0])<<8 | int(port[1])
    return fmt.Sprintf("%s:%d", addr, portNum), nil
}

func (p *SOCKS5Proxy) reply(conn net.Conn, rep byte) {
    conn.Write([]byte{socks5Version, rep, 0x00, socks5AddrIPv4})
    conn.Write([]byte{0, 0, 0, 0, 0, 0})
}