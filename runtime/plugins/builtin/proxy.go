package builtin

import (
	"encoding/json"
	"fmt"
	"net"
	"sync"
)

type ProxyPlugin struct {
	httpProxy   *net.Listener
	socksProxy  *net.Listener
	httpStatus  string
	socksStatus string
	mu          sync.RWMutex
}

type ProxyConfig struct {
	HTTPEnabled  bool `json:"http_enabled"`
	HTTPAddr     string `json:"http_addr"`
	SOCKSEnabled bool   `json:"socks_enabled"`
	SOCKSAddr    string `json:"socks_addr"`
	IdleTimeout  int    `json:"idle_timeout"`
}

func NewProxyPlugin() *ProxyPlugin {
	return &ProxyPlugin{
		httpStatus:  "stopped",
		socksStatus: "stopped",
	}
}

type StartRequest struct {
	Type string `json:"type"`
	Port int    `json:"port"`
}

func (p *ProxyPlugin) Invoke(method string, args json.RawMessage) (json.RawMessage, error) {
	switch method {
	case "start":
		return p.handleStart(args)
	case "stop":
		return p.handleStop(args)
	case "status":
		return p.handleStatus(args)
	case "configure":
		return p.handleConfigure(args)
	default:
		return nil, fmt.Errorf("unknown method: %s", method)
	}
}

func (p *ProxyPlugin) handleStart(args json.RawMessage) (json.RawMessage, error) {
	var req StartRequest
	if err := json.Unmarshal(args, &req); err != nil {
		return nil, err
	}

	addr := fmt.Sprintf(":%d", req.Port)

	switch req.Type {
	case "http":
		listener, err := net.Listen("tcp", addr)
		if err != nil {
			return nil, err
		}
		p.mu.Lock()
		p.httpProxy = &listener
		p.httpStatus = "running"
		p.mu.Unlock()

		go p.serveHTTP(&listener)
		return json.Marshal(map[string]string{"status": "started", "addr": addr})

	case "socks5":
		listener, err := net.Listen("tcp", addr)
		if err != nil {
			return nil, err
		}
		p.mu.Lock()
		p.socksProxy = &listener
		p.socksStatus = "running"
		p.mu.Unlock()

		go p.serveSOCKS5(&listener)
		return json.Marshal(map[string]string{"status": "started", "addr": addr})

	default:
		return nil, fmt.Errorf("unknown proxy type: %s", req.Type)
	}
}

func (p *ProxyPlugin) handleStop(args json.RawMessage) (json.RawMessage, error) {
	var req struct{ Type string }
	if err := json.Unmarshal(args, &req); err != nil {
		return nil, err
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	switch req.Type {
	case "http":
		if p.httpProxy != nil {
			(*p.httpProxy).Close()
			p.httpProxy = nil
			p.httpStatus = "stopped"
		}
	case "socks5":
		if p.socksProxy != nil {
			(*p.socksProxy).Close()
			p.socksProxy = nil
			p.socksStatus = "stopped"
		}
	}

	return json.Marshal(map[string]string{"status": "stopped"})
}

func (p *ProxyPlugin) handleStatus(args json.RawMessage) (json.RawMessage, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return json.Marshal(map[string]string{
		"http_proxy":  p.httpStatus,
		"socks_proxy": p.socksStatus,
	})
}

func (p *ProxyPlugin) handleConfigure(args json.RawMessage) (json.RawMessage, error) {
	var cfg ProxyConfig
	if err := json.Unmarshal(args, &cfg); err != nil {
		return nil, err
	}

	return json.Marshal(map[string]bool{"success": true})
}

func (p *ProxyPlugin) serveHTTP(listener *net.Listener) {
	for {
		conn, err := (*listener).Accept()
		if err != nil {
			return
		}
		go p.handleHTTPConn(conn)
	}
}

func (p *ProxyPlugin) handleHTTPConn(conn net.Conn) {
	defer conn.Close()

	var buf [8192]byte
	n, err := conn.Read(buf[:])
	if err != nil {
		return
	}

	conn.Write([]byte("HTTP/1.1 501 Not Implemented\r\n\r\n"))
}

func (p *ProxyPlugin) serveSOCKS5(listener *net.Listener) {
	for {
		conn, err := (*listener).Accept()
		if err != nil {
			return
		}
		go p.handleSOCKS5Conn(conn)
	}
}

func (p *ProxyPlugin) handleSOCKS5Conn(conn net.Conn) {
	defer conn.Close()

	var buf [256]byte
	n, err := conn.Read(buf[:])
	if err != nil || n < 2 {
		return
	}

	conn.Write([]byte{0x05, 0x00})
}