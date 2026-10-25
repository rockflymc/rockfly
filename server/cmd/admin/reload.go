package admin

import (
	"sync"
	"time"

	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/world"
)

// PluginReloader defines the interface for hot-reloading server plugins.
type PluginReloader interface {
	ReloadAll() (int, time.Duration, error)
}

var (
	globalReloaderMu sync.RWMutex
	globalReloader   PluginReloader
)

// SetPluginReloader configures the reloader implementation for the /reload command.
func SetPluginReloader(r PluginReloader) {
	globalReloaderMu.Lock()
	defer globalReloaderMu.Unlock()
	globalReloader = r
}

func getPluginReloader() PluginReloader {
	globalReloaderMu.RLock()
	defer globalReloaderMu.RUnlock()
	return globalReloader
}

func registerReload() {
	cmd.Register(cmd.New("reload", "Reloads server plugins and configurations", []string{"rl"},
		Reload{},
		ReloadPlugins{},
	))
}

// Reload reloads all plugins on the server.
type Reload struct{}

// Run executes the /reload command.
func (c Reload) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	if !isAllowed(src) {
		o.Errorf("You do not have permission to execute this command.")
		return
	}
	reloader := getPluginReloader()
	if reloader == nil {
		o.Errorf("No plugin manager registered on this server.")
		return
	}

	count, dur, err := reloader.ReloadAll()
	if err != nil {
		o.Errorf("Failed to reload plugins: %v", err)
		return
	}

	o.Printf("Successfully reloaded %d plugin(s) in %v.", count, dur)
}

// ReloadPlugins reloads all server plugins via /reload plugins.
type ReloadPlugins struct {
	Plugins cmd.SubCommand `cmd:"plugins"`
}

// Run executes /reload plugins.
func (c ReloadPlugins) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	Reload{}.Run(src, o, tx)
}
