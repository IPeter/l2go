package game

import (
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/stats"
)

// A player's status (Mobius PlayerStatus): CurHP, CurMP and CurCP with the
// setters below, the dead flag and the regeneration task. A player
// regenerates while it is in the game; the task starts when it enters the
// world with a value below its maximum (restoreStatus) or when a value drops
// later, and stops when all are full, at death and when the player leaves.
// Its own StatusUpdate (HP, MP and CP) goes out once at the end of the tick
// however often the values change (BroadcastStatus). Mobius sends players
// targeting a player no HP bar updates.

// IsDead reports whether the player is dead (Creature).
func (p *Player) IsDead() bool { return p.dead }

// CurrentHP returns the current HP (Creature).
func (p *Player) CurrentHP() float64 { return p.CurHP }

// MaxHP returns the maximum HP (Creature).
func (p *Player) MaxHP() int32 { return int32(p.stats.MaxHP()) } //nolint:gosec // HP fits

// Regenerating reports whether the regeneration task runs.
func (p *Player) Regenerating() bool { return p.regen.timer != 0 }

// SetCurrentHP sets the current HP: at most the maximum, which stops the HP
// regeneration, below it starts it (Mobius setCurrentHp). When the int value
// changed the player gets its StatusUpdate at the end of the tick; when the
// HP crossed 30 % of the maximum, its UserInfo. A dead player keeps its HP.
// Loop-only.
func (p *Player) SetCurrentHP(hp float64) { p.setCurrentHP(hp, true) }

// SetCurrentMP sets the current MP like SetCurrentHP (Mobius setCurrentMp).
// Loop-only.
func (p *Player) SetCurrentMP(mp float64) { p.setCurrentMP(mp, true) }

// SetCurrentCP sets the current CP, at least 0, like SetCurrentHP (Mobius
// setCurrentCp). Loop-only.
func (p *Player) SetCurrentCP(cp float64) { p.setCurrentCP(cp, true) }

// SetCurrentHPMP sets HP and MP and sends one StatusUpdate when either
// changed (Mobius setCurrentHpMp). Loop-only.
func (p *Player) SetCurrentHPMP(hp, mp float64) {
	hpChanged := p.setCurrentHP(hp, false)
	mpChanged := p.setCurrentMP(mp, false)
	if hpChanged || mpChanged {
		p.BroadcastStatus()
	}
}

// ReduceMP takes value MP away, down to 0 (Mobius reduceMp). Loop-only.
func (p *Player) ReduceMP(value float64) { p.setCurrentMP(max(p.CurMP-value, 0), true) }

// ReduceCP takes value CP away, down to 0 (Mobius PlayerStatus.reduceCp).
// Loop-only.
func (p *Player) ReduceCP(value int) {
	cp := 0.0
	if p.CurCP > float64(value) {
		cp = p.CurCP - float64(value)
	}
	p.setCurrentCP(cp, true)
}

// ReduceHP takes damage, as Mobius PlayerStatus.reduceHp does for a hit
// (neither damage over time nor HP consumption): nothing on a dead player; a
// sitting player stands up; CP absorb the damage of a player attacker first;
// an attacker other than the player itself makes the client show "$s1 hit
// you for $s2 damage." with the int damage; HP go down to 0; under 0.5 HP the
// player dies (Server.Kill). Summons, duels, the Olympiad, Arcane Shield and
// damage transfer do not exist yet. Loop-only.
func (p *Player) ReduceHP(value float64, attacker Creature) {
	if p.dead {
		return
	}
	if p.Sitting {
		p.srv.standUp(p)
	}
	amount := value
	full := javaInt32(amount)
	if attacker != nil && attacker != Creature(p) {
		if _, playable := attacker.(*Player); playable {
			if p.CurCP >= amount {
				p.setCurrentCP(p.CurCP-amount, true)
				amount = 0
			} else {
				amount -= p.CurCP
				p.setCurrentCP(0, false)
			}
		}
		if full > 0 {
			p.SendSystemMessage(sysMsgHitYouForDamage,
				serverpackets.TextParam(attacker.Name()), serverpackets.NumberParam(full))
		}
	}
	if amount > 0 {
		p.setCurrentHP(max(p.CurHP-amount, 0), true)
	}
	if p.CurHP < 0.5 {
		p.srv.Kill(p, attacker)
	}
}

// Revive clears the dead flag (the first step of Mobius doRevive). It sends
// nothing and leaves the values as they are: set them next, which restarts
// regeneration. Loop-only.
func (p *Player) Revive() { p.dead = false }

// StartRegen starts the regeneration task unless it runs or the player is
// dead or not in the game (Mobius startHpMpRegeneration). Loop-only.
func (p *Player) StartRegen() {
	if !p.dead && p.state == stateInGame {
		p.regen.start(p.srv.loop, p.regenerate)
	}
}

