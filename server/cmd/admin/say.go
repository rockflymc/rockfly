package admin

import (
	"fmt"

	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/world"
)

func registerSay(srv Server) {
	cmd.Register(cmd.New("say", "Broadcasts a message to all players on the server", nil,
		Say{srv: srv},
	))
}

// Say broadcasts a message as [Sender] Message.
type Say struct {
	srv     Server      `cmd:"-"`
	Message cmd.Varargs `cmd:"message"`
}

// Allow reports whether the source is permitted to execute /say.
func (c Say) Allow(src cmd.Source) bool {
	return isAllowed(src)
}

// Run executes the /say command.
func (c Say) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	senderName := sourceName(src)
	formatted := fmt.Sprintf("[%s] %s", senderName, string(c.Message))

	if c.srv != nil {
		for p := range c.srv.Players(tx) {
			p.Message(formatted)
		}
	} else if tx != nil {
		for e := range tx.Players() {
			if m, ok := e.(interface{ Message(...any) }); ok {
				m.Message(formatted)
			}
		}
	}

	o.Printf("%s", formatted)
}
