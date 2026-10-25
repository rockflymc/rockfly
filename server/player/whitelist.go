package player

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// WhitelistEntry represents a record in whitelist.json.
type WhitelistEntry struct {
	UUID string `json:"uuid,omitempty"`
	Name string `json:"name"`
}

// WhitelistManager manages the server whitelist and its persistence.
type WhitelistManager struct {
	mu       sync.RWMutex
	filePath string
	enabled  bool
	entries  map[string]WhitelistEntry // keyed by lowercase name
	byUUID   map[string]WhitelistEntry // keyed by lowercase UUID
}

var (
	defaultWhitelistManager     *WhitelistManager
	defaultWhitelistManagerOnce sync.Once
)

// DefaultWhitelistManager returns the global WhitelistManager singleton.
func DefaultWhitelistManager() *WhitelistManager {
	defaultWhitelistManagerOnce.Do(func() {
		defaultWhitelistManager = NewWhitelistManager("whitelist.json")
		_ = defaultWhitelistManager.Load()
	})
	return defaultWhitelistManager
}

// NewWhitelistManager creates a new WhitelistManager with the given file path.
func NewWhitelistManager(filePath string) *WhitelistManager {
	return &WhitelistManager{
		filePath: filePath,
		entries:  make(map[string]WhitelistEntry),
		byUUID:   make(map[string]WhitelistEntry),
	}
}

// Load reads whitelist entries from whitelist.json.
func (m *WhitelistManager) Load() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	data, err := os.ReadFile(m.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			m.entries = make(map[string]WhitelistEntry)
			m.byUUID = make(map[string]WhitelistEntry)
			return nil
		}
		return err
	}

	var list []WhitelistEntry
	if err := json.Unmarshal(data, &list); err != nil {
		return err
	}

	m.entries = make(map[string]WhitelistEntry, len(list))
	m.byUUID = make(map[string]WhitelistEntry, len(list))
	for _, entry := range list {
		m.entries[strings.ToLower(entry.Name)] = entry
		if entry.UUID != "" {
			m.byUUID[strings.ToLower(entry.UUID)] = entry
		}
	}
	return nil
}

// Save writes whitelist entries to whitelist.json atomically with formatting.
func (m *WhitelistManager) Save() error {
	m.mu.RLock()
	list := make([]WhitelistEntry, 0, len(m.entries))
	for _, entry := range m.entries {
		list = append(list, entry)
	}
	m.mu.RUnlock()

	data, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return err
	}

	dir := filepath.Dir(m.filePath)
	if dir != "" && dir != "." {
		_ = os.MkdirAll(dir, 0755)
	}

	tmpFile := m.filePath + ".tmp"
	if err := os.WriteFile(tmpFile, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmpFile, m.filePath)
}

// Enabled reports whether the whitelist is actively enforced.
func (m *WhitelistManager) Enabled() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.enabled
}

// SetEnabled enables or disables whitelist enforcement.
func (m *WhitelistManager) SetEnabled(enabled bool) {
	m.mu.Lock()
	m.enabled = enabled
	m.mu.Unlock()
}

// Add adds a player to the whitelist by name and optional UUID.
func (m *WhitelistManager) Add(name, uuidStr string) error {
	name = strings.TrimSpace(name)
	uuidStr = strings.TrimSpace(uuidStr)
	if name == "" {
		return fmt.Errorf("player name cannot be empty")
	}

	entry := WhitelistEntry{
		Name: name,
		UUID: uuidStr,
	}

	m.mu.Lock()
	m.entries[strings.ToLower(name)] = entry
	if uuidStr != "" {
		m.byUUID[strings.ToLower(uuidStr)] = entry
	}
	m.mu.Unlock()

	return m.Save()
}

// Remove removes a player from the whitelist by name or UUID.
func (m *WhitelistManager) Remove(nameOrUUID string) error {
	key := strings.ToLower(strings.TrimSpace(nameOrUUID))

	m.mu.Lock()
	entry, ok := m.entries[key]
	if !ok {
		entry, ok = m.byUUID[key]
	}
	if ok {
		delete(m.entries, strings.ToLower(entry.Name))
		if entry.UUID != "" {
			delete(m.byUUID, strings.ToLower(entry.UUID))
		}
	} else {
		m.mu.Unlock()
		return fmt.Errorf("player %s is not whitelisted", nameOrUUID)
	}
	m.mu.Unlock()

	return m.Save()
}

// IsWhitelisted reports whether a player name or UUID is present in the whitelist.
func (m *WhitelistManager) IsWhitelisted(name, uuidStr string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	name = strings.ToLower(strings.TrimSpace(name))
	uuidStr = strings.ToLower(strings.TrimSpace(uuidStr))

	if name != "" {
		if _, ok := m.entries[name]; ok {
			return true
		}
	}
	if uuidStr != "" {
		if _, ok := m.byUUID[uuidStr]; ok {
			return true
		}
	}
	return false
}

// All returns a slice of all whitelisted players.
func (m *WhitelistManager) All() []WhitelistEntry {
	m.mu.RLock()
	defer m.mu.RUnlock()

	list := make([]WhitelistEntry, 0, len(m.entries))
	for _, entry := range m.entries {
		list = append(list, entry)
	}
	return list
}

// IsWhitelisted checks if a player is whitelisted using the default manager.
func IsWhitelisted(name, uuid string) bool {
	return DefaultWhitelistManager().IsWhitelisted(name, uuid)
}

// WhitelistEnabled reports whether the default whitelist is enabled.
func WhitelistEnabled() bool {
	return DefaultWhitelistManager().Enabled()
}

// SetWhitelistEnabled sets whether the default whitelist is enabled.
func SetWhitelistEnabled(enabled bool) {
	DefaultWhitelistManager().SetEnabled(enabled)
}
