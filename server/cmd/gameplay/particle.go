package gameplay

import (
	"strings"

	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/particle"
	"github.com/go-gl/mathgl/mgl64"
)

func registerParticle() {
	cmd.Register(cmd.New("particle", "Creates particles at a position", nil,
		ParticleCmd{},
	))
}

// ParticleCmd spawns particles at a position.
type ParticleCmd struct {
	Name     string                   `cmd:"name"`
	Position cmd.Optional[mgl64.Vec3] `cmd:"position"`
}

func (c ParticleCmd) Allow(src cmd.Source) bool {
	return isAllowed(src)
}

func (c ParticleCmd) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	if tx == nil {
		o.Errorf("Cannot execute particle outside of an active world.")
		return
	}

	pos := src.Position()
	if customPos, ok := c.Position.Load(); ok {
		pos = customPos
	}

	if !isValidVec3(pos) {
		o.Errorf("Coordinates must not contain NaN or Infinity")
		return
	}

	p, ok := parseParticle(c.Name)
	if !ok {
		o.Errorf("Unknown particle '%s'", c.Name)
		return
	}

	tx.AddParticle(pos, p)
	o.Printf("Spawned particle %s at %.2f, %.2f, %.2f", c.Name, pos[0], pos[1], pos[2])
}

func parseParticle(name string) (world.Particle, bool) {
	clean := strings.ToLower(strings.TrimSpace(name))
	clean = strings.TrimPrefix(clean, "minecraft:")

	switch clean {
	case "flame", "entity_flame":
		return particle.EntityFlame{}, true
	case "explosion", "huge_explosion":
		return particle.HugeExplosion{}, true
	case "teleport", "enderman_teleport":
		return particle.EndermanTeleport{}, true
	case "snowball_poof", "poof":
		return particle.SnowballPoof{}, true
	case "egg_smash":
		return particle.EggSmash{}, true
	default:
		return particle.Flame{}, true
	}
}
