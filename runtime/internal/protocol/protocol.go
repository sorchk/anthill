package protocol

import (
	"encoding/binary"
	"errors"
	"io"
	"sync"

	"github.com/gorilla/websocket"
)

const (
	MessageTypeHandshake     byte = 0x01
	MessageTypeAuth          byte = 0x02
	MessageTypeAuthResponse  byte = 0x03
	MessageTypeData          byte = 0x04
	MessageTypeControl       byte = 0x05
	MessageTypeTunnelOpen    byte = 0x06
	MessageTypeTunnelData    byte = 0x07
	MessageTypeTunnelClose   byte = 0x08
	MessageTypeTunnelKeyEx   byte = 0x09
	MessageTypePluginInvoke  byte = 0x10
	MessageTypePluginResult  byte = 0x11
	MessageTypeHeartbeat     byte = 0xFE
	MessageTypeClose        byte = 0xFF
)

var (
	ErrInvalidMessage = errors.New("invalid message")
	ErrBufferTooSmall = errors.New("buffer too small")
)

type Message struct {
	Type    byte
	Payload []byte
}

type MessageReader struct {
	conn *websocket.Conn
	buf  []byte
}

func NewMessageReader(conn *websocket.Conn) *MessageReader {
	return &MessageReader{conn: conn, buf: make([]byte, 65536)}
}

func (mr *MessageReader) ReadMessage() (*Message, error) {
	_, reader, err := mr.conn.NextReader()
	if err != nil {
		return nil, err
	}

	var header [3]byte
	if _, err := io.ReadFull(reader, header[:]); err != nil {
		return nil, err
	}

	msgType := header[0]
	length := binary.BigEndian.Uint16(header[1:])

	if int(length) > len(mr.buf) {
		mr.buf = make([]byte, length)
	}

	payload := mr.buf[:length]
	if _, err := io.ReadFull(reader, payload); err != nil {
		return nil, err
	}

	return &Message{Type: msgType, Payload: payload}, nil
}

type MessageWriter struct {
	conn *websocket.Conn
	mu   sync.Mutex
}

func NewMessageWriter(conn *websocket.Conn) *MessageWriter {
	return &MessageWriter{conn: conn}
}

func (mw *MessageWriter) WriteMessage(msg *Message) error {
	mw.mu.Lock()
	defer mw.mu.Unlock()

	if len(msg.Payload) > 65535 {
		return ErrBufferTooSmall
	}

	writer, err := mw.conn.NextWriter(websocket.BinaryMessage)
	if err != nil {
		return err
	}

	header := [3]byte{
		msg.Type,
		byte(len(msg.Payload) >> 8),
		byte(len(msg.Payload) & 0xFF),
	}

	if _, err := writer.Write(header[:]); err != nil {
		return err
	}

	if _, err := writer.Write(msg.Payload); err != nil {
		return err
	}

	return writer.Close()
}