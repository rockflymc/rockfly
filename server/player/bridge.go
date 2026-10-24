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

// eventBridge wires all player actions through event.DefaultBus() while maintaining
// backward compatibility with Player.Handle() custom handlers.
type eventBridge struct {
	p *Player
}

var _ Handler = (*eventBridge)(nil)

func (b *eventBridge) HandleChat(ctx *Context, message *string) {
	e := &PlayerChatEvent{
		Player:  b.p,
		Message: *message,
	}
	event.DefaultBus().Publish(e)
	if e.Cancelled() {
		ctx.Cancel()
		return
	}
	*message = e.Message
	if b.p.h != nil {
		b.p.h.HandleChat(ctx, message)
	}
}

func (b *eventBridge) HandleCommandExecution(ctx *Context, command cmd.Command, args []string) {
	e := &PlayerCommandEvent{
		Player:    b.p,
		Command:   command,
		Arguments: args,
	}
	event.DefaultBus().Publish(e)
	if e.Cancelled() {
		ctx.Cancel()
		return
	}
	if b.p.h != nil {
		b.p.h.HandleCommandExecution(ctx, command, args)
	}
}

func (b *eventBridge) HandleMove(ctx *Context, newPos mgl64.Vec3, newRot cube.Rotation) {
	e := &PlayerMoveEvent{
		Player:  b.p,
		From:    b.p.Position(),
		To:      newPos,
		FromRot: b.p.Rotation(),
		ToRot:   newRot,
	}
	event.DefaultBus().Publish(e)
	if e.Cancelled() {
		ctx.Cancel()
		return
	}
	if b.p.h != nil {
		b.p.h.HandleMove(ctx, newPos, newRot)
	}
}

func (b *eventBridge) HandleTeleport(ctx *Context, pos mgl64.Vec3) {
	e := &PlayerTeleportEvent{
		Player: b.p,
		From:   b.p.Position(),
		To:     pos,
	}
	event.DefaultBus().Publish(e)
	if e.Cancelled() {
		ctx.Cancel()
		return
	}
	if b.p.h != nil {
		b.p.h.HandleTeleport(ctx, pos)
	}
}

func (b *eventBridge) HandleJump(p *Player) {
	e := &PlayerJumpEvent{Player: p}
	event.DefaultBus().Publish(e)
	if b.p.h != nil {
		b.p.h.HandleJump(p)
	}
}

func (b *eventBridge) HandleChangeWorld(p *Player, before, after *world.World) {
	e := &PlayerChangeWorldEvent{
		Player: p,
		Before: before,
		After:  after,
	}
	event.DefaultBus().Publish(e)
	if b.p.h != nil {
		b.p.h.HandleChangeWorld(p, before, after)
	}
}

func (b *eventBridge) HandleToggleSprint(ctx *Context, after bool) {
	e := &PlayerToggleSprintEvent{
		Player: b.p,
		After:  after,
	}
	event.DefaultBus().Publish(e)
	if e.Cancelled() {
		ctx.Cancel()
		return
	}
	if b.p.h != nil {
		b.p.h.HandleToggleSprint(ctx, after)
	}
}

func (b *eventBridge) HandleToggleSneak(ctx *Context, after bool) {
	e := &PlayerToggleSneakEvent{
		Player: b.p,
		After:  after,
	}
	event.DefaultBus().Publish(e)
	if e.Cancelled() {
		ctx.Cancel()
		return
	}
	if b.p.h != nil {
		b.p.h.HandleToggleSneak(ctx, after)
	}
}

func (b *eventBridge) HandleFoodLoss(ctx *Context, from int, to *int) {
	e := &PlayerFoodLossEvent{
		Player: b.p,
		From:   from,
		To:     to,
	}
	event.DefaultBus().Publish(e)
	if e.Cancelled() {
		ctx.Cancel()
		return
	}
	if b.p.h != nil {
		b.p.h.HandleFoodLoss(ctx, from, to)
	}
}

func (b *eventBridge) HandleHeal(ctx *Context, health *float64, src world.HealingSource) {
	e := &PlayerHealEvent{
		Player: b.p,
		Health: health,
		Source: src,
	}
	event.DefaultBus().Publish(e)
	if e.Cancelled() {
		ctx.Cancel()
		return
	}
	if b.p.h != nil {
		b.p.h.HandleHeal(ctx, health, src)
	}
}

