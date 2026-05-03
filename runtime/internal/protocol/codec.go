package protocol

import (
	"encoding/json"
)

type HandshakeRequest struct {
	NodeID    string `json:"node_id"`
	Version   string `json:"version"`
	NodeGroup string `json:"node_group"`
	Hostname  string `json:"hostname"`
}

type HandshakeResponse struct {
	SessionID string `json:"session_id"`
	Status    string `json:"status"`
}

type AuthRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Token    string `json:"token,omitempty"`
}

type AuthResponse struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

type DataMessage struct {
	SessionID string `json:"session_id"`
	Channel   string `json:"channel"`
	Data      []byte `json:"data"`
}

type ControlMessage struct {
	Command string          `json:"command"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type PluginInvoke struct {
	PluginName string          `json:"plugin_name"`
	Method     string          `json:"method"`
	Args       json.RawMessage `json:"args,omitempty"`
}

type PluginResult struct {
	PluginName string          `json:"plugin_name"`
	Success    bool            `json:"success"`
	Result     json.RawMessage `json:"result,omitempty"`
	Error      string          `json:"error,omitempty"`
}