package climate

import (
	"github.com/df-mc/dragonfly/server/world/biome"
)

// OverworldTargets returns the canonical vanilla 1.18+ climate point table mapping the 6D parameter
// space to concrete world.Biome instances.
func OverworldTargets() []Target {
	return []Target{
		// Oceans (Deep Oceanic Continentalness <= -1.05)
		{Params: Parameters{Temperature: -0.7, Humidity: 0.0, Continentalness: -1.2, Erosion: 0.0, Weirdness: 0.0, Depth: 0.0}, Biome: biome.DeepFrozenOcean{}},
		{Params: Parameters{Temperature: -0.3, Humidity: 0.0, Continentalness: -1.2, Erosion: 0.0, Weirdness: 0.0, Depth: 0.0}, Biome: biome.DeepColdOcean{}},
		{Params: Parameters{Temperature: 0.0, Humidity: 0.0, Continentalness: -1.2, Erosion: 0.0, Weirdness: 0.0, Depth: 0.0}, Biome: biome.DeepOcean{}},
		{Params: Parameters{Temperature: 0.4, Humidity: 0.0, Continentalness: -1.2, Erosion: 0.0, Weirdness: 0.0, Depth: 0.0}, Biome: biome.DeepLukewarmOcean{}},
		{Params: Parameters{Temperature: 0.8, Humidity: 0.0, Continentalness: -1.2, Erosion: 0.0, Weirdness: 0.0, Depth: 0.0}, Biome: biome.DeepWarmOcean{}},

		// Oceans (Normal Continentalness in [-1.05, -0.455])
		{Params: Parameters{Temperature: -0.7, Humidity: 0.0, Continentalness: -0.7, Erosion: 0.0, Weirdness: 0.0, Depth: 0.0}, Biome: biome.FrozenOcean{}},
		{Params: Parameters{Temperature: -0.3, Humidity: 0.0, Continentalness: -0.7, Erosion: 0.0, Weirdness: 0.0, Depth: 0.0}, Biome: biome.ColdOcean{}},
		{Params: Parameters{Temperature: 0.0, Humidity: 0.0, Continentalness: -0.7, Erosion: 0.0, Weirdness: 0.0, Depth: 0.0}, Biome: biome.Ocean{}},
		{Params: Parameters{Temperature: 0.4, Humidity: 0.0, Continentalness: -0.7, Erosion: 0.0, Weirdness: 0.0, Depth: 0.0}, Biome: biome.LukewarmOcean{}},
		{Params: Parameters{Temperature: 0.8, Humidity: 0.0, Continentalness: -0.7, Erosion: 0.0, Weirdness: 0.0, Depth: 0.0}, Biome: biome.WarmOcean{}},

		// Coasts & Beaches (Continentalness in [-0.455, -0.19])
		{Params: Parameters{Temperature: -0.6, Humidity: 0.0, Continentalness: -0.3, Erosion: 0.0, Weirdness: 0.0, Depth: 0.0}, Biome: biome.SnowyBeach{}},
		{Params: Parameters{Temperature: 0.0, Humidity: 0.0, Continentalness: -0.3, Erosion: 0.0, Weirdness: 0.0, Depth: 0.0}, Biome: biome.Beach{}},
		{Params: Parameters{Temperature: 0.0, Humidity: 0.0, Continentalness: -0.3, Erosion: -0.6, Weirdness: 0.0, Depth: 0.0}, Biome: biome.StonyShore{}},
		{Params: Parameters{Temperature: 0.8, Humidity: -0.5, Continentalness: -0.3, Erosion: 0.0, Weirdness: 0.0, Depth: 0.0}, Biome: biome.Desert{}},

		// Rivers & Valleys (Weirdness Ridges / Continental Valleys)
		{Params: Parameters{Temperature: 0.0, Humidity: 0.0, Continentalness: -0.1, Erosion: 0.0, Weirdness: 0.0, Depth: 0.0}, Biome: biome.River{}},
		{Params: Parameters{Temperature: -0.7, Humidity: 0.0, Continentalness: -0.1, Erosion: 0.0, Weirdness: 0.0, Depth: 0.0}, Biome: biome.FrozenRiver{}},

		// Freezing Inland (Temperature <= -0.45)
		{Params: Parameters{Temperature: -0.7, Humidity: -0.3, Continentalness: 0.1, Erosion: 0.2, Weirdness: 0.0, Depth: 0.0}, Biome: biome.SnowyPlains{}},
		{Params: Parameters{Temperature: -0.7, Humidity: 0.3, Continentalness: 0.1, Erosion: 0.2, Weirdness: 0.0, Depth: 0.0}, Biome: biome.SnowyTaiga{}},
		{Params: Parameters{Temperature: -0.8, Humidity: 0.0, Continentalness: 0.2, Erosion: 0.6, Weirdness: 0.6, Depth: 0.0}, Biome: biome.IceSpikes{}},

		// Cold Inland (Temperature in [-0.45, -0.15])
		{Params: Parameters{Temperature: -0.3, Humidity: -0.2, Continentalness: 0.1, Erosion: 0.1, Weirdness: 0.0, Depth: 0.0}, Biome: biome.Plains{}},
		{Params: Parameters{Temperature: -0.3, Humidity: 0.2, Continentalness: 0.1, Erosion: 0.1, Weirdness: 0.0, Depth: 0.0}, Biome: biome.Taiga{}},
		{Params: Parameters{Temperature: -0.3, Humidity: 0.6, Continentalness: 0.2, Erosion: 0.1, Weirdness: -0.3, Depth: 0.0}, Biome: biome.OldGrowthPineTaiga{}},
		{Params: Parameters{Temperature: -0.3, Humidity: 0.6, Continentalness: 0.2, Erosion: 0.1, Weirdness: 0.3, Depth: 0.0}, Biome: biome.OldGrowthSpruceTaiga{}},

		// Temperate Inland (Temperature in [-0.15, 0.2])
		{Params: Parameters{Temperature: 0.0, Humidity: -0.3, Continentalness: 0.1, Erosion: 0.2, Weirdness: 0.0, Depth: 0.0}, Biome: biome.Plains{}},
		{Params: Parameters{Temperature: 0.0, Humidity: -0.3, Continentalness: 0.1, Erosion: 0.2, Weirdness: 0.6, Depth: 0.0}, Biome: biome.SunflowerPlains{}},
		{Params: Parameters{Temperature: 0.0, Humidity: 0.2, Continentalness: 0.1, Erosion: 0.1, Weirdness: 0.0, Depth: 0.0}, Biome: biome.Forest{}},
		{Params: Parameters{Temperature: 0.0, Humidity: 0.2, Continentalness: 0.1, Erosion: 0.1, Weirdness: 0.6, Depth: 0.0}, Biome: biome.FlowerForest{}},
		{Params: Parameters{Temperature: 0.0, Humidity: 0.3, Continentalness: 0.1, Erosion: 0.2, Weirdness: -0.2, Depth: 0.0}, Biome: biome.BirchForest{}},
		{Params: Parameters{Temperature: 0.0, Humidity: 0.3, Continentalness: 0.2, Erosion: 0.2, Weirdness: -0.6, Depth: 0.0}, Biome: biome.OldGrowthBirchForest{}},
		{Params: Parameters{Temperature: 0.0, Humidity: 0.6, Continentalness: 0.1, Erosion: 0.0, Weirdness: 0.0, Depth: 0.0}, Biome: biome.DarkForest{}},
		{Params: Parameters{Temperature: 0.0, Humidity: 0.65, Continentalness: 0.2, Erosion: 0.0, Weirdness: -0.7, Depth: 0.0}, Biome: biome.PaleGarden{}},

		// Humid & Swampy Inland
		{Params: Parameters{Temperature: 0.3, Humidity: 0.5, Continentalness: 0.0, Erosion: 0.3, Weirdness: 0.0, Depth: 0.0}, Biome: biome.Swamp{}},
		{Params: Parameters{Temperature: 0.7, Humidity: 0.6, Continentalness: 0.0, Erosion: 0.3, Weirdness: 0.0, Depth: 0.0}, Biome: biome.MangroveSwamp{}},

		// Hot & Humid (Jungles)
		{Params: Parameters{Temperature: 0.4, Humidity: 0.6, Continentalness: 0.2, Erosion: 0.0, Weirdness: 0.0, Depth: 0.0}, Biome: biome.Jungle{}},
		{Params: Parameters{Temperature: 0.4, Humidity: 0.6, Continentalness: 0.2, Erosion: 0.0, Weirdness: 0.5, Depth: 0.0}, Biome: biome.BambooJungle{}},

		// Hot & Arid (Savannas, Deserts, Badlands)
		{Params: Parameters{Temperature: 0.7, Humidity: -0.1, Continentalness: 0.1, Erosion: 0.1, Weirdness: 0.0, Depth: 0.0}, Biome: biome.Savanna{}},
		{Params: Parameters{Temperature: 0.8, Humidity: -0.6, Continentalness: 0.1, Erosion: 0.2, Weirdness: 0.0, Depth: 0.0}, Biome: biome.Desert{}},
		{Params: Parameters{Temperature: 0.7, Humidity: -0.4, Continentalness: 0.4, Erosion: 0.1, Weirdness: 0.0, Depth: 0.0}, Biome: biome.Badlands{}},
		{Params: Parameters{Temperature: 0.7, Humidity: -0.4, Continentalness: 0.4, Erosion: -0.7, Weirdness: 0.0, Depth: 0.0}, Biome: biome.ErodedBadlands{}},
		{Params: Parameters{Temperature: 0.7, Humidity: 0.0, Continentalness: 0.4, Erosion: 0.1, Weirdness: 0.0, Depth: 0.0}, Biome: biome.WoodedBadlandsPlateau{}},

		// Mountains & Slopes (Erosion < -0.375)
		{Params: Parameters{Temperature: 0.0, Humidity: -0.1, Continentalness: 0.2, Erosion: -0.5, Weirdness: 0.0, Depth: 0.0}, Biome: biome.Meadow{}},
		{Params: Parameters{Temperature: 0.0, Humidity: -0.1, Continentalness: 0.2, Erosion: -0.5, Weirdness: 0.6, Depth: 0.0}, Biome: biome.CherryGrove{}},
		{Params: Parameters{Temperature: -0.3, Humidity: 0.3, Continentalness: 0.2, Erosion: -0.5, Weirdness: 0.0, Depth: 0.0}, Biome: biome.Grove{}},
		{Params: Parameters{Temperature: -0.3, Humidity: -0.3, Continentalness: 0.2, Erosion: -0.5, Weirdness: 0.0, Depth: 0.0}, Biome: biome.SnowySlopes{}},
		{Params: Parameters{Temperature: -0.4, Humidity: 0.0, Continentalness: 0.3, Erosion: -0.9, Weirdness: 0.0, Depth: 0.0}, Biome: biome.JaggedPeaks{}},
		{Params: Parameters{Temperature: -0.7, Humidity: 0.0, Continentalness: 0.3, Erosion: -0.9, Weirdness: 0.0, Depth: 0.0}, Biome: biome.FrozenPeaks{}},
		{Params: Parameters{Temperature: 0.4, Humidity: 0.0, Continentalness: 0.3, Erosion: -0.9, Weirdness: 0.0, Depth: 0.0}, Biome: biome.StonyPeaks{}},
		{Params: Parameters{Temperature: 0.0, Humidity: 0.0, Continentalness: 0.3, Erosion: -0.6, Weirdness: -0.5, Depth: 0.0}, Biome: biome.WindsweptHills{}},
		{Params: Parameters{Temperature: 0.0, Humidity: 0.0, Continentalness: 0.3, Erosion: -0.8, Weirdness: -0.7, Depth: 0.0}, Biome: biome.WindsweptGravellyHills{}},
		{Params: Parameters{Temperature: 0.0, Humidity: 0.4, Continentalness: 0.3, Erosion: -0.6, Weirdness: -0.5, Depth: 0.0}, Biome: biome.WindsweptForest{}},
		{Params: Parameters{Temperature: 0.7, Humidity: 0.0, Continentalness: 0.3, Erosion: -0.6, Weirdness: -0.5, Depth: 0.0}, Biome: biome.WindsweptSavanna{}},
		{Params: Parameters{Temperature: 0.0, Humidity: 0.3, Continentalness: 0.2, Erosion: 0.0, Weirdness: 0.3, Depth: 0.0}, Biome: biome.WoodedHills{}},

		// Rare Island Biomes
		{Params: Parameters{Temperature: 0.0, Humidity: 0.0, Continentalness: -1.3, Erosion: 0.0, Weirdness: 0.9, Depth: 0.0}, Biome: biome.MushroomFields{}},
		{Params: Parameters{Temperature: 0.0, Humidity: 0.0, Continentalness: -1.0, Erosion: 0.0, Weirdness: 0.9, Depth: 0.0}, Biome: biome.MushroomFieldShore{}},

		// Subterranean Cave Biomes (Depth >= 0.25)
		{Params: Parameters{Temperature: 0.2, Humidity: 0.5, Continentalness: 0.0, Erosion: 0.0, Weirdness: 0.0, Depth: 0.4}, Biome: biome.LushCaves{}},
		{Params: Parameters{Temperature: 0.0, Humidity: -0.4, Continentalness: 0.0, Erosion: 0.0, Weirdness: 0.0, Depth: 0.4}, Biome: biome.DripstoneCaves{}},
		{Params: Parameters{Temperature: 0.0, Humidity: 0.0, Continentalness: 0.2, Erosion: -0.4, Weirdness: 0.0, Depth: 0.8}, Biome: biome.DeepDark{}},
	}
}
