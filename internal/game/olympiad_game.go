package game

import (
	"context"
	"strconv"
	"time"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/olympiad"
	"github.com/IPeter/l2go/internal/skills"
	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/world"
)

// A match (Mobius OlympiadGame and OlympiadGameTask) as a chain of loop
// timers instead of a thread:
//
//  1. "moved to the Olympiad Stadium in $s1 second(s)" from WaitTime;
//  2. the requirements again (checkDefaulted): a noble that left, died,
//     switched to a subclass, holds a cursed weapon or carries too much
//     forfeits;
//  3. both go 900 units east and west of the stadium's centre
//     (ExOlympiadMode 2), lose their buffs, clan and hero skills, their
//     party, shots and summoned pets, and heal fully (removals);
//  4. 5 s later "The Grand Olympiad match will start in $s1 second(s)."
//     from 60; then "The match has started, fight!";
//  5. every 10 s, up to Battle, the fight is checked for a dead noble;
//  6. the result (olympiad.Config.Validate): points, the reward, the saved
//     fight; then "moved back to town in $s1 second(s)" from 40;
//  7. both come back where they were, revived and healed
//     (PlayersStatusBack, portPlayersBack), and the stadium is free.

// olyGame is a running match.
type olyGame struct {
	s       *Server
	stadium int
	typ     olympiad.CompetitionType
	ids     [2]int32
	names   [2]string
	classes [2]int // the base classes, for the saved fight
	players [2]*Player
	// disconnected: left the game after reaching the stadium; defaulted:
	// did not meet the requirements at the start.
	disconnected, defaulted [2]bool
	aborted                 bool
	// ported: the nobles went to the stadium (Mobius _gamestarted).
	ported    bool
	origin    [2]world.Position
	startTime time.Time
	damage    [2]int
	started   bool
	timer     world.TimerID
}

// newOlympiadGame makes the game of a pairing (Mobius new OlympiadGame).
func (s *Server) newOlympiadGame(pr olympiad.Pairing) {
	f := &s.olympiad
	g := &olyGame{s: s, stadium: pr.Stadium, typ: pr.Type, ids: pr.Players}
	for i, id := range pr.Players {
		p, ok := s.Player(world.ObjectID(id))
		if !ok {
			g.aborted = true
			continue
		}
		g.players[i] = p
		g.names[i] = p.Name()
		g.classes[i] = p.BaseClassID()
		s.olyState(p).gameID = pr.Stadium
	}
	f.games[pr.Stadium] = g
}

// otherSide returns the index of the other noble.
func otherSide(i int) int { return 1 - i }

// broadcast is Mobius broadcastMessage: the two nobles, and with toAll the
// stadium's spectators.
func (g *olyGame) broadcast(m olympiad.Message, toAll bool, params ...serverpackets.SystemMessageParam) {
	body := serverpackets.NewSystemMessagePacket(serverpackets.SystemMessage{ID: serverpackets.SystemMessageID(m), Params: params})
	for _, p := range g.players {
		if p != nil {
			p.Send(body)
		}
	}
	if toAll {
		for _, spec := range g.s.olympiad.spectators[g.stadium] {
			spec.Send(body)
		}
	}
}

// countdown sends the message of each tick with its seconds and runs done
// after the last wait; each, if set, runs at every tick.
func (g *olyGame) countdown(ticks []olympiad.Tick, m olympiad.Message, toAll bool, each func(sec int), done func()) {
	var step func(i int)
	step = func(i int) {
		if i == len(ticks) {
			done()
			return
		}
		t := ticks[i]
		g.broadcast(m, toAll, serverpackets.NumberParam(int32(t.Seconds))) //nolint:gosec // seconds
		if each != nil {
			each(t.Seconds)
		}
		g.timer = g.s.loop.After(t.Wait, func() { step(i + 1) })
	}
	step(0)
}

