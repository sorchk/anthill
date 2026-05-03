package client

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
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

	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	if c.cfg.TLSCertFile != "" && c.cfg.TLSKeyFile != "" {
		cert, err := tls.LoadX509KeyPair(c.cfg.TLSCertFile, c.cfg.TLSKeyFile)
		if err != nil {
			c.mu.Unlock()
			return fmt.Errorf("failed to load client cert: %w", err)
		}

		caPool, _ := c.cfg.LoadCACertPool()

		tlsConfig := &tls.Config{
			Certificates:       []tls.Certificate{cert},
			RootCAs:            caPool,
			InsecureSkipVerify: c.cfg.InsecureMode,
		}

		dialer.TLSClientConfig = tlsConfig
	}

	c.mu.Unlock()

	url := c.cfg.AdminURL
	if url == "" {
		url = "wss://localhost:8080/ws"
	}

	conn, _, err := dialer.DialContext(ctx, url, nil)
	if err != nil {
		return fmt.Errorf("failed to connect to admin: %w", err)
	}

	c.mu.Lock()
	c.conn = conn
	c.reader = protocol.NewMessageReader(conn)
	c.writer = protocol.NewMessageWriter(conn)
	c.mu.Unlock()

	go c.readLoop()

	if err := c.sendHandshake(); err != nil {
		return err
	}

	c.setStatus("connected")
	c.logger.Info("Connected to admin", zap.String("url", url))

	return nil
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
