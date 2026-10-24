package vanilla

import "github.com/df-mc/dragonfly/server/world"

// Create instantiates a new world.Structure for the specified structure type.
func Create(t Type) (world.Structure, bool) {
	switch t {
	case TypeVillage:
		return NewVillage(VillagePlains), true
	case TypePillagerOutpost:
		return NewPillagerOutpost(), true
	case TypeDesertPyramid:
		return NewDesertPyramid(), true
	case TypeWitchHut:
		return NewWitchHut(), true
	case TypeMineshaft:
		return NewMineshaft(), true
	case TypeOceanMonument:
		return NewOceanMonument(), true
	case TypeStronghold:
		return NewStronghold(), true
	case TypeAncientCity:
		return NewAncientCity(), true
	case TypeNetherFortress:
		return NewNetherFortress(), true
	case TypeBastionRemnant:
		return NewBastionRemnant(), true
	default:
		return nil, false
	}
}
