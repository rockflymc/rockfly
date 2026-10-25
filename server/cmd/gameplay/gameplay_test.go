package gameplay

import (
	"strings"
	"testing"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/entity/effect"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/google/uuid"
)

type testCaptureSource struct {
	name     string
	pos      mgl64.Vec3
	messages []string
	errors   []string
}

func (s *testCaptureSource) Name() string {
	return s.name
}

func (s *testCaptureSource) Position() mgl64.Vec3 {
	return s.pos
}

func (s *testCaptureSource) SendCommandOutput(o *cmd.Output) {
	for _, m := range o.Messages() {
		s.messages = append(s.messages, m.String())
	}
	for _, e := range o.Errors() {
		s.errors = append(s.errors, e.Error())
	}
}

func TestGameplayCommandsRegistration(t *testing.T) {
	RegisterAll(nil)

	expected := []string{
		"gamemode", "give", "teleport", "time", "weather",
		"summon", "setblock", "fill", "clone", "effect",
		"enchant", "clear", "kill", "difficulty", "defaultgamemode",
		"spawnpoint", "setworldspawn", "title", "particle", "playsound",
	}

	commands := cmd.Commands()
	for _, name := range expected {
		if _, ok := commands[name]; !ok {
			t.Errorf("expected command /%s to be registered", name)
		}
	}
}

func TestFillCommand_RelativeCoords(t *testing.T) {
	RegisterAll(nil)

	w := world.New()
	defer w.Close()

	fillCmd, ok := cmd.ByAlias("fill")
	if !ok {
		t.Fatalf("fill command not found")
	}

	src := &testCaptureSource{
		name: "CONSOLE",
		pos:  mgl64.Vec3{10, 64, 10},
	}

	// Verification requirement from Phase 48:
	// Execute /fill ~-5 ~-1 ~-5 ~5 ~-1 ~5 stone
	task := w.Do(func(tx *world.Tx) {
		fillCmd.Execute("~-5 ~-1 ~-5 ~5 ~-1 ~5 stone", src, tx)

		stone, ok := world.BlockByName("minecraft:stone", nil)
		if !ok {
			t.Fatalf("minecraft:stone block not found")
		}

		// Verify area from (5, 63, 5) to (15, 63, 15) is stone
		for x := 5; x <= 15; x++ {
			for z := 5; z <= 15; z++ {
				pos := cube.Pos{x, 63, z}
				b := tx.Block(pos)
				if b != stone {
					t.Fatalf("expected block at %v to be stone, got %v", pos, b)
				}
			}
		}
	})
	<-task.Done()

	if len(src.messages) == 0 {
		t.Fatalf("expected output from fill command, got none (errors: %v)", src.errors)
	}
	if !strings.Contains(src.messages[0], "121 block(s)") {
		t.Errorf("expected 121 blocks filled, got %v", src.messages[0])
	}
}

func TestEffectCommand_TargetSelector(t *testing.T) {
	RegisterAll(nil)

	w := world.New()
	defer w.Close()

	effectCmd, ok := cmd.ByAlias("effect")
	if !ok {
		t.Fatalf("effect command not found")
	}

	src := &testCaptureSource{name: "CONSOLE"}

	task := w.Do(func(tx *world.Tx) {
		// Spawn two players
		h1 := world.NewEntity(player.Type, player.Config{
			Name:     "Steve",
			UUID:     uuid.New(),
			Position: mgl64.Vec3{0, 64, 0},
		})
		p1 := tx.AddEntity(h1).(*player.Player)

		h2 := world.NewEntity(player.Type, player.Config{
			Name:     "Alex",
			UUID:     uuid.New(),
			Position: mgl64.Vec3{5, 64, 5},
		})
		p2 := tx.AddEntity(h2).(*player.Player)

		// Verification requirement from Phase 48:
		// Execute /effect @a speed 30 2
		effectCmd.Execute("@a speed 30 1", src, tx)

		// Verify both Steve and Alex received Speed II
		eff1, ok1 := p1.Effect(effect.Speed)
		if !ok1 {
			t.Errorf("expected Steve to have Speed effect")
		} else if eff1.Level() != 2 {
			t.Errorf("expected Steve Speed level 2, got %d", eff1.Level())
		}

		eff2, ok2 := p2.Effect(effect.Speed)
		if !ok2 {
			t.Errorf("expected Alex to have Speed effect")
		} else if eff2.Level() != 2 {
			t.Errorf("expected Alex Speed level 2, got %d", eff2.Level())
		}
	})
	<-task.Done()

	if len(src.messages) == 0 {
		t.Fatalf("expected message from effect command, errors: %v", src.errors)
	}
}

