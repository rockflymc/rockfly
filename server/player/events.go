package player

import (
	"net"
	"time"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/cmd"
	"github.com/df-mc/dragonfly/server/event"
	"github.com/df-mc/dragonfly/server/item"
	"github.com/df-mc/dragonfly/server/player/skin"
	"github.com/df-mc/dragonfly/server/session"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
)

// PlayerChatEvent is dispatched when a player attempts to send a chat message.
type PlayerChatEvent struct {
	event.BaseCancellable
	Player  *Player
	Message string
}

// PlayerCommandEvent is dispatched when a player attempts to execute a chat command.
type PlayerCommandEvent struct {
	event.BaseCancellable
	Player    *Player
	Command   cmd.Command
	Arguments []string
}

// PlayerMoveEvent is dispatched when a player moves or rotates in the world.
type PlayerMoveEvent struct {
	event.BaseCancellable
	Player   *Player
	From, To mgl64.Vec3
	FromRot  cube.Rotation
	ToRot    cube.Rotation
}

// PlayerTeleportEvent is dispatched when a player is teleported.
type PlayerTeleportEvent struct {
	event.BaseCancellable
	Player   *Player
	From, To mgl64.Vec3
}

// PlayerBlockBreakEvent is dispatched when a player breaks a block.
type PlayerBlockBreakEvent struct {
	event.BaseCancellable
	Player *Player
	Pos    cube.Pos
	Block  world.Block
	Drops  *[]item.Stack
	XP     *int
}

// PlayerBlockPlaceEvent is dispatched when a player places a block.
type PlayerBlockPlaceEvent struct {
	event.BaseCancellable
	Player *Player
	Pos    cube.Pos
	Block  world.Block
}

// PlayerBlockPickEvent is dispatched when a player picks (middle-clicks) a block.
type PlayerBlockPickEvent struct {
	event.BaseCancellable
	Player *Player
	Pos    cube.Pos
	Block  world.Block
}

// PlayerStartBreakEvent is dispatched when a player starts breaking a block.
type PlayerStartBreakEvent struct {
	event.BaseCancellable
	Player *Player
	Pos    cube.Pos
	Block  world.Block
}

// PlayerAttackEntityEvent is dispatched when a player attacks an entity.
type PlayerAttackEntityEvent struct {
	event.BaseCancellable
	Player   *Player
	Entity   world.Entity
	Force    *float64
	Height   *float64
	Critical *bool
}

// PlayerDamageEvent is dispatched when a player receives damage.
type PlayerDamageEvent struct {
	event.BaseCancellable
	Player         *Player
	Damage         *float64
	Immune         bool
	AttackImmunity *time.Duration
	Source         world.DamageSource
}

// PlayerHealEvent is dispatched when a player heals health points.
type PlayerHealEvent struct {
	event.BaseCancellable
	Player *Player
	Health *float64
	Source world.HealingSource
}

// PlayerDeathEvent is dispatched when a player dies.
type PlayerDeathEvent struct {
	Player        *Player
	Source        world.DamageSource
	KeepInventory *bool
}

// PlayerRespawnEvent is dispatched when a player respawns.
type PlayerRespawnEvent struct {
	Player *Player
	Pos    *mgl64.Vec3
	World  **world.World
}

// PlayerQuitEvent is dispatched when a player disconnects from the server.
type PlayerQuitEvent struct {
	Player *Player
}

// PlayerJoinEvent is dispatched when a player completes spawn and enters the game.
type PlayerJoinEvent struct {
	Player *Player
}

// PlayerToggleSprintEvent is dispatched when a player toggles their sprinting state.
type PlayerToggleSprintEvent struct {
	event.BaseCancellable
	Player *Player
	After  bool
}

// PlayerToggleSneakEvent is dispatched when a player toggles their sneaking state.
type PlayerToggleSneakEvent struct {
	event.BaseCancellable
	Player *Player
	After  bool
}

// PlayerJumpEvent is dispatched when a player jumps.
type PlayerJumpEvent struct {
	Player *Player
}

// PlayerItemUseEvent is dispatched when a player uses an item in the air.
type PlayerItemUseEvent struct {
	event.BaseCancellable
	Player *Player
	Item   item.Stack
}

