package admin

import (
	"math"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/structure/vanilla"
)

// StructureEnum implements cmd.Enum for Bedrock client tab completion of structure names.
type StructureEnum string

func (StructureEnum) Type() string { return "Structure" }
func (StructureEnum) Options(cmd.Source) []string {
	var opts []string
	for _, t := range vanilla.AllTypes() {
		opts = append(opts, string(t))
	}
	return opts
}

func registerLocate() {
	cmd.Register(cmd.New("locate", "Finds the nearest structure or feature", nil,
		LocateStructure{},
	))
}

// LocateStructure locates the nearest vanilla structure.
type LocateStructure struct {
	Sub       cmd.SubCommand `cmd:"structure"`
	Structure StructureEnum  `cmd:"feature"`
}

// Allow reports whether the source is permitted to execute /locate.
func (c LocateStructure) Allow(src cmd.Source) bool {
	return isAllowed(src)
}

// Run executes the /locate structure command.
func (c LocateStructure) Run(src cmd.Source, o *cmd.Output, tx *world.Tx) {
	sType := vanilla.Type(c.Structure)

	origin := cube.PosFromVec3(src.Position())

	// Try extracting world seed from generator
	var seed int64
	if tx != nil && tx.World() != nil {
		if sGen, ok := tx.World().Generator().(interface{ Seed() int64 }); ok {
			seed = sGen.Seed()
		}
	}

	pos, found := vanilla.Locate(seed, origin, sType, 200, nil)
	if !found {
		o.Errorf("Could not find structure %s within reasonable search radius", c.Structure)
		return
	}

	dx := float64(pos.X() - origin.X())
	dz := float64(pos.Z() - origin.Z())
	dist := int(math.Round(math.Sqrt(dx*dx + dz*dz)))

	o.Printf("The nearest %s is at [%d, (y?), %d] (%d blocks away)", c.Structure, pos.X(), pos.Z(), dist)
}
