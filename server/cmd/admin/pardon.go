package admin

import (
	"strings"

	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
)

func registerPardon() {
	cmd.Register(cmd.New("pardon", "Allows a player to connect to the server again", []string{"unban"},
		Pardon{},
	))
}

func registerPardonIP() {
	cmd.Register(cmd.New("pardon-ip", "Allows an IP address to connect to the server again", []string{"unban-ip"},
		PardonIP{},
	))
}

// Pardon unbans a player by name or UUID.
type Pardon struct {
	Player string `cmd:"player"`
}

// Allow reports whether the source is permitted to execute /pardon.
func (c Pardon) Allow(src cmd.Source) bool {
	return isAllowed(src)
}

// Run executes the /pardon command.
func (c Pardon) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	if err := player.DefaultBanManager().PardonPlayer(c.Player); err != nil {
		o.Errorf("Could not unban player: %v", err)
		return
	}
	o.Printf("Unbanned player %s", c.Player)
}

// PardonIP unbans an IP address.
type PardonIP struct {
	Address string `cmd:"address"`
}

// Allow reports whether the source is permitted to execute /pardon-ip.
func (c PardonIP) Allow(src cmd.Source) bool {
	return isAllowed(src)
}

// Run executes the /pardon-ip command.
func (c PardonIP) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	ip := strings.TrimSpace(c.Address)
	if err := player.DefaultBanManager().PardonIP(ip); err != nil {
		o.Errorf("Could not unban IP: %v", err)
		return
	}
	o.Printf("Unbanned IP %s", ip)
}
