package ts

import (
	"fmt"
	"strings"

	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/event"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/dop251/goja"
	"github.com/go-gl/mathgl/mgl64"
)

// BindRockflyAPI injects the rockfly host API and console helpers into the Goja runtime.
func (p *TSPlugin) bindRockflyAPI() error {
	vm := p.vm

	// 1. console object
	consoleObj := vm.NewObject()
	_ = consoleObj.Set("log", func(call goja.FunctionCall) goja.Value {
		var parts []string
		for _, arg := range call.Arguments {
			parts = append(parts, fmt.Sprint(arg.Export()))
		}
		fmt.Printf("[%s] %s\n", p.manifest.Name, strings.Join(parts, " "))
		return goja.Undefined()
	})
	_ = consoleObj.Set("warn", func(call goja.FunctionCall) goja.Value {
		var parts []string
		for _, arg := range call.Arguments {
			parts = append(parts, fmt.Sprint(arg.Export()))
		}
		fmt.Printf("[%s] [WARN] %s\n", p.manifest.Name, strings.Join(parts, " "))
		return goja.Undefined()
	})
	_ = consoleObj.Set("error", func(call goja.FunctionCall) goja.Value {
		var parts []string
		for _, arg := range call.Arguments {
			parts = append(parts, fmt.Sprint(arg.Export()))
		}
		fmt.Printf("[%s] [ERROR] %s\n", p.manifest.Name, strings.Join(parts, " "))
		return goja.Undefined()
	})
	_ = vm.Set("console", consoleObj)

	// 2. rockfly root object
	rockflyObj := vm.NewObject()
	_ = rockflyObj.Set("log", func(call goja.FunctionCall) goja.Value {
		var parts []string
		for _, arg := range call.Arguments {
			parts = append(parts, fmt.Sprint(arg.Export()))
		}
		fmt.Printf("[%s] %s\n", p.manifest.Name, strings.Join(parts, " "))
		return goja.Undefined()
	})

	// 3. rockfly.Priority constants
	priorityObj := vm.NewObject()
	_ = priorityObj.Set("Lowest", int(event.PriorityLowest))
	_ = priorityObj.Set("Low", int(event.PriorityLow))
	_ = priorityObj.Set("Normal", int(event.PriorityNormal))
	_ = priorityObj.Set("High", int(event.PriorityHigh))
	_ = priorityObj.Set("Highest", int(event.PriorityHighest))
	_ = priorityObj.Set("Monitor", int(event.PriorityMonitor))
	_ = rockflyObj.Set("Priority", priorityObj)

	// 4. rockfly.events namespace
	eventsObj := vm.NewObject()
	_ = eventsObj.Set("on", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 3 {
			panic(vm.ToValue("rockfly.events.on requires (eventName, priority, callback)"))
		}
		eventName := call.Arguments[0].String()
		priority := event.Priority(call.Arguments[1].ToInteger())
		callback, ok := goja.AssertFunction(call.Arguments[2])
		if !ok {
			panic(vm.ToValue("callback argument must be a function"))
		}

		unsub := p.registerEventListener(eventName, priority, callback)
		p.listeners = append(p.listeners, unsub)

		// Return unsubscribe function to script
		return vm.ToValue(func(call goja.FunctionCall) goja.Value {
			unsub()
			return goja.Undefined()
		})
	})
	_ = rockflyObj.Set("events", eventsObj)

	// 5. rockfly.commands namespace
	commandsObj := vm.NewObject()
	_ = commandsObj.Set("register", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 3 {
			panic(vm.ToValue("rockfly.commands.register requires (name, description, callback)"))
		}
		cmdName := call.Arguments[0].String()
		desc := call.Arguments[1].String()
		callback, ok := goja.AssertFunction(call.Arguments[2])
		if !ok {
			panic(vm.ToValue("command callback must be a function"))
		}

		p.registerCommand(cmdName, desc, callback)
		return goja.Undefined()
	})
	_ = rockflyObj.Set("commands", commandsObj)

	return vm.Set("rockfly", rockflyObj)
}

// registerEventListener wires an event listener callback to the internal dispatcher.
func (p *TSPlugin) registerEventListener(eventName string, prio event.Priority, callback goja.Callable) func() {
	entry := &tsEventListener{
		eventName: strings.ToLower(eventName),
		priority:  prio,
		callback:  callback,
		plugin:    p,
	}

	p.eventHandlers = append(p.eventHandlers, entry)

	return func() {
		for i, h := range p.eventHandlers {
			if h == entry {
				p.eventHandlers = append(p.eventHandlers[:i], p.eventHandlers[i+1:]...)
				break
			}
		}
	}
}

