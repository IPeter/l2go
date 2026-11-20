package game

import (
	"slices"
	"time"

	"github.com/IPeter/l2go/internal/config"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/stats"
)

// The PvP flag and karma (decision F13), ported from Mobius Player
// (updatePvPStatus, updatePvPFlag, startPvPFlag, stopPvPFlag, setKarma,
// broadcastKarma, onKillUpdatePvPKarma, increasePvpKills,
// increasePkKillsAndKarma), Playable.checkIfPvP and PvpFlagTaskManager.
//
// The API for the attack and AI features:
//
//   - Server.UpdatePvPStatus(p, target) flags p when it attacks target; the
//     feature registers it as an OnPlayerSwing hook (Mobius onHitTimer).
//     UpdatePvPStatusNoTarget is Mobius's updatePvPStatus() without a
//     target.
//   - Server.UpdatePvPFlag sets the flag value (0, 1, or 2 while it blinks).
//   - Server.SetKarma changes karma; Server.OnKarmaChanged hooks run after
//     every change, so the AI can wake the guards around a player whose
//     karma went from 0 up (Mobius setKarma sets IDLE guards ACTIVE).
//   - Player.Karma, Player.PvPFlag, Player.PvPKills and Player.PKKills.

// pvpFlagScan is the period of Mobius PvpFlagTaskManager.
const pvpFlagScan = time.Second

// pvpFlagBlink is how long before its end the flag blinks (value 2).
const pvpFlagBlink = 20 * time.Second

// KarmaHook runs after a player's karma changed from old to karma.
type KarmaHook func(p *Player, old, karma int32)

// OnKarmaChanged adds fn to the functions run after a player's karma
// changed (Server.SetKarma). The AI uses it to wake the guards that see a
// player whose karma went from 0 up (old == 0, karma > 0), as Mobius
// setKarma does. Call it from a feature's setup function.
func (s *Server) OnKarmaChanged(fn KarmaHook) { s.death.karmaHooks = append(s.death.karmaHooks, fn) }

// pvpConfig is the PvP section of the config, or its defaults for a config
// made in code without one.
func (s *Server) pvpConfig() config.PvP {
	c := s.gs.PvP
	if c.PvPVsNormalTime == 0 && c.PvPVsPvPTime == 0 && c.KarmaRateDrop == 0 && c.MinimumPkRequiredToDrop == 0 {
		return config.DefaultPvP()
	}
	return c
}

// combatSettings is the combat section of the config, or its defaults.
func (s *Server) combatSettings() config.Combat {
	if c := s.gs.Combat; c != (config.Combat{}) {
		return c
	}
	return config.DefaultCombat()
}

// PvPKills returns the player's PvP kill count.
func (p *Player) PvPKills() int32 { return p.char.PvPKills }

// PKKills returns the player's PK count.
func (p *Player) PKKills() int32 { return p.char.PKKills }

// UpdatePvPStatus flags p for attacking target (Mobius
// Player.updatePvPStatus(target)): nothing unless target is another player;
// outside a common PvP zone and against a karma-free target the flag lasts
// PvPVsPvPTime (60 s) from now when target is flagged (checkIfPvP),
// otherwise PvPVsNormalTime (120 s), and starts when p was not flagged.
// The attack feature calls it on the swing. Loop-only.
func (s *Server) UpdatePvPStatus(p *Player, target Creature) {
	t, ok := target.(*Player)
	if !ok || t == p {
		return
	}
	if (!p.Zones.Inside(ZonePvP) || !t.Zones.Inside(ZonePvP)) && t.Karma() == 0 {
		cfg := s.pvpConfig()
		d := time.Duration(cfg.PvPVsNormalTime)
		if checkIfPvP(p, t) {
			d = time.Duration(cfg.PvPVsPvPTime)
		}
		s.death.flagLasts[p] = s.Now().Add(d)
		if p.PvPFlag == 0 {
			s.startPvPFlag(p)
		}
	}
}

// UpdatePvPStatusNoTarget flags p for PvPVsNormalTime outside PvP zones
// (Mobius Player.updatePvPStatus(), for attacks on raids and champions when
// the config asks for it). Loop-only.
func (s *Server) UpdatePvPStatusNoTarget(p *Player) {
	if p.Zones.Inside(ZonePvP) {
		return
	}
	s.death.flagLasts[p] = s.Now().Add(time.Duration(s.pvpConfig().PvPVsNormalTime))
	if p.PvPFlag == 0 {
		s.startPvPFlag(p)
	}
}

// checkIfPvP is Mobius Playable.checkIfPvP: a karma-free p against a
// karma-free, flagged other player.
func checkIfPvP(p *Player, target Creature) bool {
	t, ok := target.(*Player)
	switch {
	case !ok || t == p:
		return false
	case p.Karma() != 0 || t.Karma() != 0:
		return false
	}
	return t.PvPFlag != 0
}

// startPvPFlag sets the flag and adds p to the flag scan (Mobius
// startPvPFlag).
func (s *Server) startPvPFlag(p *Player) {
	s.UpdatePvPFlag(p, 1)
	if !slices.Contains(s.death.flagged, p) {
		s.death.flagged = append(s.death.flagged, p)
	}
}

// stopPvPFlag takes p out of the flag scan and clears the flag (Mobius
// stopPvPFlag).
func (s *Server) stopPvPFlag(p *Player) {
	s.dropFromFlagScan(p)
	s.UpdatePvPFlag(p, 0)
}

func (s *Server) dropFromFlagScan(p *Player) {
	if i := slices.Index(s.death.flagged, p); i >= 0 {
		s.death.flagged = slices.Delete(s.death.flagged, i, i+1)
	}
}

