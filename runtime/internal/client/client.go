package client

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"

	"anthill-runtime/internal/config"
	"anthill-runtime/internal/protocol"
)

type Client struct {
	cfg       *config.Config
	logger    *zap.Logger
	conn      *websocket.Conn
	connTLS   *tls.Conn
	reader    *protocol.MessageReader
	writer    *protocol.MessageWriter
	nodeID    string
	sessionID string
	status    string

	mu          sync.RWMutex
	pendingReqs map[string]chan *protocol.Message
	handlers    map[byte]MessageHandler
}

type MessageHandler func(msg *protocol.Message)

func NewClient(cfg *config.Config, logger *zap.Logger) *Client {
	return &Client{
		cfg:         cfg,
		logger:      logger,
		pendingReqs: make(map[string]chan *protocol.Message),
		handlers:    make(map[byte]MessageHandler),
		status:      "disconnected",
	}
}

func (c *Client) Connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	var tlsConfig *tls.Config
	if c.cfg.TLSCertFile != "" && c.cfg.TLSKeyFile != "" {
		cert, err := tls.LoadX509KeyPair(c.cfg.TLSCertFile, c.cfg.TLSKeyFile)
		if err != nil {
			return fmt.Errorf("failed to load client cert: %w", err)
		}

		caPool, _ := c.cfg.LoadCACertPool()

		tlsConfig = &tls.Config{
			Certificates:       []tls.Certificate{cert},
			RootCAs:            caPool,
			InsecureSkipVerify: c.cfg.InsecureMode,
		}

		dialer.TLSClientConfig = tlsConfig
	}

	url := c.cfg.AdminURL
	if url == "" {
		if c.cfg.ConnectionMode == config.ModePassiveTLS || c.cfg.ConnectionMode == config.ModeActiveTLS {
			url = "tls://localhost:8080/runtime/conn"
		} else {
			url = "wss://localhost:8080/runtime/conn"
		}
	}

	var err error
	if c.cfg.ConnectionMode == config.ModePassiveTLS || c.cfg.ConnectionMode == config.ModeActiveTLS {
		err = c.connectTLS(ctx, url, tlsConfig)
	} else {
		err = c.connectWSS(ctx, url, &dialer)
	}

	if err != nil {
		return err
	}

	go c.readLoop()

	if err := c.sendHandshake(); err != nil {
		return err
	}

	c.setStatus("connected")
	c.logger.Info("Connected to admin", zap.String("url", url), zap.String("mode", c.cfg.ConnectionMode.String()))

	return nil
}

func (c *Client) connectWSS(ctx context.Context, url string, dialer *websocket.Dialer) error {
	conn, _, err := dialer.DialContext(ctx, url, nil)
	if err != nil {
		return fmt.Errorf("failed to connect to admin (WSS): %w", err)
	}

	c.conn = conn
	c.reader = protocol.NewMessageReader(conn)
	c.writer = protocol.NewMessageWriter(conn)
	return nil
}

func (c *Client) connectTLS(ctx context.Context, url string, tlsConfig *tls.Config) error {
	host := extractHost(url)
	if host == "" {
		host = "localhost:8080"
	}

	dialer := net.Dialer{Timeout: 10 * time.Second}
	conn, err := tls.DialWithDialer(&dialer, "tcp", host, tlsConfig)
	if err != nil {
		return fmt.Errorf("failed to connect to admin (TLS): %w", err)
	}

	c.connTLS = conn
	return nil
}

func extractHost(urlStr string) string {
	u, err := url.Parse(urlStr)
	if err != nil {
		return ""
	}
	if u.Host != "" {
		return u.Host
	}
	return ""
}

func (c *Client) sendHandshake() error {
	req := protocol.HandshakeRequest{
		NodeID:    c.cfg.NodeID,
		Version:   "1.0.0",
		NodeGroup: c.cfg.NodeGroup,
		Hostname:  getHostname(),
	}

	data, err := json.Marshal(req)
	if err != nil {
		return err
	}

	msg := &protocol.Message{
		Type:    protocol.MessageTypeHandshake,
		Payload: data,
	}

	c.mu.RLock()
	err = c.writer.WriteMessage(msg)
	c.mu.RUnlock()

	return err
}

func (c *Client) readLoop() {
	for {
		c.mu.RLock()
		conn := c.conn
		c.mu.RUnlock()

		if conn == nil {
			return
		}

		msg, err := c.reader.ReadMessage()
		if err != nil {
			c.logger.Error("Read error", zap.Error(err))
			c.setStatus("disconnected")
			return
		}

		c.handleMessage(msg)
	}
}

func (c *Client) handleMessage(msg *protocol.Message) {
	if handler, ok := c.handlers[msg.Type]; ok {
		handler(msg)
	}
}

func (c *Client) RegisterHandler(msgType byte, handler MessageHandler) {
	c.handlers[msgType] = handler
}

func (c *Client) SendPluginResult(result *protocol.PluginResult) error {
	data, err := json.Marshal(result)
	if err != nil {
		return err
	}

	msg := &protocol.Message{
		Type:    protocol.MessageTypePluginResult,
		Payload: data,
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.writer.WriteMessage(msg)
}

func (c *Client) SendHeartbeat() error {
	msg := &protocol.Message{
		Type:    protocol.MessageTypeHeartbeat,
		Payload: nil,
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.writer.WriteMessage(msg)
}

func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.setStatus("disconnected")

	if c.conn != nil {
		return c.conn.Close()
	}
	if c.connTLS != nil {
		return c.connTLS.Close()
	}
	return nil
}

func (c *Client) Status() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.status
}

func (c *Client) setStatus(status string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.status = status
}

func getHostname() string {
	hostname, _ := os.Hostname()
	return hostname
}