func (b *eventBridge) HandleHurt(ctx *Context, damage *float64, immune bool, attackImmunity *time.Duration, src world.DamageSource) {
	e := &PlayerDamageEvent{
		Player:         b.p,
		Damage:         damage,
		Immune:         immune,
		AttackImmunity: attackImmunity,
		Source:         src,
	}
	event.DefaultBus().Publish(e)
	if e.Cancelled() {
		ctx.Cancel()
		return
	}
	if b.p.h != nil {
		b.p.h.HandleHurt(ctx, damage, immune, attackImmunity, src)
	}
}

func (b *eventBridge) HandleSetOnFire(ctx *Context, duration *time.Duration) {
	e := &PlayerSetOnFireEvent{
		Player:   b.p,
		Duration: duration,
	}
	event.DefaultBus().Publish(e)
	if e.Cancelled() {
		ctx.Cancel()
		return
	}
	if b.p.h != nil {
		b.p.h.HandleSetOnFire(ctx, duration)
	}
}

func (b *eventBridge) HandleDeath(p *Player, src world.DamageSource, keepInv *bool) {
	e := &PlayerDeathEvent{
		Player:        p,
		Source:        src,
		KeepInventory: keepInv,
	}
	event.DefaultBus().Publish(e)
	if b.p.h != nil {
		b.p.h.HandleDeath(p, src, keepInv)
	}
}

func (b *eventBridge) HandleRespawn(p *Player, pos *mgl64.Vec3, w **world.World) {
	e := &PlayerRespawnEvent{
		Player: p,
		Pos:    pos,
		World:  w,
	}
	event.DefaultBus().Publish(e)
	if b.p.h != nil {
		b.p.h.HandleRespawn(p, pos, w)
	}
}

func (b *eventBridge) HandleSkinChange(ctx *Context, skin *skin.Skin) {
	e := &PlayerSkinChangeEvent{
		Player: b.p,
		Skin:   skin,
	}
	event.DefaultBus().Publish(e)
	if e.Cancelled() {
		ctx.Cancel()
		return
	}
	if b.p.h != nil {
		b.p.h.HandleSkinChange(ctx, skin)
	}
}

func (b *eventBridge) HandleFireExtinguish(ctx *Context, pos cube.Pos) {
	e := &PlayerFireExtinguishEvent{
		Player: b.p,
		Pos:    pos,
	}
	event.DefaultBus().Publish(e)
	if e.Cancelled() {
		ctx.Cancel()
		return
	}
	if b.p.h != nil {
		b.p.h.HandleFireExtinguish(ctx, pos)
	}
}

func (b *eventBridge) HandleStartBreak(ctx *Context, pos cube.Pos) {
	e := &PlayerStartBreakEvent{
		Player: b.p,
		Pos:    pos,
		Block:  b.p.tx.Block(pos),
	}
	event.DefaultBus().Publish(e)
	if e.Cancelled() {
		ctx.Cancel()
		return
	}
	if b.p.h != nil {
		b.p.h.HandleStartBreak(ctx, pos)
	}
}

func (b *eventBridge) HandleBlockBreak(ctx *Context, pos cube.Pos, drops *[]item.Stack, xp *int) {
	e := &PlayerBlockBreakEvent{
		Player: b.p,
		Pos:    pos,
		Block:  b.p.tx.Block(pos),
		Drops:  drops,
		XP:     xp,
	}
	event.DefaultBus().Publish(e)
	if e.Cancelled() {
		ctx.Cancel()
		return
	}
	if b.p.h != nil {
		b.p.h.HandleBlockBreak(ctx, pos, drops, xp)
	}
}

func (b *eventBridge) HandleBlockPlace(ctx *Context, pos cube.Pos, bl world.Block) {
	e := &PlayerBlockPlaceEvent{
		Player: b.p,
		Pos:    pos,
		Block:  bl,
	}
	event.DefaultBus().Publish(e)
	if e.Cancelled() {
		ctx.Cancel()
		return
	}
	if b.p.h != nil {
		b.p.h.HandleBlockPlace(ctx, pos, bl)
	}
}

