package server

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"time"

	"github.com/gorilla/websocket"
)

type ActiveClient struct {
	nodeID     string
	addr       string
	port       int
	protocol   string
	mode       string
	cert       *tls.Certificate
	caCert     *x509.Certificate
	conn       net.Conn
	reconnect  bool
	backoff    time.Duration
	maxBackoff time.Duration
	stopCh     chan struct{}
	connMgr    *ConnManager
}

func NewActiveClient(nodeID, addr string, port int, protocol string, cert *tls.Certificate, caCert *x509.Certificate, connMgr *ConnManager) *ActiveClient {
	mode := "active_tls"
	if protocol == "wss" {
		mode = "active_wss"
	}

	return &ActiveClient{
		nodeID:     nodeID,
		addr:       addr,
		port:       port,
		protocol:   protocol,
		mode:       mode,
		cert:       cert,
		caCert:     caCert,
		reconnect:  true,
		backoff:    10 * time.Second,
		maxBackoff: 5 * time.Minute,
		stopCh:     make(chan struct{}),
		connMgr:    connMgr,
	}
}

func (c *ActiveClient) Start() error {
	go c.run()
	return nil
}

func (c *ActiveClient) run() {
	for {
		select {
		case <-c.stopCh:
			return
		default:
			if err := c.connect(); err != nil {
				fmt.Printf("ActiveClient[%s] connection error: %v\n", c.nodeID, err)
			}

			if !c.reconnect {
				return
			}

			fmt.Printf("ActiveClient[%s] waiting %v before reconnect...\n", c.nodeID, c.backoff)
			select {
			case <-c.stopCh:
				return
			case <-time.After(c.backoff):
			}

			c.backoff = c.backoff * 2
			if c.backoff > c.maxBackoff {
				c.backoff = c.maxBackoff
			}
		}
	}
}

func (c *ActiveClient) connect() error {
	var conn net.Conn
	var err error

	if c.protocol == "tls" {
		conn, err = c.dialTLS()
	} else if c.protocol == "wss" {
		conn, err = c.dialWSS()
	} else {
		return fmt.Errorf("unsupported protocol: %s", c.protocol)
	}

	if err != nil {
		return fmt.Errorf("failed to dial: %w", err)
	}

	if err := c.verifyNodeCert(conn); err != nil {
		conn.Close()
		return fmt.Errorf("certificate verification failed: %w", err)
	}

	c.conn = conn

	if c.connMgr != nil {
		if err := c.connMgr.AddConnection(c.nodeID, conn, c.protocol, c.mode); err != nil {
			conn.Close()
			return fmt.Errorf("failed to register connection: %w", err)
		}
		defer c.connMgr.RemoveConnection(c.nodeID)
	}

	fmt.Printf("ActiveClient[%s] connected successfully\n", c.nodeID)

	c.backoff = 10 * time.Second

	return c.handleConnection()
}

func (c *ActiveClient) dialTLS() (net.Conn, error) {
	tlsConfig := &tls.Config{
		Certificates:       []tls.Certificate{*c.cert},
		RootCAs:            c.getCertPool(),
		MinVersion:         tls.VersionTLS12,
		InsecureSkipVerify: false,
	}

	addr := fmt.Sprintf("%s:%d", c.addr, c.port)
	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		return nil, err
	}

	if err := conn.Handshake(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("handshake failed: %w", err)
	}

	return conn, nil
}

func (c *ActiveClient) dialWSS() (net.Conn, error) {
	url := fmt.Sprintf("wss://%s:%d/ws", c.addr, c.port)

	tlsConfig := &tls.Config{
		Certificates:       []tls.Certificate{*c.cert},
		RootCAs:            c.getCertPool(),
		MinVersion:         tls.VersionTLS12,
		InsecureSkipVerify: false,
	}

	dialer := websocket.Dialer{
		NetDial: func(network, addr string) (net.Conn, error) {
			return tls.Dial("tcp", addr, tlsConfig)
		},
	}

	wsConn, resp, err := dialer.Dial(url, nil)
	if err != nil {
		return nil, fmt.Errorf("websocket dial failed: %w", err)
	}

	if resp != nil {
		resp.Body.Close()
	}

	return NewWebSocketConn(wsConn), nil
}

func (c *ActiveClient) getCertPool() *x509.CertPool {
	pool := x509.NewCertPool()
	if c.caCert != nil {
		pool.AddCert(c.caCert)
	}
	return pool
}

func (c *ActiveClient) verifyNodeCert(conn net.Conn) error {
	if c.protocol == "tls" {
		tlsConn, ok := conn.(*tls.Conn)
		if !ok {
			return fmt.Errorf("not a TLS connection")
		}

		state := tlsConn.ConnectionState()
		if len(state.PeerCertificates) == 0 {
			return fmt.Errorf("no peer certificates")
		}

		peerCert := state.PeerCertificates[0]
		if peerCert.Subject.CommonName != c.nodeID {
			return fmt.Errorf("certificate CN mismatch: expected %s, got %s", c.nodeID, peerCert.Subject.CommonName)
		}

		opts := x509.VerifyOptions{
			DNSName: c.nodeID,
			Roots:   c.getCertPool(),
		}

		if _, err := peerCert.Verify(opts); err != nil {
			return fmt.Errorf("certificate verification failed: %w", err)
		}
	}

	return nil
}

func (c *ActiveClient) handleConnection() error {
	buf := make([]byte, 4096)
	for {
		select {
		case <-c.stopCh:
			return nil
		default:
			if c.conn == nil {
				return fmt.Errorf("connection is nil")
			}

			c.conn.SetReadDeadline(time.Now().Add(30 * time.Second))
			n, err := c.conn.Read(buf)
			if err != nil {
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					continue
				}
				return fmt.Errorf("read error: %w", err)
			}
			fmt.Printf("ActiveClient[%s] received %d bytes\n", c.nodeID, n)
		}
	}
}

func (c *ActiveClient) Stop() error {
	close(c.stopCh)
	c.reconnect = false

	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}

	return nil
}

func (c *ActiveClient) SendMessage(msg []byte) error {
	if c.conn == nil {
		return fmt.Errorf("not connected")
	}

	_, err := c.conn.Write(msg)
	return err
}