// StopRegen stops the regeneration task (Mobius stopHpMpRegeneration).
// Loop-only.
func (p *Player) StopRegen() {
	if p.srv != nil {
		p.regen.stop(p.srv.loop)
	}
}

// BroadcastStatus sends the player its StatusUpdate with its HP, MP and CP at
// the end of the tick, once however often it is called (Mobius
// Player.broadcastStatusUpdate, coalesced for 50 ms there). Loop-only.
func (p *Player) BroadcastStatus() {
	if p.statusQueued || p.srv == nil {
		return
	}
	p.statusQueued = true
	p.srv.status.queue = append(p.srv.status.queue, p)
}

// setRegen records whether a value is below its maximum and starts or stops
// the task to match.
func (p *Player) setRegen(flag uint8, below bool) {
	if below {
		p.regen.flags |= flag
		p.StartRegen()
		return
	}
	p.regen.flags &^= flag
	if p.regen.flags == 0 {
		p.StopRegen()
	}
}

// setCurrentHP is Mobius CreatureStatus.setCurrentHp for a player. It
// reports whether the HP changed, comparing the old value cut to an int with
// the new one, as Mobius does.
func (p *Player) setCurrentHP(hp float64, broadcast bool) bool {
	old := float64(javaInt32(p.CurHP))
	maxHP := float64(p.stats.MaxHP())
	thirty := maxHP * 0.3
	userInfo := (old < thirty && hp > thirty) || (old > thirty && hp < thirty)
	if p.dead {
		return false
	}
	hp = max(hp, 0)
	if hp >= maxHP {
		p.CurHP = maxHP
		p.setRegen(regenHP, false)
	} else {
		p.CurHP = hp
		p.setRegen(regenHP, true)
	}
	changed := old != p.CurHP
	if changed && broadcast {
		p.BroadcastStatus()
	}
	if userInfo {
		p.UpdateUserInfo()
	}
	return changed
}

// setCurrentMP is Mobius setCurrentMp for a player.
func (p *Player) setCurrentMP(mp float64, broadcast bool) bool {
	old := float64(javaInt32(p.CurMP))
	if p.dead {
		return false
	}
	maxMP := float64(p.stats.MaxMP())
	mp = max(mp, 0)
	if mp >= maxMP {
		p.CurMP = maxMP
		p.setRegen(regenMP, false)
	} else {
		p.CurMP = mp
		p.setRegen(regenMP, true)
	}
	changed := old != p.CurMP
	if changed && broadcast {
		p.BroadcastStatus()
	}
	return changed
}

// setCurrentCP is Mobius PlayerStatus.setCurrentCp: the CP are at least 0.
func (p *Player) setCurrentCP(cp float64, broadcast bool) {
	old := float64(javaInt32(p.CurCP))
	if p.dead {
		return
	}
	maxCP := float64(p.stats.MaxCP())
	cp = max(cp, 0)
	if cp >= maxCP {
		p.CurCP = maxCP
		p.setRegen(regenCP, false)
	} else {
		p.CurCP = cp
		p.setRegen(regenCP, true)
	}
	if old != p.CurCP && broadcast {
		p.BroadcastStatus()
	}
}

// regenerate is one tick of the regeneration task (Mobius
// PlayerStatus.doRegeneration): CP, then HP, then MP grow by their regen
// while below their recoverable maximum, and the player gets its
// StatusUpdate. The task goes on while a value is below its maximum, even
// when it cannot recover further.
func (p *Player) regenerate() {
	r := p.srv.combat.PlayerRegen(p.stats, p.activity(), p.srv.regenZone(p))
	if p.CurCP < float64(p.stats.MaxRecoverableCP()) {
		p.setCurrentCP(p.CurCP+r.CP, false)
	}
	if p.CurHP < float64(p.stats.MaxRecoverableHP()) {
		p.setCurrentHP(p.CurHP+r.HP, false)
	}
	if p.CurMP < float64(p.stats.MaxRecoverableMP()) {
		p.setCurrentMP(p.CurMP+r.MP, false)
	}
	p.BroadcastStatus()
}

// activity is what the player does for its regeneration (Mobius
// calcHpRegen): sitting (also while standing up), standing still, walking or
// running.
func (p *Player) activity() stats.Activity {
	switch {
	case p.Sitting:
		return stats.Sitting
	case p.Move == nil:
		return stats.Standing
	case p.Running():
		return stats.Running
	}
	return stats.Walking
}

// restoreStatus is the OnPlayerEnter hook of the status: the loaded values go
// through the setters, which start regeneration when one is below its
// maximum, and a player with less than 0.5 HP is dead (Mobius
// Player.restore). Nothing is sent: the enter-world packets show the values.
func (s *Server) restoreStatus(p *Player) {
	p.setCurrentCP(p.CurCP, false)
	p.setCurrentHP(p.CurHP, false)
	p.setCurrentMP(p.CurMP, false)
	if p.CurHP < 0.5 {
		p.dead = true
		p.StopRegen()
	}
}
