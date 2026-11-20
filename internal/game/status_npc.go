package game

// An NPC's status (Mobius NpcStatus and CreatureStatus): CurHP and CurMP with
// the setters below, the Dead flag and the regeneration task, which runs
// while the HP or MP are below their maximum, and also starts for the
// attackables of a region that becomes active. The players targeting the NPC
// get its HP bar when the HP cross a pixel of the bar (needHpUpdate).

// IsDead reports whether the NPC is dead (Creature).
func (n *Npc) IsDead() bool { return n.Dead }

// CurrentHP returns the current HP (Creature).
func (n *Npc) CurrentHP() float64 { return n.CurHP }

// Regenerating reports whether the regeneration task runs.
func (n *Npc) Regenerating() bool { return n.regen.timer != 0 }

// SetCurrentMP sets the current MP, at most the maximum, starting or
// stopping regeneration (Mobius setCurrentMp). MP changes send the HP bar
// only when the bar needs it anyway (Mobius broadcastStatusUpdate).
// Loop-only.
func (n *Npc) SetCurrentMP(mp float64) { n.setCurrentMP(mp, true) }

// SetCurrentHPMP sets HP and MP and sends the HP bar once when either changed
// (Mobius setCurrentHpMp). Loop-only.
func (n *Npc) SetCurrentHPMP(hp, mp float64) {
	hpChanged := n.setCurrentHP(hp, false)
	mpChanged := n.setCurrentMP(mp, false)
	if hpChanged || mpChanged {
		n.BroadcastStatus()
	}
}

// ReduceHP takes damage (Mobius NpcStatus.reduceHp through
// CreatureStatus.reduceHp): nothing on a dead NPC; the HP go down to 0;
// under 0.5 HP the NPC dies (Server.Kill). The attacker is the killer; hate,
// the attack-by list and overhit are the AI's. Loop-only.
func (n *Npc) ReduceHP(value float64, attacker Creature) {
	if n.Dead {
		return
	}
	if attacker != nil && n.srv != nil {
		// Mobius Attackable.reduceCurrentHp and NpcStatus.reduceHp: the AI
		// hears of the attack and hates the attacker, which joins the
		// attack-by list (ai.go, hate.go).
		n.srv.npcDamaged(n, attacker, value)
	}
	if n.Dead {
		return
	}
	if value > 0 {
		n.setCurrentHP(max(n.CurHP-value, 0), true)
	}
	if n.CurHP < 0.5 && n.srv != nil {
		n.srv.Kill(n, attacker)
	}
}

// ReduceMP takes value MP away, down to 0 (Mobius reduceMp). Loop-only.
func (n *Npc) ReduceMP(value float64) { n.setCurrentMP(max(n.CurMP-value, 0), true) }

// Revive clears the dead flag. It sends nothing and leaves the values as
// they are. Loop-only.
func (n *Npc) Revive() { n.Dead = false }

// StartRegen starts the regeneration task unless it runs or the NPC is dead
// (Mobius startHpMpRegeneration). A task started at full HP and MP stops at
// its first tick. Loop-only.
func (n *Npc) StartRegen() {
	if !n.Dead && n.srv != nil {
		n.regen.start(n.srv.loop, n.regenerate)
	}
}

// StopRegen stops the regeneration task (Mobius stopHpMpRegeneration).
// Loop-only.
func (n *Npc) StopRegen() {
	if n.srv != nil {
		n.regen.stop(n.srv.loop)
	}
}

// BroadcastStatus sends the HP bar to the players targeting the NPC when it
// moved by a pixel (Mobius Creature.broadcastStatusUpdate and needHpUpdate).
// Loop-only.
func (n *Npc) BroadcastStatus() {
	if n.srv == nil || len(n.srv.npc.watchers[n.ID()]) == 0 || !n.hpBar.need(n.CurHP, float64(n.MaxHP())) {
		return
	}
	n.srv.sendHPBar(n)
}

// setRegen records whether a value is below its maximum and starts or stops
// the task to match.
func (n *Npc) setRegen(flag uint8, below bool) {
	if below {
		n.regen.flags |= flag
		n.StartRegen()
		return
	}
	n.regen.flags &^= flag
	if n.regen.flags == 0 {
		n.StopRegen()
	}
}

// setCurrentHP is Mobius CreatureStatus.setCurrentHp: it reports whether the
// HP changed, comparing the old value cut to an int with the new one.
func (n *Npc) setCurrentHP(hp float64, broadcast bool) bool {
	old := float64(javaInt32(n.CurHP))
	if n.Dead {
		return false
	}
	maxHP := float64(n.MaxHP())
	hp = max(hp, 0)
	if hp >= maxHP {
		n.CurHP = maxHP
		n.setRegen(regenHP, false)
	} else {
		n.CurHP = hp
		n.setRegen(regenHP, true)
	}
	changed := old != n.CurHP
	if changed && broadcast {
		n.BroadcastStatus()
	}
	return changed
}

// setCurrentMP is Mobius CreatureStatus.setCurrentMp.
func (n *Npc) setCurrentMP(mp float64, broadcast bool) bool {
	old := float64(javaInt32(n.CurMP))
	if n.Dead {
		return false
	}
	maxMP := float64(n.MaxMP())
	mp = max(mp, 0)
	if mp >= maxMP {
		n.CurMP = maxMP
		n.setRegen(regenMP, false)
	} else {
		n.CurMP = mp
		n.setRegen(regenMP, true)
	}
	changed := old != n.CurMP
	if changed && broadcast {
		n.BroadcastStatus()
	}
	return changed
}

// regenerate is one tick of the regeneration task (Mobius
// CreatureStatus.doRegeneration): while the NPC is alive and its HP or MP are
// below their recoverable maximum, both grow by their regen; otherwise the
// task stops.
func (n *Npc) regenerate() {
	if n.Dead || (n.CurHP >= float64(n.Stats.MaxRecoverableHP()) && n.CurMP >= float64(n.Stats.MaxRecoverableMP())) {
		n.StopRegen()
		return
	}
	r := n.srv.combat.NpcRegen(n.Stats, n.class != nil && n.class.raid)
	n.SetCurrentHPMP(n.CurHP+r.HP, n.CurMP+r.MP)
}
