package world

import (
	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/event"
	"github.com/go-gl/mathgl/mgl64"
)

// eventWorldBridge bridges world events to event.DefaultBus() while maintaining
// backward compatibility with any custom Handler set with World.Handle().
type eventWorldBridge struct {
	w *World
}

var _ Handler = (*eventWorldBridge)(nil)

func (b *eventWorldBridge) inner() Handler {
	if b.w == nil {
		return NopHandler{}
	}
	h := b.w.handler.Load()
	if h == nil {
		return NopHandler{}
	}
	return *h
}

func (b *eventWorldBridge) HandleLiquidFlow(ctx *Context, from, into cube.Pos, liquid Liquid, replaced Block) {
	e := &LiquidFlowEvent{
		World:    b.w,
		From:     from,
		Into:     into,
		Liquid:   liquid,
		Replaced: replaced,
	}
	event.DefaultBus().Publish(e)
	if e.Cancelled() {
		ctx.Cancel()
		return
	}
	b.inner().HandleLiquidFlow(ctx, from, into, liquid, replaced)
}

func (b *eventWorldBridge) HandleLiquidDecay(ctx *Context, pos cube.Pos, before, after Liquid) {
	e := &LiquidDecayEvent{
		World:  b.w,
		Pos:    pos,
		Before: before,
		After:  after,
	}
	event.DefaultBus().Publish(e)
	if e.Cancelled() {
		ctx.Cancel()
		return
	}
	b.inner().HandleLiquidDecay(ctx, pos, before, after)
}

func (b *eventWorldBridge) HandleLiquidHarden(ctx *Context, hardenedPos cube.Pos, liquidHardened, otherLiquid, newBlock Block) {
	e := &LiquidHardenEvent{
		World:          b.w,
		HardenedPos:    hardenedPos,
		LiquidHardened: liquidHardened,
		OtherLiquid:    otherLiquid,
		NewBlock:       newBlock,
	}
	event.DefaultBus().Publish(e)
	if e.Cancelled() {
		ctx.Cancel()
		return
	}
	b.inner().HandleLiquidHarden(ctx, hardenedPos, liquidHardened, otherLiquid, newBlock)
}

func (b *eventWorldBridge) HandleSound(ctx *Context, s Sound, pos mgl64.Vec3) {
	e := &SoundPlayEvent{
		World: b.w,
		Sound: s,
		Pos:   pos,
	}
	event.DefaultBus().Publish(e)
	if e.Cancelled() {
		ctx.Cancel()
		return
	}
	b.inner().HandleSound(ctx, s, pos)
}

func (b *eventWorldBridge) HandleFireSpread(ctx *Context, from, to cube.Pos) {
	e := &FireSpreadEvent{
		World: b.w,
		From:  from,
		To:    to,
	}
	event.DefaultBus().Publish(e)
	if e.Cancelled() {
		ctx.Cancel()
		return
	}
	b.inner().HandleFireSpread(ctx, from, to)
}

func (b *eventWorldBridge) HandleBlockBurn(ctx *Context, pos cube.Pos) {
	e := &BlockBurnEvent{
		World: b.w,
		Pos:   pos,
	}
	event.DefaultBus().Publish(e)
	if e.Cancelled() {
		ctx.Cancel()
		return
	}
	b.inner().HandleBlockBurn(ctx, pos)
}

func (b *eventWorldBridge) HandleCropTrample(ctx *Context, pos cube.Pos) {
	e := &CropTrampleEvent{
		World: b.w,
		Pos:   pos,
	}
	event.DefaultBus().Publish(e)
	if e.Cancelled() {
		ctx.Cancel()
		return
	}
	b.inner().HandleCropTrample(ctx, pos)
}

func (b *eventWorldBridge) HandleLeavesDecay(ctx *Context, pos cube.Pos) {
	e := &LeavesDecayEvent{
		World: b.w,
		Pos:   pos,
	}
	event.DefaultBus().Publish(e)
	if e.Cancelled() {
		ctx.Cancel()
		return
	}
	b.inner().HandleLeavesDecay(ctx, pos)
}

func (b *eventWorldBridge) HandlePortalCreate(ctx *Context, portalType Dimension, positions []cube.Pos) {
	e := &PortalCreateEvent{
		World:      b.w,
		PortalType: portalType,
		Positions:  positions,
	}
	event.DefaultBus().Publish(e)
	if e.Cancelled() {
		ctx.Cancel()
		return
	}
	b.inner().HandlePortalCreate(ctx, portalType, positions)
}

func (b *eventWorldBridge) HandlePortalActivate(ctx *Context, portalType Dimension, positions []cube.Pos) {
	e := &PortalActivateEvent{
		World:      b.w,
		PortalType: portalType,
		Positions:  positions,
	}
	event.DefaultBus().Publish(e)
	if e.Cancelled() {
		ctx.Cancel()
		return
	}
	b.inner().HandlePortalActivate(ctx, portalType, positions)
}

func (b *eventWorldBridge) HandleEntitySpawn(tx *Tx, e Entity) {
	ev := &EntitySpawnEvent{
		Tx:     tx,
		World:  b.w,
		Entity: e,
	}
	event.DefaultBus().Publish(ev)
	b.inner().HandleEntitySpawn(tx, e)
}

func (b *eventWorldBridge) HandleEntityDespawn(tx *Tx, e Entity) {
	ev := &EntityDespawnEvent{
		Tx:     tx,
		World:  b.w,
		Entity: e,
	}
	event.DefaultBus().Publish(ev)
	b.inner().HandleEntityDespawn(tx, e)
}

func (b *eventWorldBridge) HandleExplosion(ctx *Context, src ExplosionSource, entities *[]Entity, blocks *[]cube.Pos, itemDropChance *float64, spawnFire *bool) {
	e := &ExplosionEvent{
		World:          b.w,
		Source:         src,
		Entities:       entities,
		Blocks:         blocks,
		ItemDropChance: itemDropChance,
		SpawnFire:      spawnFire,
	}
	event.DefaultBus().Publish(e)
	if e.Cancelled() {
		ctx.Cancel()
		return
	}
	b.inner().HandleExplosion(ctx, src, entities, blocks, itemDropChance, spawnFire)
}

func (b *eventWorldBridge) HandleRedstoneUpdate(ctx *Context, update RedstoneUpdate) {
	e := &RedstoneUpdateEvent{
		World:  b.w,
		Update: update,
	}
	event.DefaultBus().Publish(e)
	if e.Cancelled() {
		ctx.Cancel()
		return
	}
	b.inner().HandleRedstoneUpdate(ctx, update)
}

func (b *eventWorldBridge) HandleClose(tx *Tx) {
	e := &WorldCloseEvent{
		Tx:    tx,
		World: b.w,
	}
	event.DefaultBus().Publish(e)
	b.inner().HandleClose(tx)
}
