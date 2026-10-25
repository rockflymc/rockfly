package admin

import (
	"time"

	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/world"
)

// PluginReloader defines the interface for hot-reloading server plugins.
type PluginReloader interface {
	ReloadAll() (int, time.Duration, error)
}

var globalReloader PluginReloader

// SetPluginReloader configures the reloader implementation for the /reload command.
func SetPluginReloader(r PluginReloader) {
	globalReloader = r
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
	if globalReloader == nil {
		o.Errorf("No plugin manager registered on this server.")
		return
	}

	count, dur, err := globalReloader.ReloadAll()
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
