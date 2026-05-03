package tunnel

import (
	"crypto/tls"
	"io"
	"math/rand"
	"net/http"
	"sync"
	"time"
)

type ObfuscationMode string

const (
	ObfuscationNone          ObfuscationMode = "none"
	ObfuscationHTTP2        ObfuscationMode = "http2_masquerade"
	ObfuscationDomainFront  ObfuscationMode = "domain_fronting"
	ObfuscationPadding      ObfuscationMode = "traffic_padding"
)

type ObfuscationConfig struct {
	Mode   ObfuscationMode        `json:"mode"`
	Config map[string]interface{} `json:"config,omitempty"`
}

type Obfuscator interface {
	WrapConn(conn io.ReadWriteCloser) (io.ReadWriteCloser, error)
	WrapTLS(conn *tls.Conn) (*tls.Conn, error)
}

type HTTP2Masquerade struct {
	websiteRoot    string
	tunnelEndpoint string
	enabled        bool
	fakeRequests   bool
	fakeInterval   time.Duration
}

func NewHTTP2Masquerade(cfg *ObfuscationConfig) *HTTP2Masquerade {
	return &HTTP2Masquerade{
		websiteRoot:    getString(cfg.Config, "website_root", "/var/www/html"),
		tunnelEndpoint: getString(cfg.Config, "tunnel_endpoint", "/api/v1/stream"),
		fakeRequests:   getBool(cfg.Config, "fake_requests", true),
		fakeInterval:   time.Duration(getInt(cfg.Config, "interval", 30)) * time.Second,
	}
}

func (h *HTTP2Masquerade) Start() {
	if !h.fakeRequests {
		return
	}

	go func() {
		ticker := time.NewTicker(h.fakeInterval)
		for range ticker.C {
			h.injectFakeRequest()
		}
	}()
}

func (h *HTTP2Masquerade) injectFakeRequest() {
	endpoints := []string{"/", "/index.html", "/about.html"}
	endpoint := endpoints[rand.Intn(len(endpoints))]

	client := &http.Client{Timeout: 5 * time.Second}
	req, _ := http.NewRequest("GET", endpoint, nil)
	client.Do(req)
}

func (h *HTTP2Masquerade) WrapConn(conn io.ReadWriteCloser) (io.ReadWriteCloser, error) {
	return &HTTP2Wrapper{
		conn:           conn,
		websiteRoot:    h.websiteRoot,
		tunnelEndpoint: h.tunnelEndpoint,
	}, nil
}

func (h *HTTP2Masquerade) WrapTLS(conn *tls.Conn) (*tls.Conn, error) {
	return conn, nil
}

type HTTP2Wrapper struct {
	conn           io.ReadWriteCloser
	websiteRoot    string
	tunnelEndpoint string
}

func (w *HTTP2Wrapper) Read(p []byte) (n int, err error) {
	return w.conn.Read(p)
}

func (w *HTTP2Wrapper) Write(p []byte) (n int, err error) {
	return w.conn.Write(p)
}

func (w *HTTP2Wrapper) Close() error {
	return w.conn.Close()
}

type DomainFronting struct {
	sniDomain  string
	hostDomain string
}

func NewDomainFronting(cfg *ObfuscationConfig) *DomainFronting {
	return &DomainFronting{
		sniDomain:  getString(cfg.Config, "sni_domain", "ajax.googleapis.com"),
		hostDomain: getString(cfg.Config, "host_domain", "tunnel.example.com"),
	}
}

func (d *DomainFronting) WrapConn(conn io.ReadWriteCloser) (io.ReadWriteCloser, error) {
	return &DomainFrontWrapper{
		conn:      conn,
		sniDomain:  d.sniDomain,
		hostDomain: d.hostDomain,
	}, nil
}

func (d *DomainFronting) WrapTLS(conn *tls.Conn) (*tls.Conn, error) {
	return conn, nil
}

