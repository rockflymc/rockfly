package gameplay

import (
	"strings"
	"time"

	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/entity/effect"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
)

func registerEffect() {
	cmd.Register(cmd.New("effect", "Applies or removes status effects", nil,
		EffectClear{},
		EffectGive{},
	))
}

// EffectClear removes all effects from the target players.
type EffectClear struct {
	Player []cmd.Target   `cmd:"player"`
	Clear  cmd.SubCommand `cmd:"clear"`
}

func (c EffectClear) Allow(src cmd.Source) bool {
	return isAllowed(src)
}

func (c EffectClear) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	count := 0
	for _, target := range c.Player {
		if p, ok := target.(*player.Player); ok {
			for _, eff := range p.Effects() {
				p.RemoveEffect(eff.Type())
			}
			count++
		}
	}
	if count == 0 {
		o.Errorf("No valid players found.")
		return
	}
	o.Printf("Cleared all effects from %d player(s)", count)
}

// EffectGive applies a status effect to target players.
type EffectGive struct {
	Player        []cmd.Target          `cmd:"player"`
	Effect        string                `cmd:"effect"`
	Seconds       cmd.Optional[int]     `cmd:"seconds"`
	Amplifier     cmd.Optional[int]     `cmd:"amplifier"`
	HideParticles cmd.Optional[bool]    `cmd:"hideParticles"`
}

func (c EffectGive) Allow(src cmd.Source) bool {
	return isAllowed(src)
}

func (c EffectGive) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	effType, ok := parseEffectType(c.Effect)
	if !ok {
		o.Errorf("Unknown effect '%s'", c.Effect)
		return
	}

	seconds := 30
	if s, ok := c.Seconds.Load(); ok && s > 0 {
		seconds = s
	}

	level := 1
	if a, ok := c.Amplifier.Load(); ok && a >= 0 {
		// In Minecraft commands, amplifier 0 is level 1, amplifier 1 is level 2 (Speed II), etc.
		level = a + 1
	}

	dur := time.Duration(seconds) * time.Second

	var eff effect.Effect
	if instant, ok := effType.(effect.Type); ok {
		if lasting, ok := effType.(effect.LastingType); ok {
			eff = effect.New(lasting, level, dur)
		} else {
			eff = effect.NewInstant(instant, level)
		}
	}

	if hp, ok := c.HideParticles.Load(); ok && hp {
		eff = eff.WithoutParticles()
	}

	count := 0
	for _, target := range c.Player {
		if p, ok := target.(*player.Player); ok {
			p.AddEffect(eff)
			count++
		}
	}

	if count == 0 {
		o.Errorf("No valid players found.")
		return
	}
	o.Printf("Applied effect %s (level %d) to %d player(s) for %ds", c.Effect, level, count, seconds)
}

func parseEffectType(str string) (effect.Type, bool) {
	name := strings.ToLower(strings.TrimSpace(str))
	name = strings.TrimPrefix(name, "minecraft:")

	switch name {
	case "speed":
		return effect.Speed, true
	case "slowness":
		return effect.Slowness, true
	case "haste":
		return effect.Haste, true
	case "mining_fatigue":
		return effect.MiningFatigue, true
	case "strength":
		return effect.Strength, true
	case "instant_health":
		return effect.InstantHealth, true
	case "instant_damage":
		return effect.InstantDamage, true
	case "jump_boost":
		return effect.JumpBoost, true
	case "nausea":
		return effect.Nausea, true
	case "regeneration":
		return effect.Regeneration, true
	case "resistance":
		return effect.Resistance, true
	case "fire_resistance":
		return effect.FireResistance, true
	case "water_breathing":
		return effect.WaterBreathing, true
	case "invisibility":
		return effect.Invisibility, true
	case "blindness":
		return effect.Blindness, true
	case "night_vision":
		return effect.NightVision, true
	case "hunger":
		return effect.Hunger, true
	case "weakness":
		return effect.Weakness, true
	case "poison":
		return effect.Poison, true
	case "wither":
		return effect.Wither, true
	case "health_boost":
		return effect.HealthBoost, true
	case "absorption":
		return effect.Absorption, true
	case "saturation":
		return effect.Saturation, true
	case "levitation":
		return effect.Levitation, true
	case "fatal_poison":
		return effect.FatalPoison, true
	case "conduit_power":
		return effect.ConduitPower, true
	case "slow_falling":
		return effect.SlowFalling, true
	case "darkness":
		return effect.Darkness, true
	default:
		return nil, false
	}
}