func TestTimeAndWeatherCommands(t *testing.T) {
	RegisterAll(nil)

	w := world.New()
	defer w.Close()

	timeCmd, ok := cmd.ByAlias("time")
	if !ok {
		t.Fatalf("time command not found")
	}
	weatherCmd, ok := cmd.ByAlias("weather")
	if !ok {
		t.Fatalf("weather command not found")
	}

	src := &testCaptureSource{name: "CONSOLE"}

	task := w.Do(func(tx *world.Tx) {
		timeCmd.Execute("set noon", src, tx)
		if tx.World().Time() != 6000 {
			t.Errorf("expected time 6000, got %d", tx.World().Time())
		}

		weatherCmd.Execute("rain 300", src, tx)
		// Check that weather command executed without errors
	})
	<-task.Done()

	if len(src.errors) > 0 {
		t.Errorf("unexpected errors during time/weather commands: %v", src.errors)
	}
}

func TestGameModeCommand(t *testing.T) {
	RegisterAll(nil)

	w := world.New()
	defer w.Close()

	gmCmd, ok := cmd.ByAlias("gamemode")
	if !ok {
		t.Fatalf("gamemode command not found")
	}

	src := &testCaptureSource{name: "CONSOLE"}

	task := w.Do(func(tx *world.Tx) {
		h := world.NewEntity(player.Type, player.Config{
			Name:     "Steve",
			UUID:     uuid.New(),
			GameMode: world.GameModeSurvival,
		})
		p := tx.AddEntity(h).(*player.Player)

		gmCmd.Execute("creative Steve", src, tx)
		if p.GameMode() != world.GameModeCreative {
			t.Errorf("expected creative game mode, got %v", p.GameMode())
		}
	})
	<-task.Done()
}

func TestGameMode_PermissionEnforcement(t *testing.T) {
	RegisterAll(nil)

	w := world.New()
	defer w.Close()

	gmCmd, ok := cmd.ByAlias("gamemode")
	if !ok {
		t.Fatalf("gamemode command not found")
	}

	task := w.Do(func(tx *world.Tx) {
		h := world.NewEntity(player.Type, player.Config{
			Name:     "NonOpPlayer",
			UUID:     uuid.New(),
			GameMode: world.GameModeSurvival,
		})
		p := tx.AddEntity(h).(*player.Player)

		// Non-op player attempts to run /gamemode creative
		gmCmd.Execute("creative", p, tx)
		if p.GameMode() != world.GameModeSurvival {
			t.Errorf("vulnerability: non-op player was able to set creative gamemode!")
		}
	})
	<-task.Done()
}

func TestSpatialCommands_NaNAndBoundsSanitization(t *testing.T) {
	RegisterAll(nil)

	w := world.New()
	defer w.Close()

	src := &testCaptureSource{name: "CONSOLE"}

	task := w.Do(func(tx *world.Tx) {
		// Test setblock with NaN coordinates
		sbCmd, _ := cmd.ByAlias("setblock")
		sbCmd.Execute("NaN 64 0 stone", src, tx)
		if len(src.errors) == 0 {
			t.Errorf("expected error for NaN setblock coordinates")
		}

		// Test setblock out of bounds
		src.errors = nil
		sbCmd.Execute("0 5000 0 stone", src, tx)
		if len(src.errors) == 0 {
			t.Errorf("expected error for out of bounds setblock")
		}

		// Test fill with NaN coordinates
		src.errors = nil
		fillCmd, _ := cmd.ByAlias("fill")
		fillCmd.Execute("0 NaN 0 10 64 10 stone", src, tx)
		if len(src.errors) == 0 {
			t.Errorf("expected error for NaN fill coordinates")
		}

		// Test fill out of bounds height
		src.errors = nil
		fillCmd.Execute("0 1000 0 10 2000 10 stone", src, tx)
		if len(src.errors) == 0 {
			t.Errorf("expected error for out of bounds fill coordinates")
		}
	})
	<-task.Done()
}

func TestValueValidation(t *testing.T) {
	RegisterAll(nil)

	w := world.New()
	defer w.Close()

	src := &testCaptureSource{name: "CONSOLE"}

	task := w.Do(func(tx *world.Tx) {
		h := world.NewEntity(player.Type, player.Config{
			Name: "Player1",
			UUID: uuid.New(),
		})
		_ = tx.AddEntity(h)

		// Give invalid amount
		giveCmd, _ := cmd.ByAlias("give")
		giveCmd.Execute("Player1 diamond 999999", src, tx)
		if len(src.errors) == 0 {
			t.Errorf("expected error for give amount exceeding limit")
		}

		// Effect invalid duration
		src.errors = nil
		effCmd, _ := cmd.ByAlias("effect")
		effCmd.Execute("Player1 speed 9999999", src, tx)
		if len(src.errors) == 0 {
			t.Errorf("expected error for effect duration exceeding limit")
		}
	})
	<-task.Done()
}
