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
	p.mu.Lock()
	defer p.mu.Unlock()

	entry := &tsEventListener{
		eventName: strings.ToLower(eventName),
		priority:  prio,
		callback:  callback,
		plugin:    p,
	}

	p.eventHandlers = append(p.eventHandlers, entry)

	return func() {
		p.mu.Lock()
		defer p.mu.Unlock()
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
	p.mu.RLock()
	handlers := make([]*tsEventListener, len(p.eventHandlers))
	copy(handlers, p.eventHandlers)
	p.mu.RUnlock()

	name := strings.ToLower(eventName)
	for _, h := range handlers {
		if h.eventName == name {
			val := p.vm.ToValue(eventObj)
			_, err := p.callWithTimeout(h.callback, goja.Undefined(), val)
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
	var pObj any
	if p, ok := src.(*player.Player); ok {
		pObj = map[string]any{
			"name": p.Name(),
			"uuid": p.UUID().String(),
			"heal": func(amount float64) {
				p.Heal(amount, entityHealSource{})
			},
			"teleport": func(x, y, z float64) {
				p.Teleport(mgl64.Vec3{x, y, z})
			},
			"sendMessage": func(msg string) {
				p.Message(msg)
			},
		}
	}

	rawArgs := ""
	if a, ok := c.Args.Load(); ok {
		rawArgs = string(a)
	}
	args := strings.Fields(rawArgs)

	valPlayer := c.plugin.vm.ToValue(pObj)
	valArgs := c.plugin.vm.ToValue(args)

	_, err := c.plugin.callWithTimeout(c.callback, goja.Undefined(), valPlayer, valArgs)
	if err != nil {
		o.Errorf("Error executing script command: %v", err)
	}
}

type entityHealSource struct{}

func (entityHealSource) HealingSource() {}