// run is OlympiadGameTask.run: the teleport countdown, then the game.
func (g *olyGame) run() {
	if g.aborted || g.players[0] == nil || g.players[1] == nil {
		g.finish()
		return
	}
	g.countdown(g.s.olympiad.cfg.TeleportCountdown(), olympiad.MsgMovedToStadium, false, nil, g.runGame)
}

// lookup refreshes the nobles from the game's registry (Mobius
// World.getPlayer): nil for one that left.
func (g *olyGame) lookup() {
	for i, id := range g.ids {
		p, ok := g.s.Player(world.ObjectID(id))
		if !ok || !p.InGame() {
			p = nil
		}
		g.players[i] = p
	}
}

// checkDefaulted is OlympiadGameTask.checkDefaulted: each noble must still
// be there, alive, on its base class, without a cursed weapon and under 80
// % of its slots; the other one of a forfeit is told.
func (g *olyGame) checkDefaulted() bool {
	s := g.s
	g.lookup()
	for i, p := range g.players {
		if p != nil {
			s.olyState(p).gameID = g.stadium
		}
		other := g.players[otherSide(i)]
		var msg olympiad.Message
		var item int32
		defaulted := false
		switch {
		case p == nil:
			defaulted = true
		case p.IsDead():
			msg, defaulted = olympiad.MsgDeadNoOlympiad, true
		case p.IsSubClassActive():
			msg, defaulted = olympiad.MsgSubclassRemoved, true
		case p.CursedWeaponEquipped():
			msg, item, defaulted = olympiad.MsgCursedWeaponNoOlympiad, p.CursedWeaponEquippedID(), true
		case float64(p.stats.View().InventoryLimit)*0.8 <= float64(p.inv.Len()):
			msg, defaulted = olympiad.MsgTooMuchStuff, true
		}
		if !defaulted {
			continue
		}
		if p != nil && msg != 0 {
			if item != 0 {
				p.SendSystemMessage(serverpackets.SystemMessageID(msg), s.itemNameParam(item))
			} else {
				p.SendSystemMessage(serverpackets.SystemMessageID(msg))
			}
		}
		if other != nil {
			other.SendSystemMessage(serverpackets.SystemMessageID(olympiad.MsgOpponentDefaulted))
		}
		g.defaulted[i] = true
	}
	return g.defaulted[0] || g.defaulted[1]
}

// runGame is OlympiadGameTask.runGame up to the fight.
func (g *olyGame) runGame() {
	s, f := g.s, &g.s.olympiad
	if g.checkDefaulted() {
		g.finish()
		return
	}
	g.portToArena()
	g.removals()
	if f.cfg.AnnounceGames {
		g.announce()
	}
	g.timer = s.loop.After(olympiad.ArenaWait, func() {
		f.battleStarted = true
		g.countdown(olympiad.StartCountdown(), olympiad.MsgMatchStartsIn, true, func(sec int) {
			if sec == 10 {
				g.damage = [2]int{}
			}
		}, g.startBattle)
	})
}

// crashed reports whether a noble is gone (Mobius _playerOne == null ||
// _playerOneDisconnected).
func (g *olyGame) crashed(i int) bool { return g.players[i] == nil || g.disconnected[i] }

// battleOn is Mobius checkBattleStatus.
func (g *olyGame) battleOn() bool { return !g.crashed(0) && !g.crashed(1) && !g.aborted }

