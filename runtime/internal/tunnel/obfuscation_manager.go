package tunnel

import (
	"sync"

	"go.uber.org/zap"
)

type ObfuscationManager struct {
	mu         sync.RWMutex
	obfuscators map[string]Obfuscator
	logger     *zap.Logger
}

func NewObfuscationManager(logger *zap.Logger) *ObfuscationManager {
	return &ObfuscationManager{
		obfuscators: make(map[string]Obfuscator),
		logger:     logger,
	}
}

func (m *ObfuscationManager) Create(cfg *ObfuscationConfig) (Obfuscator, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	name := string(cfg.Mode)

	if _, exists := m.obfuscators[name]; exists {
		return m.obfuscators[name], nil
	}

	obf := CreateObfuscator(cfg)
	m.obfuscators[name] = obf

	m.logger.Info("Obfuscator created", zap.String("mode", string(cfg.Mode)))

	return obf, nil
}

func (m *ObfuscationManager) Get(mode ObfuscationMode) Obfuscator {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.obfuscators[string(mode)]
}

func (m *ObfuscationManager) List() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	modes := make([]string, 0, len(m.obfuscators))
	for mode := range m.obfuscators {
		modes = append(modes, mode)
	}
	return modes
}
