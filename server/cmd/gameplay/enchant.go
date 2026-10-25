package gameplay

import (
	"strings"

	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/item/enchantment"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
)

func registerEnchant() {
	cmd.Register(cmd.New("enchant", "Adds an enchantment to a player's selected item", nil,
		Enchant{},
	))
}

// Enchant enchants the item currently held in the target player's main hand.
type Enchant struct {
	Player      []cmd.Target      `cmd:"player"`
	Enchantment string            `cmd:"enchantmentName"`
	Level       cmd.Optional[int] `cmd:"level"`
}

func (c Enchant) Allow(src cmd.Source) bool {
	return isAllowed(src)
}

func (c Enchant) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	enchType, ok := parseEnchantmentType(c.Enchantment)
	if !ok {
		o.Errorf("Unknown enchantment '%s'", c.Enchantment)
		return
	}

	lvl := 1
	if l, ok := c.Level.Load(); ok {
		if l <= 0 || l > 255 {
			o.Errorf("Enchantment level must be between 1 and 255 (got %d)", l)
			return
		}
		lvl = l
	}

	count := 0
	for _, target := range c.Player {
		if p, ok := target.(*player.Player); ok {
			mainHand, offHand := p.HeldItems()
			if mainHand.Empty() {
				continue
			}
			ench := item.NewEnchantment(enchType, lvl)
			mainHand = mainHand.WithEnchantments(ench)
			p.SetHeldItems(mainHand, offHand)
			count++
		}
	}

	if count == 0 {
		o.Errorf("No valid target players holding items found.")
		return
	}
	o.Printf("Enchanted item with %s %d for %d player(s)", c.Enchantment, lvl, count)
}

func parseEnchantmentType(str string) (item.EnchantmentType, bool) {
	name := strings.ToLower(strings.TrimSpace(str))
	name = strings.TrimPrefix(name, "minecraft:")

	switch name {
	case "protection":
		return enchantment.Protection, true
	case "fire_protection":
		return enchantment.FireProtection, true
	case "feather_falling":
		return enchantment.FeatherFalling, true
	case "blast_protection":
		return enchantment.BlastProtection, true
	case "projectile_protection":
		return enchantment.ProjectileProtection, true
	case "thorns":
		return enchantment.Thorns, true
	case "respiration":
		return enchantment.Respiration, true
	case "depth_strider":
		return enchantment.DepthStrider, true
	case "aqua_affinity":
		return enchantment.AquaAffinity, true
	case "sharpness":
		return enchantment.Sharpness, true
	case "knockback":
		return enchantment.Knockback, true
	case "fire_aspect":
		return enchantment.FireAspect, true
	case "efficiency":
		return enchantment.Efficiency, true
	case "silk_touch":
		return enchantment.SilkTouch, true
	case "unbreaking":
		return enchantment.Unbreaking, true
	case "fortune":
		return enchantment.Fortune, true
	case "power":
		return enchantment.Power, true
	case "punch":
		return enchantment.Punch, true
	case "flame":
		return enchantment.Flame, true
	case "infinity":
		return enchantment.Infinity, true
	case "mending":
		return enchantment.Mending, true
	case "vanishing", "curse_of_vanishing":
		return enchantment.CurseOfVanishing, true
	case "multishot":
		return enchantment.Multishot, true
	case "piercing":
		return enchantment.Piercing, true
	case "quick_charge":
		return enchantment.QuickCharge, true
	case "soul_speed":
		return enchantment.SoulSpeed, true
	case "swift_sneak":
		return enchantment.SwiftSneak, true
	default:
		return nil, false
	}
}
