package admin

import (
	"runtime"

	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

// RockflyVersion represents the Rockfly server software release version.
const RockflyVersion = "1.0.0"

func registerVersion() {
	cmd.Register(cmd.New("version", "Displays the server and protocol versions", []string{"ver", "about"},
		Version{},
	))
}

// Version prints detailed build and protocol version information.
type Version struct{}

// Run executes the /version command.
func (c Version) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	o.Printf("This server is running Rockfly version %s (Minecraft v%s, Protocol %d, Go %s on %s/%s)",
		RockflyVersion,
		protocol.CurrentVersion,
		protocol.CurrentProtocol,
		runtime.Version(),
		runtime.GOOS,
		runtime.GOARCH,
	)
}