func (b *eventBridge) HandleBlockPick(ctx *Context, pos cube.Pos, bl world.Block) {
	e := &PlayerBlockPickEvent{
		Player: b.p,
		Pos:    pos,
		Block:  bl,
	}
	event.DefaultBus().Publish(e)
	if e.Cancelled() {
		ctx.Cancel()
		return
	}
	if b.p.h != nil {
		b.p.h.HandleBlockPick(ctx, pos, bl)
	}
}

func (b *eventBridge) HandleItemUse(ctx *Context) {
	held, _ := b.p.HeldItems()
	e := &PlayerItemUseEvent{
		Player: b.p,
		Item:   held,
	}
	event.DefaultBus().Publish(e)
	if e.Cancelled() {
		ctx.Cancel()
		return
	}
	if b.p.h != nil {
		b.p.h.HandleItemUse(ctx)
	}
}

func (b *eventBridge) HandleItemUseOnBlock(ctx *Context, pos cube.Pos, face cube.Face, clickPos mgl64.Vec3) {
	held, _ := b.p.HeldItems()
	e := &PlayerItemUseOnBlockEvent{
		Player:   b.p,
		Item:     held,
		Pos:      pos,
		Face:     face,
		ClickPos: clickPos,
	}
	event.DefaultBus().Publish(e)
	if e.Cancelled() {
		ctx.Cancel()
		return
	}
	if b.p.h != nil {
		b.p.h.HandleItemUseOnBlock(ctx, pos, face, clickPos)
	}
}

func (b *eventBridge) HandleItemUseOnEntity(ctx *Context, target world.Entity) {
	held, _ := b.p.HeldItems()
	e := &PlayerItemUseOnEntityEvent{
		Player: b.p,
		Item:   held,
		Entity: target,
	}
	event.DefaultBus().Publish(e)
	if e.Cancelled() {
		ctx.Cancel()
		return
	}
	if b.p.h != nil {
		b.p.h.HandleItemUseOnEntity(ctx, target)
	}
}

func (b *eventBridge) HandleItemRelease(ctx *Context, it item.Stack, dur time.Duration) {
	e := &PlayerItemReleaseEvent{
		Player:   b.p,
		Item:     it,
		Duration: dur,
	}
	event.DefaultBus().Publish(e)
	if e.Cancelled() {
		ctx.Cancel()
		return
	}
	if b.p.h != nil {
		b.p.h.HandleItemRelease(ctx, it, dur)
	}
}

func (b *eventBridge) HandleItemConsume(ctx *Context, it item.Stack) {
	e := &PlayerItemConsumeEvent{
		Player: b.p,
		Item:   it,
	}
	event.DefaultBus().Publish(e)
	if e.Cancelled() {
		ctx.Cancel()
		return
	}
	if b.p.h != nil {
		b.p.h.HandleItemConsume(ctx, it)
	}
}

func (b *eventBridge) HandleAttackEntity(ctx *Context, e world.Entity, force, height *float64, critical *bool) {
	ev := &PlayerAttackEntityEvent{
		Player:   b.p,
		Entity:   e,
		Force:    force,
		Height:   height,
		Critical: critical,
	}
	event.DefaultBus().Publish(ev)
	if ev.Cancelled() {
		ctx.Cancel()
		return
	}
	if b.p.h != nil {
		b.p.h.HandleAttackEntity(ctx, e, force, height, critical)
	}
}

func (b *eventBridge) HandleExperienceGain(ctx *Context, amount *int) {
	e := &PlayerExperienceGainEvent{
		Player: b.p,
		Amount: amount,
	}
	event.DefaultBus().Publish(e)
	if e.Cancelled() {
		ctx.Cancel()
		return
	}
	if b.p.h != nil {
		b.p.h.HandleExperienceGain(ctx, amount)
	}
}

func (b *eventBridge) HandlePunchAir(ctx *Context) {
	e := &PlayerPunchAirEvent{Player: b.p}
	event.DefaultBus().Publish(e)
	if e.Cancelled() {
		ctx.Cancel()
		return
	}
	if b.p.h != nil {
		b.p.h.HandlePunchAir(ctx)
	}
}