// portToArena is Mobius portPlayersToArena: both stand up, drop their
// target and go to the stadium (ExOlympiadMode 2), sides 1 and 2.
func (g *olyGame) portToArena() {
	s, f := g.s, &g.s.olympiad
	if !g.battleOn() || g.stadium >= len(f.stadiums) {
		g.players = [2]*Player{}
		g.aborted = true
		return
	}
	center := s.olympiadStadiumPos(g.stadium)
	for i, p := range g.players {
		g.origin[i] = p.Position()
		if p.Sitting {
			s.standUp(p)
		}
		s.SetTarget(p, nil)
	}
	g.ported = true
	for i, p := range g.players {
		dx := int32(900)
		if i == 1 {
			dx = -900
		}
		// The mode first: the stadium zone sends anyone else to town.
		st := s.olyState(p)
		st.inMode, st.start, st.side = true, false, i+1
		pos := world.Position{X: center.X + dx, Y: center.Y, Z: center.Z, Heading: p.Position().Heading}
		if err := s.Teleport(p, pos); err != nil {
			s.log.Warn("game: olympiad teleport failed", append(p.logArgs(), "err", err)...)
		}
		p.Send(serverpackets.NewExOlympiadModePacket(serverpackets.OlympiadModeFighter))
	}
	f.gameIsStarted = false
}

// removals is Mobius removals: the buffs, the clan and hero skills, casts,
// a pet, the party and the shots go; the nobles heal fully; skills with a
// reuse up to 15 minutes are ready again.
func (g *olyGame) removals() {
	s := g.s
	if g.aborted || g.players[0] == nil || g.players[1] == nil || g.disconnected[0] || g.disconnected[1] {
		return
	}
	for _, p := range g.players {
		p.Effects().StopAllExceptStayAfterDeath()
		s.takeClanSkills(p)
		p.skill.AbortCast()
		if p.IsHero() {
			s.takeHeroSkills(p) // hero.go
		}
		s.healFully(p)
		if sm := s.SummonOf(p); sm != nil {
			if b := s.skillBaseOf(sm.Npc()); b != nil {
				b.el.StopAll()
			}
			if sm.pet != nil {
				s.Unsummon(p)
			}
		}
		s.LeaveParty(p)
		for _, id := range p.itemSkill.autoShots.list() {
			s.disableAutoShot(p, id)
		}
		s.unchargeShots(p)
		for _, sk := range p.Skills() {
			if sk.ReuseDelay <= 900000 { // milliseconds: 15 minutes
				s.EnableSkill(p, sk)
			}
		}
		p.SendSkillList()
		p.Send(s.SkillCoolTimePacket(p))
	}
}

// healFully sets CP, HP and MP to their maximum.
func (s *Server) healFully(p *Player) {
	v := p.stats.View()
	p.SetCurrentCP(float64(v.MaxCP))
	p.SetCurrentHP(float64(v.MaxHP))
	p.SetCurrentMP(float64(v.MaxMP))
}

// announce is Mobius announceGame: every Grand Olympiad Manager shouts the
// match.
func (g *olyGame) announce() {
	s := g.s
	kind := "class-specific individual match"
	if g.typ == olympiad.NonClassed {
		kind = "class-free individual match"
	}
	text := "Olympiad " + kind + " is going to begin in Arena " + strconv.Itoa(g.stadium+1) + " in a moment."
	for x := range s.world.Objects() {
		n, ok := x.(*Npc)
		if !ok || n.Template == nil || n.Template.ID != olympiadManagerNpc || !n.InWorld() {
			continue
		}
		// Mobius ChatType.NPC_SHOUT is the shout channel.
		s.world.Broadcast(n, serverpackets.NewNpcSayPacket(serverpackets.NpcSay{
			ObjectID: int32(n.ID()), Type: serverpackets.ChatShout, NpcID: int32(n.Template.DisplayID), Text: text, //nolint:gosec // IDs fit
		}), false)
	}
}

// unchargeShots is Item.unChargeAllShots on the weapon.
func (s *Server) unchargeShots(p *Player) {
	for _, t := range []skills.ShotType{skills.Soulshot, skills.Spiritshot, skills.BlessedSpiritshot} {
		setPlayerChargedShot(p, t, false)
	}
}

