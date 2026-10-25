package player

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// BanEntry represents a player ban record in banned-players.json.
type BanEntry struct {
	UUID    string `json:"uuid,omitempty"`
	Name    string `json:"name"`
	Created string `json:"created"`
	Source  string `json:"source"`
	Expires string `json:"expires"`
	Reason  string `json:"reason"`
}

// IPBanEntry represents an IP ban record in banned-ips.json.
type IPBanEntry struct {
	IP      string `json:"ip"`
	Created string `json:"created"`
	Source  string `json:"source"`
	Expires string `json:"expires"`
	Reason  string `json:"reason"`
}

// BanManager manages player and IP bans and persists them to JSON files.
type BanManager struct {
	mu             sync.RWMutex
	playerFilePath string
	ipFilePath     string
	players        map[string]BanEntry   // keyed by lowercase name
	byUUID         map[string]BanEntry   // keyed by lowercase UUID
	ips            map[string]IPBanEntry // keyed by IP string
}

var (
	defaultBanManager     *BanManager
	defaultBanManagerOnce sync.Once
)

// DefaultBanManager returns the global BanManager singleton.
func DefaultBanManager() *BanManager {
	defaultBanManagerOnce.Do(func() {
		defaultBanManager = NewBanManager("banned-players.json", "banned-ips.json")
		_ = defaultBanManager.Load()
	})
	return defaultBanManager
}

// NewBanManager creates a new BanManager for the specified file paths.
func NewBanManager(playerFilePath, ipFilePath string) *BanManager {
	return &BanManager{
		playerFilePath: playerFilePath,
		ipFilePath:     ipFilePath,
		players:        make(map[string]BanEntry),
		byUUID:         make(map[string]BanEntry),
		ips:            make(map[string]IPBanEntry),
	}
}

// Load loads both player and IP bans from their respective JSON files.
func (m *BanManager) Load() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.players = make(map[string]BanEntry)
	m.byUUID = make(map[string]BanEntry)
	m.ips = make(map[string]IPBanEntry)

	if pData, err := os.ReadFile(m.playerFilePath); err == nil {
		var pEntries []BanEntry
		if err := json.Unmarshal(pData, &pEntries); err == nil {
			for _, entry := range pEntries {
				m.players[strings.ToLower(entry.Name)] = entry
				if entry.UUID != "" {
					m.byUUID[strings.ToLower(entry.UUID)] = entry
				}
			}
		}
	}

	if ipData, err := os.ReadFile(m.ipFilePath); err == nil {
		var ipEntries []IPBanEntry
		if err := json.Unmarshal(ipData, &ipEntries); err == nil {
			for _, entry := range ipEntries {
				m.ips[entry.IP] = entry
			}
		}
	}
	return nil
}

// Save writes both player and IP bans to disk atomically.
func (m *BanManager) Save() error {
	m.mu.RLock()
	pEntries := make([]BanEntry, 0, len(m.players))
	for _, entry := range m.players {
		pEntries = append(pEntries, entry)
	}

	ipEntries := make([]IPBanEntry, 0, len(m.ips))
	for _, entry := range m.ips {
		ipEntries = append(ipEntries, entry)
	}
	m.mu.RUnlock()

	pData, err := json.MarshalIndent(pEntries, "", "  ")
	if err != nil {
		return err
	}
	dirP := filepath.Dir(m.playerFilePath)
	if dirP != "" && dirP != "." {
		_ = os.MkdirAll(dirP, 0755)
	}
	tmpPlayerFile := m.playerFilePath + ".tmp"
	if err := os.WriteFile(tmpPlayerFile, pData, 0644); err != nil {
		return err
	}
	if err := os.Rename(tmpPlayerFile, m.playerFilePath); err != nil {
		return err
	}

	ipData, err := json.MarshalIndent(ipEntries, "", "  ")
	if err != nil {
		return err
	}
	dirIP := filepath.Dir(m.ipFilePath)
	if dirIP != "" && dirIP != "." {
		_ = os.MkdirAll(dirIP, 0755)
	}
	tmpIPFile := m.ipFilePath + ".tmp"
	if err := os.WriteFile(tmpIPFile, ipData, 0644); err != nil {
		return err
	}
	return os.Rename(tmpIPFile, m.ipFilePath)
}

