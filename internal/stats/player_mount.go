package stats

// Mount is the strider or wyvern a player rides (Mobius PlayerStat with
// isMounted): its level's speed_on_ride replaces the class template's base
// speeds, its NPC's collision replaces the player's, and the run and walk
// speeds are halved for each penalty.
type Mount struct {
	// OnRide is set when the mount's level has speed_on_ride data (Mobius
	// PetDataTable.getPetLevelData not null); without it the template's
	// base speeds stay.
	OnRide bool
	// The base speeds of speed_on_ride.
	Walk, Run, SlowSwim, FastSwim float64
	// Penalties counts the halvings of the run and walk speeds: one when
	// the mount is 10 or more levels above the player, one when it is
	// hungry.
	Penalties int
	// CollisionRadius and CollisionHeight are the mount NPC's collision.
	CollisionRadius, CollisionHeight float64
}

// SetMount sets the ride, nil for none.
func (p *Player) SetMount(m *Mount) {
	if m == nil && p.mount == nil {
		return
	}
	if m != nil && p.mount != nil && *m == *p.mount {
		return
	}
	if m != nil {
		cp := *m
		m = &cp
	}
	p.mount = m
	p.changed()
}

// Mount returns a copy of the ride, nil for none.
func (p *Player) Mount() *Mount {
	if p.mount == nil {
		return nil
	}
	cp := *p.mount
	return &cp
}

// ride returns the ride whose speeds replace the template's, or nil.
func (p *Player) ride() *Mount {
	if p.mount == nil || !p.mount.OnRide {
		return nil
	}
	return p.mount
}

// baseRun is getBaseMoveSpeed(RUN).
func (p *Player) baseRun() float64 {
	if m := p.ride(); m != nil {
		return m.Run
	}
	return p.tmpl.RunSpeed
}

// baseWalk is getBaseMoveSpeed(WALK).
func (p *Player) baseWalk() float64 {
	if m := p.ride(); m != nil {
		return m.Walk
	}
	return p.tmpl.WalkSpeed
}

// baseSwimRun is getBaseMoveSpeed(FAST_SWIM).
func (p *Player) baseSwimRun() float64 {
	if m := p.ride(); m != nil {
		return m.FastSwim
	}
	return p.tmpl.swimRunSpeed()
}

// baseSwimWalk is getBaseMoveSpeed(SLOW_SWIM).
func (p *Player) baseSwimWalk() float64 {
	if m := p.ride(); m != nil {
		return m.SlowSwim
	}
	return p.tmpl.swimWalkSpeed()
}

// mountPenalty halves v once per mount penalty (PlayerStat.getRunSpeed and
// getWalkSpeed after the cap).
func (p *Player) mountPenalty(v float64) float64 {
	if p.mount == nil {
		return v
	}
	for range p.mount.Penalties {
		v /= 2
	}
	return v
}