// startBattle is the end of runGame: the fight starts and is checked every
// 10 s up to Battle.
func (g *olyGame) startBattle() {
	s, f := g.s, &g.s.olympiad
	g.startTime = s.Now()
	if !g.battleOn() {
		g.finish()
		return
	}
	g.broadcast(olympiad.MsgMatchStarted, true)
	f.gameIsStarted = true
	for _, p := range g.players {
		s.olyState(p).start = true
	}
	for i, p := range g.players {
		p.BroadcastUserInfo()
		p.Send(olympiadUserInfo(g.players[otherSide(i)], 1))
	}
	for _, spec := range f.spectators[g.stadium] {
		spec.Send(olympiadUserInfo(g.players[0], 1))
		spec.Send(olympiadUserInfo(g.players[1], 2))
		for _, p := range g.players {
			spec.Send(olympiadSpelledInfo(p))
		}
	}
	var elapsed time.Duration
	var check func()
	check = func() {
		elapsed += olympiad.BattleCheck
		if g.haveWinner() || elapsed >= f.cfg.Battle {
			for _, p := range g.players {
				if p != nil {
					p.BroadcastUserInfo()
				}
			}
			g.finish()
			return
		}
		g.timer = s.loop.After(olympiad.BattleCheck, check)
	}
	g.timer = s.loop.After(olympiad.BattleCheck, check)
}

// haveWinner is Mobius haveWinner: a noble is gone or down to 0 HP.
func (g *olyGame) haveWinner() bool {
	if g.aborted || g.crashed(0) || g.crashed(1) {
		return true
	}
	for _, p := range g.players {
		if p.olympiadGameID() != -1 && p.CurHP <= 0 {
			return true
		}
		if p.IsDead() {
			return true
		}
	}
	return false
}

// handleDisconnect is Mobius handleDisconnect: a noble that leaves after
// reaching the stadium is gone for the result.
func (g *olyGame) handleDisconnect(p *Player) {
	if !g.ported {
		return
	}
	for i, q := range g.players {
		if q == p {
			g.disconnected[i] = true
		}
	}
}

// addDamage is Mobius addDamage: damage to the opponent counts while it is
// not invulnerable.
func (g *olyGame) addDamage(p *Player, damage int) {
	if g.players[0] == nil || g.players[1] == nil {
		return
	}
	for i, q := range g.players {
		other := g.players[otherSide(i)]
		if q == p && !other.skill.IsInvul() {
			g.damage[i] += damage
		}
	}
}

// olympiadHit is the OnHit hook: a fighter's hit on its opponent counts
// (Mobius notifyCompetitorDamage).
func (s *Server) olympiadHit(target, attacker Creature, damage int) {
	a, ok := attacker.(*Player)
	if !ok {
		return
	}
	t, ok := target.(*Player)
	if !ok {
		return
	}
	sa, st := s.olyOf(a), s.olyOf(t)
	if sa == nil || st == nil || !sa.inMode || !st.inMode || sa.gameID != st.gameID || sa.gameID < 0 {
		return
	}
	if g := s.olympiad.games[sa.gameID]; g != nil {
		g.addDamage(a, damage)
	}
}

// finish is the end of OlympiadGameTask.run: the result, then the way back.
func (g *olyGame) finish() {
	s := g.s
	s.cancelTimer(&g.timer)
	res, ok := g.validateWinner()
	if ok && res.Countdown {
		g.countdown(olympiad.BackCountdown(), olympiad.MsgBackToTown, false, nil, g.end)
		return
	}
	g.end()
}

