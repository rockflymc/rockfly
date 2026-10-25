package vanilla

import "github.com/df-mc/dragonfly/server/block/cube"

// Type represents a canonical Minecraft vanilla structure type.
type Type string

const (
	TypeVillage         Type = "village"
	TypePillagerOutpost Type = "pillager_outpost"
	TypeDesertPyramid   Type = "desert_pyramid"
	TypeWitchHut        Type = "swamp_hut"
	TypeMineshaft       Type = "mineshaft"
	TypeOceanMonument   Type = "monument"
	TypeStronghold      Type = "stronghold"
	TypeAncientCity     Type = "ancient_city"
	TypeTrialChambers   Type = "trial_chambers"
	TypeNetherFortress  Type = "fortress"
	TypeBastionRemnant  Type = "bastion_remnant"
)

// AllTypes returns a slice of all supported vanilla structure types.
func AllTypes() []Type {
	return []Type{
		TypeVillage,
		TypePillagerOutpost,
		TypeDesertPyramid,
		TypeWitchHut,
		TypeMineshaft,
		TypeOceanMonument,
		TypeStronghold,
		TypeAncientCity,
		TypeTrialChambers,
		TypeNetherFortress,
		TypeBastionRemnant,
	}
}

// String returns the string representation of the structure type.
func (t Type) String() string {
	return string(t)
}

// PlacementConfig defines the chunk grid spacing, separation, and PRNG salt
// for deterministic structure generation.
type PlacementConfig struct {
	// Spacing is the grid cell width and depth in chunks.
	Spacing int
	// Separation is the minimum chunk distance between structures of this type.
	Separation int
	// Salt is the unique numeric salt added to the world seed.
	Salt int64
	// MinY and MaxY define the valid vertical placement envelope.
	MinY int
	MaxY int
}

// DefaultPlacements holds the canonical vanilla placement parameters for each structure.
var DefaultPlacements = map[Type]PlacementConfig{
	TypeVillage: {
		Spacing:    34,
		Separation: 8,
		Salt:       10387312,
		MinY:       60,
		MaxY:       120,
	},
	TypePillagerOutpost: {
		Spacing:    32,
		Separation: 8,
		Salt:       165745296,
		MinY:       60,
		MaxY:       140,
	},
	TypeDesertPyramid: {
		Spacing:    32,
		Separation: 8,
		Salt:       14357617,
		MinY:       60,
		MaxY:       90,
	},
	TypeWitchHut: {
		Spacing:    32,
		Separation: 8,
		Salt:       14357620,
		MinY:       60,
		MaxY:       75,
	},
	TypeMineshaft: {
		Spacing:    1,
		Separation: 0,
		Salt:       0,
		MinY:       -50,
		MaxY:       30,
	},
	TypeOceanMonument: {
		Spacing:    32,
		Separation: 5,
		Salt:       10387313,
		MinY:       30,
		MaxY:       60,
	},
	TypeStronghold: {
		Spacing:    32,
		Separation: 8,
		Salt:       0,
		MinY:       -40,
		MaxY:       20,
	},
	TypeAncientCity: {
		Spacing:    24,
		Separation: 8,
		Salt:       20083232,
		MinY:       -52,
		MaxY:       -30,
	},
	TypeTrialChambers: {
		Spacing:    32,
		Separation: 8,
		Salt:       94251324,
		MinY:       -40,
		MaxY:       -10,
	},
	TypeNetherFortress: {
		Spacing:    27,
		Separation: 4,
		Salt:       30084232,
		MinY:       30,
		MaxY:       80,
	},
	TypeBastionRemnant: {
		Spacing:    27,
		Separation: 4,
		Salt:       30084235,
		MinY:       30,
		MaxY:       80,
	},
}

// BoundingBox defines an integer 3D Axis-Aligned Bounding Box.
type BoundingBox struct {
	Min cube.Pos
	Max cube.Pos
}

// NewBoundingBox constructs a BoundingBox ensuring min <= max.
func NewBoundingBox(min, max cube.Pos) BoundingBox {
	return BoundingBox{
		Min: cube.Pos{
			minInt(min.X(), max.X()),
			minInt(min.Y(), max.Y()),
			minInt(min.Z(), max.Z()),
		},
		Max: cube.Pos{
			maxInt(min.X(), max.X()),
			maxInt(min.Y(), max.Y()),
			maxInt(min.Z(), max.Z()),
		},
	}
}

// Contains reports whether pos is inside or on the boundary of the bounding box.
func (b BoundingBox) Contains(pos cube.Pos) bool {
	return pos.X() >= b.Min.X() && pos.X() <= b.Max.X() &&
		pos.Y() >= b.Min.Y() && pos.Y() <= b.Max.Y() &&
		pos.Z() >= b.Min.Z() && pos.Z() <= b.Max.Z()
}

// Intersects reports whether b intersects another bounding box other.
func (b BoundingBox) Intersects(other BoundingBox) bool {
	return b.Min.X() <= other.Max.X() && b.Max.X() >= other.Min.X() &&
		b.Min.Y() <= other.Max.Y() && b.Max.Y() >= other.Min.Y() &&
		b.Min.Z() <= other.Max.Z() && b.Max.Z() >= other.Min.Z()
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