func (b *eventBridge) HandleSignEdit(ctx *Context, pos cube.Pos, frontSide bool, oldText, newText string) {
	e := &PlayerSignEditEvent{
		Player:    b.p,
		Pos:       pos,
		FrontSide: frontSide,
		OldText:   oldText,
		NewText:   newText,
	}
	event.DefaultBus().Publish(e)
	if e.Cancelled() {
		ctx.Cancel()
		return
	}
	if b.p.h != nil {
		b.p.h.HandleSignEdit(ctx, pos, frontSide, oldText, newText)
	}
}

func (b *eventBridge) HandleSleep(ctx *Context, sendReminder *bool) {
	e := &PlayerSleepEvent{
		Player:       b.p,
		SendReminder: sendReminder,
	}
	event.DefaultBus().Publish(e)
	if e.Cancelled() {
		ctx.Cancel()
		return
	}
	if b.p.h != nil {
		b.p.h.HandleSleep(ctx, sendReminder)
	}
}

func (b *eventBridge) HandleLecternPageTurn(ctx *Context, pos cube.Pos, oldPage int, newPage *int) {
	e := &PlayerLecternPageTurnEvent{
		Player:  b.p,
		Pos:     pos,
		OldPage: oldPage,
		NewPage: newPage,
	}
	event.DefaultBus().Publish(e)
	if e.Cancelled() {
		ctx.Cancel()
		return
	}
	if b.p.h != nil {
		b.p.h.HandleLecternPageTurn(ctx, pos, oldPage, newPage)
	}
}

func (b *eventBridge) HandleItemDamage(ctx *Context, i item.Stack, damage *int) {
	e := &PlayerItemDamageEvent{
		Player: b.p,
		Item:   i,
		Damage: damage,
	}
	event.DefaultBus().Publish(e)
	if e.Cancelled() {
		ctx.Cancel()
		return
	}
	if b.p.h != nil {
		b.p.h.HandleItemDamage(ctx, i, damage)
	}
}

func (b *eventBridge) HandleItemPickup(ctx *Context, i *item.Stack) {
	e := &PlayerItemPickupEvent{
		Player: b.p,
		Item:   i,
	}
	event.DefaultBus().Publish(e)
	if e.Cancelled() {
		ctx.Cancel()
		return
	}
	if b.p.h != nil {
		b.p.h.HandleItemPickup(ctx, i)
	}
}

func (b *eventBridge) HandleHeldSlotChange(ctx *Context, from, to int) {
	e := &PlayerHeldSlotChangeEvent{
		Player: b.p,
		From:   from,
		To:     to,
	}
	event.DefaultBus().Publish(e)
	if e.Cancelled() {
		ctx.Cancel()
		return
	}
	if b.p.h != nil {
		b.p.h.HandleHeldSlotChange(ctx, from, to)
	}
}

func (b *eventBridge) HandleItemDrop(ctx *Context, s item.Stack) {
	e := &PlayerItemDropEvent{
		Player: b.p,
		Item:   s,
	}
	event.DefaultBus().Publish(e)
	if e.Cancelled() {
		ctx.Cancel()
		return
	}
	if b.p.h != nil {
		b.p.h.HandleItemDrop(ctx, s)
	}
}

func (b *eventBridge) HandleTransfer(ctx *Context, addr *net.UDPAddr) {
	e := &PlayerTransferEvent{
		Player:  b.p,
		Address: addr,
	}
	event.DefaultBus().Publish(e)
	if e.Cancelled() {
		ctx.Cancel()
		return
	}
	if b.p.h != nil {
		b.p.h.HandleTransfer(ctx, addr)
	}
}

func (b *eventBridge) HandleQuit(p *Player) {
	e := &PlayerQuitEvent{Player: p}
	event.DefaultBus().Publish(e)
	if b.p.h != nil {
		b.p.h.HandleQuit(p)
	}
}

func (b *eventBridge) HandleDiagnostics(p *Player, d session.Diagnostics) {
	e := &PlayerDiagnosticsEvent{
		Player:      p,
		Diagnostics: d,
	}
	event.DefaultBus().Publish(e)
	if b.p.h != nil {
		b.p.h.HandleDiagnostics(p, d)
	}
}