// BanPlayer adds a player to the banned players list.
func (m *BanManager) BanPlayer(name, uuidStr, reason, source string) error {
	if reason == "" {
		reason = "Banned by an operator."
	}
	if source == "" {
		source = "Console"
	}

	entry := BanEntry{
		UUID:    uuidStr,
		Name:    name,
		Created: time.Now().Format("2006-01-02 15:04:05 -0700"),
		Source:  source,
		Expires: "forever",
		Reason:  reason,
	}

	m.mu.Lock()
	m.players[strings.ToLower(name)] = entry
	if uuidStr != "" {
		m.byUUID[strings.ToLower(uuidStr)] = entry
	}
	m.mu.Unlock()

	return m.Save()
}

// PardonPlayer removes a player from the banned players list.
func (m *BanManager) PardonPlayer(nameOrUUID string) error {
	key := strings.ToLower(nameOrUUID)

	m.mu.Lock()
	entry, ok := m.players[key]
	if !ok {
		entry, ok = m.byUUID[key]
	}
	if ok {
		delete(m.players, strings.ToLower(entry.Name))
		if entry.UUID != "" {
			delete(m.byUUID, strings.ToLower(entry.UUID))
		}
	} else {
		m.mu.Unlock()
		return fmt.Errorf("player %s is not banned", nameOrUUID)
	}
	m.mu.Unlock()

	return m.Save()
}

// IsPlayerBanned reports whether a player name or UUID is banned, returning the reason.
func (m *BanManager) IsPlayerBanned(nameOrUUID string) (bool, string) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	key := strings.ToLower(nameOrUUID)
	if entry, ok := m.players[key]; ok {
		return true, entry.Reason
	}
	if entry, ok := m.byUUID[key]; ok {
		return true, entry.Reason
	}
	return false, ""
}

// BanIP adds an IP address to the banned IPs list.
func (m *BanManager) BanIP(ip, reason, source string) error {
	ip = strings.TrimSpace(ip)
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return fmt.Errorf("invalid IP address %q", ip)
	}
	canonicalIP := parsed.String()

	if reason == "" {
		reason = "Banned by an operator."
	}
	if source == "" {
		source = "Console"
	}

	entry := IPBanEntry{
		IP:      canonicalIP,
		Created: time.Now().Format("2006-01-02 15:04:05 -0700"),
		Source:  source,
		Expires: "forever",
		Reason:  reason,
	}

	m.mu.Lock()
	m.ips[canonicalIP] = entry
	m.mu.Unlock()

	return m.Save()
}

// PardonIP removes an IP from the banned IPs list.
func (m *BanManager) PardonIP(ip string) error {
	ip = strings.TrimSpace(ip)
	if parsed := net.ParseIP(ip); parsed != nil {
		ip = parsed.String()
	}

	m.mu.Lock()
	if _, ok := m.ips[ip]; !ok {
		m.mu.Unlock()
		return fmt.Errorf("ip %s is not banned", ip)
	}
	delete(m.ips, ip)
	m.mu.Unlock()

	return m.Save()
}

// IsIPBanned reports whether an IP address is banned, returning the reason.
func (m *BanManager) IsIPBanned(ip string) (bool, string) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	ip = strings.TrimSpace(ip)
	if parsed := net.ParseIP(ip); parsed != nil {
		ip = parsed.String()
	}
	if entry, ok := m.ips[ip]; ok {
		return true, entry.Reason
	}
	return false, ""
}

// AllPlayerBans returns a list of all active player ban entries.
func (m *BanManager) AllPlayerBans() []BanEntry {
	m.mu.RLock()
	defer m.mu.RUnlock()

	entries := make([]BanEntry, 0, len(m.players))
	for _, entry := range m.players {
		entries = append(entries, entry)
	}
	return entries
}

// AllIPBans returns a list of all active IP ban entries.
func (m *BanManager) AllIPBans() []IPBanEntry {
	m.mu.RLock()
	defer m.mu.RUnlock()

	entries := make([]IPBanEntry, 0, len(m.ips))
	for _, entry := range m.ips {
		entries = append(entries, entry)
	}
	return entries
}
