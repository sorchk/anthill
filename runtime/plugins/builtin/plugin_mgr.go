package builtin

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type PluginMgr struct {
	pluginDir string
}

func NewPluginMgr(pluginDir string) *PluginMgr {
	return &PluginMgr{pluginDir: pluginDir}
}

type PluginInfo struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Type        string `json:"type"`
	Description string `json:"description"`
	Status      string `json:"status"`
}

type InstallRequest struct {
	Path string `json:"path"`
	URL  string `json:"url,omitempty"`
}

func (p *PluginMgr) HandleInstall(req *InstallRequest) (string, error) {
	if req.Path == "" && req.URL == "" {
		return "", fmt.Errorf("path or url required")
	}

	if req.Path != "" {
		destPath := filepath.Join(p.pluginDir, filepath.Base(req.Path))

		src, err := os.Open(req.Path)
		if err != nil {
			return "", fmt.Errorf("failed to open source: %w", err)
		}
		defer src.Close()

		dst, err := os.Create(destPath)
		if err != nil {
			return "", fmt.Errorf("failed to create dest: %w", err)
		}
		defer dst.Close()

		if _, err := src.WriteTo(dst); err != nil {
			return "", fmt.Errorf("failed to copy: %w", err)
		}

		return fmt.Sprintf("Installed to %s", destPath), nil
	}

	return "URL install not yet implemented", nil
}

func (p *PluginMgr) HandleList() ([]PluginInfo, error) {
	entries, err := os.ReadDir(p.pluginDir)
	if err != nil {
		return nil, err
	}

	var plugins []PluginInfo
	for _, entry := range entries {
		if entry.IsDir() {
			plugins = append(plugins, PluginInfo{
				Name:   entry.Name(),
				Status: "available",
			})
		}
	}

	return plugins, nil
}

func (p *PluginMgr) HandleUninstall(name string) (string, error) {
	path := filepath.Join(p.pluginDir, name)
	if err := os.RemoveAll(path); err != nil {
		return "", err
	}
	return fmt.Sprintf("Uninstalled %s", name), nil
}

func (p *PluginMgr) Invoke(method string, args json.RawMessage) (json.RawMessage, error) {
	switch method {
	case "install":
		var req InstallRequest
		if err := json.Unmarshal(args, &req); err != nil {
			return nil, err
		}
		msg, err := p.HandleInstall(&req)
		return json.Marshal(map[string]string{"message": msg, "error": "", "success": err == nil})

	case "list":
		plugins, err := p.HandleList()
		if err != nil {
			return nil, err
		}
		return json.Marshal(plugins)

	case "uninstall":
		var req struct{ Name string }
		if err := json.Unmarshal(args, &req); err != nil {
			return nil, err
		}
		msg, err := p.HandleUninstall(req.Name)
		return json.Marshal(map[string]string{"message": msg, "error": "", "success": err == nil})

	case "info":
		var req struct{ Name string }
		if err := json.Unmarshal(args, &req); err != nil {
			return nil, err
		}
		return json.Marshal(PluginInfo{Name: req.Name, Status: "active"})

	default:
		return nil, fmt.Errorf("unknown method: %s", method)
	}
}