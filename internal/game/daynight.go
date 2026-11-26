package game

import (
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/skills"
)

// Time of day (Plan 13, A13): what Mobius GameTimeTaskManager does when
// night starts (game hour < 6) or ends, besides the day/night spawns
// (spawn.go): Shadow Sense for Dark Elves, then the scripts'
// on_day_night_change. Mobius never sends SunRise or SunSet; neither do we.
func init() { registerFeature("daynight", setupDayNight) }

const (
	// shadowSenseSkill is Mobius CommonSkill.SHADOW_SENSE, the Dark Elf
	// passive that works at night.
	shadowSenseSkill = 294
	// sysMsgMidnightEffect is "It is now midnight and the effect of $s1 can
	// be felt." (Mobius IT_IS_NOW_MIDNIGHT_AND_THE_EFFECT_OF_S1_CAN_BE_FELT).
	sysMsgMidnightEffect serverpackets.SystemMessageID = 1131
	// sysMsgDawnEffect is "It is dawn and the effect of $s1 will now
	// disappear." (Mobius IT_IS_DAWN_AND_THE_EFFECT_OF_S1_WILL_NOW_DISAPPEAR).
	sysMsgDawnEffect serverpackets.SystemMessageID = 1132
)

// setupDayNight registers the day/night hook. It runs before the spawn
// switch (the npc feature registers later), as in Mobius, where Shadow
// Sense comes first and the spawns and scripts follow on other threads.
func setupDayNight(s *Server) error {
	s.world.GameTime().OnDayNightChange(s.dayNightChanged)
	s.OnPlayerEnter(s.shadowSenseOnEnter)
	return nil
}

// shadowSenseOnEnter is Mobius EnterWorld's Shadow Sense check: a Dark Elf
// who knows it is told whether it works now (the passive's own night
// condition does the rest).
func (s *Server) shadowSenseOnEnter(p *Player) {
	if data.Race(p.Race()) != data.RaceDarkElf { //nolint:gosec // player races are 0-4
		return
	}
	sk := p.KnownSkill(shadowSenseSkill)
	if sk == nil {
		return
	}
	msg := sysMsgDawnEffect
	if s.world.GameTime().IsNight() {
		msg = sysMsgMidnightEffect
	}
	p.SendSystemMessage(msg, serverpackets.SkillNameParam(int32(sk.DisplayID), int32(sk.DisplayLevel))) //nolint:gosec // skill IDs and levels are small
}

// dayNightChanged is Mobius GameTimeTaskManager's change of mode: Shadow
// Sense, then (after this tick's other hooks, the spawn switch among them)
// the scripts' on_day_night_change.
func (s *Server) dayNightChanged(night bool) {
	for p := range s.Players() {
		if p.InGame() {
			s.shadowSense(p, night)
		}
	}
	if s.scripts != nil && s.scripts.Listens("day_night_change", 0) {
		// Mobius notifies it asynchronously: a timer due now runs once this
		// tick's hooks (the spawn switch among them) are done, before the
		// next command or tick.
		s.loop.After(0, func() {
			if s.scripts != nil {
				_ = s.scripts.DayNightChange(s.background(), night) // failures are logged
			}
		})
	}
}

// shadowSense turns Shadow Sense on at night and off at dawn for a Dark Elf
// who knows it, with the message (Mobius: addSkill at night, which starts
// the passive again; stopSkillEffects at dawn).
func (s *Server) shadowSense(p *Player, night bool) {
	if data.Race(p.Race()) != data.RaceDarkElf { //nolint:gosec // player races are 0-4
		return
	}
	sk := p.KnownSkill(shadowSenseSkill)
	if sk == nil {
		return
	}
	name := serverpackets.SkillNameParam(int32(sk.DisplayID), int32(sk.DisplayLevel)) //nolint:gosec // skill IDs and levels are small
	if night {
		p.SendSystemMessage(sysMsgMidnightEffect, name)
		s.AddSkill(p, sk, false)
	} else {
		p.SendSystemMessage(sysMsgDawnEffect, name)
		p.skill.el.StopSkill(skills.FinishNormal, sk.ID)
	}
	p.RefreshStats()
}
