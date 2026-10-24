package admin

import (
	"slices"
	"strings"

	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
)

func registerWhitelist() {
	cmd.Register(cmd.New("whitelist", "Manages the server whitelist", nil,
		WhitelistOn{},
		WhitelistOff{},
		WhitelistList{},
		WhitelistAdd{},
		WhitelistRemove{},
		WhitelistReload{},
	))
}

// WhitelistOn turns whitelist enforcement on.
type WhitelistOn struct {
	On cmd.SubCommand `cmd:"on"`
}

func (c WhitelistOn) Allow(src cmd.Source) bool { return isAllowed(src) }
func (c WhitelistOn) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	player.SetWhitelistEnabled(true)
	o.Printf("Whitelist is now turned on")
}

// WhitelistOff turns whitelist enforcement off.
type WhitelistOff struct {
	Off cmd.SubCommand `cmd:"off"`
}

func (c WhitelistOff) Allow(src cmd.Source) bool { return isAllowed(src) }
func (c WhitelistOff) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	player.SetWhitelistEnabled(false)
	o.Printf("Whitelist is now turned off")
}

// WhitelistList displays all whitelisted players.
type WhitelistList struct {
	List cmd.SubCommand `cmd:"list"`
}

func (c WhitelistList) Allow(src cmd.Source) bool { return isAllowed(src) }
func (c WhitelistList) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	entries := player.DefaultWhitelistManager().All()
	if len(entries) == 0 {
		o.Printf("There are no whitelisted players")
		return
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.Name)
	}
	slices.Sort(names)
	o.Printf("There are %d whitelisted player(s): %s", len(names), strings.Join(names, ", "))
}

// WhitelistAdd adds a player to the whitelist.
type WhitelistAdd struct {
	Add    cmd.SubCommand `cmd:"add"`
	Player string         `cmd:"player"`
}

func (c WhitelistAdd) Allow(src cmd.Source) bool { return isAllowed(src) }
func (c WhitelistAdd) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	target := strings.TrimSpace(c.Player)
	if err := player.DefaultWhitelistManager().Add(target, ""); err != nil {
		o.Errorf("Could not whitelist player: %v", err)
		return
	}
	o.Printf("Added %s to the whitelist", target)
}

// WhitelistRemove removes a player from the whitelist.
type WhitelistRemove struct {
	Remove cmd.SubCommand `cmd:"remove"`
	Player string         `cmd:"player"`
}

func (c WhitelistRemove) Allow(src cmd.Source) bool { return isAllowed(src) }
func (c WhitelistRemove) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	target := strings.TrimSpace(c.Player)
	if err := player.DefaultWhitelistManager().Remove(target); err != nil {
		o.Errorf("Could not remove player from whitelist: %v", err)
		return
	}
	o.Printf("Removed %s from the whitelist", target)
}

// WhitelistReload reloads whitelist.json from disk.
type WhitelistReload struct {
	Reload cmd.SubCommand `cmd:"reload"`
}

func (c WhitelistReload) Allow(src cmd.Source) bool { return isAllowed(src) }
func (c WhitelistReload) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	if err := player.DefaultWhitelistManager().Load(); err != nil {
		o.Errorf("Failed to reload whitelist.json: %v", err)
		return
	}
	o.Printf("Reloaded the whitelist from disk")
}
