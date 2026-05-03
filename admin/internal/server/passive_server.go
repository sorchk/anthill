package server

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/gorilla/websocket"

	"anthill/admin/internal/handler"
)

type PassiveServer struct {
	port       int
	ca         *handler.CertCA
	connMgr    *ConnManager
	server     *http.Server
	stopCh     chan struct{}
	tlListener net.Listener
}

func NewPassiveServer(port int, ca *handler.CertCA, connMgr *ConnManager) *PassiveServer {
	return &PassiveServer{
		port:    port,
		ca:      ca,
		connMgr: connMgr,
		stopCh:  make(chan struct{}),
	}
}

func (s *PassiveServer) Start() error {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", s.port))
	if err != nil {
		return fmt.Errorf("failed to listen on port %d: %w", s.port, err)
	}

	tlsConfig, err := s.createTLSConfig()
	if err != nil {
		ln.Close()
		return fmt.Errorf("failed to create TLS config: %w", err)
	}

	tlsListener := tls.NewListener(ln, tlsConfig)
	s.tlListener = tlsListener

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", s.handleWebSocket)
	mux.HandleFunc("/", s.handleTLS)

	s.server = &http.Server{
		Handler: mux,
	}

	go func() {
		if err := s.server.Serve(tlsListener); err != nil && err != http.ErrServerClosed {
			fmt.Printf("PassiveServer error: %v\n", err)
		}
	}()

	fmt.Printf("PassiveServer started on port %d\n", s.port)
	return nil
}

func (s *PassiveServer) Stop() error {
	close(s.stopCh)

	if s.tlListener != nil {
		s.tlListener.Close()
	}

	if s.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return s.server.Shutdown(ctx)
	}
	return nil
}

func (s *PassiveServer) createTLSConfig() (*tls.Config, error) {
	caCert, err := s.getCACertPool()
	if err != nil {
		return nil, err
	}

	return &tls.Config{
		Certificates: []tls.Certificate{*s.ca.GetCACertTLS()},
		ClientCAs:   caCert,
		ClientAuth:  tls.RequireAndVerifyClientCert,
		MinVersion:  tls.VersionTLS12,
		NextProtos:  []string{"h2", "http/1.1"},
	}, nil
}

func (s *PassiveServer) getCACertPool() (*x509.CertPool, error) {
	certPEM := s.ca.GetCACert()
	if certPEM == nil {
		return nil, fmt.Errorf("CA certificate is nil")
	}

	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(certPEM) {
		return nil, fmt.Errorf("failed to append CA certificate to pool")
	}

	return pool, nil
}

func (s *PassiveServer) handleTLS(w http.ResponseWriter, r *http.Request) {
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "hijack not supported", http.StatusInternalServerError)
		return
	}

	conn, _, err := hijacker.Hijack()
	if err != nil {
		http.Error(w, "failed to hijack", http.StatusInternalServerError)
		return
	}
	defer conn.Close()

	tlsConn, ok := conn.(*tls.Conn)
	if !ok {
		http.Error(w, "not a TLS connection", http.StatusBadRequest)
		return
	}

	if err := tlsConn.Handshake(); err != nil {
		fmt.Printf("TLS handshake error: %v\n", err)
		return
	}

	state := tlsConn.ConnectionState()
	if len(state.PeerCertificates) == 0 {
		fmt.Printf("No peer certificates found\n")
		return
	}

	nodeID := state.PeerCertificates[0].Subject.CommonName
	if nodeID == "" {
		fmt.Printf("Node ID (CN) is empty\n")
		return
	}

	if err := s.handleConnection(tlsConn, nodeID, "tls"); err != nil {
		fmt.Printf("TLS connection handler error: %v\n", err)
	}
}

func (s *PassiveServer) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	if r.TLS == nil || len(r.TLS.PeerCertificates) == 0 {
		fmt.Printf("WebSocket TLS verification failed: no peer certificates\n")
		return
	}

	nodeID := r.TLS.PeerCertificates[0].Subject.CommonName
	if nodeID == "" {
		fmt.Printf("WebSocket TLS verification failed: empty node ID\n")
		return
	}

	upgrader := websocket.Upgrader{
		CheckOrigin: s.checkOrigin,
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}

	wsConn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Printf("WebSocket upgrade failed: %v\n", err)
		return
	}
	defer wsConn.Close()

	wrapper := NewWebSocketConn(wsConn)
	if err := s.handleConnection(wrapper, nodeID, "wss"); err != nil {
		fmt.Printf("WebSocket connection handler error: %v\n", err)
	}
}

func (s *PassiveServer) checkOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	return false
}

func (s *PassiveServer) handleConnection(conn net.Conn, nodeID string, protocol string) error {
	defer conn.Close()

	if s.connMgr != nil {
		if err := s.connMgr.AddConnection(nodeID, conn, protocol, "passive"); err != nil {
			return fmt.Errorf("failed to add connection: %w", err)
		}
		defer s.connMgr.RemoveConnection(nodeID)
	}

	fmt.Printf("Node %s connected successfully\n", nodeID)

	buf := make([]byte, 4096)
	for {
		select {
		case <-s.stopCh:
			return nil
		default:
			conn.SetReadDeadline(time.Now().Add(30 * time.Second))
			n, err := conn.Read(buf)
			if err != nil {
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					continue
				}
				return nil
			}
			fmt.Printf("Received %d bytes from node\n", n)
		}
	}
}

type WebSocketConn struct {
	conn *websocket.Conn
}

func NewWebSocketConn(conn *websocket.Conn) *WebSocketConn {
	return &WebSocketConn{conn: conn}
}

func (w *WebSocketConn) Read(b []byte) (n int, err error) {
	messageType, msg, err := w.conn.ReadMessage()
	if err != nil {
		return 0, err
	}
	if messageType == websocket.TextMessage || messageType == websocket.BinaryMessage {
		copy(b, msg)
		return len(msg), nil
	}
	return 0, nil
}

func (w *WebSocketConn) Write(b []byte) (n int, err error) {
	err = w.conn.WriteMessage(websocket.BinaryMessage, b)
	if err != nil {
		return 0, err
	}
	return len(b), nil
}

func (w *WebSocketConn) Close() error {
	return w.conn.Close()
}

func (w *WebSocketConn) SetReadDeadline(t time.Time) error {
	return w.conn.SetReadDeadline(t)
}

func (w *WebSocketConn) SetWriteDeadline(t time.Time) error {
	return w.conn.SetWriteDeadline(t)
}

func (w *WebSocketConn) LocalAddr() net.Addr {
	return w.conn.LocalAddr()
}

func (w *WebSocketConn) RemoteAddr() net.Addr {
	return w.conn.RemoteAddr()
}

func (w *WebSocketConn) SetDeadline(t time.Time) error {
	return w.conn.UnderlyingConn().SetDeadline(t)
}