// PlayerItemUseOnBlockEvent is dispatched when a player uses an item on a block.
type PlayerItemUseOnBlockEvent struct {
	event.BaseCancellable
	Player   *Player
	Item     item.Stack
	Pos      cube.Pos
	Face     cube.Face
	ClickPos mgl64.Vec3
}

// PlayerItemUseOnEntityEvent is dispatched when a player uses an item on an entity.
type PlayerItemUseOnEntityEvent struct {
	event.BaseCancellable
	Player *Player
	Item   item.Stack
	Entity world.Entity
}

// PlayerItemConsumeEvent is dispatched when a player consumes an item.
type PlayerItemConsumeEvent struct {
	event.BaseCancellable
	Player *Player
	Item   item.Stack
}

// PlayerItemReleaseEvent is dispatched when a player releases an item after charging/using.
type PlayerItemReleaseEvent struct {
	event.BaseCancellable
	Player   *Player
	Item     item.Stack
	Duration time.Duration
}

// PlayerItemDropEvent is dispatched when a player drops an item stack.
type PlayerItemDropEvent struct {
	event.BaseCancellable
	Player *Player
	Item   item.Stack
}

// PlayerItemPickupEvent is dispatched when a player picks up an item from the ground.
type PlayerItemPickupEvent struct {
	event.BaseCancellable
	Player *Player
	Item   *item.Stack
}

// PlayerItemDamageEvent is dispatched when an item held or worn takes durability damage.
type PlayerItemDamageEvent struct {
	event.BaseCancellable
	Player *Player
	Item   item.Stack
	Damage *int
}

// PlayerExperienceGainEvent is dispatched when a player gains experience points.
type PlayerExperienceGainEvent struct {
	event.BaseCancellable
	Player *Player
	Amount *int
}

// PlayerPunchAirEvent is dispatched when a player punches the air.
type PlayerPunchAirEvent struct {
	event.BaseCancellable
	Player *Player
}

// PlayerFoodLossEvent is dispatched when a player's food level depletes.
type PlayerFoodLossEvent struct {
	event.BaseCancellable
	Player *Player
	From   int
	To     *int
}

// PlayerSetOnFireEvent is dispatched when a player is ignited.
type PlayerSetOnFireEvent struct {
	event.BaseCancellable
	Player   *Player
	Duration *time.Duration
}

// PlayerFireExtinguishEvent is dispatched when a player extinguishes a fire block.
type PlayerFireExtinguishEvent struct {
	event.BaseCancellable
	Player *Player
	Pos    cube.Pos
}

// PlayerSignEditEvent is dispatched when a player edits a sign.
type PlayerSignEditEvent struct {
	event.BaseCancellable
	Player    *Player
	Pos       cube.Pos
	FrontSide bool
	OldText   string
	NewText   string
}

// PlayerSleepEvent is dispatched when a player attempts to sleep in a bed.
type PlayerSleepEvent struct {
	event.BaseCancellable
	Player       *Player
	SendReminder *bool
}

// PlayerLecternPageTurnEvent is dispatched when a player turns a page in a lectern.
type PlayerLecternPageTurnEvent struct {
	event.BaseCancellable
	Player  *Player
	Pos     cube.Pos
	OldPage int
	NewPage *int
}

// PlayerTransferEvent is dispatched when a player is transferred to another server address.
type PlayerTransferEvent struct {
	event.BaseCancellable
	Player  *Player
	Address *net.UDPAddr
}

// PlayerHeldSlotChangeEvent is dispatched when a player changes their active hotbar slot.
type PlayerHeldSlotChangeEvent struct {
	event.BaseCancellable
	Player *Player
	From   int
	To     int
}

// PlayerSkinChangeEvent is dispatched when a player updates their skin.
type PlayerSkinChangeEvent struct {
	event.BaseCancellable
	Player *Player
	Skin   *skin.Skin
}

// PlayerChangeWorldEvent is dispatched when a player transitions between worlds or dimensions.
type PlayerChangeWorldEvent struct {
	Player *Player
	Before *world.World
	After  *world.World
}

// PlayerDiagnosticsEvent is dispatched when client diagnostics are received.
type PlayerDiagnosticsEvent struct {
	Player      *Player
	Diagnostics session.Diagnostics
}
