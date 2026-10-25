package gameplay

import (
	"strconv"
	"strings"

	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/world"
)

func registerTime() {
	cmd.Register(cmd.New("time", "Changes or queries the world's game time", nil,
		TimeSet{},
		TimeAdd{},
		TimeQuery{},
	))
}

// TimeSet sets the time of day.
type TimeSet struct {
	Set   cmd.SubCommand `cmd:"set"`
	Value string         `cmd:"time"`
}

func (c TimeSet) Allow(src cmd.Source) bool {
	return isAllowed(src)
}

func (c TimeSet) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	if tx == nil {
		o.Errorf("Cannot execute time command outside of an active world.")
		return
	}
	t := parseTimeSpec(c.Value)
	tx.World().SetTime(t)
	o.Printf("Set the time to %d", t)
}

// TimeAdd adds to the world's time.
type TimeAdd struct {
	Add    cmd.SubCommand `cmd:"add"`
	Amount int            `cmd:"amount"`
}

func (c TimeAdd) Allow(src cmd.Source) bool {
	return isAllowed(src)
}

func (c TimeAdd) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	if tx == nil {
		o.Errorf("Cannot execute time command outside of an active world.")
		return
	}
	newTime := tx.World().Time() + c.Amount
	tx.World().SetTime(newTime)
	o.Printf("Added %d to the time (new time: %d)", c.Amount, newTime)
}

// TimeQuery queries the world's time.
type TimeQuery struct {
	Query cmd.SubCommand `cmd:"query"`
	Type  string         `cmd:"type"`
}

func (c TimeQuery) Allow(src cmd.Source) bool {
	return isAllowed(src)
}

func (c TimeQuery) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	if tx == nil {
		o.Errorf("Cannot execute time command outside of an active world.")
		return
	}
	currentTime := tx.World().Time()
	switch strings.ToLower(c.Type) {
	case "daytime":
		dayTime := currentTime % 24000
		o.Printf("The time is %d", dayTime)
	case "gametime":
		o.Printf("The game time is %d", currentTime)
	case "day":
		days := currentTime / 24000
		o.Printf("The day count is %d", days)
	default:
		o.Errorf("Unknown query type '%s' (expected daytime, gametime, or day)", c.Type)
	}
}

func parseTimeSpec(spec string) int {
	switch strings.ToLower(spec) {
	case "day":
		return 1000
	case "noon":
		return 6000
	case "sunset":
		return 12000
	case "night":
		return 13000
	case "midnight":
		return 18000
	case "sunrise":
		return 23000
	default:
		if n, err := strconv.Atoi(spec); err == nil && n >= 0 {
			return n
		}
		return 1000
	}
}
