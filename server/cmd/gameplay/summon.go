package gameplay

import (
	"strings"
	"time"

	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/entity"
	"github.com/df-mc/dragonfly/server/item/potion"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
)

func registerSummon() {
	cmd.Register(cmd.New("summon", "Summons an entity at a given position", nil,
		Summon{},
	))
}

// Summon spawns an entity in the world.
type Summon struct {
	Entity   string                    `cmd:"entity"`
	Position cmd.Optional[mgl64.Vec3]  `cmd:"position"`
}

func (c Summon) Allow(src cmd.Source) bool {
	return isAllowed(src)
}

func (c Summon) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	if tx == nil {
		o.Errorf("Cannot execute summon outside of an active world.")
		return
	}

	pos := src.Position()
	if p, ok := c.Position.Load(); ok {
		pos = p
	}

	opts := world.EntitySpawnOpts{Position: pos}
	entityName := strings.ToLower(strings.TrimSpace(c.Entity))
	entityName = strings.TrimPrefix(entityName, "minecraft:")

	var handle *world.EntityHandle
	switch entityName {
	case "lightning", "lightning_bolt":
		handle = entity.NewLightning(opts)
	case "tnt":
		handle = entity.NewTNT(opts, 4*time.Second)
	case "end_crystal", "ender_crystal":
		handle = entity.NewEndCrystal(opts)
	case "experience_orb", "xp_orb":
		handle = entity.NewExperienceOrb(opts, 10)
	case "arrow":
		handle = entity.NewArrow(opts, nil)
	case "snowball":
		handle = entity.NewSnowball(opts, nil)
	case "egg":
		handle = entity.NewEgg(opts, nil)
	case "ender_pearl":
		handle = entity.NewEnderPearl(opts, nil)
	case "area_effect_cloud":
		handle = entity.NewAreaEffectCloud(opts, potion.Awkward())
	default:
		o.Errorf("Unsupported or unknown entity '%s' for summon", c.Entity)
		return
	}

	if handle != nil {
		tx.AddEntity(handle)
		o.Printf("Summoned new %s at %.2f, %.2f, %.2f", c.Entity, pos[0], pos[1], pos[2])
	}
}
