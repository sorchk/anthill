package builtin

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"time"
)

type ShellPlugin struct {
	idleTimeout time.Duration
	maxSessions int
}

func NewShellPlugin() *ShellPlugin {
	return &ShellPlugin{
		idleTimeout: 300 * time.Second,
		maxSessions: 10,
	}
}

type ExecRequest struct {
	Command string `json:"command"`
	Timeout int    `json:"timeout,omitempty"`
}

type ExecResult struct {
	Output   string `json:"output"`
	ExitCode int    `json:"exit_code"`
	Duration int64  `json:"duration_ms"`
	Error    string `json:"error,omitempty"`
}

func (p *ShellPlugin) Invoke(method string, args json.RawMessage) (json.RawMessage, error) {
	switch method {
	case "exec":
		return p.handleExec(args)
	case "start":
		return json.Marshal(map[string]string{"status": "started"})
	case "stop":
		return json.Marshal(map[string]string{"status": "stopped"})
	case "status":
		return json.Marshal(map[string]string{"status": "running"})
	default:
		return nil, fmt.Errorf("unknown method: %s", method)
	}
}

func (p *ShellPlugin) handleExec(args json.RawMessage) (json.RawMessage, error) {
	var req ExecRequest
	if err := json.Unmarshal(args, &req); err != nil {
		return nil, err
	}

	timeout := 30 * time.Second
	if req.Timeout > 0 {
		timeout = time.Duration(req.Timeout) * time.Second
	}

	start := time.Now()

	cmd := exec.Command("sh", "-c", req.Command)
	output, err := cmd.CombinedOutput()
	duration := time.Since(start).Milliseconds()

	result := ExecResult{
		Output:   string(output),
		ExitCode: cmd.ProcessState.ExitCode(),
		Duration: duration,
	}

	if err != nil {
		result.Error = err.Error()
	}

	return json.Marshal(result)
}