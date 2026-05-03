package transport

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"

	"anthill-runtime/internal/protocol"
)

type Server struct {
	addr     string
	tlsCert  *tls.Certificate
	upgrader websocket.Upgrader
	httpServ *http.Server

	mu       sync.Mutex
	sessions map[string]*Session
	handler  MessageHandler
	logger   *zap.Logger
}

type Session struct {
	ID        string
	NodeID    string
	conn      *websocket.Conn
	rawConn   net.Conn
	reader    *protocol.MessageReader
	writer    *protocol.MessageWriter
	handler   MessageHandler
	closeChan chan struct{}
}

type MessageHandler interface {
	HandleMessage(session *Session, msg *protocol.Message) error
}

func NewServer(addr string, cert *tls.Certificate, logger *zap.Logger) *Server {
	return &Server{
		addr:     addr,
		tlsCert:  cert,
		sessions: make(map[string]*Session),
		logger:   logger,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
	}
}

func (s *Server) Start(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/runtime/conn", s.handleWebSocket)

	s.httpServ = &http.Server{
		Addr:    s.addr,
		Handler: mux,
	}

	var listener net.Listener
	var err error

	if s.tlsCert != nil {
		listener, err = tls.Listen("tcp", s.addr, &tls.Config{Certificates: []tls.Certificate{*s.tlsCert}})
	} else {
		listener, err = net.Listen("tcp", s.addr)
	}

	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	s.logger.Info("Server started", zap.String("addr", s.addr))

	go s.acceptLoop(listener)

	<-ctx.Done()
	return s.httpServ.Close()
}

func (s *Server) acceptLoop(listener net.Listener) {
	for {
		conn, err := listener.Accept()
		if err != nil {
			s.logger.Error("Accept error", zap.Error(err))
			return
		}

		go s.handleConn(conn)
	}
}

func (s *Server) handleConn(conn net.Conn) {
	defer conn.Close()

	tlsConn, ok := conn.(*tls.Conn)
	if ok {
		if err := tlsConn.Handshake(); err != nil {
			s.logger.Error("TLS handshake failed", zap.Error(err))
			return
		}
		state := tlsConn.ConnectionState()
		s.logger.Info("TLS connection accepted",
			zap.Bool("mutual_tls", len(state.PeerCertificates) > 0),
		)
	}

	session := &Session{
		conn:      nil,
		rawConn:   conn,
		reader:    nil,
		writer:    nil,
		closeChan: make(chan struct{}),
	}

	s.mu.Lock()
	s.sessions[session.ID] = session
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		delete(s.sessions, session.ID)
		s.mu.Unlock()
	}()

	buf := make([]byte, 4096)
	for {
		select {
		case <-session.closeChan:
			return
		default:
		}

		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		n, err := conn.Read(buf)
		if err != nil {
			return
		}
		if s.handler != nil && session.rawConn != nil {
			msg := &protocol.Message{Type: protocol.MessageTypeData, Payload: buf[:n]}
			s.handler.HandleMessage(session, msg)
		}
	}
}

func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.logger.Error("WebSocket upgrade failed", zap.Error(err))
		return
	}

	session := &Session{
		conn:      conn,
		reader:    protocol.NewMessageReader(conn),
		writer:    protocol.NewMessageWriter(conn),
		closeChan: make(chan struct{}),
	}

	s.mu.Lock()
	s.sessions[session.ID] = session
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		delete(s.sessions, session.ID)
		s.mu.Unlock()
	}()

	s.readLoop(session)
}

func (s *Server) readLoop(session *Session) {
	for {
		select {
		case <-session.closeChan:
			return
		default:
		}

		session.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		msg, err := session.reader.ReadMessage()
		if err != nil {
			return
		}

		if s.handler != nil {
			s.handler.HandleMessage(session, msg)
		}
	}
}

func (s *Server) SetHandler(h MessageHandler) {
	s.handler = h
}

func (s *Server) Send(sessionID string, msg *protocol.Message) error {
	s.mu.Lock()
	session, ok := s.sessions[sessionID]
	s.mu.Unlock()

	if !ok {
		return fmt.Errorf("session not found")
	}

	return session.writer.WriteMessage(msg)
}

func (s *Server) CloseSession(sessionID string) error {
	s.mu.Lock()
	session, ok := s.sessions[sessionID]
	s.mu.Unlock()

	if !ok {
		return fmt.Errorf("session not found")
	}

	close(session.closeChan)
	return session.conn.Close()
}

func (s *Session) Close() error {
	select {
	case <-s.closeChan:
	default:
		close(s.closeChan)
	}
	return s.conn.Close()
}

func (s *Session) Send(msg *protocol.Message) error {
	if s.writer != nil {
		return s.writer.WriteMessage(msg)
	}
	if s.rawConn != nil {
		_, err := s.rawConn.Write(msg.Payload)
		return err
	}
	return fmt.Errorf("no connection available")
}