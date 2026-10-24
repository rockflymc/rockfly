package player

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// OperatorEntry represents an operator record in ops.json.
type OperatorEntry struct {
	UUID                string `json:"uuid"`
	Name                string `json:"name"`
	Level               int    `json:"level"`
	BypassesPlayerLimit bool   `json:"bypassesPlayerLimit"`
}

// OperatorManager manages server operators and persists them to ops.json.
type OperatorManager struct {
	mu       sync.RWMutex
	filePath string
	ops      map[string]OperatorEntry // keyed by lowercase name
	byUUID   map[string]OperatorEntry // keyed by UUID string
}

var (
	defaultOpManager     *OperatorManager
	defaultOpManagerOnce sync.Once
)

// DefaultOperatorManager returns the global operator manager singleton.
func DefaultOperatorManager() *OperatorManager {
	defaultOpManagerOnce.Do(func() {
		defaultOpManager = NewOperatorManager("ops.json")
		_ = defaultOpManager.Load()
	})
	return defaultOpManager
}

// NewOperatorManager creates a new OperatorManager for the specified file path.
func NewOperatorManager(filePath string) *OperatorManager {
	return &OperatorManager{
		filePath: filePath,
		ops:      make(map[string]OperatorEntry),
		byUUID:   make(map[string]OperatorEntry),
	}
}

// Load reads operators from ops.json. If the file does not exist, an empty list is initialized.
func (m *OperatorManager) Load() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	data, err := os.ReadFile(m.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			m.ops = make(map[string]OperatorEntry)
			m.byUUID = make(map[string]OperatorEntry)
			return nil
		}
		return err
	}

	var entries []OperatorEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return err
	}

	m.ops = make(map[string]OperatorEntry, len(entries))
	m.byUUID = make(map[string]OperatorEntry, len(entries))
	for _, entry := range entries {
		m.ops[strings.ToLower(entry.Name)] = entry
		if entry.UUID != "" {
			m.byUUID[strings.ToLower(entry.UUID)] = entry
		}
	}
	return nil
}

// Save writes operators to ops.json with pretty indentation.
func (m *OperatorManager) Save() error {
	m.mu.RLock()
	entries := make([]OperatorEntry, 0, len(m.ops))
	for _, entry := range m.ops {
		entries = append(entries, entry)
	}
	m.mu.RUnlock()

	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}

	if dir := filepath.Dir(m.filePath); dir != "" && dir != "." {
		_ = os.MkdirAll(dir, 0755)
	}
	return os.WriteFile(m.filePath, data, 0644)
}

// SetOp adds or updates an operator entry.
func (m *OperatorManager) SetOp(name, uuidStr string, level int, bypassesLimit bool) error {
	if level < 1 {
		level = 1
	} else if level > 4 {
		level = 4
	}

	entry := OperatorEntry{
		UUID:                uuidStr,
		Name:                name,
		Level:               level,
		BypassesPlayerLimit: bypassesLimit,
	}

	m.mu.Lock()
	m.ops[strings.ToLower(name)] = entry
	if uuidStr != "" {
		m.byUUID[strings.ToLower(uuidStr)] = entry
	}
	m.mu.Unlock()

	return m.Save()
}

// Deop removes an operator by name or UUID.
func (m *OperatorManager) Deop(nameOrUUID string) error {
	key := strings.ToLower(nameOrUUID)

	m.mu.Lock()
	entry, ok := m.ops[key]
	if !ok {
		entry, ok = m.byUUID[key]
	}
	if ok {
		delete(m.ops, strings.ToLower(entry.Name))
		if entry.UUID != "" {
			delete(m.byUUID, strings.ToLower(entry.UUID))
		}
	}
	m.mu.Unlock()

	return m.Save()
}

// IsOp reports whether a player name or UUID is registered as an operator.
func (m *OperatorManager) IsOp(nameOrUUID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	key := strings.ToLower(nameOrUUID)
	if _, ok := m.ops[key]; ok {
		return true
	}
	_, ok := m.byUUID[key]
	return ok
}

// OpLevel returns the permission level for a player (1-4). Returns 0 if not an operator.
func (m *OperatorManager) OpLevel(nameOrUUID string) int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	key := strings.ToLower(nameOrUUID)
	if entry, ok := m.ops[key]; ok {
		return entry.Level
	}
	if entry, ok := m.byUUID[key]; ok {
		return entry.Level
	}
	return 0
}

// BypassesPlayerLimit reports whether the operator can join even when the server is full.
func (m *OperatorManager) BypassesPlayerLimit(nameOrUUID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	key := strings.ToLower(nameOrUUID)
	if entry, ok := m.ops[key]; ok {
		return entry.BypassesPlayerLimit
	}
	if entry, ok := m.byUUID[key]; ok {
		return entry.BypassesPlayerLimit
	}
	return false
}

// All returns a slice of all operator entries.
func (m *OperatorManager) All() []OperatorEntry {
	m.mu.RLock()
	defer m.mu.RUnlock()

	entries := make([]OperatorEntry, 0, len(m.ops))
	for _, entry := range m.ops {
		entries = append(entries, entry)
	}
	return entries
}

// IsOp checks if a player name or UUID is an operator using the default manager.
func IsOp(nameOrUUID string) bool {
	return DefaultOperatorManager().IsOp(nameOrUUID)
}

// OpLevel returns the operator level using the default manager.
func OpLevel(nameOrUUID string) int {
	return DefaultOperatorManager().OpLevel(nameOrUUID)
}

// SetOp adds an operator using the default manager.
func SetOp(name, uuid string, level int) error {
	return DefaultOperatorManager().SetOp(name, uuid, level, false)
}

// Deop removes an operator using the default manager.
func Deop(nameOrUUID string) error {
	return DefaultOperatorManager().Deop(nameOrUUID)
}

// AllOps returns all operators using the default manager.
func AllOps() []OperatorEntry {
	return DefaultOperatorManager().All()
}