type DomainFrontWrapper struct {
	conn       io.ReadWriteCloser
	sniDomain  string
	hostDomain string
}

func (w *DomainFrontWrapper) Read(p []byte) (n int, err error) {
	return w.conn.Read(p)
}

func (w *DomainFrontWrapper) Write(p []byte) (n int, err error) {
	return w.conn.Write(p)
}

func (w *DomainFrontWrapper) Close() error {
	return w.conn.Close()
}

type TrafficPadding struct {
	minPacketSize int
	maxPacketSize int
	minInterval   time.Duration
	maxInterval   time.Duration
	fakeRatio     float64
	enabled       bool
	mu            sync.Mutex
}

func NewTrafficPadding(cfg *ObfuscationConfig) *TrafficPadding {
	tp := &TrafficPadding{
		minPacketSize: getInt(cfg.Config, "min_packet_size", 1024),
		maxPacketSize: getInt(cfg.Config, "max_packet_size", 16384),
		minInterval:   time.Duration(getInt(cfg.Config, "min_interval", 0)) * time.Millisecond,
		maxInterval:   time.Duration(getInt(cfg.Config, "max_interval", 500)) * time.Millisecond,
		fakeRatio:     getFloat(cfg.Config, "fake_traffic_ratio", 0.1),
		enabled:       true,
	}
	return tp
}

func (t *TrafficPadding) WrapConn(conn io.ReadWriteCloser) (io.ReadWriteCloser, error) {
	return &PaddingWrapper{
		conn:    conn,
		padding: t,
	}, nil
}

func (t *TrafficPadding) WrapTLS(conn *tls.Conn) (*tls.Conn, error) {
	return conn, nil
}

type PaddingWrapper struct {
	conn    io.ReadWriteCloser
	padding *TrafficPadding
}

func (w *PaddingWrapper) Read(p []byte) (n int, err error) {
	return w.conn.Read(p)
}

func (w *PaddingWrapper) Write(p []byte) (n int, err error) {
	dataLen := len(p)

	if dataLen < w.padding.minPacketSize {
		padding := make([]byte, w.padding.minPacketSize-dataLen)
		rand.Read(padding)
		p = append(p, padding...)
	} else if dataLen > w.padding.maxPacketSize {
		p = p[:w.padding.maxPacketSize]
	}

	return w.conn.Write(p)
}

func (w *PaddingWrapper) Close() error {
	return w.conn.Close()
}

func CreateObfuscator(cfg *ObfuscationConfig) Obfuscator {
	switch cfg.Mode {
	case ObfuscationHTTP2:
		return NewHTTP2Masquerade(cfg)
	case ObfuscationDomainFront:
		return NewDomainFronting(cfg)
	case ObfuscationPadding:
		return NewTrafficPadding(cfg)
	default:
		return &NoOpObfuscator{}
	}
}

type NoOpObfuscator struct{}

func (n *NoOpObfuscator) WrapConn(conn io.ReadWriteCloser) (io.ReadWriteCloser, error) {
	return conn, nil
}

func (n *NoOpObfuscator) WrapTLS(conn *tls.Conn) (*tls.Conn, error) {
	return conn, nil
}

func getString(cfg map[string]interface{}, key, defaultVal string) string {
	if v, ok := cfg[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return defaultVal
}

func getInt(cfg map[string]interface{}, key string, defaultVal int) int {
	if v, ok := cfg[key]; ok {
		if f, ok := v.(float64); ok {
			return int(f)
		}
	}
	return defaultVal
}

func getBool(cfg map[string]interface{}, key string, defaultVal bool) bool {
	if v, ok := cfg[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return defaultVal
}

func getFloat(cfg map[string]interface{}, key string, defaultVal float64) float64 {
	if v, ok := cfg[key]; ok {
		if f, ok := v.(float64); ok {
			return f
		}
	}
	return defaultVal
}