// pvpFlagTick is one run of Mobius PvpFlagTaskManager (every second): a
// flag past its end stops, one within its last 20 s blinks (2), others
// stay 1.
func (s *Server) pvpFlagTick() {
	if len(s.death.flagged) == 0 {
		return
	}
	now := s.Now()
	for _, p := range slices.Clone(s.death.flagged) {
		lasts := s.death.flagLasts[p]
		switch {
		case now.After(lasts):
			s.stopPvPFlag(p)
		case now.After(lasts.Add(-pvpFlagBlink)):
			s.UpdatePvPFlag(p, 2)
		default:
			s.UpdatePvPFlag(p, 1)
		}
	}
}

// UpdatePvPFlag sets p's PvP flag (Mobius updatePvPFlag): when it changes,
// p gets its UserInfo and the players that see p RelationChanged.
// //set_pvp_flag uses it directly. Loop-only.
func (s *Server) UpdatePvPFlag(p *Player, value int32) {
	if p.PvPFlag == value {
		return
	}
	p.PvPFlag = value
	p.UpdateUserInfo()
	s.sendRelations(p)
}

// sendRelations sends RelationChanged about p to every player that sees it
// (Mobius updatePvPFlag, setKarmaFlag and broadcastKarma) and remembers what
// each was told.
func (s *Server) sendRelations(p *Player) {
	if !p.InWorld() {
		return
	}
	for v := range s.world.VisiblePlayers(p) {
		vp, ok := v.(*Player)
		if !ok || !s.world.CanSee(v, p) {
			continue
		}
		r := relation{bits: p.relationTo(vp), autoAttackable: p.autoAttackableBy(vp)}
		p.relations[vp.ID()] = r
		vp.Send(serverpackets.NewRelationChangedPacket(serverpackets.RelationChanged{
			ObjectID:       p.ObjectID(),
			Relation:       r.bits,
			AutoAttackable: r.autoAttackable,
			Karma:          p.char.Karma,
			PvPFlag:        p.PvPFlag,
		}))
	}
}

// SetKarma sets p's karma, at least 0 (Mobius Player.setKarma): losing the
// last karma first sends p its UserInfo and the players around
// RelationChanged (setKarmaFlag, with the old karma, as Mobius does); then
// p gets StatusUpdate with its karma and the players around
// RelationChanged (broadcastKarma). The OnKarmaChanged hooks run last.
// It sends no system message. Loop-only.
func (s *Server) SetKarma(p *Player, karma int32) {
	karma = max(karma, 0)
	old := p.char.Karma
	if old > 0 && karma == 0 {
		p.UpdateUserInfo()
		s.sendRelations(p)
	}
	p.char.Karma = karma
	p.Send(serverpackets.NewStatusUpdatePacket(serverpackets.StatusUpdate{
		ObjectID: p.ObjectID(),
		Values:   []serverpackets.StatusValue{{Attr: serverpackets.StatusKarma, Value: karma}},
	}))
	s.sendRelations(p)
	for _, fn := range s.death.karmaHooks {
		s.safely("OnKarmaChanged hook", func() { fn(p, old, karma) })
	}
}

// onKillUpdatePvPKarma is Mobius Player.onKillUpdatePvPKarma for killer
// having killed the player victim: nothing in PvP zones (or when both are
// in a siege zone); a flagged victim (killer and victim without karma)
// counts as PvP; a victim with karma counts as PvP only with
// AwardPKKillPVPPoint; an unflagged victim without karma is a PK. Clan
// wars, duels, the Olympiad and cursed weapons do not exist yet.
func (s *Server) onKillUpdatePvPKarma(killer, victim *Player) {
	switch {
	case victim == killer:
		return
	case killer.Zones.Inside(ZoneSiege) && victim.Zones.Inside(ZoneSiege):
		return
	case killer.Zones.Inside(ZonePvP) || victim.Zones.Inside(ZonePvP):
		return
	}
	if checkIfPvP(killer, victim) && victim.PvPFlag != 0 {
		s.increasePvPKills(killer)
		return
	}
	switch {
	case victim.Karma() > 0:
		if s.pvpConfig().AwardPKKillPVPPoint {
			s.increasePvPKills(killer)
		}
	case victim.PvPFlag == 0:
		s.increasePKKillsAndKarma(killer)
	}
}

// increasePvPKills is Mobius increasePvpKills (anti-feed and PvP title
// colours off): one more PvP kill and UserInfo.
func (s *Server) increasePvPKills(p *Player) {
	p.char.PvPKills++
	p.UpdateUserInfo()
}

// increasePKKillsAndKarma is Mobius increasePkKillsAndKarma for a killed
// player: the karma of stats.KarmaGain for the PKs so far (720 for the
// first), one more PK and UserInfo.
func (s *Server) increasePKKillsAndKarma(p *Player) {
	gain := stats.KarmaGain(int(p.char.PKKills), false)
	s.SetKarma(p, p.char.Karma+int32(min(gain, 1<<30))) //nolint:gosec // capped
	p.char.PKKills++
	p.UpdateUserInfo()
}

// pvpLeave forgets a player that leaves the game.
func (s *Server) pvpLeave(p *Player) {
	s.dropFromFlagScan(p)
	delete(s.death.flagLasts, p)
}

// pvpFlagTimeLeft returns how long p's flag lasts, for tests and GM info.
func (s *Server) pvpFlagTimeLeft(p *Player) time.Duration {
	if lasts, ok := s.death.flagLasts[p]; ok {
		return lasts.Sub(s.Now())
	}
	return 0
}
