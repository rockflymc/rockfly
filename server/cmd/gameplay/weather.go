package gameplay

import (
	"strings"
	"time"

	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/world"
)

func registerWeather() {
	cmd.Register(cmd.New("weather", "Sets the world weather", nil,
		Weather{},
	))
}

// Weather sets the weather condition in the world.
type Weather struct {
	Type     string            `cmd:"type"`
	Duration cmd.Optional[int] `cmd:"duration"`
}

func (c Weather) Allow(src cmd.Source) bool {
	return isAllowed(src)
}

func (c Weather) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	if tx == nil {
		o.Errorf("Cannot execute weather command outside of an active world.")
		return
	}

	dur := 5 * time.Minute
	if d, ok := c.Duration.Load(); ok && d > 0 {
		dur = time.Duration(d) * time.Second
	}

	w := tx.World()
	switch strings.ToLower(c.Type) {
	case "clear":
		w.StopRaining()
		o.Printf("Weather set to clear")
	case "rain":
		w.StartRaining(dur)
		o.Printf("Weather set to rain for %v", dur)
	case "thunder":
		w.StartThundering(dur)
		o.Printf("Weather set to thunder for %v", dur)
	default:
		o.Errorf("Unknown weather type '%s' (expected clear, rain, or thunder)", c.Type)
	}
}
