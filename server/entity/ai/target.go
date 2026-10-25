package ai

// TargetGoal is a specialized Goal used by TargetSelectors to acquire combat targets.
type TargetGoal interface {
	Goal
}

// NearestAttackableTargetGoal selects the nearest eligible enemy within detection radius.
type NearestAttackableTargetGoal struct {
	BaseGoal
	mob         Mob
	holder      TargetHolder
	radius      float64
	findNearest func(radius float64) Mob
}

// NewNearestAttackableTargetGoal creates a NearestAttackableTargetGoal.
func NewNearestAttackableTargetGoal(mob Mob, holder TargetHolder, radius float64, findNearest func(radius float64) Mob) *NearestAttackableTargetGoal {
	return &NearestAttackableTargetGoal{
		BaseGoal:    NewBaseGoal(0), // Target goals do not conflict with physical movement/look
		mob:         mob,
		holder:      holder,
		radius:      radius,
		findNearest: findNearest,
	}
}

func (g *NearestAttackableTargetGoal) CanUse() bool {
	if g.mob == nil || g.mob.Dead() || g.holder == nil || g.findNearest == nil {
		return false
	}
	// If already targeting a living enemy, no need to switch immediately
	curr := g.holder.Target()
	if curr != nil && !curr.Dead() {
		return false
	}
	target := g.findNearest(g.radius)
	return target != nil && !target.Dead()
}

func (g *NearestAttackableTargetGoal) CanContinueToUse() bool {
	if g.mob == nil || g.mob.Dead() || g.holder == nil {
		return false
	}
	target := g.holder.Target()
	if target == nil || target.Dead() {
		return false
	}
	dist := target.Position().Sub(g.mob.Position()).Len()
	return dist <= g.radius*1.5
}

func (g *NearestAttackableTargetGoal) Start() {
	if g.holder == nil || g.findNearest == nil {
		return
	}
	target := g.findNearest(g.radius)
	if target != nil {
		g.holder.SetTarget(target)
	}
}

func (g *NearestAttackableTargetGoal) Stop() {
	if g.holder != nil {
		g.holder.SetTarget(nil)
	}
}

func (g *NearestAttackableTargetGoal) Tick() {}

// HurtByTargetGoal retaliates against the last attacker that inflicted damage.
type HurtByTargetGoal struct {
	BaseGoal
	mob          Mob
	holder       TargetHolder
	lastAttacker func() Mob
}

// NewHurtByTargetGoal creates a HurtByTargetGoal.
func NewHurtByTargetGoal(mob Mob, holder TargetHolder, lastAttacker func() Mob) *HurtByTargetGoal {
	return &HurtByTargetGoal{
		BaseGoal:     NewBaseGoal(0),
		mob:          mob,
		holder:       holder,
		lastAttacker: lastAttacker,
	}
}

func (g *HurtByTargetGoal) CanUse() bool {
	if g.mob == nil || g.mob.Dead() || g.holder == nil || g.lastAttacker == nil {
		return false
	}
	attacker := g.lastAttacker()
	return attacker != nil && !attacker.Dead()
}

func (g *HurtByTargetGoal) CanContinueToUse() bool {
	if g.mob == nil || g.mob.Dead() || g.holder == nil {
		return false
	}
	target := g.holder.Target()
	return target != nil && !target.Dead()
}

func (g *HurtByTargetGoal) Start() {
	if g.holder == nil || g.lastAttacker == nil {
		return
	}
	attacker := g.lastAttacker()
	if attacker != nil {
		g.holder.SetTarget(attacker)
	}
}

func (g *HurtByTargetGoal) Stop() {
	if g.holder != nil {
		g.holder.SetTarget(nil)
	}
}

func (g *HurtByTargetGoal) Tick() {}
