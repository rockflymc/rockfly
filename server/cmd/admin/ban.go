package admin

import (
	"net"
	"strings"

	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
)

func registerBan(srv Server) {
	cmd.Register(cmd.New("ban", "Prevents a player from connecting to the server", nil,
		Ban{srv: srv},
		BanOffline{srv: srv},
	))
}

func registerBanIP(srv Server) {
	cmd.Register(cmd.New("ban-ip", "Prevents an IP address from connecting to the server", nil,
		BanIP{srv: srv},
		BanIPAddress{srv: srv},
	))
}

// Ban bans target online players and disconnects them.
type Ban struct {
	srv    Server                    `cmd:"-"`
	Target []cmd.Target              `cmd:"player"`
	Reason cmd.Optional[cmd.Varargs] `cmd:"reason"`
}

// Allow reports whether the source is permitted to execute /ban.
func (c Ban) Allow(src cmd.Source) bool {
	return isAllowed(src)
}

// Run executes the /ban command for matched online players.
func (c Ban) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	reason := "Banned by an operator."
	if r, ok := c.Reason.Load(); ok && string(r) != "" {
		reason = string(r)
	}

	for _, target := range c.Target {
		if p, ok := target.(*player.Player); ok {
			name := p.Name()
			u := p.UUID().String()
			_ = player.DefaultBanManager().BanPlayer(name, u, reason, sourceName(src))
			p.Disconnect(reason)
			o.Printf("Banned %s: %s", name, reason)
		} else if nt, ok := target.(cmd.NamedTarget); ok {
			_ = player.DefaultBanManager().BanPlayer(nt.Name(), "", reason, sourceName(src))
			o.Printf("Banned %s: %s", nt.Name(), reason)
		}
	}
}

// BanOffline bans a player specified by name.
type BanOffline struct {
	srv    Server                    `cmd:"-"`
	Player string                    `cmd:"player"`
	Reason cmd.Optional[cmd.Varargs] `cmd:"reason"`
}

// Allow reports whether the source is permitted to execute /ban.
func (c BanOffline) Allow(src cmd.Source) bool {
	return isAllowed(src)
}

// Run executes the /ban command for a player specified by name.
func (c BanOffline) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	reason := "Banned by an operator."
	if r, ok := c.Reason.Load(); ok && string(r) != "" {
		reason = string(r)
	}

	if c.srv != nil {
		if handle, ok := c.srv.PlayerByName(c.Player); ok {
			if e, ok := handle.Entity(tx); ok {
				if p, ok := e.(*player.Player); ok {
					name := p.Name()
					u := p.UUID().String()
					_ = player.DefaultBanManager().BanPlayer(name, u, reason, sourceName(src))
					p.Disconnect(reason)
					o.Printf("Banned %s: %s", name, reason)
					return
				}
			}
		}
	}

	_ = player.DefaultBanManager().BanPlayer(c.Player, "", reason, sourceName(src))
	o.Printf("Banned %s: %s", c.Player, reason)
}

// BanIP bans the IP address of target players.
type BanIP struct {
	srv    Server                    `cmd:"-"`
	Target []cmd.Target              `cmd:"target"`
	Reason cmd.Optional[cmd.Varargs] `cmd:"reason"`
}

// Allow reports whether the source is permitted to execute /ban-ip.
func (c BanIP) Allow(src cmd.Source) bool {
	return isAllowed(src)
}

// Run executes the /ban-ip command for target players.
func (c BanIP) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	reason := "Banned by an operator."
	if r, ok := c.Reason.Load(); ok && string(r) != "" {
		reason = string(r)
	}

	for _, target := range c.Target {
		if p, ok := target.(*player.Player); ok {
			host, _, err := net.SplitHostPort(p.Addr().String())
			if err != nil {
				host = p.Addr().String()
			}
			_ = player.DefaultBanManager().BanIP(host, reason, sourceName(src))
			if parsed := net.ParseIP(host); parsed != nil {
				host = parsed.String()
			}
			if c.srv != nil {
				for pl := range c.srv.Players(tx) {
					h, _, err := net.SplitHostPort(pl.Addr().String())
					if err != nil {
						h = pl.Addr().String()
					}
					if parsed := net.ParseIP(h); parsed != nil {
						h = parsed.String()
					}
					if h == host {
						pl.Disconnect("IP-banned: " + reason)
					}
				}
			} else {
				p.Disconnect("IP-banned: " + reason)
			}
			o.Printf("Banned IP %s: %s", host, reason)
		}
	}
}

// BanIPAddress bans an IP address directly.
type BanIPAddress struct {
	srv     Server                    `cmd:"-"`
	Address string                    `cmd:"address"`
	Reason  cmd.Optional[cmd.Varargs] `cmd:"reason"`
}

// Allow reports whether the source is permitted to execute /ban-ip.
func (c BanIPAddress) Allow(src cmd.Source) bool {
	return isAllowed(src)
}

// Run executes the /ban-ip command for a raw IP address.
func (c BanIPAddress) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	reason := "Banned by an operator."
	if r, ok := c.Reason.Load(); ok && string(r) != "" {
		reason = string(r)
	}

	ip := strings.TrimSpace(c.Address)
	if err := player.DefaultBanManager().BanIP(ip, reason, sourceName(src)); err != nil {
		o.Errorf("Could not ban IP: %v", err)
		return
	}

	if parsed := net.ParseIP(ip); parsed != nil {
		ip = parsed.String()
	}

	if c.srv != nil {
		for pl := range c.srv.Players(tx) {
			h, _, err := net.SplitHostPort(pl.Addr().String())
			if err != nil {
				h = pl.Addr().String()
			}
			if parsed := net.ParseIP(h); parsed != nil {
				h = parsed.String()
			}
			if h == ip {
				pl.Disconnect("IP-banned: " + reason)
			}
		}
	}
	o.Printf("Banned IP %s: %s", ip, reason)
}