// validateWinner is Mobius validateWinner: the result and its messages,
// the reward and the saved fight. ok is false for an aborted game.
func (g *olyGame) validateWinner() (olympiad.Result, bool) {
	s, f := g.s, &g.s.olympiad
	if g.aborted {
		return olympiad.Result{}, false
	}
	var sides [2]olympiad.Side
	for i, id := range g.ids {
		p := g.players[i]
		n := f.nobles.Get(id)
		if n == nil {
			cls := g.classes[i]
			if p != nil {
				cls = p.ClassID()
			}
			n = f.nobles.Add(f.cfg, id, cls, g.names[i])
		}
		sides[i] = olympiad.Side{Noble: n, Name: g.names[i], Defaulted: g.defaulted[i], Crashed: g.crashed(i)}
		if p != nil {
			sides[i].Dead = p.IsDead()
			sides[i].HP = p.CurHP + p.CurCP
		}
		sides[i].Damage = g.damage[i]
	}
	if !g.defaulted[0] && !g.defaulted[1] && !g.crashed(0) && !g.crashed(1) {
		// Mobius looks the nobles up again: one that relogged is back.
		g.lookup()
		for i, p := range g.players {
			sides[i].Present = p != nil
		}
	}
	res := f.cfg.Validate(g.typ, sides)
	for _, n := range res.Notices {
		var params []serverpackets.SystemMessageParam
		if n.Name != "" || n.Msg == olympiad.MsgYouWin {
			params = append(params, serverpackets.TextParam(n.Name))
		}
		if n.Msg == olympiad.MsgEarnedPoints || n.Msg == olympiad.MsgLostPoints {
			params = append(params, serverpackets.NumberParam(int32(n.Num))) //nolint:gosec // points
		}
		g.broadcast(n.Msg, n.ToAll, params...)
	}
	if res.Fight >= 0 {
		s.saveOlympiadFight(storage.OlympiadFight{
			CharOneID: g.ids[0], CharTwoID: g.ids[1], CharOneName: g.names[0], CharTwoName: g.names[1],
			CharOneClass: int32(g.classes[0]), CharTwoClass: int32(g.classes[1]), //nolint:gosec // class IDs
			Winner: int32(res.Fight), Start: g.startTime.UnixMilli(), //nolint:gosec // 0-2
			Time: s.Now().Sub(g.startTime).Milliseconds(), Classed: g.typ == olympiad.Classed,
		})
	}
	if res.Kind == olympiad.ResultWin && res.Reward > 0 {
		if w := g.players[res.Winner]; w != nil {
			if _, err := s.AddItem(w, int32(f.cfg.BattleRewardItem), int64(res.Reward), -1, true, true); err != nil { //nolint:gosec // an item ID
				s.log.Warn("game: cannot give the olympiad reward", append(w.logArgs(), "err", err)...)
			}
			s.RequestSave(w)
		}
	}
	if f.cfg.LogFights {
		s.log.Info("olympiad fight", "one", g.names[0], "two", g.names[1], "result", res.Kind, "winner", res.Winner,
			"damage", g.damage, "points", res.PointDiff, "classed", g.typ == olympiad.Classed)
	}
	s.saveOlympiad()
	return res, true
}

// end is the rest of OlympiadGameTask.run: the nobles get their state back
// (PlayersStatusBack, cleanEffects) and go home (portPlayersBack), the
// spectators get ExOlympiadMatchEnd and the stadium is free.
func (g *olyGame) end() {
	s, f := g.s, &g.s.olympiad
	g.timer = 0
	g.statusBack()
	g.cleanEffects()
	if g.ported {
		g.ported = false
		for i, p := range g.players {
			if p == nil || !p.InGame() {
				continue
			}
			p.Send(serverpackets.NewExOlympiadMatchEndPacket())
			loc := g.origin[i]
			s.teleportAround(p, data.Location{X: int(loc.X), Y: int(loc.Y), Z: int(loc.Z)}, maxOffsetOnTeleport)
		}
	}
	for _, spec := range f.spectators[g.stadium] {
		spec.Send(serverpackets.NewExOlympiadMatchEndPacket())
	}
	if f.games[g.stadium] == g {
		f.games[g.stadium] = nil
	}
}

