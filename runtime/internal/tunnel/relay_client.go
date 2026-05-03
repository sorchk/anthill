package tunnel

import (
	"encoding/binary"
	"fmt"
	"net"
	"sync"
	"time"

	"go.uber.org/zap"
)

type RelayClient struct {
	tunnelID    string
	adminAddr   string
	conn        net.Conn
	e2eEnabled  bool
	encryptor   *E2EEncryptor
	logger      *zap.Logger
	readCh      chan []byte
	writeCh     chan []byte
	closeCh     chan struct{}
	sessionID   uint64
	mu          sync.RWMutex
}

func NewRelayClient(tunnelID, adminAddr string, e2eEnabled bool, encryptor *E2EEncryptor, logger *zap.Logger) *RelayClient {
	return &RelayClient{
		tunnelID:   tunnelID,
		adminAddr:  adminAddr,
		e2eEnabled: e2eEnabled,
		encryptor:  encryptor,
		logger:     logger,
		readCh:     make(chan []byte, 100),
		writeCh:    make(chan []byte, 100),
		closeCh:    make(chan struct{}),
		sessionID:  0,
	}
}

func (c *RelayClient) Connect() error {
	conn, err := net.DialTimeout("tcp", c.adminAddr, 10*time.Second)
	if err != nil {
		return fmt.Errorf("failed to connect to admin: %w", err)
	}
	c.conn = conn

	go c.readLoop()
	go c.writeLoop()

	return nil
}

func (c *RelayClient) readLoop() {
	buf := make([]byte, 65536)
	for {
		select {
		case <-c.closeCh:
			return
		default:
			c.conn.SetReadDeadline(time.Now().Add(30 * time.Second))
			n, err := c.conn.Read(buf)
			if err != nil {
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					continue
				}
				c.logger.Error("Relay read error", zap.Error(err))
				close(c.closeCh)
				return
			}

			data := make([]byte, n)
			copy(data, buf[:n])

			if c.e2eEnabled && c.encryptor != nil {
				plaintext, err := c.encryptor.Decrypt(data)
				if err != nil {
					c.logger.Error("Failed to decrypt relay data", zap.Error(err))
					continue
				}
				c.readCh <- plaintext
			} else {
				c.readCh <- data
			}
		}
	}
}

func (c *RelayClient) writeLoop() {
	for {
		select {
		case <-c.closeCh:
			return
		case data := <-c.writeCh:
			var writeData []byte
			if c.e2eEnabled && c.encryptor != nil {
				encrypted, err := c.encryptor.Encrypt(data)
				if err != nil {
					c.logger.Error("Failed to encrypt data", zap.Error(err))
					continue
				}
				writeData = encrypted
			} else {
				writeData = data
			}

			c.mu.Lock()
			sessionID := c.sessionID
			c.mu.Unlock()

			msg := c.buildTunnelDataMsg(sessionID, writeData)
			if _, err := c.conn.Write(msg); err != nil {
				c.logger.Error("Relay write error", zap.Error(err))
				close(c.closeCh)
				return
			}
		}
	}
}

func (c *RelayClient) buildTunnelDataMsg(sessionID uint64, data []byte) []byte {
	msgLen := 1 + 8 + 4 + len(data)
	msg := make([]byte, msgLen)

	msg[0] = 0x07

	binary.BigEndian.PutUint64(msg[1:9], sessionID)
	binary.BigEndian.PutUint32(msg[9:13], uint32(len(data)))
	copy(msg[13:], data)

	return msg
}

func (c *RelayClient) Write(data []byte) error {
	select {
	case c.writeCh <- data:
		return nil
	case <-c.closeCh:
		return fmt.Errorf("relay client closed")
	default:
		return fmt.Errorf("write buffer full")
	}
}

func (c *RelayClient) Read() ([]byte, error) {
	select {
	case data := <-c.readCh:
		return data, nil
	case <-c.closeCh:
		return nil, fmt.Errorf("relay client closed")
	}
}

func (c *RelayClient) Close() error {
	close(c.closeCh)
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

func (c *RelayClient) LocalAddr() net.Addr {
	if c.conn != nil {
		return c.conn.LocalAddr()
	}
	return nil
}

func (c *RelayClient) RemoteAddr() net.Addr {
	if c.conn != nil {
		return c.conn.RemoteAddr()
	}
	return nil
}

type RelayStats struct {
	BytesIn  int64
	BytesOut int64
}

func (c *RelayClient) GetStats() *RelayStats {
	return &RelayStats{
		BytesIn:  0,
		BytesOut: 0,
	}
}

type RelayDialer struct {
	AdminAddr string
	TLSConfig interface{}
	Logger    *zap.Logger
}

func NewRelayDialer(adminAddr string, logger *zap.Logger) *RelayDialer {
	return &RelayDialer{
		AdminAddr: adminAddr,
		Logger:    logger,
	}
}

func (d *RelayDialer) Dial(tunnelID string, e2eEnabled bool, encryptor *E2EEncryptor) (*RelayClient, error) {
	client := NewRelayClient(tunnelID, d.AdminAddr, e2eEnabled, encryptor, d.Logger)
	if err := client.Connect(); err != nil {
		return nil, err
	}
	return client, nil
}

func (d *RelayDialer) DialWithTimeout(tunnelID string, e2eEnabled bool, encryptor *E2EEncryptor, timeout time.Duration) (*RelayClient, error) {
	client := NewRelayClient(tunnelID, d.AdminAddr, e2eEnabled, encryptor, d.Logger)

	conn, err := net.DialTimeout("tcp", d.AdminAddr, timeout)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to admin: %w", err)
	}
	client.conn = conn

	go client.readLoop()
	go client.writeLoop()

	return client, nil
}

type RelayConnection struct {
	Client    *RelayClient
	TunnelID  string
	SessionID uint64
	Status    string
	CreatedAt time.Time
}

func (c *RelayConnection) IsActive() bool {
	return c.Status == "active"
}

func (c *RelayConnection) Close() error {
	c.Status = "closed"
	return c.Client.Close()
}

type RelayPool struct {
	mu        sync.RWMutex
	clients   map[string]*RelayConnection
	maxConns  int
	adminAddr string
	logger    *zap.Logger
}

func NewRelayPool(adminAddr string, maxConns int, logger *zap.Logger) *RelayPool {
	return &RelayPool{
		clients:   make(map[string]*RelayConnection),
		maxConns:  maxConns,
		adminAddr: adminAddr,
		logger:    logger,
	}
}

func (p *RelayPool) Get(tunnelID string) (*RelayConnection, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	conn, ok := p.clients[tunnelID]
	return conn, ok
}

func (p *RelayPool) Add(tunnelID string, conn *RelayConnection) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(p.clients) >= p.maxConns {
		return fmt.Errorf("relay pool full")
	}

	p.clients[tunnelID] = conn
	return nil
}

func (p *RelayPool) Remove(tunnelID string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if conn, ok := p.clients[tunnelID]; ok {
		conn.Close()
		delete(p.clients, tunnelID)
	}
	return nil
}

func (p *RelayPool) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, conn := range p.clients {
		conn.Close()
	}
	p.clients = make(map[string]*RelayConnection)
	return nil
}

type Hop struct {
	Address string
	IsNode  bool
	IsAdmin bool
}

func ParseHop(address string) *Hop {
	return &Hop{
		Address: address,
		IsAdmin: true,
	}
}