package builtin

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"sync"
	"time"
)

type TerminalPlugin struct {
	activeSessions map[string]*TerminalSession
	mu             sync.RWMutex
}

type TerminalSession struct {
	ID      string
	Cmd     *exec.Cmd
	Created time.Time
	Status  string
}

type TermStartRequest struct {
	Command string `json:"command,omitempty"`
	Shell   string `json:"shell,omitempty"`
}

func NewTerminalPlugin() *TerminalPlugin {
	return &TerminalPlugin{
		activeSessions: make(map[string]*TerminalSession),
	}
}

func (t *TerminalPlugin) Invoke(method string, args json.RawMessage) (json.RawMessage, error) {
	switch method {
	case "start":
		return t.handleStart(args)
	case "stop":
		return t.handleStop(args)
	case "status":
		return t.handleStatus(args)
	case "resize":
		return t.handleResize(args)
	default:
		return nil, fmt.Errorf("unknown method: %s", method)
	}
}

func (t *TerminalPlugin) handleStart(args json.RawMessage) (json.RawMessage, error) {
	var req TermStartRequest
	if err := json.Unmarshal(args, &req); err != nil {
		return nil, err
	}

	shell := req.Shell
	if shell == "" {
		shell = "/bin/sh"
	}

	cmd := exec.Command(shell)

	sessionID := fmt.Sprintf("term-%d", time.Now().UnixNano())

	session := &TerminalSession{
		ID:      sessionID,
		Cmd:     cmd,
		Created: time.Now(),
		Status:  "running",
	}

	t.mu.Lock()
	t.activeSessions[sessionID] = session
	t.mu.Unlock()

	return json.Marshal(map[string]interface{}{
		"session_id": sessionID,
		"shell":      shell,
		"status":     "started",
	})
}

func (t *TerminalPlugin) handleStop(args json.RawMessage) (json.RawMessage, error) {
	var req struct{ SessionID string }
	if err := json.Unmarshal(args, &req); err != nil {
		return nil, err
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	session, ok := t.activeSessions[req.SessionID]
	if !ok {
		return nil, fmt.Errorf("session not found")
	}

	if session.Cmd.Process != nil {
		session.Cmd.Process.Kill()
	}

	session.Status = "stopped"
	delete(t.activeSessions, req.SessionID)

	return json.Marshal(map[string]string{"status": "stopped"})
}

func (t *TerminalPlugin) handleStatus(args json.RawMessage) (json.RawMessage, error) {
	var req struct{ SessionID string }
	if err := json.Unmarshal(args, &req); err != nil {
		return nil, err
	}

	t.mu.RLock()
	session, ok := t.activeSessions[req.SessionID]
	t.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("session not found")
	}

	return json.Marshal(map[string]interface{}{
		"session_id": session.ID,
		"status":     session.Status,
		"created":    session.Created,
		"uptime":     time.Since(session.Created).Seconds(),
	})
}

func (t *TerminalPlugin) handleResize(args json.RawMessage) (json.RawMessage, error) {
	var req struct{ SessionID string; Rows, Cols int }
	if err := json.Unmarshal(args, &req); err != nil {
		return nil, err
	}

	return json.Marshal(map[string]bool{"success": true})
}