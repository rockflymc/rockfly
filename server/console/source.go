package console

import (
	"log/slog"

	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/go-gl/mathgl/mgl64"
)

// Source implements cmd.Source and cmd.NamedTarget for commands executed directly
// from the server console.
type Source struct {
	log *slog.Logger
}

// NewSource creates a new console command Source.
func NewSource(log *slog.Logger) Source {
	if log == nil {
		log = slog.Default()
	}
	return Source{log: log}
}

// Name returns "CONSOLE".
func (s Source) Name() string {
	return "CONSOLE"
}

// Position returns the origin vector.
func (s Source) Position() mgl64.Vec3 {
	return mgl64.Vec3{}
}

// SendCommandOutput routes command output and errors to the console logger.
func (s Source) SendCommandOutput(o *cmd.Output) {
	for _, msg := range o.Messages() {
		s.log.Info(msg.String())
	}
	for _, err := range o.Errors() {
		s.log.Error(err.Error())
	}
}
