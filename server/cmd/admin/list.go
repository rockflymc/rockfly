package admin

import (
	"fmt"
	"slices"
	"strings"

	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/world"
)

func registerList(srv Server) {
	cmd.Register(cmd.New("list", "Lists players currently connected to the server", nil,
		List{srv: srv},
		ListUUIDs{srv: srv},
	))
}

// List lists all players currently online on the server.
type List struct {
	srv Server `cmd:"-"`
}

// Run executes the /list command.
func (c List) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	count := 0
	maxCount := 0
	var names []string

	if c.srv != nil {
		count = c.srv.PlayerCount()
		maxCount = c.srv.MaxPlayerCount()
		for p := range c.srv.Players(tx) {
			names = append(names, p.Name())
		}
	} else if tx != nil {
		for e := range tx.Players() {
			if nt, ok := e.(cmd.NamedTarget); ok {
				names = append(names, nt.Name())
			}
		}
		count = len(names)
		maxCount = count
	}

	slices.Sort(names)
	playerList := strings.Join(names, ", ")
	if playerList == "" {
		o.Printf("There are %d of a max of %d players online", count, maxCount)
	} else {
		o.Printf("There are %d of a max of %d players online: %s", count, maxCount, playerList)
	}
}

// ListUUIDs lists all online players along with their UUIDs.
type ListUUIDs struct {
	srv   Server         `cmd:"-"`
	UUIDs cmd.SubCommand `cmd:"uuids"`
}

// Run executes the /list uuids command.
func (c ListUUIDs) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	count := 0
	maxCount := 0
	var entries []string

	if c.srv != nil {
		count = c.srv.PlayerCount()
		maxCount = c.srv.MaxPlayerCount()
		for p := range c.srv.Players(tx) {
			entries = append(entries, fmt.Sprintf("%s (%s)", p.Name(), p.UUID()))
		}
	} else if tx != nil {
		for e := range tx.Players() {
			if nt, ok := e.(cmd.NamedTarget); ok {
				entries = append(entries, nt.Name())
			}
		}
		count = len(entries)
		maxCount = count
	}

	slices.Sort(entries)
	entryList := strings.Join(entries, ", ")
	if entryList == "" {
		o.Printf("There are %d of a max of %d players online", count, maxCount)
	} else {
		o.Printf("There are %d of a max of %d players online: %s", count, maxCount, entryList)
	}
}
