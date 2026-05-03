package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"go.uber.org/zap"
)

type Manager struct {
	mu         sync.RWMutex
	plugins    map[string]*Plugin
	runtime    wazero.Runtime
	logger     *zap.Logger
	pluginDir  string
}

type Plugin struct {
	Name        string
	Version     string
	Description string
	Type        string
	compiled    api.Module
}

type Result struct {
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data,omitempty"`
	Error   string          `json:"error,omitempty"`
}

func NewManager(logger *zap.Logger, pluginDir string) (*Manager, error) {
	ctx := context.Background()

	runtime := wazero.NewRuntime(ctx)

	builder := runtime.NewHostModuleBuilder("env")

	m := &Manager{
		plugins:   make(map[string]*Plugin),
		runtime:   runtime,
		logger:    logger,
		pluginDir: pluginDir,
	}

	builder.NewFunctionBuilder().WithFunc(m.printMessage).Export("print")

	if _, err := builder.Instantiate(ctx); err != nil {
		return nil, fmt.Errorf("failed to instantiate env module: %w", err)
	}

	return m, nil
}

func (m *Manager) printMessage(ctx context.Context, mod api.Module) {
	m.logger.Info("Plugin print called")
}

func (m *Manager) LoadPlugin(name string, wasmData []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.plugins[name]; exists {
		return fmt.Errorf("plugin %s already loaded", name)
	}

	module, err := m.runtime.Instantiate(context.Background(), wasmData)
	if err != nil {
		return fmt.Errorf("failed to instantiate plugin: %w", err)
	}

	plugin := &Plugin{
		Name:     name,
		compiled: module,
	}

	m.plugins[name] = plugin
	m.logger.Info("Plugin loaded", zap.String("name", name))

	return nil
}

func (m *Manager) LoadPluginFromFile(name, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read plugin file: %w", err)
	}

	return m.LoadPlugin(name, data)
}

func (m *Manager) LoadPluginFromDir(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".wasm" {
			continue
		}

		path := filepath.Join(dir, entry.Name())
		name := filepath.Base(entry.Name())
		name = name[:len(name)-5]

		if err := m.LoadPluginFromFile(name, path); err != nil {
			m.logger.Warn("Failed to load plugin",
				zap.String("name", name),
				zap.Error(err))
		}
	}

	return nil
}

func (m *Manager) InvokePlugin(name string, method string, args json.RawMessage) *Result {
	m.mu.RLock()
	plugin, exists := m.plugins[name]
	m.mu.RUnlock()

	if !exists {
		return &Result{Success: false, Error: fmt.Sprintf("plugin %s not found", name)}
	}

	invoke := plugin.compiled.ExportedFunction("invoke")
	if invoke == nil {
		return &Result{Success: false, Error: "plugin has no invoke function"}
	}

	argsData, err := json.Marshal(args)
	if err != nil {
		return &Result{Success: false, Error: fmt.Sprintf("failed to marshal args: %v", err)}
	}

	alloc := plugin.compiled.ExportedFunction("alloc")
	if alloc == nil {
		return &Result{Success: false, Error: "plugin missing alloc function"}
	}

	allocResult, err := alloc.Call(context.Background(), uint64(len(argsData)))
	if err != nil {
		return &Result{Success: false, Error: fmt.Sprintf("alloc failed: %v", err)}
	}

	ptr := uint32(allocResult[0])

	if !plugin.compiled.Memory().Write(uint32(ptr), argsData) {
		return &Result{Success: false, Error: "failed to write args to memory"}
	}

	invokeResult, err := invoke.Call(context.Background(), uint64(ptr), uint64(len(argsData)))
	if err != nil {
		return &Result{Success: false, Error: fmt.Sprintf("invoke failed: %v", err)}
	}

	resultPtr := uint32(invokeResult[0])
	resultSize := uint32(invokeResult[1])

	resultData, ok := plugin.compiled.Memory().Read(resultPtr, resultSize)
	if !ok {
		return &Result{Success: false, Error: "failed to read result from memory"}
	}

	var result Result
	if err := json.Unmarshal(resultData, &result); err != nil {
		return &Result{Success: false, Error: fmt.Sprintf("failed to parse result: %v", err)}
	}

	return &result
}

func (m *Manager) UnloadPlugin(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	plugin, exists := m.plugins[name]
	if !exists {
		return fmt.Errorf("plugin %s not loaded", name)
	}

	plugin.compiled.Close(context.Background())
	delete(m.plugins, name)

	m.logger.Info("Plugin unloaded", zap.String("name", name))
	return nil
}

func (m *Manager) ListPlugins() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	names := make([]string, 0, len(m.plugins))
	for name := range m.plugins {
		names = append(names, name)
	}
	return names
}

func (m *Manager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for name := range m.plugins {
		m.plugins[name].compiled.Close(context.Background())
	}

	return m.runtime.Close(context.Background())
}