// DispatchEvent dispatches an event object to registered TypeScript listeners.
func (p *TSPlugin) DispatchEvent(eventName string, eventObj any) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.enabled {
		return nil
	}

	name := strings.ToLower(eventName)
	handlers := make([]*tsEventListener, len(p.eventHandlers))
	copy(handlers, p.eventHandlers)

	for _, h := range handlers {
		if h.eventName == name {
			val := p.vm.ToValue(eventObj)
			_, err := p.callWithTimeoutLocked(h.callback, goja.Undefined(), val)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// registerCommand dynamically registers a command that dispatches to the TS callback.
func (p *TSPlugin) registerCommand(name, desc string, callback goja.Callable) {
	runnable := &dynamicTSCommand{
		plugin:   p,
		callback: callback,
	}
	cmd.Register(cmd.New(name, desc, nil, runnable))
	p.commands = append(p.commands, strings.ToLower(name))
}

type dynamicTSCommand struct {
	plugin   *TSPlugin
	callback goja.Callable
	Args     cmd.Optional[cmd.Varargs] `cmd:"args"`
}

func (c *dynamicTSCommand) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	c.plugin.mu.Lock()
	defer c.plugin.mu.Unlock()

	if !c.plugin.enabled {
		o.Errorf("Command is unavailable because plugin is disabled")
		return
	}

	var pObj any
	if p, ok := src.(*player.Player); ok {
		pObj = wrapPlayer(p, tx)
	}

	rawArgs := ""
	if a, ok := c.Args.Load(); ok {
		rawArgs = string(a)
	}
	args := strings.Fields(rawArgs)

	valPlayer := c.plugin.vm.ToValue(pObj)
	valArgs := c.plugin.vm.ToValue(args)

	_, err := c.plugin.callWithTimeoutLocked(c.callback, goja.Undefined(), valPlayer, valArgs)
	if err != nil {
		o.Errorf("Error executing script command: %v", err)
	}
}

func wrapPlayer(p *player.Player, tx *world.Tx) map[string]any {
	if p == nil {
		return nil
	}
	pos := p.Position()
	return map[string]any{
		"name": p.Name(),
		"uuid": p.UUID().String(),
		"xuid": p.XUID(),
		"position": map[string]any{
			"x": pos.X(),
			"y": pos.Y(),
			"z": pos.Z(),
		},
		"health":    p.Health(),
		"maxHealth": p.MaxHealth(),
		"gameMode":  gameModeString(p.GameMode()),
		"sendMessage": func(msg string) {
			p.Message(msg)
		},
		"teleport": func(val any) {
			switch v := val.(type) {
			case map[string]any:
				var x, y, z float64
				if fx, ok := toFloat(v["x"]); ok {
					x = fx
				}
				if fy, ok := toFloat(v["y"]); ok {
					y = fy
				}
				if fz, ok := toFloat(v["z"]); ok {
					z = fz
				}
				p.Teleport(mgl64.Vec3{x, y, z})
			}
		},
		"heal": func(amount float64) {
			p.Heal(amount, entityHealSource{})
		},
		"setGameMode": func(mode string) {
			switch strings.ToLower(mode) {
			case "survival":
				p.SetGameMode(world.GameModeSurvival)
			case "creative":
				p.SetGameMode(world.GameModeCreative)
			case "adventure":
				p.SetGameMode(world.GameModeAdventure)
			case "spectator":
				p.SetGameMode(world.GameModeSpectator)
			}
		},
	}
}

func toFloat(v any) (float64, bool) {
	switch val := v.(type) {
	case float64:
		return val, true
	case float32:
		return float64(val), true
	case int:
		return float64(val), true
	case int64:
		return float64(val), true
	default:
		return 0, false
	}
}

func gameModeString(gm world.GameMode) string {
	switch gm {
	case world.GameModeSurvival:
		return "survival"
	case world.GameModeCreative:
		return "creative"
	case world.GameModeAdventure:
		return "adventure"
	case world.GameModeSpectator:
		return "spectator"
	default:
		return "survival"
	}
}

type entityHealSource struct{}

func (entityHealSource) HealingSource() {}