// statusBack is Mobius PlayersStatusBack: a noble still in the stadium is
// revived and healed; every noble leaves the olympiad mode
// (ExOlympiadMode 0) and gets its clan and hero skills back.
func (g *olyGame) statusBack() {
	s := g.s
	for _, p := range g.players {
		if p == nil || !p.InGame() {
			continue // gone: Mobius changes the object that left
		}
		if s.inOlympiadStadium(p) {
			if p.IsDead() {
				// Mobius setIsDead(false); the Revive packet keeps the
				// client in step.
				p.Revive()
				p.Send(serverpackets.NewRevivePacket(p.ObjectID()))
			}
			p.StartRegen()
			s.healFully(p)
		}
		st := s.olyState(p)
		st.inMode, st.start, st.side, st.gameID = false, false, -1, -1
		p.Send(serverpackets.NewExOlympiadModePacket(serverpackets.OlympiadModeNone))
		s.giveClanSkills(p)
		if p.IsHero() {
			s.giveHeroSkills(p)
		}
		p.SendSkillList()
	}
}

// cleanEffects is Mobius cleanEffects: the buffs of the fight end.
func (g *olyGame) cleanEffects() {
	s := g.s
	if g.players[0] == nil || g.players[1] == nil || g.disconnected[0] || g.disconnected[1] {
		return
	}
	for _, p := range g.players {
		p.Effects().StopAllExceptStayAfterDeath()
		if sm := s.SummonOf(p); sm != nil {
			if b := s.skillBaseOf(sm.Npc()); b != nil {
				b.el.StopAll()
			}
		}
	}
}

// saveOlympiadFight stores a validated fight.
func (s *Server) saveOlympiadFight(fight storage.OlympiadFight) {
	f := &s.olympiad
	if f.store == nil {
		return
	}
	f.writer.add(f.store, func(ctx context.Context, store storage.OlympiadStore) error {
		return store.AddOlympiadFight(ctx, fight)
	})
}

// olympiadUserInfo builds a fighter's ExOlympiadUserInfo.
func olympiadUserInfo(p *Player, side uint8) []byte {
	v := p.stats.View()
	return serverpackets.NewExOlympiadUserInfoPacket(serverpackets.OlympiadUserInfo{
		Side: side, ObjectID: p.ObjectID(), Name: p.Name(), ClassID: int32(p.ClassID()), //nolint:gosec // class IDs
		HP: javaInt32(p.CurHP), MaxHP: v.MaxHP, CP: javaInt32(p.CurCP), MaxCP: v.MaxCP,
	})
}

// olympiadSpelledInfo builds a fighter's ExOlympiadSpelledInfo with the
// effects on it.
func olympiadSpelledInfo(p *Player) []byte {
	var effects []serverpackets.SpelledEffect
	for _, info := range p.Effects().Effects() {
		sk := info.Skill()
		if sk == nil {
			continue
		}
		effects = append(effects, serverpackets.SpelledEffect{
			SkillID: int32(sk.ID), Level: int16(sk.Level), Seconds: int32(info.Time()), //nolint:gosec // skill IDs and levels are small
		})
	}
	return serverpackets.NewExOlympiadSpelledInfoPacket(p.ObjectID(), effects)
}

// olympiadStatusUpdate is Player.broadcastStatusUpdate's Olympiad part: a
// fighter whose HP or CP changed during the fight is shown to its opponent
// (side 1) and to the spectators (its side).
func (s *Server) olympiadStatusUpdate(p *Player) {
	st := s.olyOf(p)
	if st == nil || !st.inMode || !st.start || st.gameID < 0 {
		return
	}
	g := s.olympiad.games[st.gameID]
	if g == nil {
		return
	}
	for _, q := range g.players {
		if q != nil && q != p {
			q.Send(olympiadUserInfo(p, 1))
		}
	}
	for _, spec := range s.olympiad.spectators[st.gameID] {
		spec.Send(olympiadUserInfo(p, uint8(st.side))) //nolint:gosec // 1 or 2
	}
}
